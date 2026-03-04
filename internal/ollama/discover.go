package ollama

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	manifestsDirName     = "manifests"
	modelLayerMediaType  = "application/vnd.ollama.image.model"
	projectorLayerType   = "application/vnd.ollama.image.projector"
	minManifestPathParts = 4 // registry/namespace/repo/tag
)

type ModelID struct {
	Registry   string
	Namespace  string
	Repository string
	Tag        string
}

type DiscoveredModel struct {
	ID               ModelID
	ManifestPath     string
	ModelLayerDigest string
	ProjectorDigests []string
}

type manifestFile struct {
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

func DiscoverModels(ollamaModelsDir string) ([]DiscoveredModel, error) {
	manifestsDir := filepath.Join(ollamaModelsDir, manifestsDirName)
	if _, err := os.Stat(manifestsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var models []DiscoveredModel
	var errs []error

	walkErr := filepath.WalkDir(manifestsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		model, err := discoverOneManifest(manifestsDir, path)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if model == nil {
			return nil
		}

		models = append(models, *model)
		return nil
	})

	if walkErr != nil {
		errs = append(errs, walkErr)
	}
	return models, errors.Join(errs...)
}

func discoverOneManifest(manifestsDir, manifestPath string) (*DiscoveredModel, error) {
	id, ok := modelIDFromManifestPath(manifestsDir, manifestPath)
	if !ok {
		return nil, nil
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var mf manifestFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, err
	}

	modelDigest, projectorDigests := digestsFromLayers(mf.Layers)
	if modelDigest == "" {
		return nil, nil
	}

	return &DiscoveredModel{
		ID:               id,
		ManifestPath:     manifestPath,
		ModelLayerDigest: modelDigest,
		ProjectorDigests: projectorDigests,
	}, nil
}

func modelIDFromManifestPath(manifestsDir, manifestPath string) (ModelID, bool) {
	rel, err := filepath.Rel(manifestsDir, manifestPath)
	if err != nil {
		return ModelID{}, false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < minManifestPathParts {
		return ModelID{}, false
	}

	repoParts := parts[2 : len(parts)-1]
	if len(repoParts) == 0 {
		return ModelID{}, false
	}

	return ModelID{
		Registry:   parts[0],
		Namespace:  parts[1],
		Repository: strings.Join(repoParts, "/"),
		Tag:        parts[len(parts)-1],
	}, true
}

func digestsFromLayers(layers []struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
}) (string, []string) {
	var modelDigest string
	var projectors []string

	for _, layer := range layers {
		switch layer.MediaType {
		case modelLayerMediaType:
			if modelDigest == "" {
				modelDigest = layer.Digest
			}
		case projectorLayerType:
			projectors = append(projectors, layer.Digest)
		}
	}

	return modelDigest, projectors
}
