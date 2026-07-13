package orquestaruntimeworktree

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
)

const GoalWorkspaceIntegrationSchemaVersionV0 = "goal_workspace_integration.v0"

const (
	goalWorkspaceIntegrationGitUserNameV0  = "Orquesta Integration"
	goalWorkspaceIntegrationGitUserEmailV0 = "orquesta-integration@localhost.invalid"
)

const (
	GoalWorkspaceIntegrationStatusIntegratedV0 = "integrated"
	GoalWorkspaceIntegrationStatusReplayedV0   = "replayed"
	GoalWorkspaceIntegrationStatusBlockedV0    = "blocked"
)

type GoalWorkspaceIntegrationRequestV0 struct {
	IntegrationRef         string   `json:"integration_ref"`
	SourceWorkspaceDir     string   `json:"source_workspace_dir"`
	CanonicalWorkDir       string   `json:"canonical_work_dir"`
	BaseRevision           string   `json:"base_revision"`
	ExpectedParentRevision string   `json:"expected_parent_revision"`
	WriteSet               []string `json:"write_set"`
	CommitMessage          string   `json:"commit_message"`
	ReceiptDir             string   `json:"receipt_dir"`
}

type GoalWorkspaceIntegrationResultV0 struct {
	SchemaVersion          string            `json:"schema_version"`
	Status                 string            `json:"status"`
	IntegrationRef         string            `json:"integration_ref,omitempty"`
	BaseRevision           string            `json:"base_revision,omitempty"`
	ExpectedParentRevision string            `json:"expected_parent_revision,omitempty"`
	SourceCommit           string            `json:"source_commit,omitempty"`
	IntegratedCommit       string            `json:"integrated_commit,omitempty"`
	ChangedPaths           []string          `json:"changed_paths,omitempty"`
	EvidenceRefs           []string          `json:"evidence_refs,omitempty"`
	Issues                 []WorktreeIssueV0 `json:"issues,omitempty"`
}

type GoalWorkspaceIntegrationPortV0 interface {
	IntegrateGoalWorkspaceV0(context.Context, GoalWorkspaceIntegrationRequestV0) (GoalWorkspaceIntegrationResultV0, []WorktreeIssueV0)
}

type GitGoalWorkspaceIntegrationConnectorV0 struct {
	VCS GitAppVCSConnectorV0
}

