package gui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
