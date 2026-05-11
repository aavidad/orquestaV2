package orquestaappcodexstack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestDrainRunV0IgnoraArtefactoYaRegistradoPorLoopGestionado(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	form := codexStackFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("web status=%d body=%s", rec.Code, rec.Body.String())
	}

	store := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	descriptors := codexStackRealSmokeDescriptorsV0(t, store)
	if len(descriptors) == 0 {
		t.Fatalf("descriptors vacios")
	}
	observation := drainObservationFromDescriptorForTestV0(descriptors[0])

	_, err := stack.applyDrainObservationsV0(context.Background(), DrainRunRequestV0{
		RunRef:        descriptors[0].RunID,
		CorrelationID: "corr-drain-idempotent-001",
	}, []orquestacionnucleoapp.AgentDeliveryObservationV0{observation})
	if err != nil {
		t.Fatalf("applyDrainObservationsV0: %v", err)
	}
	assertPhaseArtifactEventsForRunV0(t, stack, descriptors[0].RunID, 4)
}

func drainObservationFromDescriptorForTestV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.AgentDeliveryObservationV0 {
	packet := descriptor.Spec.AgentPacket
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + packet.DeliveryRefs.AckRef,
		ArtifactRef:  packet.DeliveryRefs.AckRef,
		DeliveryRef:  packet.DeliveryRefs.AckRef,
		PhaseID:      packet.Phase,
		TaskID:       packet.Task.TaskRef,
		AgentRef:     descriptor.AgentRef,
		Summary:      "Entrega compacta validada por recibo de agente.",
		EvidenceRefs: []string{packet.DeliveryRefs.AckRef},
	}
}
