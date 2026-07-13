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
	unauthorizedDestructiveChanges := worktreeUnauthorizedDestructiveChangesV0(
		result.DestructiveChanges,
		request.DestructiveAuthorizations,
	)
	if paths := worktreeDestructivePathsByKindV0(
		unauthorizedDestructiveChanges,
		WorktreeDestructiveRenamedOrMovedV0,
	); len(paths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueRenamedOrMovedV0,
				"renamed_or_moved_paths",
				paths...,
			),
		}
	}
	if paths := worktreeDestructivePathsByKindV0(
		unauthorizedDestructiveChanges,
		WorktreeDestructiveTruncatedV0,
	); len(paths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueTruncatedPathV0,
				"truncated_paths",
				paths...,
			),
		}
	}
	if paths := worktreeUnauthorizedRemovedPathsV0(
		result.RemovedPaths,
		result.AddedPaths,
		result.DestructiveChanges,
		request.DestructiveAuthorizations,
	); len(paths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueRemovedPathV0,
				"removed_paths",
				paths...,
			),
		}
	}
	if paths := worktreeDestructivePathsByKindV0(
		unauthorizedDestructiveChanges,
		WorktreeDestructiveReplacedLargeDeltaV0,
	); len(paths) > 0 {
		return result, []WorktreeIssueV0{
			worktreeIssueV0(
				WorktreeIssueReplacedLargeV0,
				"replaced_large_delta",
				paths...,
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
	issues = append(issues, worktreeControlPathIssuesV0(request.WriteSet)...)
	if len(request.AckFiles) > 0 {
		var ackIssues []WorktreeIssueV0
		request.AckFiles, ackIssues = normalizeWorktreePathListV0(request.AckFiles, false)
		issues = append(issues, ackIssues...)
		issues = append(issues, worktreeControlPathIssuesV0(request.AckFiles)...)
	}
	var destructiveAuthorizationIssues []WorktreeIssueV0
	request.DestructiveAuthorizations, destructiveAuthorizationIssues = normalizeWorktreeDestructiveAuthorizationsV0(
		request.DestructiveAuthorizations,
	)
	issues = append(issues, destructiveAuthorizationIssues...)
	if len(request.WriteSet) == 0 {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "write_set"))
	}
	return request, issues
}

func normalizeWorktreeDestructiveAuthorizationsV0(
	authorizations []WorktreeDestructiveAuthorizationV0,
) ([]WorktreeDestructiveAuthorizationV0, []WorktreeIssueV0) {
	result := make([]WorktreeDestructiveAuthorizationV0, 0, len(authorizations))
	var issues []WorktreeIssueV0
	for _, authorization := range authorizations {
		normalized, ok := normalizeWorktreeDestructiveAuthorizationV0(authorization)
		if !ok {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "destructive_authorizations"))
			continue
		}
		if !worktreeDestructiveAuthorizationInSetV0(result, normalized) {
			result = append(result, normalized)
		}
	}
	return result, issues
}

func normalizeWorktreeDestructiveAuthorizationV0(
	authorization WorktreeDestructiveAuthorizationV0,
) (WorktreeDestructiveAuthorizationV0, bool) {
	path, pathOK := normalizeWorktreeOptionalRelPathV0(authorization.Path)
	previousPath, previousPathOK := normalizeWorktreeOptionalRelPathV0(authorization.PreviousPath)
	currentPath, currentPathOK := normalizeWorktreeOptionalRelPathV0(authorization.CurrentPath)
	switch authorization.Kind {
	case WorktreeDestructiveAuthorizationRenameV0:
		if authorization.Path != "" || !previousPathOK || !currentPathOK {
			return WorktreeDestructiveAuthorizationV0{}, false
		}
		return WorktreeDestructiveAuthorizationV0{
			Kind:         authorization.Kind,
			PreviousPath: previousPath,
			CurrentPath:  currentPath,
		}, true
	case WorktreeDestructiveAuthorizationRemoveV0,
		WorktreeDestructiveAuthorizationTruncateV0,
		WorktreeDestructiveAuthorizationReplaceV0:
		if !pathOK || authorization.PreviousPath != "" || authorization.CurrentPath != "" {
			return WorktreeDestructiveAuthorizationV0{}, false
		}
		return WorktreeDestructiveAuthorizationV0{
			Kind: authorization.Kind,
			Path: path,
		}, true
	default:
		return WorktreeDestructiveAuthorizationV0{}, false
	}
}

