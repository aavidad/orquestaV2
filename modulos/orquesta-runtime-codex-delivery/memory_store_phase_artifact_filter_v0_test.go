package orquestaruntimecodexdelivery

import (
	"context"
	"testing"
)

func TestInMemoryCodexReceiptDescriptorStoreV0FiltraArtefactoFaseRegistrado(t *testing.T) {
	registered := codexReceiptDescriptorForFilterTestV0(
		"ack-ref-store-registered-001",
		"agent-ref-store-registered-001",
	)
	pending := codexReceiptDescriptorForFilterTestV0(
		"ack-ref-store-pending-001",
		"agent-ref-store-pending-001",
	)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(registered, pending)

	descriptors, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID: "run-ref-001",
		StartedAgents: []string{
			registered.AgentRef,
			pending.AgentRef,
		},
		PhaseArtifacts: []string{
			codexReceiptPhaseArtifactProjectionForTestV0(registered.Spec),
		},
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 || descriptors[0].AgentRef != pending.AgentRef {
		t.Fatalf("descriptors=%+v", descriptors)
	}
}

func codexReceiptDescriptorForFilterTestV0(ackRef string, agentRef string) CodexReceiptDescriptorV0 {
	spec := codexDeliverySpecForTestV0()
	spec.RequestID = agentRef
	spec.AgentPacket.RequestID = agentRef
	spec.AgentPacket.DeliveryRefs.AckRef = ackRef
	return CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-" + agentRef,
		RunID:         "run-ref-001",
		AgentRef:      agentRef,
		Spec:          spec,
		AckPath:       "/tmp/" + agentRef + "/agent_ack.json",
	}
}
