package gui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/config"
)

func TestReadJSON_RejectsNonJSONContentType(t *testing.T) {
	t.Parallel()

	var v scanRequest
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/scan", strings.NewReader(`{"direction":"ollama_to_lmstudio"}`))
	r.Header.Set("Content-Type", "text/plain")

	if err := readJSON(w, r, &v); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestReadJSON_AllowsJSONContentType(t *testing.T) {
	t.Parallel()

	var v scanRequest
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/scan", strings.NewReader(`{"direction":"ollama_to_lmstudio"}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	if err := readJSON(w, r, &v); err != nil {
		t.Fatalf("readJSON: %v", err)
	}
	if v.Direction != "ollama_to_lmstudio" {
		t.Fatalf("Direction: expected %q, got %q", "ollama_to_lmstudio", v.Direction)
	}
}

func TestServer_RootServesIndexHTML(t *testing.T) {
	t.Parallel()

	srv := New("unused.json", config.Default())
	h, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:48289/", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<title>Ollama ↔ LM Studio 同步</title>") {
		t.Fatalf("expected HTML title to be present")
	}
	if !strings.Contains(body, `href="/styles.css"`) {
		t.Fatalf("expected styles.css href to be present")
	}
	if !strings.Contains(body, `src="/app.js"`) {
		t.Fatalf("expected app.js src to be present")
	}

	assetPaths := []string{"/styles.css", "/app.js"}
	for _, p := range assetPaths {
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:48289"+p, nil)
		h.ServeHTTP(w2, r2)
		if w2.Code != http.StatusOK {
			t.Fatalf("%s: expected %d, got %d", p, http.StatusOK, w2.Code)
		}
	}
}
