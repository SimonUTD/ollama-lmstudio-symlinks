package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	cfg.OllamaBin = "/Applications/Ollama.app/Contents/Resources/ollama"
	cfg.OllamaHost = "127.0.0.1:11434"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.OllamaBin != cfg.OllamaBin {
		t.Fatalf("OllamaBin: expected %q, got %q", cfg.OllamaBin, loaded.OllamaBin)
	}
	if loaded.OllamaHost != cfg.OllamaHost {
		t.Fatalf("OllamaHost: expected %q, got %q", cfg.OllamaHost, loaded.OllamaHost)
	}
}
