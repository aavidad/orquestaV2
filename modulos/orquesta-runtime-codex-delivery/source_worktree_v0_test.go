package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexDeliveryObservationSourceV0AceptaAckFilesEnWorktreeCompartido(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "entrega")
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "web/index.html", "otro agente vivo")
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-001",
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			Mode: CodexReceiptWorktreeAckFilesV0,
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))

	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0AceptaAckFilesConWriteSetRaiz(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Task.WriteSet = []string{"."}
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = []string{"docs/extra.md"}
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "docs/extra.md", "entrega")
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-root-001",
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			Mode: CodexReceiptWorktreeAckFilesV0,
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))

	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0AckFilesNoCortaRailBlandoWriteSet(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = []string{"README.md", "docs/no-autorizado.md"}
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "entrega")
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "docs/no-autorizado.md", "advisory")
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-soft-write-set-001",
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			Mode: CodexReceiptWorktreeAckFilesV0,
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))

	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0RechazaAckFileInexistente(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-001",
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		}},
	}

	_, err := (CodexDeliveryObservationSourceV0{
		Store: store,
		WorktreeVerifier: CodexReceiptWorktreeVerifierV0{
			Mode: CodexReceiptWorktreeAckFilesV0,
		},
	}).BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))

	if err == nil {
		t.Fatalf("esperaba rechazo por fichero ACK inexistente")
	}
	if strings.Contains(err.Error(), projectDir) || strings.Contains(err.Error(), ackPath) {
		t.Fatalf("error filtra path operacional: %v", err)
	}
	if !strings.Contains(err.Error(), "ack_file_missing") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestCodexDeliveryObservationSourceV0ConservaCambioRealFueraDeWriteSetComoRailBlando(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = append(ack.Files, "docs/extra.md")
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v1")
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v2")
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "docs/extra.md", "fuera")
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-001",
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
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:file_outside_write_set") {
		t.Fatalf("rail blando fuera de write-set no conservado: %+v", observations[0].EvidenceRefs)
	}
}

func TestCodexDeliveryObservationSourceV0ConservaAckFilesMismatchComoRailBlando(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = append(ack.Files, "docs/extra.md")
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v1")
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v2")
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-ack-mismatch-001",
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
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:ack_files_mismatch") {
		t.Fatalf("rail blando ack_files_mismatch no conservado: %+v", observations[0].EvidenceRefs)
	}
}

func TestCodexDeliveryObservationSourceV0ConservaBorradoRealComoIssueBloqueante(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "v1")
	baseline := captureCodexDeliveryWorktreeBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	if err := os.Remove(filepath.Join(projectDir, "README.md")); err != nil {
		t.Fatalf("remove README.md: %v", err)
	}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:       "receipt-ref-removed-001",
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
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:removed_path") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:removed_path:README.md") {
		t.Fatalf("borrado real sin evidencia bloqueante: %+v", observations[0].EvidenceRefs)
	}
}

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

func captureCodexDeliveryWorktreeBaselineForTestV0(
	t *testing.T,
	projectDir string,
) orquestaruntimeworktree.WorktreeSnapshotV0 {
	t.Helper()
	snapshot, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(
		context.Background(),
		orquestaruntimeworktree.WorktreeSnapshotRequestV0{
			SnapshotRef:    "worktree-snapshot-ref-test",
			ProjectWorkDir: projectDir,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("baseline issues=%+v", issues)
	}
	return snapshot
}

func writeCodexDeliveryProjectFileForTestV0(
	t *testing.T,
	projectDir string,
	rel string,
	content string,
) {
	t.Helper()
	path := filepath.Join(projectDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
