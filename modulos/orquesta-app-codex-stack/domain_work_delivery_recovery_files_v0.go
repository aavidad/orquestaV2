package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"strings"
)

func domainWorkRecoveryArtifactFilesV0(
	projectDir string,
	writeSet []string,
) ([]string, bool) {
	files := make([]string, 0)
	for _, entry := range writeSet {
		entryFiles, ok := domainWorkRecoveryArtifactFilesForEntryV0(projectDir, entry)
		if !ok {
			continue
		}
		files = append(files, entryFiles...)
	}
	files = compactCodexStackStringsV0(files)
	return files, len(files) > 0
}

func domainWorkRecoveryArtifactFilesForEntryV0(
	projectDir string,
	entry string,
) ([]string, bool) {
	entry = filepath.ToSlash(filepath.Clean(strings.TrimSpace(entry)))
	path, ok := safeDomainWorkDeliveryFilePathV0(projectDir, entry)
	if !ok {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if !info.IsDir() {
		if info.Size() <= 0 {
			return nil, false
		}
		return []string{entry}, true
	}
	return domainWorkRecoveryFilesUnderDirV0(projectDir, entry, path)
}

func domainWorkRecoveryFilesUnderDirV0(
	projectDir string,
	entry string,
	dir string,
) ([]string, bool) {
	files := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() <= 0 {
			return nil
		}
		rel, relErr := filepath.Rel(projectDir, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == entry || strings.HasPrefix(rel, entry+"/") {
			files = append(files, rel)
		}
		return nil
	})
	files = compactCodexStackStringsV0(files)
	return files, err == nil && len(files) > 0
}
