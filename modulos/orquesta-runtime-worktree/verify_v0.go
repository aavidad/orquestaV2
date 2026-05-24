package orquestaruntimeworktree

import (
	"context"
	"strings"
)

func VerifyWorktreeWriteSetV0(
	ctx context.Context,
	request WorktreeVerifyRequestV0,
) (WorktreeVerifyResultV0, []WorktreeIssueV0) {
	request, issues := normalizeWorktreeVerifyRequestV0(request)
	if len(issues) > 0 {
		return WorktreeVerifyResultV0{}, issues
	}
	current, captureIssues := CaptureWorktreeSnapshotV0(ctx, WorktreeSnapshotRequestV0{
		SnapshotRef:    request.Baseline.SnapshotRef + "-current",
		ProjectWorkDir: request.ProjectWorkDir,
		IgnorePrefixes: request.IgnorePrefixes,
	})
	if len(captureIssues) > 0 {
		return WorktreeVerifyResultV0{}, captureIssues
	}
	result := diffWorktreeSnapshotsV0(request.Baseline, current, request.WriteSet)
	if len(result.RemovedPaths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueRemovedPathV0,
				"removed_paths",
				result.RemovedPaths...,
			),
		}
	}
	if len(result.OutsideWriteSet) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueOutsideWriteSetV0,
				"write_set",
				"changed_file_outside_write_set",
			),
		}
	}
	result.OK = true
	return result, nil
}

func normalizeWorktreeVerifyRequestV0(
	request WorktreeVerifyRequestV0,
) (WorktreeVerifyRequestV0, []WorktreeIssueV0) {
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.IgnorePrefixes, _ = normalizeWorktreePathListV0(request.IgnorePrefixes, false)
	var issues []WorktreeIssueV0
	if request.Baseline.SchemaVersion != WorktreeSnapshotSchemaVersionV0 ||
		strings.TrimSpace(request.Baseline.SnapshotRef) == "" {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "baseline"))
	}
	var writeIssues []WorktreeIssueV0
	request.WriteSet, writeIssues = normalizeWorktreePathListV0(request.WriteSet, true)
	issues = append(issues, writeIssues...)
	if len(request.WriteSet) == 0 {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "write_set"))
	}
	return request, issues
}

func diffWorktreeSnapshotsV0(
	baseline WorktreeSnapshotV0,
	current WorktreeSnapshotV0,
	writeSet []string,
) WorktreeVerifyResultV0 {
	base := worktreeSnapshotMapV0(baseline)
	now := worktreeSnapshotMapV0(current)
	result := WorktreeVerifyResultV0{}
	for path, currentFile := range now {
		baseFile, existed := base[path]
		if !existed {
			result.AddedPaths = append(result.AddedPaths, path)
			result.ChangedPaths = append(result.ChangedPaths, path)
			continue
		}
		if baseFile.Digest != currentFile.Digest || baseFile.Size != currentFile.Size {
			result.ChangedPaths = append(result.ChangedPaths, path)
		}
	}
	for path := range base {
		if _, ok := now[path]; !ok {
			result.RemovedPaths = append(result.RemovedPaths, path)
			result.ChangedPaths = append(result.ChangedPaths, path)
		}
	}
	result.ChangedPaths = compactWorktreeStringsV0(result.ChangedPaths)
	result.AddedPaths = compactWorktreeStringsV0(result.AddedPaths)
	result.RemovedPaths = compactWorktreeStringsV0(result.RemovedPaths)
	result.OutsideWriteSet = changedPathsOutsideWriteSetV0(result.ChangedPaths, writeSet)
	result.EvidenceRefs = []string{"evidence-ref-worktree-write-set-verified-v0"}
	return result
}

func worktreeSnapshotMapV0(
	snapshot WorktreeSnapshotV0,
) map[string]WorktreeSnapshotFileV0 {
	result := make(map[string]WorktreeSnapshotFileV0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if path, ok := normalizeWorktreeRelPathV0(file.Path, false); ok {
			result[path] = file
		}
	}
	return result
}

func changedPathsOutsideWriteSetV0(paths []string, writeSet []string) []string {
	out := make([]string, 0)
	for _, path := range paths {
		if !worktreePathAllowedV0(path, writeSet) {
			out = append(out, path)
		}
	}
	return compactWorktreeStringsV0(out)
}
