package orquestaruntimecodexdelivery

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexDeliveryObservationSourceV0OmiteArtefactoFaseYaRegistrado(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	spec.AgentPacket.Phase = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-phase-registered-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}
	request := codexDeliveryRequestForTestV0(spec, nil)
	request.Run.PhaseArtifacts = []string{codexReceiptPhaseArtifactProjectionForTestV0(spec)}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexReceiptArtifactRefRegisteredV0(
		store.LastRequest.PhaseArtifacts,
		spec.AgentPacket.DeliveryRefs.AckRef,
	) {
		t.Fatalf("phase_artifacts no enviados al store: %+v", store.LastRequest)
	}
}

func codexReceiptPhaseArtifactProjectionForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) string {
	return spec.AgentPacket.DeliveryRefs.AckRef +
		"#phase:" + spec.AgentPacket.Phase +
		"#agent:" + spec.RequestID
}
