package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexDeliveryObservationSourceV0NoTiraEntregaPorArtifactFueraDeWriteSet(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ack := codexDeliveryAckForTestV0(spec)
	ack.Files = orquestaruntimecodex.EvidenceListV0{"README.md", "docs/no-autorizado.md"}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-soft-outside-write-set-001",
			Spec:          spec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, spec, ack),
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
}
