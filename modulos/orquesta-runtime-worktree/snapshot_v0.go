package orquestaruntimeworktree

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var errWorktreeSnapshotStoppedV0 = errors.New("worktree_snapshot_stopped")

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
	budget := worktreeSnapshotRequestBudgetV0(request)
	files, omittedPaths, receipts, issues := collectWorktreeFilesV0(
		ctx,
		request.ProjectWorkDir,
		request.IgnorePrefixes,
		budget,
		request.AllowPartial,
	)
	if len(issues) > 0 {
		return WorktreeSnapshotV0{}, issues
	}
	return WorktreeSnapshotV0{
		SchemaVersion:     WorktreeSnapshotSchemaVersionV0,
		SnapshotRef:       request.SnapshotRef,
		ReadBudget:        budget,
		Files:             files,
		OmittedPaths:      omittedPaths,
		ExclusionReceipts: receipts,
	}, nil
}

func normalizeWorktreeSnapshotRequestV0(
	request WorktreeSnapshotRequestV0,
) WorktreeSnapshotRequestV0 {
	request.SnapshotRef = strings.TrimSpace(request.SnapshotRef)
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.IgnorePrefixes = normalizeWorktreeIgnorePrefixesV0(request.IgnorePrefixes)
	budget := worktreeSnapshotRequestBudgetV0(request)
	request.MaxFiles = budget.MaxFiles
	request.MaxFileBytes = budget.MaxFileBytes
	request.MaxTotalBytes = budget.MaxTotalBytes
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
	budget WorktreeSnapshotReadBudgetV0,
	allowPartial bool,
) ([]WorktreeSnapshotFileV0, []string, []WorktreeLocalArtifactExclusionReceiptV0, []WorktreeIssueV0) {
	files := make([]WorktreeSnapshotFileV0, 0)
	var excluded []string
	var omittedPaths []string
	var budgetReceipts []WorktreeLocalArtifactExclusionReceiptV0
	var issues []WorktreeIssueV0
	var totalBytes int64
	partialStopped := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, worktreeSnapshotUnreadableIssueV0(root, path))
			return errWorktreeSnapshotStoppedV0
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
		if worktreeControlPathV0(rel) {
			excluded = append(excluded, rel)
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if worktreePathIgnoredV0(rel, ignorePrefixes) {
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
			issues = append(issues, worktreeIssueV0(
				WorktreeIssueSnapshotUnreadableV0,
				"file",
				rel,
			))
			return errWorktreeSnapshotStoppedV0
		}
		if len(files) >= budget.MaxFiles {
			issue := worktreeIssueV0(
				WorktreeIssueSnapshotTooManyFilesV0,
				"max_files",
				"worktree_snapshot_file_count_budget_exceeded",
			)
			if allowPartial {
				if receipt, ok := worktreeSnapshotBudgetReceiptForIssueV0(issue); ok {
					budgetReceipts = append(budgetReceipts, receipt)
				}
				partialStopped = true
				return errWorktreeSnapshotStoppedV0
			}
			issues = append(issues, issue)
			return errWorktreeSnapshotStoppedV0
		}
		if info.Size() > budget.MaxFileBytes {
			issue := worktreeIssueV0(
				WorktreeIssueSnapshotFileTooLargeV0,
				"file",
				rel,
			)
			if allowPartial {
				if receipt, ok := worktreeSnapshotBudgetReceiptForIssueV0(issue); ok {
					budgetReceipts = append(budgetReceipts, receipt)
				}
				omittedPaths = append(omittedPaths, rel)
				return nil
			}
			issues = append(issues, issue)
			return errWorktreeSnapshotStoppedV0
		}
		if totalBytes+info.Size() > budget.MaxTotalBytes {
			issue := worktreeIssueV0(
				WorktreeIssueSnapshotTooLargeV0,
				"max_total_bytes",
				"worktree_snapshot_total_budget_exceeded",
			)
			if allowPartial {
				if receipt, ok := worktreeSnapshotBudgetReceiptForIssueV0(issue); ok {
					budgetReceipts = append(budgetReceipts, receipt)
				}
				omittedPaths = append(omittedPaths, rel)
				return nil
			}
			issues = append(issues, issue)
			return errWorktreeSnapshotStoppedV0
		}
		file, readBytes, issue := hashWorktreeFileV0(path, rel, info.Size(), budget, totalBytes)
		if issue != nil {
			if allowPartial {
				if receipt, ok := worktreeSnapshotBudgetReceiptForIssueV0(*issue); ok {
					budgetReceipts = append(budgetReceipts, receipt)
					omittedPaths = append(omittedPaths, rel)
					return nil
				}
			}
			issues = append(issues, *issue)
			return errWorktreeSnapshotStoppedV0
		}
		totalBytes += readBytes
		files = append(files, file)
		return nil
	})
	if errors.Is(err, errWorktreeSnapshotStoppedV0) {
		if allowPartial && partialStopped && len(issues) == 0 {
			sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
			return files, compactWorktreeStringsV0(omittedPaths), mergeWorktreeLocalArtifactReceiptsV0(
				worktreeLocalArtifactReceiptsV0(excluded),
				budgetReceipts,
			), nil
		}
		return nil, nil, nil, issues
	}
	if err != nil {
		return nil, nil, nil, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueFilesystemV0, "project_work_dir")}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, compactWorktreeStringsV0(omittedPaths), mergeWorktreeLocalArtifactReceiptsV0(
		worktreeLocalArtifactReceiptsV0(excluded),
		budgetReceipts,
	), nil
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

func worktreeSnapshotUnreadableIssueV0(root string, path string) WorktreeIssueV0 {
	if rel, ok := worktreeRelativePathV0(root, path); ok {
		return worktreeIssueV0(WorktreeIssueSnapshotUnreadableV0, "file", rel)
	}
	return worktreeIssueV0(WorktreeIssueSnapshotUnreadableV0, "file")
}
