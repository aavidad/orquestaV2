package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
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
