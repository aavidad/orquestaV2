package orquestaservershutdown

import (
	"context"
	"reflect"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestShutdownServerV0SolicitaStopDrenaYQuedaReady(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-a", AppRef: "app-a", Status: "ready"},
		{RunRef: "run-b", AppRef: "app-b", Status: "ready"},
	})
	deps.stats.stats["run-a"] = RunShutdownStatsV0{RunRef: "run-a"}
	deps.stats.stats["run-b"] = RunShutdownStatsV0{RunRef: "run-b"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		QueueRef:      "global",
		Forced:        true,
		RequestedBy:   "operator",
		Reason:        "apagado controlado",
		CorrelationID: "corr-shutdown-test-001",
		OccurredAt:    time.Date(2026, 5, 13, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.RunsRequested != 2 ||
		result.RunsStopped != 2 ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if got := deps.control.stopped; !reflect.DeepEqual(got, []string{"run-a", "run-b"}) {
		t.Fatalf("stopped=%v", got)
	}
}

func TestShutdownServerV0NoDrenaSiFaltaCheckpoint(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-checkpoint", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.pending["run-checkpoint"] = []string{"agent-ref-a"}
	deps.checkpoint.evidence["run-checkpoint"] = []string{"shutdown-checkpoint-issue-pending-agent_ack"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy: "operator",
		Reason:      "apagado no forzado",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusWaitingCheckpointV0 ||
		result.CheckpointsPending != 1 ||
		result.CheckpointAgentsPending != 1 ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if len(result.Runs) != 1 ||
		!reflect.DeepEqual(result.Runs[0].PendingCheckpointAgentRefs, []string{"agent-ref-a"}) ||
		!reflect.DeepEqual(result.Runs[0].CheckpointEvidenceRefs, []string{"shutdown-checkpoint-issue-pending-agent_ack"}) {
		t.Fatalf("pending checkpoint no expuesto: %+v", result.Runs)
	}
	if !reflect.DeepEqual(deps.control.stopped, []string{"run-checkpoint"}) {
		t.Fatalf("stop requests=%+v", deps.control.stopped)
	}
}

func TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-graceful", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.recorded["run-graceful"] = true
	deps.stats.stats["run-graceful"] = RunShutdownStatsV0{RunRef: "run-graceful"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:   "operator",
		Reason:        "apagado graceful",
		CorrelationID: "corr-shutdown-test-002",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.CheckpointsPending != 0 ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if !deps.control.states["run-graceful"].CheckpointRecorded ||
		!reflect.DeepEqual(deps.control.stopped, []string{"run-graceful"}) {
		t.Fatalf("control=%+v", deps.control)
	}
}

func TestShutdownServerV0SinRunsQuedaReady(t *testing.T) {
	deps := newServerShutdownDepsForTestV0(nil)

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.RunsRequested != 0 ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
}

type serverShutdownDepsForTestV0 struct {
	queue      fakeShutdownQueueV0
	control    *fakeShutdownControlV0
	checkpoint *fakeShutdownCheckpointPreparerV0
	supervisor *fakeShutdownSupervisorV0
	stats      *fakeShutdownStatsV0
}

func newServerShutdownDepsForTestV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) serverShutdownDepsForTestV0 {
	return serverShutdownDepsForTestV0{
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
	}
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
	fake.states[command.RunRef] = state
	return state, nil
}

func (fake *fakeShutdownControlV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
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
}

func (fake *fakeShutdownCheckpointPreparerV0) PrepareAgentShutdownV0(
	_ context.Context,
	command PrepareAgentShutdownCommandV0,
) (PrepareAgentShutdownResultV0, error) {
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

type fakeShutdownSupervisorV0 struct {
	calls int
}

func (fake *fakeShutdownSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
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
