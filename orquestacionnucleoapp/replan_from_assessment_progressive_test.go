package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressiveLoopV0ReplansFromStoppedAgentAssessment(t *testing.T) {
	runRef := "run-nucleo-replan-assessment-001"
	oldAgentRef := "agent-ref-001"
	replacementAgentRef := "agent-ref-replacement-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = append(run.Tasks, "task-ref-001")
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	provider := AgentAssessmentReplanCandidateProviderV0{
		Base: replanFromAssessmentBaseProviderV0{
			OldAgentRef: oldAgentRef,
		},
		PlanSource: replanFromAssessmentPlanSourceV0{
			OldAgentRef:         oldAgentRef,
			ReplacementAgentRef: replacementAgentRef,
		},
		RequestedBy: "orquesta-nucleo-test",
	}
	service := ServiceV0{
		RunStore:          store,
		EventSink:         sink,
		CandidateProvider: provider,
		OutboxLedger:      ledger,
		MaxCommands:       4,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T16:00:00Z",
		MaxBursts:            12,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-replan-assessment-progressive-001",
		EvidenceRefs:         []string{"evidence-ref-replan-assessment-progressive-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
			agentLifecycleDispatcherForReplanAssessmentTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v result=%+v", err, result)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !containsNucleoAssessmentRefV0(result.Run.AgentAssessments, replanAssessmentRefV0()) ||
		!containsNucleoRefV0(result.Run.StoppedAgents, oldAgentRef) ||
		!containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, oldAgentRef) ||
		!containsReplanProjectionRefV0(result.Run.ReplanDecisions, replanAssessmentDecisionRefV0()) ||
		!containsNucleoRefV0(result.Run.CapacityRequests, replanAssessmentCapacityRefV0()) ||
		!containsProjectionPrefixV0(result.Run.CapacityDecisions, replanAssessmentCapacityRefV0()) ||
		!containsNucleoRefV0(result.Run.Agents, replacementAgentRef) ||
		!containsNucleoRefV0(result.Run.StartedAgents, replacementAgentRef) {
		t.Fatalf("run final incompleto: %+v", result.Run)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStartedV0,
	} {
		if !sinkHasEventTypeV0(sink, eventType) {
			t.Fatalf("sink no contiene %s: %+v", eventType, sink.EventsV0())
		}
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
}

type replanFromAssessmentBaseProviderV0 struct {
	OldAgentRef string
}

func (provider replanFromAssessmentBaseProviderV0) BuildSchedulerCandidatesV0(
	_ context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	run := request.Run
	if !containsNucleoRefV0(run.Agents, provider.OldAgentRef) {
		return SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(run.RunID),
			},
		}, nil
	}
	if containsNucleoRefV0(run.StartedAgents, provider.OldAgentRef) &&
		!containsNucleoAssessmentRefV0(run.AgentAssessments, replanAssessmentRefV0()) {
		return SchedulerCandidateSetV0{
			ProgressSupervisionCandidates: []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
				progressLoopCandidateV0(run.RunID, provider.OldAgentRef),
			},
		}, nil
	}
	return SchedulerCandidateSetV0{}, nil
}

type replanFromAssessmentPlanSourceV0 struct {
	OldAgentRef         string
	ReplacementAgentRef string
}

func (source replanFromAssessmentPlanSourceV0) BuildAgentAssessmentReplanPlansV0(
	_ context.Context,
	request AgentAssessmentReplanPlanRequestV0,
) ([]AgentAssessmentReplanPlanV0, error) {
	run := request.Run
	if !containsNucleoRefV0(run.ConfirmedStoppedAgents, source.OldAgentRef) ||
		!containsNucleoAssessmentRefV0(run.AgentAssessments, replanAssessmentRefV0()) ||
		containsNucleoRefV0(run.Agents, source.ReplacementAgentRef) {
		return nil, nil
	}
	return []AgentAssessmentReplanPlanV0{{
		CandidateRef:               "replan-followup-candidate-ref-assessment-001",
		ReplanRef:                  replanAssessmentDecisionRefV0(),
		SignalRef:                  "signal-ref-from-assessment-001",
		TaskRef:                    "task-ref-001",
		RequestedAction:            orquestacorereplanner.ReplanActionReplaceAgentV0,
		ReplacementRole:            "implementacion",
		CapacityRequestRef:         replanAssessmentCapacityRefV0(),
		AgentRequestID:             source.ReplacementAgentRef,
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		Summary:                    "Reemplazar agente detenido tras evaluacion de bucle.",
		EvidenceRefs:               []string{"evidence-ref-replan-from-assessment-001"},
		Assessment: orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  replanAssessmentRefV0(),
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: source.OldAgentRef,
			TaskRef:        "task-ref-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
			Summary:        "Evidencia compacta de bucle en el trabajo del agente.",
			EvidenceRefs:   []string{"evidence-ref-progress-loop-nucleo-001"},
		},
	}}, nil
}

func agentLifecycleDispatcherForReplanAssessmentTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runtime *orquestaruntime.RuntimeFakeLifecycleV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: AgentLifecycleDispatchExecutorV0{
			Launch: AgentLauncherExecutorV0{
				RunStore:   store,
				EventSink:  sink,
				Launcher:   &FakeLifecycleAgentLauncherV0{Runtime: runtime},
				OccurredAt: "2026-05-09T16:01:00Z",
			},
			Stop: AgentStopperExecutorV0{
				RunStore:   store,
				EventSink:  sink,
				Stopper:    &FakeLifecycleAgentStopperV0{Runtime: runtime},
				ObservedAt: "2026-05-09T16:02:00Z",
			},
		},
		Acker: ledger,
	}
}

type AgentLifecycleDispatchExecutorV0 struct {
	Launch AgentLauncherExecutorV0
	Stop   AgentStopperExecutorV0
}

func (executor AgentLifecycleDispatchExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	switch intent.MessageType {
	case orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0:
		return executor.Launch.ExecuteOutboxDispatchV0(intent)
	case orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0:
		return executor.Stop.ExecuteOutboxDispatchV0(intent)
	default:
		return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "message_type", "tipo de outbox no soportado")
	}
}

func containsReplanProjectionRefV0(values []string, replanRef string) bool {
	return containsProjectionPrefixV0(values, replanRef)
}

func containsProjectionPrefixV0(values []string, ref string) bool {
	for _, value := range values {
		if len(value) >= len(ref) && value[:len(ref)] == ref {
			return true
		}
	}
	return false
}

func replanAssessmentRefV0() string {
	return "assessment-ref-nucleo-loop-001"
}

func replanAssessmentDecisionRefV0() string {
	return "replan-ref-from-assessment-001"
}

func replanAssessmentCapacityRefV0() string {
	return "capacity-ref-from-assessment-001"
}
