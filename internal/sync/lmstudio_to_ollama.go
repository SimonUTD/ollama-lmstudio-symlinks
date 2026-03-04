package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollama"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/symlink"
)

const defaultOllamaTag = "latest"

type OllamaRunner interface {
	CreateFromGGUF(ctx context.Context, modelName string, ggufPath string) error
}

type LMStudioToOllamaSpec struct {
	ModelName string
	GGUFPath  string
}

type ApplyLMStudioToOllamaOptions struct {
	AllowReplaceExistingBlob bool
}

type ApplyLMStudioToOllamaResult struct {
	ModelName string
	Note      string
}

func ApplyLMStudioToOllama(ctx context.Context, specs []LMStudioToOllamaSpec, ollamaModelsDir string, runner OllamaRunner, opts ApplyLMStudioToOllamaOptions) ([]ApplyLMStudioToOllamaResult, error) {
	existingBlobs, err := snapshotBlobFiles(ollamaModelsDir)
	if err != nil {
		return nil, err
	}

	var results []ApplyLMStudioToOllamaResult
	var errs []error
	for _, spec := range specs {
		r, err := applyOneLMStudioToOllama(ctx, spec, ollamaModelsDir, runner, existingBlobs, opts)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", spec.ModelName, err))
			continue
		}
		results = append(results, r)
	}
	return results, errors.Join(errs...)
}

func applyOneLMStudioToOllama(ctx context.Context, spec LMStudioToOllamaSpec, ollamaModelsDir string, runner OllamaRunner, existingBlobs map[string]bool, opts ApplyLMStudioToOllamaOptions) (ApplyLMStudioToOllamaResult, error) {
	ref := parseModelName(spec.ModelName)

	models, err := ollama.DiscoverModels(ollamaModelsDir)
	if err != nil {
		return ApplyLMStudioToOllamaResult{}, err
	}
	model, ok := findModel(models, ref.Repository, ref.Tag)
	if ok {
		alreadyLinked, err := isOllamaModelLinkedToGGUF(ollamaModelsDir, model, spec.GGUFPath)
		if err != nil {
			return ApplyLMStudioToOllamaResult{}, err
		}
		if alreadyLinked {
			return ApplyLMStudioToOllamaResult{ModelName: spec.ModelName, Note: "already linked"}, nil
		}
		return ApplyLMStudioToOllamaResult{}, fmt.Errorf("model exists but is not linked to the selected gguf")
	}

	if err := runner.CreateFromGGUF(ctx, spec.ModelName, spec.GGUFPath); err != nil {
		return ApplyLMStudioToOllamaResult{}, err
	}

	updatedModels, err := ollama.DiscoverModels(ollamaModelsDir)
	if err != nil {
		return ApplyLMStudioToOllamaResult{}, err
	}
	created, ok := findModel(updatedModels, ref.Repository, ref.Tag)
	if !ok {
		return ApplyLMStudioToOllamaResult{}, fmt.Errorf("ollama create succeeded but manifest was not found for %s:%s", ref.Repository, ref.Tag)
	}

	blobFilename := strings.Replace(created.ModelLayerDigest, ":", "-", 1)
	blobPath := filepath.Join(ollamaModelsDir, "blobs", blobFilename)
	if existingBlobs[blobFilename] && !opts.AllowReplaceExistingBlob {
		return ApplyLMStudioToOllamaResult{
			ModelName: spec.ModelName,
			Note:      "blob already existed before sync; left as-is to avoid impacting other models",
		}, nil
	}

	if err := replaceFileWithSymlink(blobPath, spec.GGUFPath); err != nil {
		return ApplyLMStudioToOllamaResult{}, err
	}
	existingBlobs[blobFilename] = true

	return ApplyLMStudioToOllamaResult{ModelName: spec.ModelName, Note: "created and linked"}, nil
}

type modelRef struct {
	Repository string
	Tag        string
}

func parseModelName(name string) modelRef {
	name = strings.TrimSpace(name)
	if name == "" {
		return modelRef{Repository: "", Tag: defaultOllamaTag}
	}

	lastColon := strings.LastIndex(name, ":")
	if lastColon <= 0 || lastColon >= len(name)-1 {
		return modelRef{Repository: name, Tag: defaultOllamaTag}
	}
	return modelRef{
		Repository: name[:lastColon],
		Tag:        name[lastColon+1:],
	}
}

func findModel(models []ollama.DiscoveredModel, repo string, tag string) (ollama.DiscoveredModel, bool) {
	for _, m := range models {
		if m.ID.Repository == repo && m.ID.Tag == tag {
			return m, true
		}
	}
	return ollama.DiscoveredModel{}, false
}

func isOllamaModelLinkedToGGUF(ollamaModelsDir string, model ollama.DiscoveredModel, ggufPath string) (bool, error) {
	blobPath := blobPathFromDigest(ollamaModelsDir, model.ModelLayerDigest)
	status, err := symlink.InspectTarget(blobPath, ggufPath)
	if err != nil {
		return false, err
	}
	return status.Kind == symlink.KindSymlink && status.IsSymlinkMatch && !status.IsSymlinkBroken, nil
}

func snapshotBlobFiles(ollamaModelsDir string) (map[string]bool, error) {
	dir := filepath.Join(ollamaModelsDir, "blobs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}

	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[e.Name()] = true
	}
	return set, nil
}

func replaceFileWithSymlink(targetPath string, sourcePath string) error {
	sourceAbs, err := absClean(sourcePath)
	if err != nil {
		return err
	}

	if err := os.Remove(targetPath); err != nil {
		return err
	}
	return os.Symlink(sourceAbs, targetPath)
}
