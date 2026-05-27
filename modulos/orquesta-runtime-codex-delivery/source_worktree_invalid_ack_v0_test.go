package orquestaruntimecodexdelivery

import (
	"context"
	"strings"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexDeliveryObservationSourceV0WorktreeVerifierConservaFailedTestEvidenceSinAckInvalid(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	failedExitCode := 1
	outputRedacted := true
	ack.TestReceipts[0] = orquestaruntimecodex.CodexRequiredTestReceiptV0{
		SchemaVersion:  orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
		Command:        "go test ./...",
		Status:         "failed",
		ExitCode:       &failedExitCode,
		EvidenceRefs:   []string{"required-test-receipt-ref-codex-delivery-failed-test"},
		OccurredAt:     "2026-05-27T17:00:00Z",
		Sequence:       1,
		OutputRedacted: &outputRedacted,
	}
	projectDir := t.TempDir()
	writeCodexDeliveryProjectFileForTestV0(t, projectDir, "README.md", "entrega con test fallido")
	ackPath := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef:  "receipt-ref-failed-test-worktree-001",
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
		t.Fatalf("BuildAgentDeliveryObservationsV0 no debe tumbar el tick por failed_test_evidence: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
	if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:failed_test_evidence") {
		t.Fatalf("rail failed_test_evidence no conservado: %+v", observations[0].EvidenceRefs)
	}
	joined := strings.Join(observations[0].EvidenceRefs, " ")
	if strings.Contains(joined, projectDir) || strings.Contains(joined, ackPath) {
		t.Fatalf("evidencia filtra ruta operacional: %+v", observations[0].EvidenceRefs)
	}
}
