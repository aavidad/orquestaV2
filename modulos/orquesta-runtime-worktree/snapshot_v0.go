package orquestaruntimeworktree

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func CaptureWorktreeSnapshotV0(
	ctx context.Context,
	request WorktreeSnapshotRequestV0,
) (WorktreeSnapshotV0, []WorktreeIssueV0) {
	request = normalizeWorktreeSnapshotRequestV0(request)
	if issue := validateWorktreeRootV0(request.ProjectWorkDir); issue != nil {
		return WorktreeSnapshotV0{}, []WorktreeIssueV0{*issue}
	}
	if strings.TrimSpace(request.SnapshotRef) == "" {
		return WorktreeSnapshotV0{}, []WorktreeIssueV0{
			worktreeIssueV0(WorktreeIssueInvalidRequestV0, "snapshot_ref"),
		}
	}
	files, issues := collectWorktreeFilesV0(ctx, request.ProjectWorkDir, request.IgnorePrefixes)
	if len(issues) > 0 {
		return WorktreeSnapshotV0{}, issues
	}
	return WorktreeSnapshotV0{
		SchemaVersion: WorktreeSnapshotSchemaVersionV0,
		SnapshotRef:   request.SnapshotRef,
		Files:         files,
	}, nil
}

func normalizeWorktreeSnapshotRequestV0(
	request WorktreeSnapshotRequestV0,
) WorktreeSnapshotRequestV0 {
	request.SnapshotRef = strings.TrimSpace(request.SnapshotRef)
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.IgnorePrefixes = normalizeWorktreeIgnorePrefixesV0(request.IgnorePrefixes)
	return request
}

func validateWorktreeRootV0(projectWorkDir string) *WorktreeIssueV0 {
	if projectWorkDir == "" || !filepath.IsAbs(projectWorkDir) {
		issue := worktreeIssueV0(WorktreeIssueInvalidRequestV0, "project_work_dir")
		return &issue
	}
	info, err := os.Stat(projectWorkDir)
	if err != nil || !info.IsDir() {
		issue := worktreeIssueV0(WorktreeIssueFilesystemV0, "project_work_dir")
		return &issue
	}
	return nil
}

func collectWorktreeFilesV0(
	ctx context.Context,
	root string,
	ignorePrefixes []string,
) ([]WorktreeSnapshotFileV0, []WorktreeIssueV0) {
	files := make([]WorktreeSnapshotFileV0, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if path == root {
			return nil
		}
		rel, ok := worktreeRelativePathV0(root, path)
		if !ok {
			return fs.SkipDir
		}
		if worktreePathIgnoredV0(rel, ignorePrefixes) || worktreeControlPathV0(rel) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return err
		}
		file, err := hashWorktreeFileV0(path, rel, info.Size())
		if err != nil {
			return err
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueFilesystemV0, "project_work_dir")}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func worktreeRelativePathV0(root string, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", false
	}
	return rel, true
}

func hashWorktreeFileV0(path string, rel string, size int64) (WorktreeSnapshotFileV0, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WorktreeSnapshotFileV0{}, err
	}
	sum := sha256.Sum256(data)
	return WorktreeSnapshotFileV0{
		Path:      rel,
		Digest:    hex.EncodeToString(sum[:]),
		Size:      size,
		LineCount: worktreeGoLineCountV0(rel, data),
	}, nil
}

func worktreeGoLineCountV0(rel string, data []byte) int {
	if !strings.HasSuffix(rel, ".go") || len(data) == 0 {
		return 0
	}
	lines := bytes.Count(data, []byte{'\n'})
	if !bytes.HasSuffix(data, []byte{'\n'}) {
		lines++
	}
	return lines
}
