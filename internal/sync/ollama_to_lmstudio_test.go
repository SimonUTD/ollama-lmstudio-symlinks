package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollama"
)

func TestApplyOllamaToLMStudio_CreatesSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	lmstudioModelsDir := filepath.Join(root, "lmstudio")

	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-aaaaaaaa")
	if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
		t.Fatalf("mkdir blobs: %v", err)
	}
	if err := os.WriteFile(blobFile, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}

	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"llama3",
		"8b",
	)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifests: %v", err)
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

	models, err := ollama.DiscoverModels(ollamaModelsDir)
	if err != nil {
		t.Fatalf("discover ollama: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	results, err := ApplyOllamaToLMStudio(models, ollamaModelsDir, lmstudioModelsDir, ApplyOllamaToLMStudioOptions{})
	if err != nil {
		t.Fatalf("ApplyOllamaToLMStudio: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	target := filepath.Join(lmstudioModelsDir, "ollama", "llama3-8b", "llama3-8b.gguf")
	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("readlink target: %v", err)
	}
	if linkTarget != blobFile {
		t.Fatalf("expected symlink to %q, got %q", blobFile, linkTarget)
	}
}
