package lmstudio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverGGUFFiles_CollectsAndMarksPrimaryBySize(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modelsDir := filepath.Join(root, "models")
	modelDir := filepath.Join(modelsDir, "providerA", "Model-One")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	small := filepath.Join(modelDir, "small.gguf")
	large := filepath.Join(modelDir, "large.gguf")
	if err := os.WriteFile(small, []byte("1"), 0o644); err != nil {
		t.Fatalf("write small: %v", err)
	}
	if err := os.WriteFile(large, []byte(strings.Repeat("x", 10)), 0o644); err != nil {
		t.Fatalf("write large: %v", err)
	}

	files, err := DiscoverGGUFFiles(modelsDir)
	if err != nil {
		t.Fatalf("DiscoverGGUFFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 gguf files, got %d", len(files))
	}

	var foundLargePrimary bool
	var foundSmallPrimary bool
	for _, f := range files {
		if f.Provider != "providerA" {
			t.Fatalf("provider: expected providerA, got %q", f.Provider)
		}
		if f.ModelDir != "Model-One" {
			t.Fatalf("model dir: expected Model-One, got %q", f.ModelDir)
		}
		switch f.FileName {
		case "large.gguf":
			foundLargePrimary = f.IsPrimary
		case "small.gguf":
			foundSmallPrimary = f.IsPrimary
		default:
			t.Fatalf("unexpected file: %q", f.FileName)
		}
	}

	if !foundLargePrimary {
		t.Fatalf("expected large.gguf to be primary")
	}
	if foundSmallPrimary {
		t.Fatalf("expected small.gguf to not be primary")
	}
}
