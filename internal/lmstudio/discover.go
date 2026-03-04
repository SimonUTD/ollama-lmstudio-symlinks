package lmstudio

import (
	"io/fs"
	"path/filepath"
	"strings"
)

const ggufExt = ".gguf"

type GGUFFile struct {
	Provider  string
	ModelDir  string
	FileName  string
	FullPath  string
	SizeBytes int64
	IsPrimary bool
}

func DiscoverGGUFFiles(modelsDir string) ([]GGUFFile, error) {
	var files []GGUFFile

	err := filepath.WalkDir(modelsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ggufExt) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(modelsDir, path)
		if err != nil {
			return err
		}

		provider, modelDir := providerAndModelDirFromRel(rel)
		files = append(files, GGUFFile{
			Provider:  provider,
			ModelDir:  modelDir,
			FileName:  d.Name(),
			FullPath:  path,
			SizeBytes: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	markPrimaryByDirectory(files)
	return files, nil
}

func providerAndModelDirFromRel(rel string) (string, string) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= 3 {
		return parts[0], parts[1]
	}
	return "", ""
}

func markPrimaryByDirectory(files []GGUFFile) {
	primaryIndex := map[string]int{}

	for i, f := range files {
		dir := filepath.Dir(f.FullPath)
		bestIdx, ok := primaryIndex[dir]
		if !ok || f.SizeBytes > files[bestIdx].SizeBytes {
			primaryIndex[dir] = i
		}
	}

	for dir, idx := range primaryIndex {
		_ = dir
		files[idx].IsPrimary = true
	}
}
