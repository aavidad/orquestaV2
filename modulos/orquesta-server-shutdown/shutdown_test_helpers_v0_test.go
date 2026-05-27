package orquestaservershutdown

import (
	"context"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type serverShutdownDepsForTestV0 struct {
	queue      fakeShutdownQueueV0
	control    *fakeShutdownControlV0
	checkpoint *fakeShutdownCheckpointPreparerV0
	supervisor *fakeShutdownSupervisorV0
	stats      *fakeShutdownStatsV0
	events     *[]string
}

func newServerShutdownDepsForTestV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) serverShutdownDepsForTestV0 {
	events := []string{}
	deps := serverShutdownDepsForTestV0{
		queue: fakeShutdownQueueV0{candidates: candidates},
		control: &fakeShutdownControlV0{
			states: map[string]orquestaruncontrol.RunControlStateV0{},
		},
		checkpoint: &fakeShutdownCheckpointPreparerV0{
			recorded: map[string]bool{},
			pending:  map[string][]string{},
			evidence: map[string][]string{},
		},
		supervisor: &fakeShutdownSupervisorV0{},
		stats:      &fakeShutdownStatsV0{stats: map[string]RunShutdownStatsV0{}},
		events:     &events,
	}
	deps.control.events = deps.events
	deps.checkpoint.events = deps.events
	return deps
}

func (deps serverShutdownDepsForTestV0) deps() ServerShutdownDepsV0 {
	return ServerShutdownDepsV0{
		QueueReader:         deps.queue,
		RunControlReader:    deps.control,
		RunControlWriter:    deps.control,
		RunCheckpointWriter: deps.control,
		CheckpointPreparer:  deps.checkpoint,
		Supervisor:          deps.supervisor,
		StatsReader:         deps.stats,
	}
}

type fakeShutdownQueueV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
}

func (fake fakeShutdownQueueV0) ListRunSchedulingCandidatesV0(
	context.Context,
	orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

type fakeShutdownControlV0 struct {
	states  map[string]orquestaruncontrol.RunControlStateV0
	stopped []string
	events  *[]string
}

func (fake *fakeShutdownControlV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if state, ok := fake.states[request.RunRef]; ok {
		return state, nil
	}
	return orquestaruncontrol.RunControlStateV0{},
		orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: request.RunRef}
}

func (fake *fakeShutdownControlV0) StopRunV0(
	_ context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	appendShutdownEventForTestV0(fake.events, "stop:"+command.RunRef)
	state := orquestaruncontrol.RunControlStateV0{
		RunRef: command.RunRef,
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: command.Forced,
		Meta: orquestaruncontrol.RunControlMetaV0{
			RequestedBy: command.RequestedBy,
			Reason:      command.Reason,
		},
	}
	fake.stopped = append(fake.stopped, command.RunRef)
	if previous, ok := fake.states[command.RunRef]; ok {
		state.CheckpointRecorded = previous.CheckpointRecorded
		state.EvidenceRefs = append([]string(nil), previous.EvidenceRefs...)
	}
	state.EvidenceRefs = compactServerShutdownStringsV0(append(state.EvidenceRefs, command.EvidenceRefs...))
	fake.states[command.RunRef] = state
	return state, nil
}

func (fake *fakeShutdownControlV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	appendShutdownEventForTestV0(fake.events, "record_checkpoint:"+command.RunRef)
	state, ok := fake.states[command.RunRef]
	if !ok {
		state = orquestaruncontrol.DefaultRunControlStateV0(command.RunRef)
	}
	state.CheckpointRecorded = true
	state.EvidenceRefs = append([]string(nil), command.EvidenceRefs...)
	fake.states[command.RunRef] = state
	return state, nil
}

func (fake *fakeShutdownControlV0) PauseRunV0(
	context.Context,
	orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *fakeShutdownControlV0) ResumeRunV0(
	context.Context,
	orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *fakeShutdownControlV0) CancelRunV0(
	context.Context,
	orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

type fakeShutdownCheckpointPreparerV0 struct {
	recorded map[string]bool
	pending  map[string][]string
	evidence map[string][]string
	events   *[]string
}

func (fake *fakeShutdownCheckpointPreparerV0) PrepareAgentShutdownV0(
	_ context.Context,
	command PrepareAgentShutdownCommandV0,
) (PrepareAgentShutdownResultV0, error) {
	appendShutdownEventForTestV0(fake.events, "prepare:"+command.RunRef)
	if fake.recorded[command.RunRef] {
		return PrepareAgentShutdownResultV0{
			RunRef:             command.RunRef,
			CheckpointRecorded: true,
			CheckpointRef:      "checkpoint-ref-" + command.RunRef,
			EvidenceRefs:       []string{"evidence-ref-" + command.RunRef},
		}, nil
	}
	return PrepareAgentShutdownResultV0{
		RunRef:           command.RunRef,
		PendingAgentRefs: append([]string(nil), fake.pending[command.RunRef]...),
		EvidenceRefs:     append([]string(nil), fake.evidence[command.RunRef]...),
	}, nil
}

func appendShutdownEventForTestV0(events *[]string, event string) {
	if events != nil {
		*events = append(*events, event)
	}
}

func shutdownEventsForTestV0(events *[]string) []string {
	if events == nil {
		return nil
	}
	return append([]string(nil), (*events)...)
}

type fakeShutdownSupervisorV0 struct {
	calls int
	err   error
}

func (fake *fakeShutdownSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	if fake.err != nil {
		return orquestarunsupervisor.RunSupervisorResultV0{}, fake.err
	}
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

type fakeShutdownStatsV0 struct {
	stats map[string]RunShutdownStatsV0
}

func (fake *fakeShutdownStatsV0) ReadRunShutdownStatsV0(
	_ context.Context,
	request RunShutdownStatsRequestV0,
) (RunShutdownStatsV0, error) {
	return fake.stats[request.RunRef], nil
}
