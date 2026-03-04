package symlink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectTarget_Missing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "missing-link")
	status, err := InspectTarget(target, filepath.Join(root, "src.gguf"))
	if err != nil {
		t.Fatalf("InspectTarget: %v", err)
	}
	if status.Kind != KindMissing {
		t.Fatalf("expected KindMissing, got %v", status.Kind)
	}
}

func TestInspectTarget_File(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "file.gguf")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	status, err := InspectTarget(target, filepath.Join(root, "src.gguf"))
	if err != nil {
		t.Fatalf("InspectTarget: %v", err)
	}
	if status.Kind != KindFile {
		t.Fatalf("expected KindFile, got %v", status.Kind)
	}
}

func TestInspectTarget_SymlinkCorrect(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "src.gguf")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	target := filepath.Join(root, "target.gguf")
	if err := os.Symlink(src, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	status, err := InspectTarget(target, src)
	if err != nil {
		t.Fatalf("InspectTarget: %v", err)
	}
	if status.Kind != KindSymlink {
		t.Fatalf("expected KindSymlink, got %v", status.Kind)
	}
	if !status.IsSymlinkMatch {
		t.Fatalf("expected symlink to match expected source")
	}
	if status.IsSymlinkBroken {
		t.Fatalf("expected symlink to not be broken")
	}
}

func TestInspectTarget_SymlinkMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	expected := filepath.Join(root, "expected.gguf")
	actual := filepath.Join(root, "actual.gguf")
	if err := os.WriteFile(expected, []byte("x"), 0o644); err != nil {
		t.Fatalf("write expected: %v", err)
	}
	if err := os.WriteFile(actual, []byte("y"), 0o644); err != nil {
		t.Fatalf("write actual: %v", err)
	}

	target := filepath.Join(root, "target.gguf")
	if err := os.Symlink(actual, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	status, err := InspectTarget(target, expected)
	if err != nil {
		t.Fatalf("InspectTarget: %v", err)
	}
	if status.Kind != KindSymlink {
		t.Fatalf("expected KindSymlink, got %v", status.Kind)
	}
	if status.IsSymlinkMatch {
		t.Fatalf("expected symlink to mismatch expected source")
	}
	if status.IsSymlinkBroken {
		t.Fatalf("expected symlink to not be broken")
	}
}

func TestInspectTarget_SymlinkBroken(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	expected := filepath.Join(root, "missing.gguf")
	target := filepath.Join(root, "target.gguf")
	if err := os.Symlink(expected, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	status, err := InspectTarget(target, expected)
	if err != nil {
		t.Fatalf("InspectTarget: %v", err)
	}
	if status.Kind != KindSymlink {
		t.Fatalf("expected KindSymlink, got %v", status.Kind)
	}
	if !status.IsSymlinkMatch {
		t.Fatalf("expected broken symlink to still match expected source path")
	}
	if !status.IsSymlinkBroken {
		t.Fatalf("expected symlink to be broken")
	}
}
