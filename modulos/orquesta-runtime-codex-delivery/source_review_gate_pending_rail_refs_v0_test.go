package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateObservationSourceV0PropagaCategoriasDeRailPendiente(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Notes = orquestaruntimecodex.EvidenceListV0{
		"rail pendiente: token provider home prompt access_token=redacted sin valor",
	}
	path := writeCodexDeliveryAckForTestV0(t, spec, ack)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-pending-rail-categories-001",
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
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	for _, want := range []string{
		orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefV0,
		"ack-pending-rail:token",
		"ack-pending-rail:provider",
		"ack-pending-rail:home",
		"ack-pending-rail:prompt",
	} {
		if !stringInCodexDeliverySetV0(observations[0].EvidenceRefs, want) {
			t.Fatalf("evidence_refs sin %q: %v", want, observations[0].EvidenceRefs)
		}
	}
}
