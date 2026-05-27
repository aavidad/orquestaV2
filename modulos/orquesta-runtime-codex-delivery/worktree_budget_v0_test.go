package orquestaruntimecodexdelivery

import (
	"context"
	"strings"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexDeliveryObservationSourceV0ConservaSnapshotBudgetComoRailV0(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v1")
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", strings.Repeat("x", 32))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-snapshot-budget-001",
			Spec:                spec,
			AckPath:             ackPath,
			ProjectWorkDir:      projectDir,
			WorktreeBaselineRef: baseline.SnapshotRef,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			SnapshotStore: snapshotStore,
			SnapshotReadBudget: orquestaruntimeworktree.WorktreeSnapshotReadBudgetFromLimitsV0(
				10,
				8,
				1024,
			),
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:worktree_snapshot_file_too_large") {
		t.Fatalf("observations=%+v", observations)
	}
}
