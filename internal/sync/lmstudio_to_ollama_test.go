package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeOllamaRunner struct {
	createCalls int
	onCreate    func(modelName string, ggufPath string) error
}

func (f *fakeOllamaRunner) CreateFromGGUF(ctx context.Context, modelName string, ggufPath string) error {
	f.createCalls++
	if f.onCreate == nil {
		return nil
	}
	return f.onCreate(modelName, ggufPath)
}

func TestApplyLMStudioToOllama_CreatesModelAndReplacesBlobWithSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	gguf := filepath.Join(root, "model.gguf")
	if err := os.WriteFile(gguf, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	const digest = "sha256:dddddddd"
	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-dddddddd")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"my-model",
		"latest",
	)

	runner := &fakeOllamaRunner{
		onCreate: func(modelName string, ggufPath string) error {
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
				return err
			}
			const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "` + digest + `", "size": 123 }
  ]
}`
			if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
				return err
			}
			return os.WriteFile(blobFile, []byte("copied"), 0o644)
		},
	}

	specs := []LMStudioToOllamaSpec{{
		ModelName: "my-model",
		GGUFPath:  gguf,
	}}

	results, err := ApplyLMStudioToOllama(context.Background(), specs, ollamaModelsDir, runner, ApplyLMStudioToOllamaOptions{})
	if err != nil {
		t.Fatalf("ApplyLMStudioToOllama: %v", err)
	}
	if runner.createCalls != 1 {
		t.Fatalf("expected createCalls=1, got %d", runner.createCalls)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	target, err := os.Readlink(blobFile)
	if err != nil {
		t.Fatalf("blob is not symlink: %v", err)
	}
	if target != gguf {
		t.Fatalf("expected blob symlink to %q, got %q", gguf, target)
	}
}

func TestApplyLMStudioToOllama_SkipsWhenAlreadyLinked(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	gguf := filepath.Join(root, "model.gguf")
	if err := os.WriteFile(gguf, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	const digest = "sha256:eeeeeeee"
	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-eeeeeeee")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"already",
		"latest",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest: %v", err)
	}
	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "` + digest + `", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
		t.Fatalf("mkdir blob: %v", err)
	}
	if err := os.Symlink(gguf, blobFile); err != nil {
		t.Fatalf("symlink blob: %v", err)
	}

	runner := &fakeOllamaRunner{}
	specs := []LMStudioToOllamaSpec{{
		ModelName: "already",
		GGUFPath:  gguf,
	}}

	_, err := ApplyLMStudioToOllama(context.Background(), specs, ollamaModelsDir, runner, ApplyLMStudioToOllamaOptions{})
	if err != nil {
		t.Fatalf("ApplyLMStudioToOllama: %v", err)
	}
	if runner.createCalls != 0 {
		t.Fatalf("expected createCalls=0, got %d", runner.createCalls)
	}
}

func TestApplyLMStudioToOllama_FailsWhenModelExistsButBlobIsRegularFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	gguf := filepath.Join(root, "model.gguf")
	if err := os.WriteFile(gguf, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	const digest = "sha256:ffffffff"
	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-ffffffff")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"conflict",
		"latest",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest: %v", err)
	}
	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "` + digest + `", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
		t.Fatalf("mkdir blob: %v", err)
	}
	if err := os.WriteFile(blobFile, []byte("real file"), 0o644); err != nil {
		t.Fatalf("write blob file: %v", err)
	}

	runner := &fakeOllamaRunner{}
	specs := []LMStudioToOllamaSpec{{
		ModelName: "conflict",
		GGUFPath:  gguf,
	}}

	_, err := ApplyLMStudioToOllama(context.Background(), specs, ollamaModelsDir, runner, ApplyLMStudioToOllamaOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestApplyLMStudioToOllama_SupportsNamespaceModelsInManifestsDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	gguf := filepath.Join(root, "model.gguf")
	if err := os.WriteFile(gguf, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	const digest = "sha256:abababab"
	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-abababab")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"teichai",
		"glm-4.7-flash",
		"latest",
	)

	runner := &fakeOllamaRunner{
		onCreate: func(modelName string, ggufPath string) error {
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
				return err
			}
			const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "` + digest + `", "size": 123 }
  ]
}`
			if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
				return err
			}
			return os.WriteFile(blobFile, []byte("copied"), 0o644)
		},
	}

	specs := []LMStudioToOllamaSpec{{
		ModelName: "teichai/glm-4.7-flash",
		GGUFPath:  gguf,
	}}

	_, err := ApplyLMStudioToOllama(context.Background(), specs, ollamaModelsDir, runner, ApplyLMStudioToOllamaOptions{})
	if err != nil {
		t.Fatalf("ApplyLMStudioToOllama: %v", err)
	}

	target, err := os.Readlink(blobFile)
	if err != nil {
		t.Fatalf("blob is not symlink: %v", err)
	}
	if target != gguf {
		t.Fatalf("expected blob symlink to %q, got %q", gguf, target)
	}
}

func TestApplyLMStudioToOllama_RelinksExistingModelWhenAllowed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ollamaModelsDir := filepath.Join(root, "ollama")
	gguf := filepath.Join(root, "model.gguf")
	if err := os.WriteFile(gguf, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	const digest = "sha256:cdcdcdcd"
	blobFile := filepath.Join(ollamaModelsDir, "blobs", "sha256-cdcdcdcd")
	manifestPath := filepath.Join(
		ollamaModelsDir,
		"manifests",
		"registry.ollama.ai",
		"library",
		"exists",
		"latest",
	)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest: %v", err)
	}
	const manifestJSON = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "mediaType": "application/vnd.ollama.image.config", "digest": "sha256:cfg", "size": 1 },
  "layers": [
    { "mediaType": "application/vnd.ollama.image.model", "digest": "` + digest + `", "size": 123 }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(blobFile), 0o755); err != nil {
		t.Fatalf("mkdir blob: %v", err)
	}
	if err := os.WriteFile(blobFile, []byte("copied"), 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}

	runner := &fakeOllamaRunner{}
	specs := []LMStudioToOllamaSpec{{
		ModelName: "exists",
		GGUFPath:  gguf,
	}}

	_, err := ApplyLMStudioToOllama(context.Background(), specs, ollamaModelsDir, runner, ApplyLMStudioToOllamaOptions{
		AllowReplaceExistingBlob: true,
	})
	if err != nil {
		t.Fatalf("ApplyLMStudioToOllama: %v", err)
	}
	if runner.createCalls != 0 {
		t.Fatalf("expected createCalls=0, got %d", runner.createCalls)
	}

	target, err := os.Readlink(blobFile)
	if err != nil {
		t.Fatalf("blob is not symlink: %v", err)
	}
	if target != gguf {
		t.Fatalf("expected blob symlink to %q, got %q", gguf, target)
	}
}
