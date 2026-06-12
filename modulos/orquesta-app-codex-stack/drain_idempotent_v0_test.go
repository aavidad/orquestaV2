package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
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

func TestDrainRunV0ReingestaACKBlockedYaObservadoNoBloquea(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "run-drain-blocked-ack-001",
		ProjectRef:    "project-ref-drain-blocked-ack-001",
		AppSpecRef:    "app-spec-ref-drain-blocked-ack-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			OpenedAt:            "2026-05-10T11:59:00Z",
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		}},
		Tasks:  []string{"task-drain-blocked-ack-001"},
		Agents: []string{"agent-drain-blocked-ack-001"},
		StartedAgents: []string{
			"agent-drain-blocked-ack-001",
		},
	}
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	observation := orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-ack-drain-blocked-001",
		ArtifactRef:  "ack-drain-blocked-001",
		DeliveryRef:  "ack-drain-blocked-001",
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-drain-blocked-ack-001",
		AgentRef:     "agent-drain-blocked-ack-001",
		Summary:      "Entrega blocked conservada para replan causal.",
		EvidenceRefs: []string{"ack-drain-blocked-001", "gate-issue:agent_blocked"},
	}

	first, err := stack.applyDrainObservationsV0(context.Background(), DrainRunRequestV0{
		RunRef:        run.RunID,
		OccurredAt:    "2026-05-10T12:00:00Z",
		CorrelationID: "corr-drain-blocked-ack-001",
	}, []orquestacionnucleoapp.AgentDeliveryObservationV0{observation})
	if err != nil {
		t.Fatalf("first applyDrainObservationsV0: %v", err)
	}
	if !codexStackHasRefV0(first.Deliveries, observation.DeliveryRef) ||
		!codexStackHasRefV0(first.DeliveredTasks, observation.TaskID) {
		t.Fatalf("blocked ack no registrado: %+v", first)
	}
	if !drainDeliveryEventHasEvidenceForTestV0(stack, observation.DeliveryRef, "gate-issue:agent_blocked") {
		t.Fatalf("blocked ack sin evidencia durable de replan: %+v", first.Deliveries)
	}

	second, err := stack.applyDrainObservationsV0(context.Background(), DrainRunRequestV0{
		RunRef:        run.RunID,
		OccurredAt:    "2026-05-10T12:01:00Z",
		CorrelationID: "corr-drain-blocked-ack-002",
	}, []orquestacionnucleoapp.AgentDeliveryObservationV0{observation})
	if err != nil {
		t.Fatalf("second applyDrainObservationsV0: %v", err)
	}
	if len(second.Deliveries) != len(first.Deliveries) ||
		len(second.DeliveredTasks) != len(first.DeliveredTasks) {
		t.Fatalf("reingesta no idempotente: first=%+v second=%+v", first, second)
	}
}

func drainDeliveryEventHasEvidenceForTestV0(stack StackV0, deliveryRef, evidence string) bool {
	sink, ok := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if !ok {
		return false
	}
	for _, event := range sink.EventsV0() {
		if event.EventType != orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0 {
			continue
		}
		var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if payload.DeliveryRef == deliveryRef &&
			codexStackRefsContainPartV0(payload.EvidenceRefs, evidence) {
			return true
		}
	}
	return false
}
