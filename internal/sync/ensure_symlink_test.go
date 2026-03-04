package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSymlink_CreatesWhenMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "src.gguf")
	target := filepath.Join(root, "dst.gguf")

	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	result, err := EnsureSymlink(target, source, EnsureSymlinkOptions{})
	if err != nil {
		t.Fatalf("EnsureSymlink: %v", err)
	}
	if result.Kind != EnsureSymlinkCreated {
		t.Fatalf("expected created, got %v", result.Kind)
	}

	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if linkTarget != source {
		t.Fatalf("expected link to %q, got %q", source, linkTarget)
	}
}

func TestEnsureSymlink_SkipsWhenAlreadyCorrect(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "src.gguf")
	target := filepath.Join(root, "dst.gguf")

	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	result, err := EnsureSymlink(target, source, EnsureSymlinkOptions{})
	if err != nil {
		t.Fatalf("EnsureSymlink: %v", err)
	}
	if result.Kind != EnsureSymlinkSkippedAlreadySynced {
		t.Fatalf("expected skipped already synced, got %v", result.Kind)
	}
}

func TestEnsureSymlink_FailsOnMismatchWithoutFix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "src.gguf")
	other := filepath.Join(root, "other.gguf")
	target := filepath.Join(root, "dst.gguf")

	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(other, []byte("y"), 0o644); err != nil {
		t.Fatalf("write other: %v", err)
	}
	if err := os.Symlink(other, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := EnsureSymlink(target, source, EnsureSymlinkOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEnsureSymlink_FixesMismatchWhenAllowed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "src.gguf")
	other := filepath.Join(root, "other.gguf")
	target := filepath.Join(root, "dst.gguf")

	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(other, []byte("y"), 0o644); err != nil {
		t.Fatalf("write other: %v", err)
	}
	if err := os.Symlink(other, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	result, err := EnsureSymlink(target, source, EnsureSymlinkOptions{
		AllowFixSymlink: true,
	})
	if err != nil {
		t.Fatalf("EnsureSymlink: %v", err)
	}
	if result.Kind != EnsureSymlinkFixed {
		t.Fatalf("expected fixed, got %v", result.Kind)
	}

	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if linkTarget != source {
		t.Fatalf("expected link to %q, got %q", source, linkTarget)
	}
}

func TestEnsureSymlink_FailsOnRegularFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "src.gguf")
	target := filepath.Join(root, "dst.gguf")

	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(target, []byte("occupied"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	_, err := EnsureSymlink(target, source, EnsureSymlinkOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
