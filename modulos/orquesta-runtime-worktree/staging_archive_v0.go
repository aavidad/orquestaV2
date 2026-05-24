package orquestaruntimeworktree

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func (connector GitStagingPromotionConnectorV0) ArchiveStagingWorktreeV0(
	ctx context.Context,
	request StagingPromotionRequestV0,
) (StagingPromotionResultV0, []WorktreeIssueV0) {
	request = normalizeStagingPromotionRequestV0(request)
	if issues := validateStagingPromotionRequestV0(request, true); len(issues) > 0 {
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			issue := worktreeIssueV0(WorktreeIssueFilesystemV0, "context", err.Error())
			return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, []WorktreeIssueV0{issue}), []WorktreeIssueV0{issue}
		}
	}
	if err := os.MkdirAll(request.ArchiveDir, 0o700); err != nil {
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueFilesystemV0, "archive_dir")}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	body, err := json.MarshalIndent(stagingPromotionArchiveManifestV0(request), "", "  ")
	if err != nil {
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueInvalidRequestV0, "archive_manifest")}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	path := filepath.Join(request.ArchiveDir, request.ArchiveRef+".json")
	if existing, err := os.ReadFile(path); err == nil {
		if reflect.DeepEqual(existing, body) {
			return newStagingPromotionResultV0(request, StagingPromotionStatusArchivedV0, request.WriteSet, nil), nil
		}
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueInvalidRequestV0, "archive_ref")}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueFilesystemV0, "archive_dir")}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	return newStagingPromotionResultV0(request, StagingPromotionStatusArchivedV0, request.WriteSet, nil), nil
}

func stagingPromotionArchiveManifestV0(request StagingPromotionRequestV0) map[string]any {
	sort.Strings(request.WriteSet)
	return map[string]any{
		"schema_version": StagingPromotionResultSchemaVersionV0,
		"archive_ref":    request.ArchiveRef,
		"promotion_ref":  request.PromotionRef,
		"run_ref":        request.RunRef,
		"project_ref":    request.ProjectRef,
		"repo_ref":       request.RepoRef,
		"worktree_ref":   request.WorktreeRef,
		"branch_ref":     request.BranchRef,
		"write_set":      request.WriteSet,
		"evidence_refs":  compactWorktreeStringsV0(request.EvidenceRefs),
	}
}
