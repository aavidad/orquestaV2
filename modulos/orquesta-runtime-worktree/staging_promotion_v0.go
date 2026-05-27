package orquestaruntimeworktree

import (
	"context"
	"path/filepath"
	"strings"
)

const (
	StagingPromotionResultSchemaVersionV0 = "worktree_staging_promotion.v0"

	StagingPromotionStatusPromotedV0    = "promoted"
	StagingPromotionStatusCleanV0       = "clean"
	StagingPromotionStatusBlockedV0     = "blocked"
	StagingPromotionStatusArchivedV0    = "archived"
	StagingPromotionStatusPendingPushV0 = "pending_push"
)

type StagingPromotionRequestV0 struct {
	PromotionRef   string   `json:"promotion_ref,omitempty"`
	ArchiveRef     string   `json:"archive_ref,omitempty"`
	RunRef         string   `json:"run_ref,omitempty"`
	ProjectRef     string   `json:"project_ref"`
	AppRef         string   `json:"app_ref,omitempty"`
	RepoRef        string   `json:"repo_ref"`
	WorktreeRef    string   `json:"worktree_ref"`
	BranchRef      string   `json:"branch_ref"`
	ProjectWorkDir string   `json:"project_work_dir,omitempty"`
	ArchiveDir     string   `json:"archive_dir,omitempty"`
	CommitMessage  string   `json:"commit_message,omitempty"`
	WriteSet       []string `json:"write_set,omitempty"`
	AllowPush      bool     `json:"allow_push,omitempty"`
	RemoteName     string   `json:"remote_name,omitempty"`
	RemoteBranch   string   `json:"remote_branch,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type StagingPromotionResultV0 struct {
	SchemaVersion     string                                    `json:"schema_version"`
	Status            string                                    `json:"status"`
	PromotionRef      string                                    `json:"promotion_ref,omitempty"`
	ArchiveRef        string                                    `json:"archive_ref,omitempty"`
	RunRef            string                                    `json:"run_ref,omitempty"`
	ProjectRef        string                                    `json:"project_ref,omitempty"`
	AppRef            string                                    `json:"app_ref,omitempty"`
	RepoRef           string                                    `json:"repo_ref,omitempty"`
	WorktreeRef       string                                    `json:"worktree_ref,omitempty"`
	BranchRef         string                                    `json:"branch_ref,omitempty"`
	CommitRef         string                                    `json:"commit_ref,omitempty"`
	CommitShortRef    string                                    `json:"commit_short_ref,omitempty"`
	ChangedPaths      []string                                  `json:"changed_paths,omitempty"`
	Retryable         bool                                      `json:"retryable,omitempty"`
	EvidenceRefs      []string                                  `json:"evidence_refs,omitempty"`
	ExclusionReceipts []WorktreeLocalArtifactExclusionReceiptV0 `json:"exclusion_receipts,omitempty"`
	Issues            []WorktreeIssueV0                         `json:"issues,omitempty"`
}

type GitStagingPromotionConnectorV0 struct {
	VCS GitAppVCSConnectorV0
}

func (connector GitStagingPromotionConnectorV0) PromoteStagingWorktreeV0(
	ctx context.Context,
	request StagingPromotionRequestV0,
) (StagingPromotionResultV0, []WorktreeIssueV0) {
	request = normalizeStagingPromotionRequestV0(request)
	if issues := validateStagingPromotionRequestV0(request, false); len(issues) > 0 {
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	entries, issues := connector.gitStatusEntriesV0(ctx, request.ProjectWorkDir)
	if len(issues) > 0 {
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, nil, issues), issues
	}
	productEntries, controlIssues := stagingPromotionProductEntriesV0(entries)
	changed := stagingPromotionChangedPathsV0(productEntries)
	if issues := stagingPromotionWriteSetIssuesV0(entries, request.WriteSet); len(issues) > 0 {
		allIssues := append(issues, controlIssues...)
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, changed, allIssues), issues
	}
	if len(changed) == 0 {
		if request.AllowPush {
			return connector.pushCleanPromotionV0(ctx, request, controlIssues)
		}
		return connector.cleanPromotionResultV0(ctx, request, nil, controlIssues)
	}
	addArgs := append([]string{"add", "--"}, changed...)
	if _, issue := connector.vcsV0().gitOutputV0(ctx, request.ProjectWorkDir, addArgs...); issue != nil {
		issues := []WorktreeIssueV0{stagingPromotionGitIssueV0("git.add", issue.Evidence...)}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, changed, issues), issues
	}
	if _, issue := connector.vcsV0().gitOutputV0(ctx, request.ProjectWorkDir, "commit", "-m", request.CommitMessage); issue != nil {
		issues := []WorktreeIssueV0{stagingPromotionGitIssueV0("git.commit", issue.Evidence...)}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, changed, issues), issues
	}
	result, issues := connector.cleanPromotionResultV0(ctx, request, changed, controlIssues)
	result.Status = StagingPromotionStatusPromotedV0
	if len(issues) > 0 || !request.AllowPush {
		return result, issues
	}
	return connector.pushPromotedResultV0(ctx, request, result)
}

func (connector GitStagingPromotionConnectorV0) vcsV0() GitAppVCSConnectorV0 {
	return connector.VCS
}

func normalizeStagingPromotionRequestV0(request StagingPromotionRequestV0) StagingPromotionRequestV0 {
	request.PromotionRef = strings.TrimSpace(request.PromotionRef)
	request.ArchiveRef = strings.TrimSpace(request.ArchiveRef)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.RepoRef = strings.TrimSpace(request.RepoRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.ArchiveDir = strings.TrimSpace(request.ArchiveDir)
	request.CommitMessage = strings.TrimSpace(request.CommitMessage)
	request.RemoteName = strings.TrimSpace(request.RemoteName)
	request.RemoteBranch = strings.TrimSpace(request.RemoteBranch)
	request.WriteSet, _ = normalizeWorktreePathListV0(request.WriteSet, true)
	request.EvidenceRefs = compactWorktreeStringsV0(request.EvidenceRefs)
	return request
}

func validateStagingPromotionRequestV0(request StagingPromotionRequestV0, archive bool) []WorktreeIssueV0 {
	var issues []WorktreeIssueV0
	for _, item := range []struct {
		field string
		value string
	}{
		{"project_ref", request.ProjectRef},
		{"repo_ref", request.RepoRef},
		{"worktree_ref", request.WorktreeRef},
		{"branch_ref", request.BranchRef},
	} {
		issues = append(issues, validateWorktreeOpaqueRefV0(item.field, item.value, true)...)
	}
	if !archive {
		issues = append(issues, validateWorktreeOpaqueRefV0("promotion_ref", request.PromotionRef, true)...)
		if request.CommitMessage == "" {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "commit_message"))
		}
		if issue := validateWorktreeRootV0(request.ProjectWorkDir); issue != nil {
			issues = append(issues, *issue)
		}
	}
	if archive {
		issues = append(issues, validateWorktreeOpaqueRefV0("archive_ref", request.ArchiveRef, true)...)
		issues = append(issues, validateWorktreeOpaqueRefV0("promotion_ref", request.PromotionRef, true)...)
		if request.ArchiveDir == "" || !filepath.IsAbs(request.ArchiveDir) {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "archive_dir"))
		}
	}
	if len(request.WriteSet) == 0 {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "write_set"))
	}
	return issues
}

func (connector GitStagingPromotionConnectorV0) gitStatusEntriesV0(
	ctx context.Context,
	repo string,
) ([]stagingPromotionGitStatusEntryV0, []WorktreeIssueV0) {
	raw, issue := connector.vcsV0().gitOutputRawV0(ctx, repo, "status", "--porcelain", "--untracked-files=all")
	if issue != nil {
		return nil, []WorktreeIssueV0{stagingPromotionGitIssueFromAppVCSV0("git.status", *issue)}
	}
	if issue := connector.vcsV0().gitStatusPathBudgetIssueV0(parseAppVCSStatusPathsV0(raw)); issue != nil {
		return nil, []WorktreeIssueV0{stagingPromotionGitIssueFromAppVCSV0("git.status", *issue)}
	}
	return parseStagingPromotionGitStatusV0(raw), nil
}

type stagingPromotionGitStatusEntryV0 struct {
	Code string
	Path string
}

func parseStagingPromotionGitStatusV0(raw string) []stagingPromotionGitStatusEntryV0 {
	var out []stagingPromotionGitStatusEntryV0
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		path := filepath.ToSlash(strings.TrimSpace(line[3:]))
		if parts := strings.Split(path, " -> "); len(parts) == 2 {
			path = strings.TrimSpace(parts[1])
		}
		out = append(out, stagingPromotionGitStatusEntryV0{Code: strings.TrimSpace(line[:2]), Path: path})
	}
	return out
}

func (connector GitStagingPromotionConnectorV0) cleanPromotionResultV0(
	ctx context.Context,
	request StagingPromotionRequestV0,
	changed []string,
	controlIssues []WorktreeIssueV0,
) (StagingPromotionResultV0, []WorktreeIssueV0) {
	head, issue := connector.vcsV0().gitOutputV0(ctx, request.ProjectWorkDir, "rev-parse", "HEAD")
	if issue != nil {
		issues := []WorktreeIssueV0{stagingPromotionGitIssueV0("git.rev_parse", issue.Evidence...)}
		return newStagingPromotionResultV0(request, StagingPromotionStatusBlockedV0, changed, issues), issues
	}
	result := newStagingPromotionResultV0(request, StagingPromotionStatusCleanV0, changed, controlIssues)
	result.CommitRef = strings.TrimSpace(head)
	result.CommitShortRef = shortCommitRefV0(result.CommitRef)
	return result, nil
}

func (connector GitStagingPromotionConnectorV0) pushCleanPromotionV0(
	ctx context.Context,
	request StagingPromotionRequestV0,
	controlIssues []WorktreeIssueV0,
) (StagingPromotionResultV0, []WorktreeIssueV0) {
	result, issues := connector.cleanPromotionResultV0(ctx, request, nil, controlIssues)
	if len(issues) > 0 {
		return result, issues
	}
	return connector.pushPromotedResultV0(ctx, request, result)
}

func (connector GitStagingPromotionConnectorV0) pushPromotedResultV0(
	ctx context.Context,
	request StagingPromotionRequestV0,
	result StagingPromotionResultV0,
) (StagingPromotionResultV0, []WorktreeIssueV0) {
	args := []string{"push"}
	if request.RemoteName != "" && request.RemoteBranch != "" {
		args = append(args, request.RemoteName, "HEAD:"+request.RemoteBranch)
	}
	if _, issue := connector.vcsV0().gitOutputV0(ctx, request.ProjectWorkDir, args...); issue != nil {
		result.Status = StagingPromotionStatusPendingPushV0
		result.Retryable = true
		issues := []WorktreeIssueV0{stagingPromotionGitIssueV0("git.push", issue.Evidence...)}
		result.Issues = issues
		return result, issues
	}
	return result, nil
}

func newStagingPromotionResultV0(
	request StagingPromotionRequestV0,
	status string,
	changed []string,
	issues []WorktreeIssueV0,
) StagingPromotionResultV0 {
	result := StagingPromotionResultV0{
		SchemaVersion: StagingPromotionResultSchemaVersionV0,
		Status:        status,
		PromotionRef:  request.PromotionRef,
		ArchiveRef:    request.ArchiveRef,
		RunRef:        request.RunRef,
		ProjectRef:    request.ProjectRef,
		AppRef:        request.AppRef,
		RepoRef:       request.RepoRef,
		WorktreeRef:   request.WorktreeRef,
		BranchRef:     request.BranchRef,
		ChangedPaths:  compactWorktreeStringsV0(changed),
		EvidenceRefs: compactWorktreeStringsV0(append(
			[]string{"evidence-ref-worktree-staging-promotion-v0"},
			request.EvidenceRefs...,
		)),
		Issues: issues,
	}
	result.ExclusionReceipts = worktreeLocalArtifactReceiptsV0(worktreeControlPathIssueEvidenceV0(issues))
	return result
}

func stagingPromotionGitIssueV0(field string, evidence ...string) WorktreeIssueV0 {
	return worktreeIssueV0(WorktreeIssueFilesystemV0, field, evidence...)
}

func stagingPromotionGitIssueFromAppVCSV0(field string, issue AppVCSIssueV0) WorktreeIssueV0 {
	code := WorktreeIssueFilesystemV0
	switch issue.Code {
	case AppVCSIssueGitOutputTooLargeV0:
		code = WorktreeIssueGitOutputTooLargeV0
	case AppVCSIssueGitStatusTooManyPathsV0:
		code = WorktreeIssueGitStatusTooManyPathsV0
	case AppVCSIssueGitCommandTimeoutV0:
		code = WorktreeIssueGitCommandTimeoutV0
	}
	return worktreeIssueV0(code, field, issue.Evidence...)
}
