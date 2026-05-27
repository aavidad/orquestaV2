package orquestaappcodexstack

import (
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexStackWorktreeSnapshotBudgetV0(t *testing.T) {
	store := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	budget := orquestaruntimeworktree.WorktreeSnapshotReadBudgetFromLimitsV0(7, 8, 9)
	config := ConfigV0{
		ReviewGate: ReviewGateConfigV0{
			LineBudgetSnapshotStore: store,
			SnapshotReadBudget:      budget,
		},
	}
	verifier, ok := codexStackWorktreeVerifierV0(config).(orquestaruntimecodexdelivery.CodexReceiptWorktreeVerifierV0)
	if !ok || verifier.SnapshotReadBudget != budget {
		t.Fatalf("verifier budget=%+v ok=%v", verifier.SnapshotReadBudget, ok)
	}
	recorder, ok := codexStackWorktreeBaselineRecorderV0(config).(orquestaruntimecodexdelivery.CodexReceiptWorktreeBaselineRecorderV0)
	if !ok || recorder.SnapshotReadBudget != budget {
		t.Fatalf("recorder budget=%+v ok=%v", recorder.SnapshotReadBudget, ok)
	}
}
