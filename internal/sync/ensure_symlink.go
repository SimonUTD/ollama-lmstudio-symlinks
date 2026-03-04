package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SimonUTD/ollama-lmstudio-symlinks/internal/symlink"
)

type EnsureSymlinkOptions struct {
	AllowFixSymlink            bool
	AllowRecreateBrokenSymlink bool
}

type EnsureSymlinkResultKind int

const (
	EnsureSymlinkCreated EnsureSymlinkResultKind = iota
	EnsureSymlinkSkippedAlreadySynced
	EnsureSymlinkFixed
)

type EnsureSymlinkResult struct {
	Kind   EnsureSymlinkResultKind
	Status symlink.TargetStatus
}

func EnsureSymlink(targetPath string, sourcePath string, opts EnsureSymlinkOptions) (EnsureSymlinkResult, error) {
	sourceAbs, err := absClean(sourcePath)
	if err != nil {
		return EnsureSymlinkResult{}, err
	}

	status, err := symlink.InspectTarget(targetPath, sourceAbs)
	if err != nil {
		return EnsureSymlinkResult{}, err
	}

	switch status.Kind {
	case symlink.KindMissing:
		if err := os.Symlink(sourceAbs, targetPath); err != nil {
			return EnsureSymlinkResult{}, err
		}
		return EnsureSymlinkResult{Kind: EnsureSymlinkCreated, Status: status}, nil
	case symlink.KindSymlink:
		return ensureSymlinkWhenSymlinkExists(targetPath, sourceAbs, status, opts)
	case symlink.KindDir:
		return EnsureSymlinkResult{}, fmt.Errorf("target exists as directory: %s", targetPath)
	case symlink.KindFile:
		return EnsureSymlinkResult{}, fmt.Errorf("target exists as file: %s", targetPath)
	default:
		return EnsureSymlinkResult{}, fmt.Errorf("unknown target kind for %s: %v", targetPath, status.Kind)
	}
}

func ensureSymlinkWhenSymlinkExists(targetPath string, sourceAbs string, status symlink.TargetStatus, opts EnsureSymlinkOptions) (EnsureSymlinkResult, error) {
	if status.IsSymlinkMatch && !status.IsSymlinkBroken {
		return EnsureSymlinkResult{Kind: EnsureSymlinkSkippedAlreadySynced, Status: status}, nil
	}

	if status.IsSymlinkBroken && status.IsSymlinkMatch {
		if !opts.AllowRecreateBrokenSymlink {
			return EnsureSymlinkResult{}, fmt.Errorf("broken symlink exists (recreate disabled): %s", targetPath)
		}
		if err := replaceSymlink(targetPath, sourceAbs); err != nil {
			return EnsureSymlinkResult{}, err
		}
		return EnsureSymlinkResult{Kind: EnsureSymlinkFixed, Status: status}, nil
	}

	if !opts.AllowFixSymlink {
		return EnsureSymlinkResult{}, fmt.Errorf("symlink exists but points elsewhere (fix disabled): %s", targetPath)
	}
	if err := replaceSymlink(targetPath, sourceAbs); err != nil {
		return EnsureSymlinkResult{}, err
	}
	return EnsureSymlinkResult{Kind: EnsureSymlinkFixed, Status: status}, nil
}

func replaceSymlink(targetPath string, sourceAbs string) error {
	if err := os.Remove(targetPath); err != nil {
		return err
	}
	return os.Symlink(sourceAbs, targetPath)
}

func absClean(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
