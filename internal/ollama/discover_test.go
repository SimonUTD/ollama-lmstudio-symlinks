package ollama

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverModels_ParsesRepoAndTagFromPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "models")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"llama3",
		"8b",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "sha256:aaaaaaaa", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	models, err := DiscoverModels(ollamaModelsDir)
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	if models[0].ID.Repository != "llama3" {
		t.Fatalf("repo: expected llama3, got %q", models[0].ID.Repository)
	}
	if models[0].ID.Tag != "8b" {
		t.Fatalf("tag: expected 8b, got %q", models[0].ID.Tag)
	}
	if models[0].ModelLayerDigest != "sha256:aaaaaaaa" {
		t.Fatalf("digest: expected sha256:aaaaaaaa, got %q", models[0].ModelLayerDigest)
	}
}

func TestDiscoverModels_SupportsNestedRepository(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "models")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"myuser",
		"my-model",
		"latest",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "sha256:bbbbbbbb", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	models, err := DiscoverModels(ollamaModelsDir)
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	if models[0].ID.Repository != "myuser/my-model" {
		t.Fatalf("repo: expected myuser/my-model, got %q", models[0].ID.Repository)
	}
	if models[0].ID.Tag != "latest" {
		t.Fatalf("tag: expected latest, got %q", models[0].ID.Tag)
	}
}

func TestDiscoverModels_IgnoresManifestWithoutModelLayer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "models")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"no-model-layer",
		"latest",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.projector", "digest": "sha256:cccccccc", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	models, err := DiscoverModels(ollamaModelsDir)
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expected 0 models, got %d", len(models))
	}
}
