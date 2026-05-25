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

func TestCodexStackReviewGateStrictLineBudgetBloqueaCrecimientoBaselineV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("x\n", 320))
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("x\n", 321))

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-strict-lines-001")
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorWithBaselineForTestV0(t, receiptStore, projectDir, spec, baseline.SnapshotRef)
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
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:go_file_line_budget_strict_blocking",
	) {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGateStrictLineBudgetUsaSnapshotAunqueACKOmitaFicheroV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/generated.go", strings.Repeat("x\n", 301))

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-strict-lines-omitted-001")
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorWithBaselineForTestV0(t, receiptStore, projectDir, spec, baseline.SnapshotRef)
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
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:go_file_line_budget_strict_blocking",
	) {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGateLegacyConSnapshotMantieneLineBudgetAdvisoryV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("x\n", 301))

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-legacy-lines-001")
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorWithBaselineForTestV0(t, receiptStore, projectDir, spec, baseline.SnapshotRef)
	source := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store: receiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
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
	if !codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:file_too_large",
	) {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackReviewGateSnapshotBloqueaTruncadoFuerteV0(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("x\n", 80))
	baseline := captureStackReviewGateBaselineForTestV0(t, projectDir)
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(baseline)
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")

	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-truncate-001")
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0()
	recordStackReviewDescriptorWithBaselineForTestV0(t, receiptStore, projectDir, spec, baseline.SnapshotRef)
	source := orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0{
		Store: receiptStore,
		FileEvidenceResult: codexStackReviewGateFileEvidenceV0(ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
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
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:truncated_path",
	) {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func captureStackReviewGateBaselineForTestV0(
	t *testing.T,
	projectDir string,
) orquestaruntimeworktree.WorktreeSnapshotV0 {
	t.Helper()
	snapshot, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(
		context.Background(),
		orquestaruntimeworktree.WorktreeSnapshotRequestV0{
			SnapshotRef:    "worktree-snapshot-ref-stack-review-line-budget",
			ProjectWorkDir: projectDir,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("snapshot issues=%+v", issues)
	}
	return snapshot
}

func recordStackReviewDescriptorWithBaselineForTestV0(
	t *testing.T,
	store orquestaruntimecodexdelivery.CodexReceiptDescriptorRecorderPortV0,
	projectDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	baselineRef string,
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
		Files:         orquestaruntimecodex.EvidenceListV0{"internal/api/handler.go"},
		Tests:         orquestaruntimecodex.EvidenceListV0(spec.AgentPacket.Task.RequiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(spec.AgentPacket.Task.RequiredTests),
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
