package gui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/config"
)

const (
	maxBodyBytes      = 1 << 20
	serverDialTimeout = 500 * time.Millisecond
)

type Server struct {
	cfgPath string
	mu      sync.RWMutex
	cfg     config.Config
}

func New(cfgPath string, cfg config.Config) *Server {
	return &Server{cfgPath: cfgPath, cfg: cfg}
}

func (s *Server) Handler() (http.Handler, error) {
	sub, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/apply", s.handleApply)

	mux.Handle("/", fileServer)

	return mux, nil
}

func (s *Server) Serve(ctx context.Context, addr string) error {
	h, err := s.Handler()
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    addr,
		Handler: h,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.HasPrefix(ct, "application/json") {
		return fmt.Errorf("Content-Type 必须是 application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, apiError{Error: err.Error()})
}

func (s *Server) cfgSnapshot() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Server) updateConfig(cfg config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func drainBody(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cfgSnapshot())
	case http.MethodPut:
		var cfg config.Config
		if err := readJSON(w, r, &cfg); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := config.Save(s.cfgPath, cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		loaded, err := config.Load(s.cfgPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.updateConfig(loaded)
		writeJSON(w, http.StatusOK, loaded)
	default:
		drainBody(r)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
	}
}
