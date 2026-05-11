package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressSupervisionLoopDetectedStopsAgentThroughOutbox(t *testing.T) {
	runRef := "run-nucleo-progress-supervision-001"
	agentRef := "agent-ref-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	launchService := ServiceV0{
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
	launched := runProgressiveLoopWithFakeAgentRuntimeV0(
		t,
		launchService,
		store,
		sink,
		ledger,
		runtime,
		runRef,
	)
	if !containsNucleoRefV0(launched.Run.StartedAgents, agentRef) {
		t.Fatalf("started agents=%v", launched.Run.StartedAgents)
	}

	supervisionService := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			ProgressSupervisionCandidates: []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
				progressLoopCandidateV0(runRef, agentRef),
			},
		}},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}
	result, err := supervisionService.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T10:00:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-progress-supervision-001",
		EvidenceRefs:         []string{"evidence-ref-progress-supervision-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			agentStopperDispatcherForProgressTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop supervision: %v", err)
	}
	if result.Status != ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoAssessmentRefV0(result.Run.AgentAssessments, "assessment-ref-nucleo-loop-001") ||
		!containsNucleoRefV0(result.Run.StoppedAgents, agentRef) ||
		!containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("run final inesperado: %+v", result.Run)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0,
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

func TestProgressSupervisionLoopDetectedProtectedDirectorDoesNotStopAgent(t *testing.T) {
	runRef := "run-nucleo-progress-protected-director-001"
	agentRef := "agent-ref-protected-director-001"
	store := NewInMemoryRunStoreV0(mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, agentRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()

	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: ProgressSupervisionCandidateProviderV0{
			ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{{
				Report:       progressLoopReportV0(runRef, agentRef),
				PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
				TaskRef:      "task-ref-protected-director-001",
				EvidenceRefs: []string{"evidence-ref-progress-protected-director-observation-001"},
			}}},
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}
	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T10:05:00Z",
		MaxBursts:            2,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-progress-protected-director-001",
		EvidenceRefs:         []string{"evidence-ref-progress-protected-director-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			agentStopperDispatcherForProgressTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop protected supervision: %v", err)
	}
	if containsNucleoRefV0(result.Run.StoppedAgents, agentRef) ||
		containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("director protegido parado: %+v", result.Run)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
	} {
		if sinkHasEventTypeV0(sink, eventType) {
			t.Fatalf("sink contiene parada inesperada %s: %+v", eventType, sink.EventsV0())
		}
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
}

func progressLoopCandidateV0(
	runRef string,
	agentRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	return orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: "progress-supervision-candidate-ref-nucleo-loop-001",
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta:   commandMetaV0(runRef, "cmd-assess-agent-loop-001", "idem-assess-agent-loop-001"),
			Report:        progressLoopReportV0(runRef, agentRef),
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:       "task-ref-001",
			AssessmentRef: "assessment-ref-nucleo-loop-001",
			QuestionID:    "question-ref-nucleo-loop-001",
		},
		EvidenceRefs: []string{"evidence-ref-progress-candidate-nucleo-001"},
	}
}

func progressLoopReportV0(
	runRef string,
	agentRef string,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-nucleo-loop-001",
		RunID:               runRef,
		AgentRequestID:      agentRef,
		Status:              orquestaruntime.AgentLoopDetectedV0,
		NoProgressTicks:     4,
		RepeatedActionCount: 3,
		Summary:             "Evidencia compacta de bucle en el trabajo del agente.",
		EvidenceRefs:        []string{"evidence-ref-progress-loop-nucleo-001"},
	}
}

func agentStopperDispatcherForProgressTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runtime *orquestaruntime.RuntimeFakeLifecycleV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    &FakeLifecycleAgentStopperV0{Runtime: runtime},
			ObservedAt: "2026-05-09T10:01:00Z",
		},
		Acker: ledger,
	}
}
