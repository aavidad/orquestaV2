package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressiveLoopV0StopsAgentFromLeaseAssessment(t *testing.T) {
	runRef := "run-nucleo-lease-stop-001"
	agentRef := "agent-ref-001"
	leaseRef := "lease-ref-nucleo-timeout-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: AgentLeaseActionCandidateProviderV0{
			Base:        leaseActionBaseProviderV0{AgentRef: agentRef},
			LeaseSource: leaseStopAssessmentSourceV0{AgentRef: agentRef, LeaseRef: leaseRef},
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger:      ledger,
		MaxCommands:       4,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T17:00:00Z",
		MaxBursts:            8,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-lease-stop-progressive-001",
		EvidenceRefs:         []string{"evidence-ref-lease-stop-progressive-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
			agentLifecycleDispatcherForReplanAssessmentTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v result=%+v", err, result)
	}
	if result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsProjectionPrefixV0(result.Run.AgentLeaseExpirations, leaseRef) ||
		!containsNucleoRefV0(result.Run.StoppedAgents, agentRef) ||
		!containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("run final incompleto: %+v", result.Run)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
	} {
		if !sinkHasEventTypeV0(sink, eventType) {
			t.Fatalf("sink no contiene %s: %+v", eventType, sink.EventsV0())
		}
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
}

func TestAgentLeaseActionCandidateProviderV0SkipsAlreadyExpiredLease(t *testing.T) {
	runRef := "run-nucleo-lease-dedupe-001"
	leaseRef := "lease-ref-nucleo-timeout-dedupe-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.AgentLeaseExpirations = append(run.AgentLeaseExpirations,
		leaseRef+"#agent:agent-ref-001#action:stop_agent#reason:heartbeat_timeout#observed:2026-05-09T17:00:00Z",
	)
	provider := AgentLeaseActionCandidateProviderV0{
		LeaseSource: leaseStopAssessmentSourceV0{AgentRef: "agent-ref-001", LeaseRef: leaseRef},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-09T17:05:00Z",
		CorrelationID: "corr-lease-dedupe-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.LeaseActionCandidates) != 0 {
		t.Fatalf("lease candidates=%+v", candidates.LeaseActionCandidates)
	}
}

type leaseActionBaseProviderV0 struct {
	AgentRef string
}

func (provider leaseActionBaseProviderV0) BuildSchedulerCandidatesV0(
	_ context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if containsNucleoRefV0(request.Run.Agents, provider.AgentRef) {
		return SchedulerCandidateSetV0{}, nil
	}
	return SchedulerCandidateSetV0{
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
			workCandidateWithAgentV0(request.Run.RunID),
		},
	}, nil
}

type leaseStopAssessmentSourceV0 struct {
	AgentRef string
	LeaseRef string
}

func (source leaseStopAssessmentSourceV0) BuildAgentLeaseAssessmentsV0(
	_ context.Context,
	request AgentLeaseAssessmentRequestV0,
) ([]orquestacoreleases.AgentTimeoutAssessmentV0, error) {
	if !containsNucleoRefV0(request.Run.StartedAgents, source.AgentRef) ||
		containsNucleoRefV0(request.Run.StoppedAgents, source.AgentRef) ||
		leaseActionAlreadyExpiredV0(request.Run, source.LeaseRef) {
		return nil, nil
	}
	return []orquestacoreleases.AgentTimeoutAssessmentV0{{
		AssessmentRef:  "lease-assessment-ref-" + leaseActionSafeRefPartV0(source.LeaseRef),
		RunRef:         request.Run.RunID,
		AgentRequestID: source.AgentRef,
		LeaseRef:       source.LeaseRef,
		NowObservedAt:  "2026-05-09T17:01:00Z",
		Decision:       orquestacoreleases.AgentTimeoutDecisionStopAgentV0,
		ReasonCode:     orquestacoreleases.AgentTimeoutReasonHeartbeatTimeoutV0,
		EvidenceRefs:   []string{"evidence-ref-lease-timeout-001"},
	}}, nil
}