type goalWorkspaceIntegrationReceiptV0 struct {
	SchemaVersion          string   `json:"schema_version"`
	IntegrationRef         string   `json:"integration_ref"`
	BaseRevision           string   `json:"base_revision"`
	ExpectedParentRevision string   `json:"expected_parent_revision"`
	SourceCommit           string   `json:"source_commit"`
	IntegratedCommit       string   `json:"integrated_commit"`
	WriteSet               []string `json:"write_set"`
	CommitMessage          string   `json:"commit_message"`
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) IntegrateGoalWorkspaceV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) (GoalWorkspaceIntegrationResultV0, []WorktreeIssueV0) {
	if ctx == nil {
		ctx = context.Background()
	}
	request, issues := normalizeGoalWorkspaceIntegrationRequestV0(request)
	if len(issues) > 0 {
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	if err := os.MkdirAll(request.ReceiptDir, 0o700); err != nil {
		issues := []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("receipt_dir")}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	lock, err := os.OpenFile(goalWorkspaceIntegrationLockPathV0(request), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		issues := []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("receipt_lock")}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceLockedV0, "receipt_dir")}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	base, issue := connector.resolveGoalWorkspaceIntegrationBaseV0(ctx, request)
	if issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	request.BaseRevision = base
	receiptPath := goalWorkspaceIntegrationReceiptPathV0(request)
	receipt, receiptFound, receiptIssues := loadGoalWorkspaceIntegrationReceiptV0(receiptPath)
	if receiptFound && request.ExpectedParentRevision == "" {
		request.ExpectedParentRevision = strings.TrimSpace(receipt.ExpectedParentRevision)
		if request.ExpectedParentRevision == "" {
			issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integration_receipt")}
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", receipt.IntegratedCommit, nil, issues), issues
		}
	}
	expectedParent, issue := connector.resolveGoalWorkspaceExpectedParentV0(ctx, request)
	if issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, "", "", nil, issues), issues
	}
	request.ExpectedParentRevision = expectedParent
	sourceCommit, changedPaths, issues := connector.promoteGoalWorkspaceSourceV0(ctx, request)
	if len(issues) > 0 {
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if issue := connector.validateGoalWorkspaceSourceCommitV0(ctx, request, sourceCommit); issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if issue := connector.requireGoalWorkspaceCommitAncestorV0(ctx, request.CanonicalWorkDir, request.BaseRevision, "base_revision"); issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if receiptFound || len(receiptIssues) > 0 {
		if len(receiptIssues) > 0 {
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, receiptIssues), receiptIssues
		}
		if issue := validateGoalWorkspaceIntegrationReceiptV0(request, sourceCommit, receipt); issue != nil {
			issues := []WorktreeIssueV0{*issue}
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, receipt.IntegratedCommit, changedPaths, issues), issues
		}
		if issue := connector.validateGoalWorkspaceReceiptIntegrationV0(ctx, request, receipt); issue != nil {
			issues := []WorktreeIssueV0{*issue}
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, receipt.IntegratedCommit, changedPaths, issues), issues
		}
		if issue := connector.requireGoalWorkspaceCommitAncestorV0(ctx, request.CanonicalWorkDir, receipt.IntegratedCommit, "integrated_commit"); issue != nil {
			issues := []WorktreeIssueV0{*issue}
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, receipt.IntegratedCommit, changedPaths, issues), issues
		}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusReplayedV0, sourceCommit, receipt.IntegratedCommit, changedPaths, nil), nil
	}
	canonicalHead, issue := connector.resolveGoalWorkspaceCanonicalHeadV0(ctx, request.CanonicalWorkDir)
	if issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if integratedCommit, issue := connector.findGoalWorkspaceCherryPickV0(ctx, request.CanonicalWorkDir, sourceCommit); issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	} else if integratedCommit != "" {
		if issue := connector.validateGoalWorkspaceRecoveredIntegrationV0(ctx, request, integratedCommit, canonicalHead); issue != nil {
			issues := []WorktreeIssueV0{*issue}
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, integratedCommit, changedPaths, issues), issues
		}
		receipt := newGoalWorkspaceIntegrationReceiptV0(request, sourceCommit, integratedCommit)
		if receiptIssues := writeGoalWorkspaceIntegrationReceiptV0(receiptPath, receipt); len(receiptIssues) > 0 {
			return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, integratedCommit, changedPaths, receiptIssues), receiptIssues
		}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusReplayedV0, sourceCommit, integratedCommit, changedPaths, nil), nil
	}
	if issue := connector.requireGoalWorkspaceCanonicalCleanV0(ctx, request); issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if issue := connector.requireGoalWorkspaceCanonicalHeadV0(ctx, request); issue != nil {
		issues := []WorktreeIssueV0{*issue}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	if _, gitIssue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir,
		"-c", "user.name="+goalWorkspaceIntegrationGitUserNameV0,
		"-c", "user.email="+goalWorkspaceIntegrationGitUserEmailV0,
		"cherry-pick", "-x", sourceCommit,
	); gitIssue != nil {
		_, abortIssue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "cherry-pick", "--abort")
		issues := []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "cherry_pick", gitIssue.Evidence...)}
		if abortIssue != nil {
			issues = append(issues, goalWorkspaceGitIssueV0("git.cherry_pick_abort", *abortIssue))
		}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	integratedCommit, gitIssue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "rev-parse", "--verify", "HEAD^{commit}")
	if gitIssue != nil {
		issues := []WorktreeIssueV0{goalWorkspaceGitIssueV0("git.rev_parse", *gitIssue)}
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, "", changedPaths, issues), issues
	}
	receipt = newGoalWorkspaceIntegrationReceiptV0(request, sourceCommit, integratedCommit)
	if receiptIssues := writeGoalWorkspaceIntegrationReceiptV0(receiptPath, receipt); len(receiptIssues) > 0 {
		return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusBlockedV0, sourceCommit, integratedCommit, changedPaths, receiptIssues), receiptIssues
	}
	return newGoalWorkspaceIntegrationResultV0(request, GoalWorkspaceIntegrationStatusIntegratedV0, sourceCommit, integratedCommit, changedPaths, nil), nil
}

