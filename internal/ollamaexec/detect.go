package ollamaexec

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	envOllamaBin  = "OLLAMA_BIN"
	envOllamaHost = "OLLAMA_HOST"
)

type BinaryDetection struct {
	Found         bool
	Path          string
	Source        string
	VersionOutput string
}

func DetectBinary(ctx context.Context, configuredPath string) (BinaryDetection, error) {
	candidates := candidatePaths(configuredPath)
	for _, c := range candidates {
		ok, err := isExecutableFile(c.Path)
		if err != nil {
			return BinaryDetection{}, err
		}
		if !ok {
			continue
		}

		versionOut, versionErr := runVersion(ctx, c.Path)
		versionOut = strings.TrimSpace(versionOut)
		if versionErr != nil {
			if versionOut == "" {
				versionOut = "ERROR: " + versionErr.Error()
			} else {
				versionOut = versionOut + "\nERROR: " + versionErr.Error()
			}
		}
		return BinaryDetection{
			Found:         true,
			Path:          c.Path,
			Source:        c.Source,
			VersionOutput: versionOut,
		}, nil
	}

	return BinaryDetection{Found: false}, nil
}

type candidate struct {
	Path   string
	Source string
}

func candidatePaths(configuredPath string) []candidate {
	var out []candidate
	if strings.TrimSpace(configuredPath) != "" {
		out = append(out, candidate{Path: configuredPath, Source: "config"})
	}
	if v := strings.TrimSpace(os.Getenv(envOllamaBin)); v != "" {
		out = append(out, candidate{Path: v, Source: "env:" + envOllamaBin})
	}
	if lp, err := exec.LookPath("ollama"); err == nil {
		out = append(out, candidate{Path: lp, Source: "PATH"})
	}

	for _, p := range fixedPaths() {
		out = append(out, candidate{Path: p, Source: "fixed"})
	}
	return out
}

func fixedPaths() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/Applications/Ollama.app/Contents/Resources/ollama",
			"/opt/homebrew/bin/ollama",
			"/usr/local/bin/ollama",
		}
	}
	return []string{}
}

func isExecutableFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Ext(path), ".exe"), nil
	}
	return info.Mode()&0o111 != 0, nil
}

func runVersion(ctx context.Context, binPath string) (string, error) {
	cmd := exec.CommandContext(ctx, binPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

type ServerStatus struct {
	Host      string
	Reachable bool
	Error     string
}

func CheckServer(host string, timeout time.Duration) (ServerStatus, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		host = os.Getenv(envOllamaHost)
	}
	if host == "" {
		return ServerStatus{}, errors.New("empty host")
	}

	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return ServerStatus{
			Host:      host,
			Reachable: false,
			Error:     err.Error(),
		}, nil
	}
	_ = conn.Close()
	return ServerStatus{Host: host, Reachable: true}, nil
}
