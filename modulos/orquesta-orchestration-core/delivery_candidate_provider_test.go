package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestDeliveryCandidateProviderV0RegistersDeliveryThroughProgressiveLoop(t *testing.T) {
	runRef := "run-nucleo-delivery-receipt-001"
	run := mustDeliveryReadyRunV0(t, runRef)
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: DeliveryCandidateProviderV0{
			DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
				deliveryObservationForTestV0(),
			}},
			RequestedBy: "orquestacion-nucleo-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T13:00:00Z",
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-delivery-receipt-001",
		EvidenceRefs:         []string{"evidence-ref-delivery-receipt-001"},
	})
	if err != nil {
		t.Fatalf("progressive loop delivery: %v", err)
	}
	if result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoRefV0(result.Run.Deliveries, "delivery-ref-nucleo-001") {
		t.Fatalf("deliveries=%v", result.Run.Deliveries)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0) {
		t.Fatalf("sink no contiene DeliveryRegistered: %+v", sink.EventsV0())
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

func TestDeliveryCandidateProviderV0RegistersPhaseArtifactThroughProgressiveLoop(t *testing.T) {
	runRef := "run-nucleo-phase-artifact-receipt-001"
	run := mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, "agent-ref-nucleo-brainstorm-001")
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: DeliveryCandidateProviderV0{
			DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
				phaseArtifactObservationForTestV0(),
			}},
			RequestedBy: "orquestacion-nucleo-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T13:03:00Z",
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-phase-artifact-receipt-001",
		EvidenceRefs:         []string{"evidence-ref-phase-artifact-receipt-001"},
	})
	if err != nil {
		t.Fatalf("progressive loop phase artifact: %v", err)
	}
	if result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsProjectionRefPartV0(result.Run.PhaseArtifacts, "artifact-ref-nucleo-brainstorm-001") {
		t.Fatalf("phase_artifacts=%v", result.Run.PhaseArtifacts)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0) {
		t.Fatalf("sink no contiene PhaseArtifactRegistered: %+v", sink.EventsV0())
	}
	if sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0) {
		t.Fatalf("sink no debe contener DeliveryRegistered: %+v", sink.EventsV0())
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

func TestDeliveryCandidateProviderV0TaskRefNoRegistradaEsArtefactoDeFase(t *testing.T) {
	runRef := "run-nucleo-phase-artifact-task-ref-001"
	run := mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, "agent-ref-nucleo-brainstorm-001")
	observation := phaseArtifactObservationForTestV0()
	observation.TaskID = "task-ref-director-no-durable-001"
	provider := DeliveryCandidateProviderV0{
		DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
			observation,
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-09T13:05:00Z",
		CorrelationID: "corr-phase-artifact-task-ref-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.PhaseArtifactCandidates) != 1 || len(candidates.DeliveryCandidates) != 0 {
		t.Fatalf("candidates=%+v", candidates)
	}
}

func TestDeliveryCandidateProviderV0RejectsInvalidObservation(t *testing.T) {
	provider := DeliveryCandidateProviderV0{
		DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
			{DeliveryRef: "delivery-ref-invalid-001"},
		}},
	}

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustDeliveryReadyRunV0(t, "run-nucleo-delivery-invalid-001"),
		OccurredAt: "2026-05-09T13:01:00Z",
	})
	if err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestDeliveryCandidateProviderV0UsaRequestedByNeutralPorDefecto(t *testing.T) {
	provider := DeliveryCandidateProviderV0{
		DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
			deliveryObservationForTestV0(),
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           mustDeliveryReadyRunV0(t, "run-nucleo-delivery-default-001"),
		OccurredAt:    "2026-05-09T13:02:00Z",
		CorrelationID: "corr-delivery-default-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.DeliveryCandidates) != 1 {
		t.Fatalf("delivery_candidates=%d", len(candidates.DeliveryCandidates))
	}
	if candidates.DeliveryCandidates[0].CommandMeta.RequestedBy != "orquesta-nucleo-delivery" {
		t.Fatalf("requested_by=%q", candidates.DeliveryCandidates[0].CommandMeta.RequestedBy)
	}
}