func normalizeGoalWorkspaceIntegrationRequestV0(request GoalWorkspaceIntegrationRequestV0) (GoalWorkspaceIntegrationRequestV0, []WorktreeIssueV0) {
	request.IntegrationRef = strings.TrimSpace(request.IntegrationRef)
	request.SourceWorkspaceDir = strings.TrimSpace(request.SourceWorkspaceDir)
	request.CanonicalWorkDir = strings.TrimSpace(request.CanonicalWorkDir)
	request.BaseRevision = strings.TrimSpace(request.BaseRevision)
	request.ExpectedParentRevision = strings.TrimSpace(request.ExpectedParentRevision)
	request.CommitMessage = strings.TrimSpace(request.CommitMessage)
	request.ReceiptDir = strings.TrimSpace(request.ReceiptDir)
	writeSet, issues := normalizeWorktreePathListV0(request.WriteSet, true)
	request.WriteSet = writeSet
	issues = append(issues, worktreeControlPathIssuesV0(request.WriteSet)...)
	issues = append(issues, validateWorktreeOpaqueRefV0("integration_ref", request.IntegrationRef, true)...)
	for _, item := range []struct{ field, value string }{
		{"source_workspace_dir", request.SourceWorkspaceDir},
		{"canonical_work_dir", request.CanonicalWorkDir},
		{"receipt_dir", request.ReceiptDir},
	} {
		if item.value == "" || !filepath.IsAbs(item.value) {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, item.field))
			continue
		}
		cleaned, err := filepath.Abs(item.value)
		if err != nil {
			issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, item.field))
			continue
		}
		switch item.field {
		case "source_workspace_dir":
			request.SourceWorkspaceDir = filepath.Clean(cleaned)
		case "canonical_work_dir":
			request.CanonicalWorkDir = filepath.Clean(cleaned)
		case "receipt_dir":
			request.ReceiptDir = filepath.Clean(cleaned)
		}
	}
	if request.SourceWorkspaceDir == request.CanonicalWorkDir || goalWorkspaceIntegrationSameDirectoryV0(request.SourceWorkspaceDir, request.CanonicalWorkDir) {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "source_workspace_dir_canonical_work_dir"))
	}
	if request.BaseRevision == "" || strings.ContainsAny(request.BaseRevision, "\x00\r\n") {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "base_revision"))
	}
	if request.ExpectedParentRevision != "" && strings.ContainsAny(request.ExpectedParentRevision, "\x00\r\n") {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "expected_parent_revision"))
	}
	if request.CommitMessage == "" {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "commit_message"))
	}
	if len(request.WriteSet) == 0 {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "write_set"))
	}
	return request, issues
}

