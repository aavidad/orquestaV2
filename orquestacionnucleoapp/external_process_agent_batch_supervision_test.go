package orquestacionnucleoapp

import (
	"context"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestExternalProcessAgentBatchExecutorV0StopsOnlyLoopingProcess(t *testing.T) {
	runRef := "run-nucleo-process-batch-stop-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	registry := NewInMemoryAgentProcessRegistryV0()

	launchService := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentVariantV0(runRef, "001", "app/process_worker_a.go"),
				workCandidateWithAgentVariantV0(runRef, "002", "app/process_worker_b.go"),
			},
		}},
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	launched, err := launchService.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T12:00:00Z",
		MaxBursts:            5,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-process-batch-stop-001",
		EvidenceRefs:         []string{"evidence-ref-process-batch-stop-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{
			processAgentBatchDispatcherForSupervisionTestV0(t, store, sink, ledger, processRuntime, registry),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop process batch: %v", err)
	}
	if launched.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", launched.Status, launched)
	}
	requireBatchStartedAgentsV0(t, launched.Run.StartedAgents, "agent-ref-001", "agent-ref-002")

	looping := resolveAgentProcessForSupervisionTestV0(t, registry, runRef, "agent-ref-001")
	survivor := resolveAgentProcessForSupervisionTestV0(t, registry, runRef, "agent-ref-002")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = processRuntime.StopV0(ctx, survivor.ProcessRef)
	})
	requireProcessRuntimeStatusForSupervisionTestV0(t, processRuntime, looping.ProcessRef, orquestaruntime.ProcessRuntimeRunningV0)
	requireProcessRuntimeStatusForSupervisionTestV0(t, processRuntime, survivor.ProcessRef, orquestaruntime.ProcessRuntimeRunningV0)

	supervisionService := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: ProgressSupervisionCandidateProviderV0{
			ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
				progressLoopObservationForSupervisionTestV0(runRef, "agent-ref-001"),
			}},
			RequestedBy: "orquestacion-nucleo-test",
		},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}
	supervised, err := supervisionService.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T12:05:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-process-batch-supervision-001",
		EvidenceRefs:         []string{"evidence-ref-process-batch-supervision-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			processAgentStopperDispatcherForSupervisionTestV0(store, sink, ledger, registry, processRuntime),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop supervision: %v", err)
	}
	if supervised.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", supervised.Status, supervised)
	}

	waitExternalProcessRuntimeStatusV0(t, processRuntime, looping.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
	requireProcessRuntimeStatusForSupervisionTestV0(t, processRuntime, survivor.ProcessRef, orquestaruntime.ProcessRuntimeRunningV0)
	assertSelectiveStopForSupervisionTestV0(t, supervised.Run, sink, ledger, runRef)
}

func progressLoopObservationForSupervisionTestV0(
	runRef string,
	agentRef string,
) AgentProgressObservationV0 {
	return AgentProgressObservationV0{
		Report:        progressLoopReportV0(runRef, agentRef),
		TaskRef:       "task-ref-001",
		AssessmentRef: "assessment-ref-nucleo-loop-001",
		QuestionID:    "question-ref-nucleo-loop-001",
		EvidenceRefs:  []string{"evidence-ref-progress-candidate-nucleo-001"},
	}
}

func processAgentBatchDispatcherForSupervisionTestV0(
	t *testing.T,
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	registry AgentProcessRegistryPortV0,
) OutboxBatchDispatcherBindingV0 {
	t.Helper()
	return OutboxBatchDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: ExternalProcessAgentBatchExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			SpecResolver:    externalAgentLaunchSpecResolverForTestV0(externalProcessRuntimeRequestForTestV0(t, "wait")),
			Runtime:         processRuntime,
			ProcessStopper:  processRuntime,
			ProcessRegistry: registry,
			MaxConcurrency:  2,
			OccurredAt:      "2026-05-09T12:01:00Z",
		},
		Acker: ledger,
	}
}

func processAgentStopperDispatcherForSupervisionTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	registry AgentProcessRegistryPortV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    ProcessAgentStopperV0{Registry: registry, Runtime: processRuntime},
			ObservedAt: "2026-05-09T12:06:00Z",
		},
		Acker: ledger,
	}
}

func resolveAgentProcessForSupervisionTestV0(
	t *testing.T,
	registry AgentProcessRegistryPortV0,
	runRef string,
	agentRef string,
) AgentProcessRecordV0 {
	t.Helper()
	record, err := registry.ResolveAgentProcessV0(context.Background(), runRef, agentRef)
	if err != nil {
		t.Fatalf("resolve process %s: %v", agentRef, err)
	}
	return record
}

func requireProcessRuntimeStatusForSupervisionTestV0(
	t *testing.T,
	connector *orquestaruntime.ProcessRuntimeConnectorV0,
	processRef string,
	want orquestaruntime.ProcessRuntimeStatusV0,
) {
	t.Helper()
	snapshot, err := connector.SnapshotV0(processRef)
	if err != nil {
		t.Fatalf("snapshot process %s: %v", processRef, err)
	}
	if snapshot.Status != want {
		t.Fatalf("process_ref=%s status=%s want=%s snapshot=%+v", processRef, snapshot.Status, want, snapshot)
	}
}

func assertSelectiveStopForSupervisionTestV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runRef string,
) {
	t.Helper()
	if !containsNucleoAssessmentRefV0(run.AgentAssessments, "assessment-ref-nucleo-loop-001") {
		t.Fatalf("agent assessments=%v", run.AgentAssessments)
	}
	if !containsNucleoRefV0(run.StoppedAgents, "agent-ref-001") ||
		!containsNucleoRefV0(run.ConfirmedStoppedAgents, "agent-ref-001") {
		t.Fatalf("stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if containsNucleoRefV0(run.StoppedAgents, "agent-ref-002") ||
		containsNucleoRefV0(run.ConfirmedStoppedAgents, "agent-ref-002") {
		t.Fatalf("agent-ref-002 no debe pararse: stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
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
