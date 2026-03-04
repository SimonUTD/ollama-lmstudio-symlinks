package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollama"
)

const lmStudioOllamaProviderDir = "ollama"

type ApplyOllamaToLMStudioOptions struct {
	Symlink EnsureSymlinkOptions
}

type ApplyOllamaToLMStudioResult struct {
	ModelName   string
	LinkResults []EnsureSymlinkResult
}

func ApplyOllamaToLMStudio(models []ollama.DiscoveredModel, ollamaModelsDir string, lmstudioModelsDir string, opts ApplyOllamaToLMStudioOptions) ([]ApplyOllamaToLMStudioResult, error) {
	providerDir := filepath.Join(lmstudioModelsDir, lmStudioOllamaProviderDir)
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		return nil, err
	}

	var results []ApplyOllamaToLMStudioResult
	var errs []error
	for _, m := range models {
		r, err := applyOneOllamaToLMStudio(m, ollamaModelsDir, providerDir, opts)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ollama.RepoForCLI(m.ID), err))
			continue
		}
		results = append(results, r)
	}
	return results, errors.Join(errs...)
}

func applyOneOllamaToLMStudio(model ollama.DiscoveredModel, ollamaModelsDir string, providerDir string, opts ApplyOllamaToLMStudioOptions) (ApplyOllamaToLMStudioResult, error) {
	name := lmStudioSafeNameFromOllama(model.ID)
	modelDir := filepath.Join(providerDir, name)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return ApplyOllamaToLMStudioResult{}, err
	}

	mainSource := blobPathFromDigest(ollamaModelsDir, model.ModelLayerDigest)
	mainTarget := filepath.Join(modelDir, name+".gguf")
	mainResult, err := EnsureSymlink(mainTarget, mainSource, opts.Symlink)
	if err != nil {
		return ApplyOllamaToLMStudioResult{}, err
	}

	linkResults := []EnsureSymlinkResult{mainResult}
	projectorSpecs := projectorTargets(name, model.ProjectorDigests)
	for _, spec := range projectorSpecs {
		res, err := EnsureSymlink(filepath.Join(modelDir, spec.TargetFileName), blobPathFromDigest(ollamaModelsDir, spec.Digest), opts.Symlink)
		if err != nil {
			return ApplyOllamaToLMStudioResult{}, err
		}
		linkResults = append(linkResults, res)
	}

	return ApplyOllamaToLMStudioResult{
		ModelName:   name,
		LinkResults: linkResults,
	}, nil
}

func lmStudioSafeNameFromOllama(id ollama.ModelID) string {
	repo := strings.ReplaceAll(ollama.RepoForCLI(id), "/", "-")
	if id.Tag == "" {
		return repo
	}
	return repo + "-" + id.Tag
}

func blobPathFromDigest(ollamaModelsDir string, digest string) string {
	return filepath.Join(ollamaModelsDir, "blobs", strings.Replace(digest, ":", "-", 1))
}

type projectorSpec struct {
	Digest         string
	TargetFileName string
}

func projectorTargets(baseName string, digests []string) []projectorSpec {
	if len(digests) == 0 {
		return nil
	}
	if len(digests) == 1 {
		return []projectorSpec{{
			Digest:         digests[0],
			TargetFileName: baseName + "-projector.bin",
		}}
	}

	specs := make([]projectorSpec, 0, len(digests))
	for i, digest := range digests {
		specs = append(specs, projectorSpec{
			Digest:         digest,
			TargetFileName: baseName + "-projector-" + strconv.Itoa(i+1) + ".bin",
		})
	}
	return specs
}
