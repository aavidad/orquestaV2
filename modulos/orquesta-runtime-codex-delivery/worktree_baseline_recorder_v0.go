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
	SnapshotStore  orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	IgnorePrefixes []string
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
		},
	)
	if len(issues) > 0 {
		return CodexReceiptWorktreeBaselineResolutionV0{},
			fmt.Errorf("codex_worktree_baseline: capture_failed")
	}
	if err := recorder.SnapshotStore.RecordWorktreeSnapshotV0(ctx, snapshot); err != nil {
		return CodexReceiptWorktreeBaselineResolutionV0{},
			fmt.Errorf("codex_worktree_baseline: record_failed")
	}
	return CodexReceiptWorktreeBaselineResolutionV0{BaselineRef: snapshot.SnapshotRef}, nil
}

func codexReceiptWorktreeBaselineRefV0(descriptorRef string) string {
	descriptorRef = strings.TrimSpace(descriptorRef)
	if descriptorRef == "" {
		return "worktree-snapshot-ref-codex-receipt"
	}
	return "worktree-snapshot-ref-" + descriptorRef
}
