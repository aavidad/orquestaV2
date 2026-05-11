package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestRunManagedProgressiveLoopV0WaitsExternalAndRegistersDelivery(t *testing.T) {
	runRef := "run-nucleo-managed-loop-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = append(run.Tasks, "task-ref-001")
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	deliverySource := &gatedDeliveryObservationSourceV0{
		Observation: managedDeliveryObservationV0(),
	}
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: DeliveryCandidateProviderV0{
			Base: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
				WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
					workCandidateWithAgentV0(runRef),
				},
			}},
			DeliverySource: deliverySource,
			RequestedBy:    "orquestacion-nucleo-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunManagedProgressiveLoopV0(context.Background(), ManagedProgressiveLoopRequestV0{
		Loop: ProgressiveLoopRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-09T14:00:00Z",
			MaxBursts:            4,
			MaxStepsPerBurst:     3,
			MaxDispatchesPerWait: 2,
			CorrelationID:        "corr-managed-loop-001",
			EvidenceRefs:         []string{"evidence-ref-managed-loop-001"},
			Dispatchers: []OutboxDispatcherBindingV0{
				capacityDecisionDispatcherForTestV0(store, sink, ledger),
				agentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
		ExternalWaiter:   &enableDeliveryOnWaitV0{Source: deliverySource},
		MaxExternalWaits: 2,
	})
	if err != nil {
		t.Fatalf("RunManagedProgressiveLoopV0: %v", err)
	}
	if result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if len(result.Attempts) != 2 ||
		result.Attempts[0].Result.Status != ProgressiveLoopStatusWaitExternalV0 ||
		result.Attempts[1].Result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("attempts=%+v", result.Attempts)
	}
	if len(result.ExternalWaits) != 1 || !result.ExternalWaits[0].Continue {
		t.Fatalf("external_waits=%+v", result.ExternalWaits)
	}
	if !containsNucleoRefV0(result.Final.Run.Deliveries, "delivery-ref-managed-001") {
		t.Fatalf("deliveries=%v", result.Final.Run.Deliveries)
	}
	if !sinkHasEventTypeV0(sink, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0) {
		t.Fatalf("sink no contiene DeliveryRegistered: %+v", sink.EventsV0())
	}
}

type gatedDeliveryObservationSourceV0 struct {
	Ready       bool
	Observation AgentDeliveryObservationV0
}

func (source *gatedDeliveryObservationSourceV0) BuildAgentDeliveryObservationsV0(
	context.Context,
	AgentDeliveryObservationRequestV0,
) ([]AgentDeliveryObservationV0, error) {
	if source == nil || !source.Ready {
		return nil, nil
	}
	return []AgentDeliveryObservationV0{source.Observation}, nil
}

type enableDeliveryOnWaitV0 struct {
	Source *gatedDeliveryObservationSourceV0
}

func (waiter *enableDeliveryOnWaitV0) WaitExternalProgressV0(
	context.Context,
	ExternalProgressWaitRequestV0,
) (ExternalProgressWaitResultV0, error) {
	waiter.Source.Ready = true
	return ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-managed-wait-001"},
	}, nil
}

func managedDeliveryObservationV0() AgentDeliveryObservationV0 {
	return AgentDeliveryObservationV0{
		DeliveryRef:  "delivery-ref-managed-001",
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ref-001",
		AgentRef:     "agent-ref-001",
		Summary:      "Entrega compacta despues de espera externa.",
		EvidenceRefs: []string{"evidence-ref-managed-delivery-001"},
	}
}
