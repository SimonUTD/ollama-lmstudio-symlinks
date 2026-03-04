package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/config"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/lmstudio"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollama"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/ollamaexec"
	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/symlink"
	modelsync "github.com/SimonUTD/ollama-lmstudio-symlinks/internal/sync"
)

func scanLMStudioToOllama(cfg config.Config) ([]scanItem, error) {
	files, err := lmstudio.DiscoverGGUFFiles(cfg.LMStudioModelsDir)
	if err != nil {
		return nil, err
	}

	byKey, err := buildOllamaModelIndex(cfg.OllamaModelsDir)
	if err != nil {
		return nil, err
	}

	items := make([]scanItem, 0, len(files))
	dirsWithGGUF := ggufDirKeySet(files)
	for _, f := range files {
		items = append(items, scanItemFromGGUFFile(cfg, f, byKey))
	}

	unsupported, err := unsupportedLMStudioModelItems(cfg.LMStudioModelsDir, dirsWithGGUF)
	if err != nil {
		return nil, err
	}
	items = append(items, unsupported...)

	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items, nil
}

func applyLMStudioToOllama(ctx context.Context, cfg config.Config, imports []lmImport) (any, error) {
	if len(imports) == 0 {
		return []modelsync.ApplyLMStudioToOllamaResult{}, nil
	}

	specs, err := validateLMStudioImports(cfg, imports)
	if err != nil {
		return nil, err
	}

	runner, err := requireOllamaRunner(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return modelsync.ApplyLMStudioToOllama(ctx, specs, cfg.OllamaModelsDir, runner, modelsync.ApplyLMStudioToOllamaOptions{
		AllowReplaceExistingBlob: cfg.AllowReplaceExistingBlob,
	})
}

func buildOllamaModelIndex(ollamaModelsDir string) (map[string]ollama.DiscoveredModel, error) {
	models, err := ollama.DiscoverModels(ollamaModelsDir)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]ollama.DiscoveredModel, len(models))
	for _, m := range models {
		byKey[idFromRepoTag(m.ID.Repository, m.ID.Tag)] = m
	}
	return byKey, nil
}

func ggufDirKeySet(files []lmstudio.GGUFFile) map[string]bool {
	out := make(map[string]bool, len(files))
	for _, f := range files {
		out[lmstudioDirKey(f.Provider, f.ModelDir)] = true
	}
	return out
}

func scanItemFromGGUFFile(cfg config.Config, f lmstudio.GGUFFile, byKey map[string]ollama.DiscoveredModel) scanItem {
	suggested := suggestOllamaName(f)
	key := idFromRepoTag(suggested, "latest")
	existing, ok := byKey[key]
	status, message := statusForGGUFImport(cfg, f, existing, ok)

	label := f.Provider + "/" + f.ModelDir + "/" + f.FileName
	detail := f.FullPath + " (" + strconv.FormatInt(f.SizeBytes, 10) + " bytes)"

	return scanItem{
		ID:         f.FullPath,
		Label:      label,
		Detail:     detail,
		Status:     status,
		Selectable: true,
		Selected:   f.IsPrimary && status == "ready",
		Message:    message,
		GGUFPath:   f.FullPath,
		ModelName:  suggested,
	}
}

func statusForGGUFImport(cfg config.Config, f lmstudio.GGUFFile, existing ollama.DiscoveredModel, hasExisting bool) (string, string) {
	if !hasExisting {
		return "ready", ""
	}

	blobPath := filepath.Join(cfg.OllamaModelsDir, "blobs", strings.Replace(existing.ModelLayerDigest, ":", "-", 1))
	st, err := symlink.InspectTarget(blobPath, f.FullPath)
	if err != nil {
		return "conflict", err.Error()
	}
	if st.Kind == symlink.KindSymlink && st.IsSymlinkMatch && !st.IsSymlinkBroken {
		return "already_linked", ""
	}
	return "conflict", ""
}