func TestDeliveryCandidateProviderV0LimitaUnCandidatoPorTick(t *testing.T) {
	second := deliveryObservationForTestV0()
	second.DeliveryRef = "delivery-ref-nucleo-002"
	provider := DeliveryCandidateProviderV0{
		DeliverySource: staticAgentDeliveryObservationSourceV0{Observations: []AgentDeliveryObservationV0{
			deliveryObservationForTestV0(),
			second,
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustDeliveryReadyRunV0(t, "run-nucleo-delivery-single-tick-001"),
		OccurredAt: "2026-05-09T13:04:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.DeliveryCandidates) != 1 {
		t.Fatalf("delivery_candidates=%d", len(candidates.DeliveryCandidates))
	}
	if candidates.DeliveryCandidates[0].Payload.DeliveryRef != "delivery-ref-nucleo-001" {
		t.Fatalf("delivery_ref=%q", candidates.DeliveryCandidates[0].Payload.DeliveryRef)
	}
}

func TestDeliveryCandidateProviderV0PropagaYFiltraWaitAgentRefs(t *testing.T) {
	outOfScope := deliveryObservationForTestV0()
	outOfScope.DeliveryRef = "delivery-ref-out-of-scope-001"
	outOfScope.AgentRef = "agent-ref-out-of-scope-001"
	source := &capturingAgentDeliveryObservationSourceV0{
		Observations: []AgentDeliveryObservationV0{outOfScope, deliveryObservationForTestV0()},
	}
	provider := DeliveryCandidateProviderV0{DeliverySource: source}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           mustDeliveryReadyRunV0(t, "run-nucleo-delivery-wait-scope-001"),
		OccurredAt:    "2026-05-17T11:00:00Z",
		CorrelationID: "corr-delivery-wait-scope-001",
		WaitAgentRefs: []string{"agent-ref-nucleo-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if !containsNucleoRefV0(source.LastRequest.WaitAgentRefs, "agent-ref-nucleo-001") {
		t.Fatalf("wait_agent_refs no propagado: %+v", source.LastRequest)
	}
	if len(candidates.DeliveryCandidates) != 1 {
		t.Fatalf("delivery_candidates=%d candidates=%+v", len(candidates.DeliveryCandidates), candidates)
	}
	if candidates.DeliveryCandidates[0].Payload.AgentRef != "agent-ref-nucleo-001" {
		t.Fatalf("agent_ref=%q", candidates.DeliveryCandidates[0].Payload.AgentRef)
	}
}

func TestRunProgressiveLoopV0PropagaWaitAgentRefsAObservationSource(t *testing.T) {
	runRef := "run-nucleo-delivery-progressive-wait-scope-001"
	source := &capturingAgentDeliveryObservationSourceV0{}
	service := ServiceV0{
		RunStore: NewInMemoryRunStoreV0(mustDeliveryReadyRunV0(t, runRef)),
		CandidateProvider: DeliveryCandidateProviderV0{
			DeliverySource: source,
		},
		OutboxLedger: NewInMemoryOutboxLedgerV0(),
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-17T11:02:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-progressive-wait-scope-001",
		WaitAgentRefs:        []string{"agent-ref-nucleo-001"},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoRefV0(source.LastRequest.WaitAgentRefs, "agent-ref-nucleo-001") {
		t.Fatalf("wait_agent_refs no propagado por loop progresivo: %+v", source.LastRequest)
	}
}

type staticAgentDeliveryObservationSourceV0 struct {
	Observations []AgentDeliveryObservationV0
}

func (source staticAgentDeliveryObservationSourceV0) BuildAgentDeliveryObservationsV0(
	context.Context,
	AgentDeliveryObservationRequestV0,
) ([]AgentDeliveryObservationV0, error) {
	return source.Observations, nil
}

type capturingAgentDeliveryObservationSourceV0 struct {
	Observations []AgentDeliveryObservationV0
	LastRequest  AgentDeliveryObservationRequestV0
}

func (source *capturingAgentDeliveryObservationSourceV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request AgentDeliveryObservationRequestV0,
) ([]AgentDeliveryObservationV0, error) {
	source.LastRequest = request
	return source.Observations, nil
}

func mustDeliveryReadyRunV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = append(run.Tasks, "task-ref-nucleo-001")
	run.Agents = append(run.Agents, "agent-ref-nucleo-001")
	run.StartedAgents = append(run.StartedAgents, "agent-ref-nucleo-001")
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run delivery ready invalido: %+v", issues)
	}
	return run
}

func deliveryObservationForTestV0() AgentDeliveryObservationV0 {
	return AgentDeliveryObservationV0{
		DeliveryRef:  "delivery-ref-nucleo-001",
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ref-nucleo-001",
		AgentRef:     "agent-ref-nucleo-001",
		Summary:      "Entrega compacta aceptada por receipt.",
		EvidenceRefs: []string{"evidence-ref-delivery-nucleo-001"},
	}
}
