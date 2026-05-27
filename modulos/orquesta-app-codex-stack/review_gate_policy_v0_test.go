package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexStackReviewGatePolicyNoAplicaPresupuestoGoADocumentacionV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "docs/manual.md", "inicio\n")
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "docs/manual.md", strings.Repeat("linea\n", 350))

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-doc-policy-001")
	spec.AgentPacket.Task.WriteSet = []string{"docs"}
	spec.AgentPacket.Task.RequiredTests = nil
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorCustomAckForTestV0(t, receiptStore, projectDir, spec, baseline.SnapshotRef, []string{"docs/manual.md"}, nil)
	source := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store: receiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      true,
			LineBudgetSnapshotStore: snapshotStore,
		}),
	}

	observations, err := source.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(observations[0].EvidenceRefs, "gate-issue:file_too_large") {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
	if codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:go_file_line_budget_strict_blocking",
	) {
		t.Fatalf("strict Go budget aplicado a documentacion: %v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGatePolicyPorPuertoControlaPresupuestoGoV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("linea\n", 350))

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-policy-port-001")
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorCustomAckForTestV0(
		t,
		receiptStore,
		projectDir,
		spec,
		baseline.SnapshotRef,
		[]string{"internal/api/handler.go"},
		spec.AgentPacket.Task.RequiredTests,
	)
	source := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store: receiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      true,
			LineBudgetSnapshotStore: snapshotStore,
			Policy:                  fakeReviewGatePolicyV0{},
		}),
	}

	observations, err := source.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:go_file_line_budget_strict_blocking",
	) {
		t.Fatalf("policy por puerto no controlo presupuesto Go: %v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGatePolicyPerfilDocumentalNoActivaPresupuestoGoV0(t *testing.T) {
	policy := codexStackReviewGatePolicyV0{StrictGoLineBudget: true, MaxGoFileLines: 300}
	packet := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-doc-profile-001").AgentPacket
	packet.Policies = []string{"work_profile_kind:documentation"}
	packet.Task.WriteSet = []string{"internal/api"}
	packet.Task.RequiredTests = []string{"go test ./..."}

	if policy.StrictGoLineBudgetForDeliveryV0(
		orquestaruntimecodex.CodexAgentAckV0{Files: []string{"internal/api/handler.go"}},
		packet,
	) {
		t.Fatalf("perfil documental no debe activar presupuesto Go estricto")
	}
}

func TestCodexStackReviewGatePolicyToleraAliasYGlobsSegurosV0(t *testing.T) {
	projectDir := t.TempDir()
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/webadmin/handler.go", "package webadmin\n")

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-glob-policy-001")
	spec.AgentPacket.Task.WriteSet = []string{"web", "internal/**/*.go"}
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorCustomAckForTestV0(
		t,
		receiptStore,
		projectDir,
		spec,
		baseline.SnapshotRef,
		[]string{"internal/webadmin/handler.go"},
		spec.AgentPacket.Task.RequiredTests,
	)
	source := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store: receiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      true,
			LineBudgetSnapshotStore: snapshotStore,
		}),
	}

	observations, err := source.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if codexStackReviewGateHasEvidenceForTestV0(observations[0].EvidenceRefs, "gate-issue:file_outside_write_set") {
		t.Fatalf("alias/glob tratado como fuera de write-set: %v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGatePolicyPerfilDocumentalNoActivaGoBudgetV0(t *testing.T) {
	policy := codexStackReviewGatePolicyV0{StrictGoLineBudget: true}
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-profile-policy-001")
	spec.AgentPacket.Policies = append(spec.AgentPacket.Policies, "work_profile:documentation")
	spec.AgentPacket.Task.WriteSet = []string{"docs/manual.go"}

	if policy.StrictGoLineBudgetForDeliveryV0(orquestaruntimecodex.CodexAgentAckV0{}, spec.AgentPacket) {
		t.Fatalf("perfil documental activo presupuesto Go estricto")
	}
}

func recordStackReviewDescriptorCustomAckForTestV0(
	t *testing.T,
	store orquestaruntimecodexdelivery.CodexReceiptDescriptorRecorderPortV0,
	projectDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	baselineRef string,
	files []string,
	tests []string,
) {
	t.Helper()
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0(files),
		Tests:         orquestaruntimecodex.EvidenceListV0(tests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(tests),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	err = store.RecordCodexReceiptDescriptorV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			DescriptorRef:       "receipt-ref-" + spec.AgentPacket.DeliveryRefs.AckRef,
			RunID:               "run-ref-stack-review-001",
			AgentRef:            spec.RequestID,
			Spec:                spec,
			AckPath:             ackPath,
			ProjectWorkDir:      projectDir,
			WorktreeBaselineRef: baselineRef,
		},
	)
	if err != nil {
		t.Fatalf("RecordCodexReceiptDescriptorV0: %v", err)
	}
}

type fakeReviewGatePolicyV0 struct{}

func (fakeReviewGatePolicyV0) StrictGoLineBudgetForDeliveryV0(
	_ orquestaruntimecodex.CodexAgentAckV0,
	_ orquestaruntime.AgentStartPacketV0,
) bool {
	return false
}

func (fakeReviewGatePolicyV0) EffectiveWriteSetV0(writeSet []string) []string {
	return append([]string(nil), writeSet...)
}

func (fakeReviewGatePolicyV0) MaxGoFileLinesV0() int {
	return 300
}
