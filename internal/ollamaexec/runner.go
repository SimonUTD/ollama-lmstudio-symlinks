package ollamaexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ExecRunner struct {
	BinPath string
	Host    string
}

func (r ExecRunner) CreateFromGGUF(ctx context.Context, modelName string, ggufPath string) error {
	ggufAbs, err := filepath.Abs(ggufPath)
	if err != nil {
		return err
	}
	if strings.ContainsAny(ggufAbs, "\r\n") {
		return fmt.Errorf("ggufPath 包含换行符，无法写入 Modelfile: %q", ggufAbs)
	}

	tmpDir, err := os.MkdirTemp("", "ollama-lmstudio-symlinks-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	modelfilePath := filepath.Join(tmpDir, "Modelfile")
	modelfile := "FROM " + ggufAbs + "\n"
	if err := os.WriteFile(modelfilePath, []byte(modelfile), 0o600); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, r.BinPath, "create", modelName, "-f", modelfilePath)
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+strings.TrimSpace(r.Host))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ollama create failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
