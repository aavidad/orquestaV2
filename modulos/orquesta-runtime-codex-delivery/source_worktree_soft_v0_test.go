package orquestaruntimecodexdelivery

import (
	"context"
	"strings"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexDeliveryObservationSourceV0ConservaTruncadoComoRailBlando(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", strings.Repeat("base\n", 80))
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "resumen\n")
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-truncated-001",
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
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:truncated_path") {
		t.Fatalf("rail blando truncated_path no conservado: %+v", observations[0].EvidenceRefs)
	}
}

func TestCodexDeliveryObservationSourceV0ConservaReemplazoGrandeComoRailBlando(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", strings.Repeat("base\n", 1300))
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", strings.Repeat("current\n", 2200))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-large-replacement-001",
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
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:replaced_large_delta") {
		t.Fatalf("rail blando replaced_large_delta no conservado: %+v", observations[0].EvidenceRefs)
	}
}

func TestCodexDeliveryObservationSourceV0FallbackACKFilesSiBaselineSePierdeTrasReinicio(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "entrega tras reinicio")
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-missing-baseline-001",
			Spec:                spec,
			AckPath:             ackPath,
			ProjectWorkDir:      projectDir,
			WorktreeBaselineRef: "worktree-snapshot-ref-perdido-tras-reinicio",
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			SnapshotStore: orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(),
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:worktree_baseline_missing") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:strict_diff_unavailable") {
		t.Fatalf("fallback sin evidencia: %+v", observations[0].EvidenceRefs)
	}
}
