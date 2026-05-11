package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestRunProgressiveLoopV0StopsAtUnhandledAgentLauncher(t *testing.T) {
	runRef := "run-nucleo-progressive-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-08T10:20:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-progressive-loop-001",
		EvidenceRefs:         []string{"evidence-ref-progressive-loop-001"},
		Dispatchers: []OutboxDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor: CapacityDecisionExecutorV0{
				RunStore:      store,
				EventSink:     sink,
				OccurredAt:    "2026-05-08T10:21:00Z",
				CorrelationID: "corr-progressive-capacity-001",
			},
			Acker: ledger,
		}},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitUnhandledOutboxV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if result.TotalExecutedSteps != 2 || len(result.Bursts) != 2 {
		t.Fatalf("bursts=%+v total=%d", result.Bursts, result.TotalExecutedSteps)
	}
	if !containsNucleoRefV0(result.Run.CapacityRequests, "capacity-ref-001") ||
		len(result.Run.CapacityDecisions) != 1 ||
		!containsNucleoRefV0(result.Run.Agents, "agent-ref-001") {
		t.Fatalf("run no avanzo correctamente: %+v", result.Run)
	}
	if len(result.Dispatches) != 2 || result.Dispatches[0].Status != "dispatched" ||
		result.Dispatches[1].Status != "no_pending" {
		t.Fatalf("dispatches=%+v", result.Dispatches)
	}
	if result.PendingOutboxCount != 1 || len(result.PendingOutboxRefs) != 1 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}
