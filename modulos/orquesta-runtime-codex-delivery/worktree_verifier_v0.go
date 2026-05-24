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
	SnapshotStore  orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	IgnorePrefixes []string
	Mode           CodexReceiptWorktreeVerificationModeV0
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
	if len(issues) > 0 {
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
		return nil, fmt.Errorf("codex_worktree_verification: snapshot_store_required")
	}
	if strings.TrimSpace(request.ProjectWorkDir) == "" ||
		strings.TrimSpace(request.WorktreeBaselineRef) == "" {
		return nil, fmt.Errorf("codex_worktree_verification: baseline_required")
	}
	baseline, err := verifier.SnapshotStore.LoadWorktreeSnapshotV0(ctx, request.WorktreeBaselineRef)
	if err != nil {
		return nil, fmt.Errorf("codex_worktree_verification: baseline_not_found")
	}
	result, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(
		ctx,
		orquestaruntimeworktree.WorktreeVerifyRequestV0{
			Baseline:       baseline,
			ProjectWorkDir: strings.TrimSpace(request.ProjectWorkDir),
			WriteSet:       request.Spec.AgentPacket.Task.WriteSet,
			IgnorePrefixes: verifier.IgnorePrefixes,
		},
	)
	if len(issues) > 0 {
		if codexReceiptWorktreeIssuesOnlyOutsideWriteSetV0(issues) {
			return compactCodexDeliveryRefsV0(append(
				result.EvidenceRefs,
				"gate-issue:file_outside_write_set",
			)), nil
		}
		return nil, fmt.Errorf("codex_worktree_verification: %s", issues[0].Code)
	}
	return compactCodexDeliveryRefsV0(result.EvidenceRefs), nil
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

func codexReceiptWorktreeIssuesOnlyOutsideWriteSetV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if issue.Code != orquestaruntimeworktree.WorktreeIssueOutsideWriteSetV0 {
			return false
		}
	}
	return true
}
