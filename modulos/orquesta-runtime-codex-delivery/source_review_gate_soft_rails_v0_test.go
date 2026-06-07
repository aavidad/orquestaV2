package orquestaruntimecodexdelivery

import (
	"context"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateObservationSourceV0NoCortaPorRailsGenericosEnNotas(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Notes = orquestaruntimecodex.EvidenceListV0{
		"rail pendiente: token/provider/home/prompt solo para matriz externa",
		"rail pendiente: access_token=redacted y secret: policy sin valor",
	}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-rail-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		observations[0].AcceptedReviewRef == "" ||
		stringInCodexDeliverySetV0(
			observations[0].EvidenceRefs,
			orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefV0,
		) ||
		stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") ||
		stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-action:request_followup_review") ||
		stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:ack-pending-rail:token") {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReviewGateObservationSourceV0NoCortaPorRailsGenericosEnCamposACK(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = orquestaruntimecodex.EvidenceListV0{
		"README.md",
		"docs/prompt-policy.md",
	}
	ack.Tests = orquestaruntimecodex.EvidenceListV0{
		"go test ./...",
		"completion policy redacted",
	}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-rail-fields-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})

	observations, err := (CodexReviewGateObservationSourceV0{Store: store}).
		BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		observations[0].AcceptedReviewRef == "" ||
		stringInCodexDeliverySetV0(
			observations[0].EvidenceRefs,
			orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefV0,
		) ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:file_outside_write_set") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReviewGateObservationSourceV0ConservaAccionDeRailBlando(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-file-rail-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})
	fileEvidence := staticCodexReviewGateFileEvidenceV0{
		Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
			Path:      "README.md",
			LineCount: 301,
		}},
	}

	observations, err := (CodexReviewGateObservationSourceV0{
		Store:        store,
		FileEvidence: fileEvidence,
	}).BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-action:request_followup_review") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:file_too_large") ||
		!strings.Contains(observations[0].Summary, "rail blando") {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReviewGateObservationSourceV0AceptaFileOutsideWriteSetComoAviso(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = orquestaruntimecodex.EvidenceListV0{"README.md", "docs/extra.md"}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-outside-review-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})
	fileEvidence := staticCodexReviewGateFileEvidenceV0{
		Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
			Path:      "docs/extra.md",
			LineCount: 12,
		}},
	}

	observations, err := (CodexReviewGateObservationSourceV0{
		Store:        store,
		FileEvidence: fileEvidence,
	}).BuildReviewGateObservationsV0(context.Background(), codexReviewGateRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-action:request_followup_review") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:file_outside_write_set") {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReviewGateObservationSourceV0AceptaWriteSetFaltanteComoAviso(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-soft-missing-target-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       path,
	})
	fileEvidence := staticCodexReviewGateFileEvidenceResultV0{
		Result: CodexReviewGateFileEvidenceV0{
			Files: []orquestaautoprogramming.AutoprogrammingReviewGateFileV0{{
				Path:      "README.md",
				LineCount: 12,
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
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-followup-required") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-action:request_followup_review") ||
		!stringInCodexDeliverySetV0(observations[0].EvidenceRefs, "gate-issue:write_set_target_missing:web") {
		t.Fatalf("observations=%+v", observations)
	}
}

type staticCodexReviewGateFileEvidenceResultV0 struct {
	Result CodexReviewGateFileEvidenceV0
}

func (provider staticCodexReviewGateFileEvidenceResultV0) BuildCodexReviewGateFileEvidenceV0(
	_ context.Context,
	_ CodexReceiptDescriptorV0,
	_ orquestaruntimecodex.CodexAgentAckV0,
) (CodexReviewGateFileEvidenceV0, error) {
	return provider.Result, nil
}
