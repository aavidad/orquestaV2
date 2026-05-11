package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestAgentLauncherExecutorV0RegistersAgentStartedAndAcksOutbox(t *testing.T) {
	runRef := "run-nucleo-agent-launcher-001"
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
		OccurredAt:           "2026-05-08T10:30:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-agent-launcher-loop-001",
		EvidenceRefs:         []string{"evidence-ref-agent-launcher-loop-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
			agentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoRefV0(result.Run.StartedAgents, "agent-ref-001") {
		t.Fatalf("started agents=%v", result.Run.StartedAgents)
	}
	if result.PendingOutboxCount != 0 || len(result.PendingOutboxRefs) != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
	if !hasDispatchForTargetV0(result.Dispatches, orquestacoreworkflow.OutboxTargetAgentLauncherV0) {
		t.Fatalf("dispatches sin agent_launcher: %+v", result.Dispatches)
	}
}

func capacityDecisionDispatcherForTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: CapacityDecisionExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			OccurredAt:    "2026-05-08T10:31:00Z",
			CorrelationID: "corr-test-capacity-decision-001",
		},
		Acker: ledger,
	}
}

func agentLauncherDispatcherForTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-08T10:32:00Z",
		},
		Acker: ledger,
	}
}

func hasDispatchForTargetV0(
	dispatches []OutboxDispatchOnceResultV0,
	targetPort string,
) bool {
	for _, dispatch := range dispatches {
		if dispatch.TargetPort == targetPort && dispatch.Status == "dispatched" {
			return true
		}
	}
	return false
}
