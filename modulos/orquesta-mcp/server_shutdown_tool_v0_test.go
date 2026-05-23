package orquestamcp

import (
	"context"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestMCPServerShutdownToolExecutorV0DrenaPorCasoDeUso(t *testing.T) {
	control := &fakeMCPServerShutdownControlV0{
		states: map[string]orquestaruncontrol.RunControlStateV0{},
	}
	supervisor := &fakeMCPServerShutdownSupervisorV0{}
	executor := NewMCPServerShutdownToolExecutorV0(orquestaservershutdown.ServerShutdownDepsV0{
		QueueReader: fakeMCPServerShutdownQueueV0{
			candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
				RunRef: "run-ref-shutdown-001",
				AppRef: "app-ref-shutdown-001",
				Status: "ready",
			}},
		},
		RunControlReader: control,
		RunControlWriter: control,
		Supervisor:       supervisor,
		StatsReader: fakeMCPServerShutdownStatsV0{
			stats: map[string]orquestaservershutdown.RunShutdownStatsV0{
				"run-ref-shutdown-001": {RunRef: "run-ref-shutdown-001"},
			},
		},
	})

	result, err := executor.Execute(context.Background(), MCPServerShutdownToolInputV0{
		RequestID:     "req-server-shutdown-001",
		CorrelationID: "corr-server-shutdown-001",
		Forced:        true,
		RequestedBy:   "operator",
		Reason:        "apagado controlado",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoOKV0 ||
		!result.ShutdownReady ||
		result.RunsRequested != 1 ||
		result.RunsStopped != 1 ||
		control.stop.RunRef != "run-ref-shutdown-001" ||
		!control.stop.Forced ||
		supervisor.calls != 0 {
		t.Fatalf("result=%+v control=%+v supervisor=%+v", result, control.stop, supervisor)
	}
}

func TestMCPServerShutdownToolExecutorV0DrenaConAgentesEnVuelo(t *testing.T) {
	control := &fakeMCPServerShutdownControlV0{
		states: map[string]orquestaruncontrol.RunControlStateV0{},
	}
	supervisor := &fakeMCPServerShutdownSupervisorV0{}
	executor := NewMCPServerShutdownToolExecutorV0(orquestaservershutdown.ServerShutdownDepsV0{
		QueueReader: fakeMCPServerShutdownQueueV0{
			candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
				RunRef: "run-ref-shutdown-live-001",
				AppRef: "app-ref-shutdown-001",
				Status: "ready",
			}},
		},
		RunControlReader: control,
		RunControlWriter: control,
		Supervisor:       supervisor,
		StatsReader: fakeMCPServerShutdownStatsV0{
			stats: map[string]orquestaservershutdown.RunShutdownStatsV0{
				"run-ref-shutdown-live-001": {
					RunRef:         "run-ref-shutdown-live-001",
					AgentsInFlight: 1,
				},
			},
		},
	})

	result, err := executor.Execute(context.Background(), MCPServerShutdownToolInputV0{
		RequestID:     "req-server-shutdown-live-001",
		CorrelationID: "corr-server-shutdown-live-001",
		Forced:        true,
		RequestedBy:   "operator",
		Reason:        "apagado con agente en vuelo",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoOKV0 ||
		result.ShutdownReady ||
		result.Status != orquestaservershutdown.ServerShutdownStatusWaitingDrainV0 ||
		result.AgentsInFlight != 1 ||
		supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, supervisor)
	}
}

func TestMCPServerShutdownRunsV0ExponeCheckpointPendientePorAgente(t *testing.T) {
	runs := mcpServerShutdownRunsV0([]orquestaservershutdown.ServerShutdownRunResultV0{{
		RunRef:                        "run-ref-shutdown-pending-001",
		CheckpointRequired:            true,
		CheckpointDeadlineExpired:     true,
		ForcedAfterCheckpointDeadline: true,
		PendingCheckpointAgentRefs:    []string{"agent-ref-a", "agent-ref-a", "agent-ref-b"},
		CheckpointEvidenceRefs:        []string{"shutdown-checkpoint-issue-pending-ack", "shutdown-checkpoint-issue-pending-ack"},
	}})

	if len(runs) != 1 ||
		len(runs[0].PendingCheckpointAgentRefs) != 2 ||
		runs[0].PendingCheckpointAgentRefs[0] != "agent-ref-a" ||
		runs[0].PendingCheckpointAgentRefs[1] != "agent-ref-b" ||
		len(runs[0].CheckpointEvidenceRefs) != 1 ||
		!runs[0].CheckpointDeadlineExpired ||
		!runs[0].ForcedAfterCheckpointDeadline {
		t.Fatalf("runs=%+v", runs)
	}
}

func TestMCPServerShutdownCommandFromMCPV0IncluyeDeadlineCheckpoint(t *testing.T) {
	command := serverShutdownCommandFromMCPV0(MCPServerShutdownToolInputV0{
		OccurredAt:           "2026-05-13T12:05:00Z",
		CheckpointDeadlineAt: "2026-05-13T12:00:00Z",
	})
	if !command.OccurredAt.Equal(time.Date(2026, 5, 13, 12, 5, 0, 0, time.UTC)) ||
		!command.CheckpointDeadlineAt.Equal(time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("command=%+v", command)
	}
}

type fakeMCPServerShutdownQueueV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
}

func (fake fakeMCPServerShutdownQueueV0) ListRunSchedulingCandidatesV0(
	context.Context,
	orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

type fakeMCPServerShutdownControlV0 struct {
	states map[string]orquestaruncontrol.RunControlStateV0
	stop   orquestaruncontrol.StopRunCommandV0
}

func (fake *fakeMCPServerShutdownControlV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if state, ok := fake.states[request.RunRef]; ok {
		return state, nil
	}
	return orquestaruncontrol.RunControlStateV0{},
		orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: request.RunRef}
}

func (fake *fakeMCPServerShutdownControlV0) StopRunV0(
	_ context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.stop = command
	state := orquestaruncontrol.RunControlStateV0{
		RunRef: command.RunRef,
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: command.Forced,
	}
	fake.states[command.RunRef] = state
	return state, nil
}

func (fake *fakeMCPServerShutdownControlV0) PauseRunV0(
	context.Context,
	orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *fakeMCPServerShutdownControlV0) ResumeRunV0(
	context.Context,
	orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fake *fakeMCPServerShutdownControlV0) CancelRunV0(
	context.Context,
	orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

type fakeMCPServerShutdownSupervisorV0 struct {
	calls int
}

func (fake *fakeMCPServerShutdownSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	return orquestarunsupervisor.RunSupervisorResultV0{}, nil
}

type fakeMCPServerShutdownStatsV0 struct {
	stats map[string]orquestaservershutdown.RunShutdownStatsV0
}

func (fake fakeMCPServerShutdownStatsV0) ReadRunShutdownStatsV0(
	_ context.Context,
	request orquestaservershutdown.RunShutdownStatsRequestV0,
) (orquestaservershutdown.RunShutdownStatsV0, error) {
	return fake.stats[request.RunRef], nil
}
