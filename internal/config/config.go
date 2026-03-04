package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	appName           = "ollama-lmstudio-symlinks"
	defaultOllamaHost = "127.0.0.1:11434"
)

type Config struct {
	OllamaModelsDir   string `json:"ollamaModelsDir"`
	LMStudioModelsDir string `json:"lmStudioModelsDir"`

	OllamaBin  string `json:"ollamaBin"`
	OllamaHost string `json:"ollamaHost"`

	AllowFixSymlink            bool `json:"allowFixSymlink"`
	AllowRecreateBrokenSymlink bool `json:"allowRecreateBrokenSymlink"`
	AllowReplaceExistingBlob   bool `json:"allowReplaceExistingBlob"`
}

func Default() Config {
	return Config{
		OllamaModelsDir:   defaultOllamaModelsDir(),
		LMStudioModelsDir: defaultLMStudioModelsDir(),
		OllamaHost:        defaultOllamaHost,
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, "config.json"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg = withDefaults(cfg)
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	cfg = withDefaults(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func withDefaults(cfg Config) Config {
	if strings.TrimSpace(cfg.OllamaModelsDir) == "" {
		cfg.OllamaModelsDir = defaultOllamaModelsDir()
	}
	if strings.TrimSpace(cfg.LMStudioModelsDir) == "" {
		cfg.LMStudioModelsDir = defaultLMStudioModelsDir()
	}
	if strings.TrimSpace(cfg.OllamaHost) == "" {
		cfg.OllamaHost = defaultOllamaHost
	}
	return cfg
}

func defaultOllamaModelsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ollama", "models")
}

func defaultLMStudioModelsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "lm-studio", "models")
}
