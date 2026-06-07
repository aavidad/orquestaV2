package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateObservationSourceV0SoftRailsCombinadosNoBloqueanV0(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Notes = orquestaruntimecodex.EvidenceListV0{
		"rail pendiente: token/provider/home/prompt solo para matriz externa",
		"rail pendiente: access_token=redacted sin valor real",
	}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-rails-combined-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})
	fileEvidence := staticCodexReviewGateFileEvidenceResultV0{
		Result: CodexReviewGateFileEvidenceV0{
			Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
				Path:      "README.md",
				LineCount: 301,
			}},
			Issues: []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{{
				Code:  "write_set_target_missing:web",
				Field: "write_set",
			}},
		},
	}

	observations, err := (CodexReviewGateObservationSourceV0{
		Store:              store,
		FileEvidenceResult: fileEvidence,
	}).BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		observations[0].AcceptedReviewRef == "" ||
		stringInCodexDeliverySetV0(observations[0].EvidenceRefs, orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefV0) ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-action:request_followup_review") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:file_too_large") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:write_set_target_missing:web") {
		t.Fatalf("observations=%+v", observations)
	}
}
