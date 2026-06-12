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
		MaxFiles:       request.MaxSnapshotFiles,
		MaxFileBytes:   request.MaxSnapshotFileBytes,
		MaxTotalBytes:  request.MaxSnapshotTotalBytes,
		AllowPartial:   request.AllowPartialSnapshot,
	})
	if len(captureIssues) > 0 {
		return WorktreeVerifyResultV0{}, captureIssues
	}
	result := diffWorktreeSnapshotsV0(
		request.Baseline,
		current,
		request.WriteSet,
		request.AckFiles,
		request,
	)
	if len(result.RenamedOrMovedPaths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueRenamedOrMovedV0,
				"renamed_or_moved_paths",
				result.RenamedOrMovedPaths...,
			),
		}
	}
	if len(result.TruncatedPaths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueTruncatedPathV0,
				"truncated_paths",
				result.TruncatedPaths...,
			),
		}
	}
	if len(result.RemovedPaths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueRemovedPathV0,
				"removed_paths",
				result.RemovedPaths...,
			),
		}
	}
	if len(result.ReplacedLargeDelta) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueReplacedLargeV0,
				"replaced_large_delta",
				result.ReplacedLargeDelta...,
			),
		}
	}
	if len(result.AckFilesNotChanged) > 0 || len(result.UnreportedChangedPaths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueAckFilesMismatchV0,
				"ack_files",
				"ack_files_do_not_match_snapshot_changes",
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
	if len(result.GoLineBudgetViolations) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueGoLineBudgetV0,
				"go_line_budget",
				worktreeGoLineBudgetEvidenceV0(result.GoLineBudgetViolations)...,
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
	request.IgnorePrefixes = normalizeWorktreeIgnorePrefixesV0(request.IgnorePrefixes)
	budget := worktreeVerifySnapshotBudgetV0(request)
	request.MaxSnapshotFiles = budget.MaxFiles
	request.MaxSnapshotFileBytes = budget.MaxFileBytes
	request.MaxSnapshotTotalBytes = budget.MaxTotalBytes
	request.AcceptedPartitionFollowups, _ = normalizeWorktreePathListV0(request.AcceptedPartitionFollowups, false)
	var issues []WorktreeIssueV0
	if request.Baseline.SchemaVersion != WorktreeSnapshotSchemaVersionV0 ||
		strings.TrimSpace(request.Baseline.SnapshotRef) == "" {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "baseline"))
	}
	var writeIssues []WorktreeIssueV0
	request.WriteSet, writeIssues = normalizeWorktreePathListV0(request.WriteSet, true)
	issues = append(issues, writeIssues...)
	if len(request.AckFiles) > 0 {
		var ackIssues []WorktreeIssueV0
		request.AckFiles, ackIssues = normalizeWorktreePathListV0(request.AckFiles, false)
		issues = append(issues, ackIssues...)
		issues = append(issues, worktreeControlPathIssuesV0(request.AckFiles)...)
	}
	if len(request.WriteSet) == 0 {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "write_set"))
	}
	return request, issues
}

func diffWorktreeSnapshotsV0(
	baseline WorktreeSnapshotV0,
	current WorktreeSnapshotV0,
	writeSet []string,
	ackFiles []string,
	request WorktreeVerifyRequestV0,
) WorktreeVerifyResultV0 {
	base := worktreeSnapshotMapV0(baseline)
	now := worktreeSnapshotMapV0(current)
	currentOmitted := worktreeSnapshotOmittedPathSetV0(current)
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
			if currentOmitted[path] {
				continue
			}
			result.RemovedPaths = append(result.RemovedPaths, path)
			result.ChangedPaths = append(result.ChangedPaths, path)
		}
	}
	result.ChangedPaths = compactWorktreeStringsV0(result.ChangedPaths)
	result.AddedPaths = compactWorktreeStringsV0(result.AddedPaths)
	result.RemovedPaths = compactWorktreeStringsV0(result.RemovedPaths)
	result.DestructiveChanges = classifyWorktreeDestructiveChangesV0(base, now)
	result.TruncatedPaths = worktreeDestructivePathsByKindV0(
		result.DestructiveChanges,
		WorktreeDestructiveTruncatedV0,
	)
	result.RenamedOrMovedPaths = worktreeDestructivePathsByKindV0(
		result.DestructiveChanges,
		WorktreeDestructiveRenamedOrMovedV0,
	)
	result.ReplacedLargeDelta = worktreeDestructivePathsByKindV0(
		result.DestructiveChanges,
		WorktreeDestructiveReplacedLargeDeltaV0,
	)
	result.OutsideWriteSet = changedPathsOutsideWriteSetV0(result.ChangedPaths, writeSet)
	result.AckFilesNotChanged = ackFilesNotChangedV0(result.ChangedPaths, ackFiles)
	result.UnreportedChangedPaths = changedPathsMissingFromAckFilesV0(result.ChangedPaths, ackFiles)
	if request.StrictGoLineBudget {
		result.GoLineBudgetViolations = goLineBudgetViolationsV0(
			baseline,
			current,
			result.ChangedPaths,
			request.MaxGoFileLines,
			request.AcceptedPartitionFollowups,
		)
	}
	result.ExclusionReceipts = mergeWorktreeLocalArtifactReceiptsV0(
		baseline.ExclusionReceipts,
		current.ExclusionReceipts,
	)
	result.EvidenceRefs = compactWorktreeStringsV0(append(
		[]string{"evidence-ref-worktree-write-set-verified-v0"},
		worktreeSnapshotBudgetEvidenceRefsV0(result.ExclusionReceipts)...,
	))
	return result
}

func worktreeSnapshotOmittedPathSetV0(
	snapshot WorktreeSnapshotV0,
) map[string]bool {
	out := make(map[string]bool, len(snapshot.OmittedPaths))
	for _, item := range snapshot.OmittedPaths {
		if path, ok := normalizeWorktreeRelPathV0(item, false); ok {
			out[path] = true
		}
	}
	return out
}

func worktreeSnapshotMapV0(
	snapshot WorktreeSnapshotV0,
) map[string]WorktreeSnapshotFileV0 {
	result := make(map[string]WorktreeSnapshotFileV0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if path, ok := normalizeWorktreeRelPathV0(file.Path, false); ok {
			if worktreeControlPathV0(path) {
				continue
			}
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

func ackFilesNotChangedV0(changedPaths []string, ackFiles []string) []string {
	if len(ackFiles) == 0 {
		return nil
	}
	out := make([]string, 0)
	for _, file := range ackFiles {
		if !worktreeStringInSetV0(changedPaths, file) {
			out = append(out, file)
		}
	}
	return compactWorktreeStringsV0(out)
}

func changedPathsMissingFromAckFilesV0(changedPaths []string, ackFiles []string) []string {
	if len(ackFiles) == 0 {
		return nil
	}
	out := make([]string, 0)
	for _, path := range changedPaths {
		if !worktreeStringInSetV0(ackFiles, path) {
			out = append(out, path)
		}
	}
	return compactWorktreeStringsV0(out)
}

func worktreeStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