func goalWorkspaceIntegrationSameDirectoryV0(first string, second string) bool {
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) resolveGoalWorkspaceIntegrationBaseV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) (string, *WorktreeIssueV0) {
	base, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkspaceDir, "rev-parse", "--verify", "--end-of-options", request.BaseRevision+"^{commit}")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.rev_parse", *issue)
		return "", &result
	}
	canonicalBase, issue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "rev-parse", "--verify", "--end-of-options", request.BaseRevision+"^{commit}")
	if issue != nil || canonicalBase != base {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "base_revision")
		return "", &result
	}
	return base, nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) resolveGoalWorkspaceExpectedParentV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) (string, *WorktreeIssueV0) {
	expected := request.ExpectedParentRevision
	if expected == "" {
		return connector.resolveGoalWorkspaceCanonicalHeadV0(ctx, request.CanonicalWorkDir)
	}
	resolved, issue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "rev-parse", "--verify", "--end-of-options", expected+"^{commit}")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.rev_parse", *issue)
		return "", &result
	}
	return resolved, nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) promoteGoalWorkspaceSourceV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) (string, []string, []WorktreeIssueV0) {
	entries, issues := (GitStagingPromotionConnectorV0{VCS: connector.VCS}).gitStatusEntriesV0(ctx, request.SourceWorkspaceDir)
	if len(issues) > 0 {
		return "", nil, issues
	}
	productEntries, controlIssues := stagingPromotionProductEntriesV0(entries)
	changedPaths := stagingPromotionChangedPathsV0(productEntries)
	if writeSetIssues := stagingPromotionWriteSetIssuesV0(entries, request.WriteSet); len(writeSetIssues) > 0 {
		return "", changedPaths, append(writeSetIssues, controlIssues...)
	}
	if len(changedPaths) > 0 {
		addArgs := append([]string{"add", "--"}, changedPaths...)
		if _, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkspaceDir, addArgs...); issue != nil {
			return "", changedPaths, []WorktreeIssueV0{goalWorkspaceGitIssueV0("git.add", *issue)}
		}
		if _, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkspaceDir,
			"-c", "user.name="+goalWorkspaceIntegrationGitUserNameV0,
			"-c", "user.email="+goalWorkspaceIntegrationGitUserEmailV0,
			"commit", "-m", request.CommitMessage,
		); issue != nil {
			return "", changedPaths, []WorktreeIssueV0{goalWorkspaceGitIssueV0("git.commit", *issue)}
		}
	}
	head, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkspaceDir, "rev-parse", "--verify", "HEAD^{commit}")
	if issue != nil {
		return "", changedPaths, []WorktreeIssueV0{goalWorkspaceGitIssueV0("git.rev_parse", *issue)}
	}
	return head, changedPaths, nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) validateGoalWorkspaceSourceCommitV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
	sourceCommit string,
) *WorktreeIssueV0 {
	parent, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkspaceDir, "rev-parse", "--verify", "HEAD^{commit}^")
	if issue != nil || parent != request.BaseRevision || sourceCommit == request.BaseRevision {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "source_commit_parent")
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) requireGoalWorkspaceCanonicalCleanV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) *WorktreeIssueV0 {
	status, issue := connector.VCS.gitOutputRawV0(ctx, request.CanonicalWorkDir, "status", "--porcelain", "--untracked-files=all")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.status", *issue)
		return &result
	}
	if strings.TrimSpace(status) != "" {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "canonical_work_dir")
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) requireGoalWorkspaceCommitAncestorV0(
	ctx context.Context,
	canonicalWorkDir string,
	commit string,
	field string,
) *WorktreeIssueV0 {
	if _, issue := connector.VCS.gitOutputV0(ctx, canonicalWorkDir, "merge-base", "--is-ancestor", commit, "HEAD"); issue != nil {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, field)
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) resolveGoalWorkspaceCanonicalHeadV0(
	ctx context.Context,
	canonicalWorkDir string,
) (string, *WorktreeIssueV0) {
	head, issue := connector.VCS.gitOutputV0(ctx, canonicalWorkDir, "rev-parse", "--verify", "HEAD^{commit}")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.rev_parse", *issue)
		return "", &result
	}
	return head, nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) requireGoalWorkspaceCanonicalHeadV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
) *WorktreeIssueV0 {
	head, issue := connector.resolveGoalWorkspaceCanonicalHeadV0(ctx, request.CanonicalWorkDir)
	if issue != nil {
		return issue
	}
	if head != request.ExpectedParentRevision {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "expected_parent_revision")
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) validateGoalWorkspaceRecoveredIntegrationV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
	integratedCommit string,
	canonicalHead string,
) *WorktreeIssueV0 {
	parent, issue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "rev-parse", "--verify", integratedCommit+"^{commit}^")
	if issue != nil || parent != request.ExpectedParentRevision {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integrated_commit_parent")
		return &result
	}
	if _, issue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "merge-base", "--is-ancestor", integratedCommit, canonicalHead); issue != nil {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integrated_commit")
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) validateGoalWorkspaceReceiptIntegrationV0(
	ctx context.Context,
	request GoalWorkspaceIntegrationRequestV0,
	receipt goalWorkspaceIntegrationReceiptV0,
) *WorktreeIssueV0 {
	parent, issue := connector.VCS.gitOutputV0(ctx, request.CanonicalWorkDir, "rev-parse", "--verify", receipt.IntegratedCommit+"^{commit}^")
	if issue != nil || parent != receipt.ExpectedParentRevision {
		result := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integrated_commit_parent")
		return &result
	}
	return nil
}

func (connector GitGoalWorkspaceIntegrationConnectorV0) findGoalWorkspaceCherryPickV0(
	ctx context.Context,
	canonicalWorkDir string,
	sourceCommit string,
) (string, *WorktreeIssueV0) {
	trailer := "(cherry picked from commit " + sourceCommit + ")"
	raw, issue := connector.VCS.gitOutputV0(ctx, canonicalWorkDir, "log", "--format=%H", "--fixed-strings", "--grep="+trailer, "HEAD")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.log", *issue)
		return "", &result
	}
	for _, candidate := range strings.Fields(raw) {
		body, issue := connector.VCS.gitOutputV0(ctx, canonicalWorkDir, "show", "-s", "--format=%B", candidate)
		if issue != nil {
			result := goalWorkspaceGitIssueV0("git.show", *issue)
			return "", &result
		}
		if goalWorkspaceCommitHasCherryPickTrailerV0(body, trailer) {
			return candidate, nil
		}
	}
	return "", nil
}

