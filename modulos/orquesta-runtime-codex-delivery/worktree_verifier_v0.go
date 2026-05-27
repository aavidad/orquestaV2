package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type CodexReceiptWorktreeVerifierPortV0 interface {
	VerifyCodexReceiptWorktreeV0(
		context.Context,
		CodexReceiptWorktreeVerificationRequestV0,
	) error
}

type CodexReceiptWorktreeEvidenceVerifierPortV0 interface {
	VerifyCodexReceiptWorktreeEvidenceRefsV0(
		context.Context,
		CodexReceiptWorktreeVerificationRequestV0,
	) ([]string, error)
}

type CodexReceiptWorktreeVerificationRequestV0 struct {
	DescriptorRef       string
	RunID               string
	AgentRef            string
	ProjectWorkDir      string
	WorktreeBaselineRef string
	Spec                orquestaruntime.ExternalAgentLaunchSpecV0
	AckFiles            []string
}

type CodexReceiptWorktreeVerificationModeV0 string

const (
	CodexReceiptWorktreeStrictDiffV0 CodexReceiptWorktreeVerificationModeV0 = "strict_diff"
	CodexReceiptWorktreeAckFilesV0   CodexReceiptWorktreeVerificationModeV0 = "ack_files"
)

type CodexReceiptWorktreeVerifierV0 struct {
	SnapshotStore      orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	IgnorePrefixes     []string
	Mode               CodexReceiptWorktreeVerificationModeV0
	SnapshotReadBudget orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0
}

func (source CodexDeliveryObservationSourceV0) verifyDescriptorWorktreeV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
) ([]string, error) {
	if source.WorktreeVerifier == nil {
		return nil, nil
	}
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		strings.TrimSpace(descriptor.AckPath),
		descriptor.Spec,
	)
	if len(issues) > 0 && !codexReceiptAckIssuesOnlyReviewableFailedTestEvidenceV0(issues) {
		return nil, fmt.Errorf("codex_worktree_verification: ack_invalid")
	}
	request := CodexReceiptWorktreeVerificationRequestV0{
		DescriptorRef:       descriptor.DescriptorRef,
		RunID:               descriptor.RunID,
		AgentRef:            descriptor.AgentRef,
		ProjectWorkDir:      descriptor.ProjectWorkDir,
		WorktreeBaselineRef: descriptor.WorktreeBaselineRef,
		Spec:                descriptor.Spec,
		AckFiles:            append([]string(nil), ack.Files...),
	}
	if verifier, ok := source.WorktreeVerifier.(CodexReceiptWorktreeEvidenceVerifierPortV0); ok {
		return verifier.VerifyCodexReceiptWorktreeEvidenceRefsV0(ctx, request)
	}
	return nil, source.WorktreeVerifier.VerifyCodexReceiptWorktreeV0(ctx, request)
}

func (verifier CodexReceiptWorktreeVerifierV0) VerifyCodexReceiptWorktreeV0(
	ctx context.Context,
	request CodexReceiptWorktreeVerificationRequestV0,
) error {
	_, err := verifier.VerifyCodexReceiptWorktreeEvidenceRefsV0(ctx, request)
	return err
}

func (verifier CodexReceiptWorktreeVerifierV0) VerifyCodexReceiptWorktreeEvidenceRefsV0(
	ctx context.Context,
	request CodexReceiptWorktreeVerificationRequestV0,
) ([]string, error) {
	if verifier.modeV0() == CodexReceiptWorktreeAckFilesV0 {
		return nil, verifyCodexReceiptAckFilesV0(request)
	}
	if verifier.SnapshotStore == nil {
		return codexReceiptWorktreeFallbackAckFilesV0(request, "gate-issue:worktree_snapshot_store_missing")
	}
	if strings.TrimSpace(request.ProjectWorkDir) == "" ||
		strings.TrimSpace(request.WorktreeBaselineRef) == "" {
		return codexReceiptWorktreeFallbackAckFilesV0(request, "gate-issue:worktree_baseline_missing")
	}
	baseline, err := verifier.SnapshotStore.LoadWorktreeSnapshotV0(ctx, request.WorktreeBaselineRef)
	if err != nil {
		return codexReceiptWorktreeFallbackAckFilesV0(request, "gate-issue:worktree_baseline_missing")
	}
	result, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(
		ctx,
		orquestaruntimeworktree.WorktreeVerifyRequestV0{
			Baseline:              baseline,
			ProjectWorkDir:        strings.TrimSpace(request.ProjectWorkDir),
			WriteSet:              request.Spec.AgentPacket.Task.WriteSet,
			AckFiles:              request.AckFiles,
			IgnorePrefixes:        verifier.IgnorePrefixes,
			MaxSnapshotFiles:      verifier.SnapshotReadBudget.MaxFiles,
			MaxSnapshotFileBytes:  verifier.SnapshotReadBudget.MaxFileBytes,
			MaxSnapshotTotalBytes: verifier.SnapshotReadBudget.MaxTotalBytes,
		},
	)
	if len(issues) > 0 {
		if refs, ok := codexReceiptWorktreeSoftIssueRefsV0(issues); ok {
			return compactCodexDeliveryRefsV0(append(result.EvidenceRefs, refs...)), nil
		}
		return nil, fmt.Errorf("codex_worktree_verification: %s", issues[0].Code)
	}
	return compactCodexDeliveryRefsV0(result.EvidenceRefs), nil
}

