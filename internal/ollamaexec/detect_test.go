package ollamaexec

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCheckServer_Reachable(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	st, err := CheckServer(ln.Addr().String(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("CheckServer: %v", err)
	}
	if !st.Reachable {
		t.Fatalf("expected reachable, got false (error=%q)", st.Error)
	}
}

func TestDetectBinary_ConfiguredPathWins(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	name := "ollama"
	if runtime.GOOS == "windows" {
		name = "ollama.exe"
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	det, err := DetectBinary(context.Background(), p)
	if err != nil {
		t.Fatalf("DetectBinary: %v", err)
	}
	if !det.Found {
		t.Fatalf("expected Found=true")
	}
	if det.Source != "config" {
		t.Fatalf("expected Source=config, got %q", det.Source)
	}
	if det.Path != p {
		t.Fatalf("expected Path=%q, got %q", p, det.Path)
	}
}
