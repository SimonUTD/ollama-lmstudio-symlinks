package gui

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/config"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollama"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollamaexec"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/symlink"
	modelsync "github.com/SimonUTD/ollama-lmstudio-symlinks/internal/sync"
)

const (
	dirOllamaToLMStudio = "ollama_to_lmstudio"
	dirLMStudioToOllama = "lmstudio_to_ollama"
	binaryDetectTimeout = 2 * time.Second
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		drainBody(r)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	cfg := s.cfgSnapshot()
	ctx, cancel := context.WithTimeout(r.Context(), binaryDetectTimeout)
	defer cancel()

	binDet, err := ollamaexec.DetectBinary(ctx, cfg.OllamaBin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	srv, err := ollamaexec.CheckServer(cfg.OllamaHost, serverDialTimeout)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Binary: binaryStatus{
			Found:         binDet.Found,
			Path:          binDet.Path,
			Source:        binDet.Source,
			VersionOutput: binDet.VersionOutput,
		},
		Server: serverStatus{
			Host:      srv.Host,
			Reachable: srv.Reachable,
			Error:     srv.Error,
		},
	})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		drainBody(r)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	var req scanRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cfg := s.cfgSnapshot()
	items, err := scanItems(cfg, req.Direction)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, scanResponse{Items: items})
}

func scanItems(cfg config.Config, direction string) ([]scanItem, error) {
	switch direction {
	case dirOllamaToLMStudio:
		return scanOllamaToLMStudio(cfg)
	case dirLMStudioToOllama:
		return scanLMStudioToOllama(cfg)
	default:
		return nil, fmt.Errorf("unknown direction: %s", direction)
	}
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		drainBody(r)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	var req applyRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cfg := s.cfgSnapshot()
	resp := applyResponse{}

	switch req.Direction {
	case dirOllamaToLMStudio:
		result, err := applyOllamaToLMStudio(cfg, req.Selected)
		resp.Result = result
		if err != nil {
			resp.Error = err.Error()
		}
		writeJSON(w, http.StatusOK, resp)
	case dirLMStudioToOllama:
		result, err := applyLMStudioToOllama(r.Context(), cfg, req.Imports)
		resp.Result = result
		if err != nil {
			resp.Error = err.Error()
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown direction: %s", req.Direction))
	}
}

func scanOllamaToLMStudio(cfg config.Config) ([]scanItem, error) {
	models, err := ollama.DiscoverModels(cfg.OllamaModelsDir)
	if err != nil {
		return nil, err
	}

	items := make([]scanItem, 0, len(models))
	for _, m := range models {
		repo, tag := m.ID.Repository, m.ID.Tag
		safeName := lmStudioSafeName(repo, tag)

		source := filepath.Join(cfg.OllamaModelsDir, "blobs", strings.Replace(m.ModelLayerDigest, ":", "-", 1))
		target := filepath.Join(cfg.LMStudioModelsDir, "ollama", safeName, safeName+".gguf")

		status, err := statusForTarget(target, source)
		message := ""
		if err != nil {
			status = "error"
			message = err.Error()
		}

		items = append(items, scanItem{
			ID:         idFromRepoTag(repo, tag),
			Label:      repo + ":" + tag,
			Detail:     target,
			Status:     status,
			Selectable: true,
			Selected:   status == "ready",
			Message:    message,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items, nil
}

func applyOllamaToLMStudio(cfg config.Config, selected []string) (any, error) {
	selectedSet := make(map[string]bool, len(selected))
	for _, id := range selected {
		selectedSet[id] = true
	}

	models, err := ollama.DiscoverModels(cfg.OllamaModelsDir)
	if err != nil {
		return nil, err
	}

	var picked []ollama.DiscoveredModel
	for _, m := range models {
		id := idFromRepoTag(m.ID.Repository, m.ID.Tag)
		if selectedSet[id] {
			picked = append(picked, m)
		}
	}

	return modelsync.ApplyOllamaToLMStudio(picked, cfg.OllamaModelsDir, cfg.LMStudioModelsDir, modelsync.ApplyOllamaToLMStudioOptions{
		Symlink: modelsync.EnsureSymlinkOptions{
			AllowFixSymlink:            cfg.AllowFixSymlink,
			AllowRecreateBrokenSymlink: cfg.AllowRecreateBrokenSymlink,
		},
	})
}

func statusForTarget(targetPath string, expectedSource string) (string, error) {
	st, err := symlink.InspectTarget(targetPath, expectedSource)
	if err != nil {
		return "error", err
	}
	switch st.Kind {
	case symlink.KindMissing:
		return "ready", nil
	case symlink.KindFile, symlink.KindDir:
		return "conflict", nil
	case symlink.KindSymlink:
		if st.IsSymlinkMatch && !st.IsSymlinkBroken {
			return "already_synced", nil
		}
		if st.IsSymlinkBroken {
			return "broken_symlink", nil
		}
		return "symlink_mismatch", nil
	default:
		return "error", fmt.Errorf("unknown target kind")
	}
}

func idFromRepoTag(repo, tag string) string {
	return repo + ":" + tag
}

func withinDir(dir, path string) (bool, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return false, err
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false, err
	}
	rel = filepath.ToSlash(rel)
	return rel != ".." && !strings.HasPrefix(rel, "../"), nil
}

func lmStudioSafeName(repo, tag string) string {
	repo = strings.ReplaceAll(repo, "/", "-")
	if tag == "" {
		return repo
	}
	return repo + "-" + tag
}