func normalizeWorktreeOptionalRelPathV0(value string) (string, bool) {
	if value == "" {
		return "", true
	}
	return normalizeWorktreeRelPathV0(value, false)
}

func worktreeDestructiveAuthorizationInSetV0(
	authorizations []WorktreeDestructiveAuthorizationV0,
	want WorktreeDestructiveAuthorizationV0,
) bool {
	for _, authorization := range authorizations {
		if authorization == want {
			return true
		}
	}
	return false
}

func worktreeUnauthorizedDestructiveChangesV0(
	changes []WorktreeDestructiveChangeV0,
	authorizations []WorktreeDestructiveAuthorizationV0,
) []WorktreeDestructiveChangeV0 {
	result := make([]WorktreeDestructiveChangeV0, 0, len(changes))
	for _, change := range changes {
		if !worktreeDestructiveChangeAuthorizedV0(change, authorizations) {
			result = append(result, change)
		}
	}
	return result
}

func worktreeDestructiveChangeAuthorizedV0(
	change WorktreeDestructiveChangeV0,
	authorizations []WorktreeDestructiveAuthorizationV0,
) bool {
	for _, authorization := range authorizations {
		switch change.Kind {
		case WorktreeDestructiveRenamedOrMovedV0:
			if authorization.Kind == WorktreeDestructiveAuthorizationRenameV0 &&
				authorization.PreviousPath == change.PreviousPath &&
				authorization.CurrentPath == change.CurrentPath {
				return true
			}
		case WorktreeDestructiveRemovedV0:
			if authorization.Kind == WorktreeDestructiveAuthorizationRemoveV0 && authorization.Path == change.Path {
				return true
			}
		case WorktreeDestructiveTruncatedV0:
			if authorization.Kind == WorktreeDestructiveAuthorizationTruncateV0 && authorization.Path == change.Path {
				return true
			}
		case WorktreeDestructiveReplacedLargeDeltaV0:
			if authorization.Kind == WorktreeDestructiveAuthorizationReplaceV0 && authorization.Path == change.Path {
				return true
			}
		}
	}
	return false
}

func worktreeUnauthorizedRemovedPathsV0(
	removedPaths []string,
	addedPaths []string,
	changes []WorktreeDestructiveChangeV0,
	authorizations []WorktreeDestructiveAuthorizationV0,
) []string {
	result := make([]string, 0, len(removedPaths))
	for _, path := range removedPaths {
		if !worktreeRemovedPathAuthorizedV0(path, addedPaths, changes, authorizations) {
			result = append(result, path)
		}
	}
	return result
}

func worktreeRemovedPathAuthorizedV0(
	path string,
	addedPaths []string,
	changes []WorktreeDestructiveChangeV0,
	authorizations []WorktreeDestructiveAuthorizationV0,
) bool {
	if worktreeDestructiveChangeAuthorizedV0(
		WorktreeDestructiveChangeV0{Kind: WorktreeDestructiveRemovedV0, Path: path},
		authorizations,
	) {
		return true
	}
	for _, change := range changes {
		if change.Kind == WorktreeDestructiveRenamedOrMovedV0 &&
			change.PreviousPath == path &&
			worktreeDestructiveChangeAuthorizedV0(change, authorizations) {
			return true
		}
	}
	for _, authorization := range authorizations {
		if authorization.Kind == WorktreeDestructiveAuthorizationRenameV0 &&
			authorization.PreviousPath == path &&
			worktreePathInSetV0(authorization.CurrentPath, addedPaths) {
			return true
		}
	}
	return false
}

func worktreePathInSetV0(path string, paths []string) bool {
	for _, candidate := range paths {
		if candidate == path {
			return true
		}
	}
	return false
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
			if IsWorktreeControlPathV0(path) {
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