func validateLMStudioImports(cfg config.Config, imports []lmImport) ([]modelsync.LMStudioToOllamaSpec, error) {
	specs := make([]modelsync.LMStudioToOllamaSpec, 0, len(imports))
	for _, it := range imports {
		if strings.TrimSpace(it.GGUFPath) == "" {
			return nil, fmt.Errorf("empty ggufPath")
		}
		if strings.TrimSpace(it.ModelName) == "" {
			return nil, fmt.Errorf("empty modelName for %s", it.GGUFPath)
		}
		if !strings.EqualFold(filepath.Ext(it.GGUFPath), ".gguf") {
			return nil, fmt.Errorf("仅支持 .gguf：%s", it.GGUFPath)
		}

		ok, err := withinDir(cfg.LMStudioModelsDir, it.GGUFPath)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("ggufPath is outside lmstudioModelsDir: %s", it.GGUFPath)
		}

		info, err := os.Stat(it.GGUFPath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, fmt.Errorf("ggufPath is a directory: %s", it.GGUFPath)
		}

		specs = append(specs, modelsync.LMStudioToOllamaSpec{
			ModelName: it.ModelName,
			GGUFPath:  it.GGUFPath,
		})
	}

	return specs, nil
}

func detectOllamaBinary(ctx context.Context, cfg config.Config) (ollamaexec.BinaryDetection, error) {
	detectCtx, cancel := context.WithTimeout(ctx, binaryDetectTimeout)
	defer cancel()
	return ollamaexec.DetectBinary(detectCtx, cfg.OllamaBin)
}

func requireOllamaRunner(ctx context.Context, cfg config.Config) (ollamaexec.ExecRunner, error) {
	bin, err := detectOllamaBinary(ctx, cfg)
	if err != nil {
		return ollamaexec.ExecRunner{}, err
	}
	if !bin.Found {
		return ollamaexec.ExecRunner{}, fmt.Errorf("ollama 未安装或未找到可执行文件")
	}

	srv, err := ollamaexec.CheckServer(cfg.OllamaHost, serverDialTimeout)
	if err != nil {
		return ollamaexec.ExecRunner{}, err
	}
	if !srv.Reachable {
		return ollamaexec.ExecRunner{}, fmt.Errorf("无法连接 Ollama 服务（%s）：%s", srv.Host, srv.Error)
	}

	return ollamaexec.ExecRunner{BinPath: bin.Path, Host: srv.Host}, nil
}

func suggestOllamaName(f lmstudio.GGUFFile) string {
	base := strings.TrimSuffix(f.FileName, filepath.Ext(f.FileName))
	return sanitizeOllamaName(strings.ToLower(f.Provider + "-" + f.ModelDir + "-" + base))
}

func suggestOllamaNameForDir(provider string, modelDir string) string {
	return sanitizeOllamaName(strings.ToLower(provider + "-" + modelDir))
}

func sanitizeOllamaName(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "lmstudio-import"
	}
	return out
}

func lmstudioDirKey(provider string, modelDir string) string {
	return provider + "/" + modelDir
}

func unsupportedLMStudioModelItems(modelsDir string, dirsWithGGUF map[string]bool) ([]scanItem, error) {
	providers, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, err
	}

	var items []scanItem
	for _, p := range providers {
		if !p.IsDir() || strings.HasPrefix(p.Name(), ".") {
			continue
		}
		provider := p.Name()

		models, err := os.ReadDir(filepath.Join(modelsDir, provider))
		if err != nil {
			return nil, err
		}

		for _, m := range models {
			if !m.IsDir() || strings.HasPrefix(m.Name(), ".") {
				continue
			}
			modelDir := m.Name()
			if dirsWithGGUF[lmstudioDirKey(provider, modelDir)] {
				continue
			}

			full := filepath.Join(modelsDir, provider, modelDir)
			items = append(items, scanItem{
				ID:         "dir:" + full,
				Label:      provider + "/" + modelDir,
				Detail:     full,
				Status:     "unsupported",
				Selectable: false,
				Selected:   false,
				Message:    "不支持：目录中未发现 .gguf（可能是 MLX/其他格式）",
				ModelName:  suggestOllamaNameForDir(provider, modelDir),
			})
		}
	}

	return items, nil
}
