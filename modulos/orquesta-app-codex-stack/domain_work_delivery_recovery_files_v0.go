package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
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
	_ = dir
	result := orquestaruntimeworktree.ProjectTreeScanFilesV0(context.Background(), orquestaruntimeworktree.ProjectTreeScanRequestV0{
		ProjectRoot:    projectDir,
		Target:         entry,
		Mode:           orquestaruntimeworktree.ProjectTreeScanModeDirV0,
		IgnorePrefixes: orquestaruntimeworktree.DefaultWorktreeControlIgnorePrefixesV0(),
	})
	files := compactCodexStackStringsV0(result.MatchedPaths)
	return files, result.Found && len(files) > 0
}
