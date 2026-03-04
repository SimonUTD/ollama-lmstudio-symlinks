package symlink

import (
	"os"
	"path/filepath"
)

type Kind int

const (
	KindMissing Kind = iota
	KindFile
	KindDir
	KindSymlink
)

type TargetStatus struct {
	Kind              Kind
	Path              string
	ExpectedSource    string
	ExpectedSourceAbs string

	SymlinkTargetRaw string
	SymlinkTargetAbs string

	IsSymlinkMatch  bool
	IsSymlinkBroken bool
}

func InspectTarget(targetPath string, expectedSourcePath string) (TargetStatus, error) {
	expectedAbs, err := absClean(expectedSourcePath)
	if err != nil {
		return TargetStatus{}, err
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return TargetStatus{
				Kind:              KindMissing,
				Path:              targetPath,
				ExpectedSource:    expectedSourcePath,
				ExpectedSourceAbs: expectedAbs,
			}, nil
		}
		return TargetStatus{}, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return inspectSymlink(targetPath, expectedSourcePath, expectedAbs)
	}
	if info.IsDir() {
		return TargetStatus{
			Kind:              KindDir,
			Path:              targetPath,
			ExpectedSource:    expectedSourcePath,
			ExpectedSourceAbs: expectedAbs,
		}, nil
	}
	return TargetStatus{
		Kind:              KindFile,
		Path:              targetPath,
		ExpectedSource:    expectedSourcePath,
		ExpectedSourceAbs: expectedAbs,
	}, nil
}

func inspectSymlink(targetPath, expectedSourcePath, expectedAbs string) (TargetStatus, error) {
	rawTarget, err := os.Readlink(targetPath)
	if err != nil {
		return TargetStatus{}, err
	}

	targetAbs, err := absSymlinkTarget(targetPath, rawTarget)
	if err != nil {
		return TargetStatus{}, err
	}

	broken, err := isSymlinkBroken(targetPath)
	if err != nil {
		return TargetStatus{}, err
	}

	return TargetStatus{
		Kind:              KindSymlink,
		Path:              targetPath,
		ExpectedSource:    expectedSourcePath,
		ExpectedSourceAbs: expectedAbs,
		SymlinkTargetRaw:  rawTarget,
		SymlinkTargetAbs:  targetAbs,
		IsSymlinkMatch:    targetAbs == expectedAbs,
		IsSymlinkBroken:   broken,
	}, nil
}

func absClean(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func absSymlinkTarget(linkPath string, rawTarget string) (string, error) {
	if filepath.IsAbs(rawTarget) {
		return absClean(rawTarget)
	}
	return absClean(filepath.Join(filepath.Dir(linkPath), rawTarget))
}

func isSymlinkBroken(linkPath string) (bool, error) {
	_, err := os.Stat(linkPath)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}
