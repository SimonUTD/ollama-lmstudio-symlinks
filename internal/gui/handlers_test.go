package gui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/config"
)

func TestHandleScan_LMStudioToOllama_IncludesUnsupportedDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	lms := filepath.Join(root, "lmstudio-models")
	ollamaDir := filepath.Join(root, "ollama-models")

	okDir := filepath.Join(lms, "providerA", "Model-One")
	if err := os.MkdirAll(okDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(okDir, "one.gguf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write gguf: %v", err)
	}

	unsupportedDir := filepath.Join(lms, "providerA", "Model-Two")
	if err := os.MkdirAll(unsupportedDir, 0o755); err != nil {
		t.Fatalf("mkdir unsupported: %v", err)
	}

	cfg := config.Default()
	cfg.LMStudioModelsDir = lms
	cfg.OllamaModelsDir = ollamaDir

	srv := New(filepath.Join(root, "cfg.json"), cfg)
	h, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	body := bytes.NewBufferString(`{"direction":"lmstudio_to_ollama"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/scan", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d (%s)", http.StatusOK, w.Code, strings.TrimSpace(w.Body.String()))
	}

	var resp scanResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var foundUnsupported bool
	var foundGGUF bool
	for _, it := range resp.Items {
		if it.Status == "unsupported" && it.Label == "providerA/Model-Two" {
			foundUnsupported = true
		}
		if it.GGUFPath != "" && strings.HasSuffix(it.GGUFPath, "one.gguf") {
			foundGGUF = true
		}
	}
	if !foundUnsupported {
		t.Fatalf("expected unsupported item for providerA/Model-Two")
	}
	if !foundGGUF {
		t.Fatalf("expected gguf item to be listed")
	}
}

func TestHandleApply_LMStudioToOllama_RejectsNonGGUF(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	lms := filepath.Join(root, "lmstudio-models")
	if err := os.MkdirAll(lms, 0o755); err != nil {
		t.Fatalf("mkdir lms: %v", err)
	}

	cfg := config.Default()
	cfg.LMStudioModelsDir = lms
	cfg.OllamaModelsDir = filepath.Join(root, "ollama-models")

	srv := New(filepath.Join(root, "cfg.json"), cfg)
	h, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	reqBody := applyRequest{
		Direction: "lmstudio_to_ollama",
		Imports: []lmImport{{
			GGUFPath:  filepath.Join(lms, "providerA", "Model-One", "model.bin"),
			ModelName: "my-model",
		}},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/apply", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d (%s)", http.StatusOK, w.Code, strings.TrimSpace(w.Body.String()))
	}

	var resp applyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(resp.Error, "仅支持 .gguf") {
		t.Fatalf("expected error about .gguf, got: %q", resp.Error)
	}
}

func TestHandleApply_LMStudioToOllama_RejectsOutsideLMStudioDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	lms := filepath.Join(root, "lmstudio-models")
	if err := os.MkdirAll(lms, 0o755); err != nil {
		t.Fatalf("mkdir lms: %v", err)
	}

	outside := filepath.Join(root, "outside.gguf")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	cfg := config.Default()
	cfg.LMStudioModelsDir = lms
	cfg.OllamaModelsDir = filepath.Join(root, "ollama-models")

	srv := New(filepath.Join(root, "cfg.json"), cfg)
	h, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	reqBody := applyRequest{
		Direction: "lmstudio_to_ollama",
		Imports: []lmImport{{
			GGUFPath:  outside,
			ModelName: "my-model",
		}},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/apply", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d (%s)", http.StatusOK, w.Code, strings.TrimSpace(w.Body.String()))
	}

	var resp applyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(resp.Error, "outside lmstudioModelsDir") {
		t.Fatalf("expected error about outside dir, got: %q", resp.Error)
	}
}

func TestHandleApply_LMStudioToOllama_RejectsDuplicateModelName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	lms := filepath.Join(root, "lmstudio-models")

	dir1 := filepath.Join(lms, "providerA", "Model-One")
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p1 := filepath.Join(dir1, "one.gguf")
	if err := os.WriteFile(p1, []byte("x"), 0o644); err != nil {
		t.Fatalf("write gguf1: %v", err)
	}

	dir2 := filepath.Join(lms, "providerA", "Model-Two")
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p2 := filepath.Join(dir2, "two.gguf")
	if err := os.WriteFile(p2, []byte("x"), 0o644); err != nil {
		t.Fatalf("write gguf2: %v", err)
	}

	cfg := config.Default()
	cfg.LMStudioModelsDir = lms
	cfg.OllamaModelsDir = filepath.Join(root, "ollama-models")

	srv := New(filepath.Join(root, "cfg.json"), cfg)
	h, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	reqBody := applyRequest{
		Direction: "lmstudio_to_ollama",
		Imports: []lmImport{
			{GGUFPath: p1, ModelName: "dup"},
			{GGUFPath: p2, ModelName: "dup"},
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/apply", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d (%s)", http.StatusOK, w.Code, strings.TrimSpace(w.Body.String()))
	}

	var resp applyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(resp.Error, "重复的 modelName") {
		t.Fatalf("expected duplicate modelName error, got: %q", resp.Error)
	}
}
