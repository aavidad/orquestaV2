package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type CodexReceiptWorktreeBaselineRecorderPortV0 interface {
	CaptureCodexReceiptWorktreeBaselineV0(
		context.Context,
		CodexReceiptWorktreeBaselineRequestV0,
	) (CodexReceiptWorktreeBaselineResolutionV0, error)
}

type CodexReceiptWorktreeBaselineRequestV0 struct {
	DescriptorRef  string
	RunID          string
	AgentRef       string
	ProjectWorkDir string
	Spec           orquestaruntime.ExternalAgentLaunchSpecV0
}

type CodexReceiptWorktreeBaselineResolutionV0 struct {
	BaselineRef string
}

type CodexReceiptWorktreeBaselineRecorderV0 struct {
	SnapshotStore      orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	IgnorePrefixes     []string
	SnapshotReadBudget orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0
}

func (recorder CodexReceiptWorktreeBaselineRecorderV0) CaptureCodexReceiptWorktreeBaselineV0(
	ctx context.Context,
	request CodexReceiptWorktreeBaselineRequestV0,
) (CodexReceiptWorktreeBaselineResolutionV0, error) {
	if recorder.SnapshotStore == nil {
		return CodexReceiptWorktreeBaselineResolutionV0{},
			fmt.Errorf("codex_worktree_baseline: snapshot_store_required")
	}
	snapshot, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(
		ctx,
		orquestaruntimeworktree.WorktreeSnapshotRequestV0{
			SnapshotRef:    codexReceiptWorktreeBaselineRefV0(request.DescriptorRef),
			ProjectWorkDir: strings.TrimSpace(request.ProjectWorkDir),
			IgnorePrefixes: recorder.IgnorePrefixes,
			MaxFiles:       recorder.SnapshotReadBudget.MaxFiles,
			MaxFileBytes:   recorder.SnapshotReadBudget.MaxFileBytes,
			MaxTotalBytes:  recorder.SnapshotReadBudget.MaxTotalBytes,
		},
	)
	if len(issues) > 0 {
		return CodexReceiptWorktreeBaselineResolutionV0{},
			fmt.Errorf("codex_worktree_baseline: capture_failed: %s",
				codexWorktreeBaselineIssueSummaryV0(issues))
	}
	if err := recorder.SnapshotStore.RecordWorktreeSnapshotV0(ctx, snapshot); err != nil {
		return CodexReceiptWorktreeBaselineResolutionV0{},
			fmt.Errorf("codex_worktree_baseline: record_failed")
	}
	return CodexReceiptWorktreeBaselineResolutionV0{BaselineRef: snapshot.SnapshotRef}, nil
}

func codexWorktreeBaselineIssueSummaryV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) string {
	for _, issue := range issues {
		code := strings.TrimSpace(string(issue.Code))
		field := strings.TrimSpace(issue.Field)
		evidence := ""
		if len(issue.Evidence) > 0 {
			evidence = strings.TrimSpace(issue.Evidence[0])
		}
		parts := make([]string, 0, 3)
		if code != "" {
			parts = append(parts, "code="+code)
		}
		if field != "" {
			parts = append(parts, "field="+field)
		}
		if evidence != "" {
			parts = append(parts, "evidence="+evidence)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return "issue_unavailable"
}

func codexReceiptWorktreeBaselineRefV0(descriptorRef string) string {
	descriptorRef = strings.TrimSpace(descriptorRef)
	if descriptorRef == "" {
		return "worktree-snapshot-ref-codex-receipt"
	}
	return "worktree-snapshot-ref-" + descriptorRef
}
