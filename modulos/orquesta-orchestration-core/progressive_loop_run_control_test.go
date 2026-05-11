package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestRunProgressiveLoopV0RespetaRunPausadaAntesDeProgramar(t *testing.T) {
	runRef := "run-nucleo-progressive-control-paused-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	service := ServiceV0{
		RunStore:  store,
		EventSink: NewInMemoryEventSinkV0(),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		RunControl:   sequenceRunControlForTestV0(pausedRunControlStateForTestV0(runRef)),
		OutboxLedger: NewInMemoryOutboxLedgerV0(),
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:00:00Z",
		MaxBursts:            2,
		MaxStepsPerBurst:     2,
		MaxDispatchesPerWait: 1,
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunPausedV0 ||
		result.TotalExecutedSteps != 0 ||
		len(result.Bursts) != 0 ||
		len(result.Run.Agents) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunProgressiveLoopV0CortaAntesDeDispatchSiPidenStop(t *testing.T) {
	runRef := "run-nucleo-progressive-control-stop-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	service := ServiceV0{
		RunStore:  store,
		EventSink: NewInMemoryEventSinkV0(),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		RunControl: sequenceRunControlForTestV0(
			runningRunControlStateForTestV0(runRef),
			stopRequestedRunControlStateForTestV0(runRef),
		),
		OutboxLedger: NewInMemoryOutboxLedgerV0(),
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:05:00Z",
		MaxBursts:            2,
		MaxStepsPerBurst:     2,
		MaxDispatchesPerWait: 1,
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunStopRequestedV0 ||
		result.TotalExecutedSteps == 0 ||
		len(result.Dispatches) != 0 ||
		result.FirstPendingCount == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunProgressiveLoopV0PermiteRunSinEstadoDeControl(t *testing.T) {
	runRef := "run-nucleo-progressive-control-default-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	service := ServiceV0{
		RunStore:  store,
		EventSink: NewInMemoryEventSinkV0(),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		RunControl:   missingRunControlForTestV0{},
		OutboxLedger: NewInMemoryOutboxLedgerV0(),
		MaxCommands:  4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:10:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     2,
		MaxDispatchesPerWait: 1,
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status == ProgressiveLoopStatusRunPausedV0 ||
		result.TotalExecutedSteps == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunProgressiveLoopV0StopRequestedDrenaAgentesVivos(t *testing.T) {
	runRef := "run-nucleo-progressive-control-drain-stop-001"
	agentRef := "agent-ref-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()

	launched := runProgressiveLoopWithFakeAgentRuntimeV0(t,
		ServiceV0{
			RunStore:  store,
			EventSink: sink,
			CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
				WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
					workCandidateWithAgentV0(runRef),
				},
			}},
			OutboxLedger: ledger,
			MaxCommands:  4,
		},
		store,
		sink,
		ledger,
		runtime,
		runRef,
	)
	if !containsNucleoRefV0(launched.Run.StartedAgents, agentRef) {
		t.Fatalf("started agents=%v", launched.Run.StartedAgents)
	}

	result, err := ServiceV0{
		RunStore:          store,
		EventSink:         sink,
		CandidateProvider: StaticCandidateProviderV0{},
		RunControl:        sequenceRunControlForTestV0(stopRequestedRunControlStateForTestV0(runRef)),
		OutboxLedger:      ledger,
		MaxCommands:       4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:20:00Z",
		MaxBursts:            2,
		MaxStepsPerBurst:     2,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-run-control-stop-drain-001",
		Dispatchers: []OutboxDispatcherBindingV0{
			runControlStopDispatcherForTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop stop requested: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunStopRequestedV0 ||
		!containsNucleoRefV0(result.Run.StoppedAgents, agentRef) ||
		!containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Dispatches) == 0 {
		t.Fatalf("no despacho parada: %+v", result)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
	} {
		if !sinkHasEventTypeV0(sink, eventType) {
			t.Fatalf("sink no contiene %s: %+v", eventType, sink.EventsV0())
		}
	}
}

type missingRunControlForTestV0 struct{}

func (missingRunControlForTestV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{},
		orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: request.RunRef}
}

type sequenceRunControlV0 struct {
	states []orquestaruncontrol.RunControlStateV0
	calls  int
}

func sequenceRunControlForTestV0(
	states ...orquestaruncontrol.RunControlStateV0,
) *sequenceRunControlV0 {
	return &sequenceRunControlV0{states: states}
}

func (control *sequenceRunControlV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	index := control.calls
	control.calls++
	if index >= len(control.states) {
		index = len(control.states) - 1
	}
	state := control.states[index]
	if state.RunRef == "" {
		state.RunRef = request.RunRef
	}
	return state, nil
}

func runningRunControlStateForTestV0(runRef string) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef: runRef,
		Status: orquestaruncontrol.RunControlStatusRunningV0,
	}
}

func pausedRunControlStateForTestV0(runRef string) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef: runRef,
		Status: orquestaruncontrol.RunControlStatusPausedV0,
	}
}

func stopRequestedRunControlStateForTestV0(runRef string) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef:             runRef,
		Status:             orquestaruncontrol.RunControlStatusStopRequestedV0,
		CheckpointRecorded: true,
	}
}

func runControlStopDispatcherForTestV0(
	store *InMemoryRunStoreV0,
	sink *InMemoryEventSinkV0,
	ledger *InMemoryOutboxLedgerV0,
	runtime *orquestaruntime.RuntimeFakeLifecycleV0,
) OutboxDispatcherBindingV0 {
	return OutboxDispatcherBindingV0{
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		Reader:      ledger,
		Claimer:     ledger,
		Executor: AgentStopperExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Stopper:    &FakeLifecycleAgentStopperV0{Runtime: runtime},
			ObservedAt: "2026-05-11T09:21:00Z",
		},
		Acker: ledger,
	}
}