func codexReceiptWorktreeFallbackAckFilesV0(
	request CodexReceiptWorktreeVerificationRequestV0,
	evidenceRef string,
) ([]string, error) {
	if err := verifyCodexReceiptAckFilesV0(request); err != nil {
		return nil, err
	}
	return compactCodexDeliveryRefsV0([]string{evidenceRef, "gate-issue:strict_diff_unavailable"}), nil
}

func (verifier CodexReceiptWorktreeVerifierV0) modeV0() CodexReceiptWorktreeVerificationModeV0 {
	if verifier.Mode == "" {
		return CodexReceiptWorktreeStrictDiffV0
	}
	return verifier.Mode
}

func verifyCodexReceiptAckFilesV0(request CodexReceiptWorktreeVerificationRequestV0) error {
	projectDir := strings.TrimSpace(request.ProjectWorkDir)
	if projectDir == "" {
		return fmt.Errorf("codex_worktree_verification: project_required")
	}
	for _, raw := range request.AckFiles {
		rel, ok := cleanCodexReceiptAckFilePathV0(raw)
		if !ok {
			return fmt.Errorf("codex_worktree_verification: ack_file_invalid")
		}
		info, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("codex_worktree_verification: ack_file_missing")
		}
		if info.IsDir() {
			return fmt.Errorf("codex_worktree_verification: ack_file_invalid")
		}
	}
	return nil
}

func cleanCodexReceiptAckFilePathV0(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.Contains(trimmed, "://") ||
		strings.HasPrefix(trimmed, "~") || strings.Contains(trimmed, "$HOME") ||
		filepath.IsAbs(trimmed) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func codexReceiptPathAllowedByWriteSetV0(path string, writeSet []string) bool {
	for _, raw := range writeSet {
		if strings.TrimSpace(raw) == "." {
			return true
		}
		allowed, ok := cleanCodexReceiptAckFilePathV0(raw)
		if !ok {
			continue
		}
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

func codexReceiptWorktreeSoftIssueRefsV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) ([]string, bool) {
	if len(issues) == 0 {
		return nil, false
	}
	refs := make([]string, 0, len(issues))
	for _, issue := range issues {
		switch issue.Code {
		case orquestaruntimeworktree.WorktreeIssueRemovedPathV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("removed_path", issue)...)
		case orquestaruntimeworktree.WorktreeIssueOutsideWriteSetV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("file_outside_write_set", issue)...)
		case orquestaruntimeworktree.WorktreeIssueTruncatedPathV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("truncated_path", issue)...)
		case orquestaruntimeworktree.WorktreeIssueRenamedOrMovedV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("renamed_or_moved_path", issue)...)
		case orquestaruntimeworktree.WorktreeIssueReplacedLargeV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("replaced_large_delta", issue)...)
		case orquestaruntimeworktree.WorktreeIssueAckFilesMismatchV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("ack_files_mismatch", issue)...)
		case orquestaruntimeworktree.WorktreeIssueGoLineBudgetV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("go_file_line_budget_exceeded", issue)...)
		case orquestaruntimeworktree.WorktreeIssueSnapshotFileTooLargeV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("worktree_snapshot_file_too_large", issue)...)
		case orquestaruntimeworktree.WorktreeIssueSnapshotTooManyFilesV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("worktree_snapshot_too_many_files", issue)...)
		case orquestaruntimeworktree.WorktreeIssueSnapshotTooLargeV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("worktree_snapshot_too_large", issue)...)
		case orquestaruntimeworktree.WorktreeIssueSnapshotUnreadableV0:
			refs = append(refs, codexReceiptWorktreeIssueEvidenceRefsV0("worktree_snapshot_unreadable", issue)...)
		default:
			return nil, false
		}
	}
	return compactCodexDeliveryRefsV0(refs), true
}

func codexReceiptWorktreeIssueEvidenceRefsV0(
	stem string,
	issue orquestaruntimeworktree.WorktreeIssueV0,
) []string {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return nil
	}
	refs := []string{"gate-issue:" + stem}
	for _, evidence := range issue.Evidence {
		evidence = strings.TrimSpace(evidence)
		if !codexReceiptWorktreeIssueEvidenceSafeV0(evidence) {
			continue
		}
		refs = append(refs, "gate-issue:"+stem+":"+evidence)
	}
	return compactCodexDeliveryRefsV0(refs)
}

func codexReceiptWorktreeIssueEvidenceSafeV0(value string) bool {
	if value == "" ||
		strings.Contains(value, "://") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$HOME") ||
		filepath.IsAbs(value) {
		return false
	}
	return true
}