func goalWorkspaceCommitHasCherryPickTrailerV0(body string, trailer string) bool {
	for _, line := range strings.Split(body, "\n") {
		if line == trailer {
			return true
		}
	}
	return false
}

func goalWorkspaceIntegrationLockPathV0(request GoalWorkspaceIntegrationRequestV0) string {
	return filepath.Join(request.ReceiptDir, ".goal-workspace-integration.lock")
}

func goalWorkspaceIntegrationReceiptPathV0(request GoalWorkspaceIntegrationRequestV0) string {
	return filepath.Join(request.ReceiptDir, request.IntegrationRef+".json")
}

func loadGoalWorkspaceIntegrationReceiptV0(path string) (goalWorkspaceIntegrationReceiptV0, bool, []WorktreeIssueV0) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return goalWorkspaceIntegrationReceiptV0{}, false, nil
	}
	if err != nil {
		return goalWorkspaceIntegrationReceiptV0{}, false, []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("integration_receipt")}
	}
	var receipt goalWorkspaceIntegrationReceiptV0
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return goalWorkspaceIntegrationReceiptV0{}, false, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integration_receipt")}
	}
	return receipt, true, nil
}

func validateGoalWorkspaceIntegrationReceiptV0(
	request GoalWorkspaceIntegrationRequestV0,
	sourceCommit string,
	receipt goalWorkspaceIntegrationReceiptV0,
) *WorktreeIssueV0 {
	if receipt.SchemaVersion != GoalWorkspaceIntegrationSchemaVersionV0 ||
		receipt.IntegrationRef != request.IntegrationRef ||
		receipt.BaseRevision != request.BaseRevision ||
		receipt.ExpectedParentRevision != request.ExpectedParentRevision ||
		receipt.SourceCommit != sourceCommit ||
		receipt.CommitMessage != request.CommitMessage ||
		!reflect.DeepEqual(receipt.WriteSet, request.WriteSet) ||
		receipt.IntegratedCommit == "" {
		issue := worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "integration_receipt")
		return &issue
	}
	return nil
}

func newGoalWorkspaceIntegrationReceiptV0(
	request GoalWorkspaceIntegrationRequestV0,
	sourceCommit string,
	integratedCommit string,
) goalWorkspaceIntegrationReceiptV0 {
	return goalWorkspaceIntegrationReceiptV0{
		SchemaVersion:          GoalWorkspaceIntegrationSchemaVersionV0,
		IntegrationRef:         request.IntegrationRef,
		BaseRevision:           request.BaseRevision,
		ExpectedParentRevision: request.ExpectedParentRevision,
		SourceCommit:           sourceCommit,
		IntegratedCommit:       integratedCommit,
		WriteSet:               request.WriteSet,
		CommitMessage:          request.CommitMessage,
	}
}

func writeGoalWorkspaceIntegrationReceiptV0(path string, receipt goalWorkspaceIntegrationReceiptV0) []WorktreeIssueV0 {
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("integration_receipt")}
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".goal-workspace-integration-*.tmp")
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("integration_receipt")}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(raw)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err == nil {
		if directory, openErr := os.Open(filepath.Dir(path)); openErr == nil {
			err = directory.Sync()
			_ = directory.Close()
		}
	}
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("integration_receipt")}
	}
	return nil
}

func newGoalWorkspaceIntegrationResultV0(
	request GoalWorkspaceIntegrationRequestV0,
	status string,
	sourceCommit string,
	integratedCommit string,
	changedPaths []string,
	issues []WorktreeIssueV0,
) GoalWorkspaceIntegrationResultV0 {
	return GoalWorkspaceIntegrationResultV0{
		SchemaVersion:          GoalWorkspaceIntegrationSchemaVersionV0,
		Status:                 status,
		IntegrationRef:         request.IntegrationRef,
		BaseRevision:           request.BaseRevision,
		ExpectedParentRevision: request.ExpectedParentRevision,
		SourceCommit:           sourceCommit,
		IntegratedCommit:       integratedCommit,
		ChangedPaths:           compactWorktreeStringsV0(changedPaths),
		EvidenceRefs: compactWorktreeStringsV0([]string{
			"evidence-ref-goal-workspace-integration-v0",
			"evidence-ref-goal-workspace-integration:" + request.IntegrationRef,
		}),
		Issues: issues,
	}
}
