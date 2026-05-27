package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type codexStackReviewGateLineBudgetEvidenceV0 struct {
	Inner  orquestaruntimecodexdelivery.CodexReviewGateFileEvidenceResultProviderPortV0
	Config ReviewGateConfigV0
}

func codexStackReviewGateFileEvidenceV0(
	config ReviewGateConfigV0,
) orquestaruntimecodexdelivery.CodexReviewGateFileEvidenceResultProviderPortV0 {
	if !config.StrictGoLineBudget {
		if config.LineBudgetSnapshotStore != nil {
			return codexStackReviewGateLineBudgetEvidenceV0{
				Inner:  config.FileEvidence,
				Config: config,
			}
		}
		return config.FileEvidence
	}
	return codexStackReviewGateLineBudgetEvidenceV0{
		Inner:  config.FileEvidence,
		Config: config,
	}
}

func (provider codexStackReviewGateLineBudgetEvidenceV0) BuildCodexReviewGateFileEvidenceV0(
	ctx context.Context,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) (orquestaruntimecodexdelivery.CodexReviewGateFileEvidenceV0, error) {
	evidence, err := provider.Inner.BuildCodexReviewGateFileEvidenceV0(ctx, descriptor, ack)
	if err != nil {
		return evidence, err
	}
	baselineSnapshot, ok := provider.baselineSnapshotV0(ctx, descriptor.WorktreeBaselineRef)
	evidence.Issues = append(evidence.Issues, provider.destructiveIssuesV0(
		ctx,
		descriptor,
		ack,
		baselineSnapshot,
		ok,
	)...)
	baseline := worktreeLineCountsFromSnapshotV0(baselineSnapshot)
	for index := range evidence.Files {
		if !strings.HasSuffix(strings.TrimSpace(evidence.Files[index].Path), ".go") {
			continue
		}
		evidence.Files[index].LineCountSource = "project_file"
		evidence.Files[index].BaselineLineCount = baseline[strings.TrimSpace(evidence.Files[index].Path)]
	}
	return evidence, nil
}

func (provider codexStackReviewGateLineBudgetEvidenceV0) baselineSnapshotV0(
	ctx context.Context,
	baselineRef string,
) (orquestaruntimeworktree.WorktreeSnapshotV0, bool) {
	if provider.Config.LineBudgetSnapshotStore == nil || strings.TrimSpace(baselineRef) == "" {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, false
	}
	snapshot, err := provider.Config.LineBudgetSnapshotStore.LoadWorktreeSnapshotV0(ctx, baselineRef)
	if err != nil {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, false
	}
	return snapshot, true
}

func (provider codexStackReviewGateLineBudgetEvidenceV0) destructiveIssuesV0(
	ctx context.Context,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	baseline orquestaruntimeworktree.WorktreeSnapshotV0,
	ok bool,
) []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0 {
	if !ok || strings.TrimSpace(descriptor.ProjectWorkDir) == "" {
		return nil
	}
	policy := codexStackReviewGatePolicyFromConfigV0(provider.Config)
	_, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(ctx, orquestaruntimeworktree.WorktreeVerifyRequestV0{
		Baseline:              baseline,
		ProjectWorkDir:        strings.TrimSpace(descriptor.ProjectWorkDir),
		WriteSet:              policy.EffectiveWriteSetV0(descriptor.Spec.AgentPacket.Task.WriteSet),
		IgnorePrefixes:        codexStackWorktreeIgnorePrefixesV0(),
		MaxSnapshotFiles:      provider.Config.SnapshotReadBudget.MaxFiles,
		MaxSnapshotFileBytes:  provider.Config.SnapshotReadBudget.MaxFileBytes,
		MaxSnapshotTotalBytes: provider.Config.SnapshotReadBudget.MaxTotalBytes,
		StrictGoLineBudget:    policy.StrictGoLineBudgetForDeliveryV0(ack, descriptor.Spec.AgentPacket),
		MaxGoFileLines:        policy.MaxGoFileLinesV0(),
	})
	return codexStackReviewGateWorktreeIssuesV0(issues, ack.AckRef)
}

func worktreeLineCountsFromSnapshotV0(
	snapshot orquestaruntimeworktree.WorktreeSnapshotV0,
) map[string]int {
	out := map[string]int{}
	for _, file := range snapshot.Files {
		if path := strings.TrimSpace(file.Path); strings.HasSuffix(path, ".go") {
			out[path] = file.LineCount
		}
	}
	return out
}

func codexStackReviewGateWorktreeIssuesV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
	deliveryRef string,
) []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0 {
	var out []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0
	for _, issue := range issues {
		code := codexStackReviewGateWorktreeIssueCodeV0(issue.Code)
		if code == "" {
			continue
		}
		out = append(out, orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(deliveryRef),
			Message: strings.Join(issue.Evidence, ","),
		})
	}
	return out
}

func codexStackReviewGateWorktreeIssueCodeV0(
	code orquestaruntimeworktree.WorktreeIssueCodeV0,
) string {
	switch code {
	case orquestaruntimeworktree.WorktreeIssueOutsideWriteSetV0:
		return "file_outside_write_set"
	case orquestaruntimeworktree.WorktreeIssueRemovedPathV0:
		return "removed_path"
	case orquestaruntimeworktree.WorktreeIssueTruncatedPathV0:
		return "truncated_path"
	case orquestaruntimeworktree.WorktreeIssueRenamedOrMovedV0:
		return "renamed_or_moved_path"
	case orquestaruntimeworktree.WorktreeIssueReplacedLargeV0:
		return "replaced_large_delta"
	case orquestaruntimeworktree.WorktreeIssueGoLineBudgetV0:
		return orquestaautoprogramming.AutoprogrammingReviewGateStrictLineBudgetIssueV0
	case orquestaruntimeworktree.WorktreeIssueSnapshotFileTooLargeV0:
		return "worktree_snapshot_file_too_large"
	case orquestaruntimeworktree.WorktreeIssueSnapshotTooManyFilesV0:
		return "worktree_snapshot_too_many_files"
	case orquestaruntimeworktree.WorktreeIssueSnapshotTooLargeV0:
		return "worktree_snapshot_too_large"
	case orquestaruntimeworktree.WorktreeIssueSnapshotUnreadableV0:
		return "worktree_snapshot_unreadable"
	default:
		return ""
	}
}
