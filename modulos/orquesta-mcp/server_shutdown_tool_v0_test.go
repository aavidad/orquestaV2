package orquestamcp

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0(t *testing.T) {
	descriptor := MCPServerShutdownDescriptorV0()
	if descriptor.Name != MCPServerShutdownToolNameV0 ||
		descriptor.ResourceURI != MCPServerShutdownResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if !strings.Contains(descriptor.InputSchema, "evidence_refs?") ||
		!strings.Contains(descriptor.Output, "evidence_refs?") ||
		!strings.Contains(descriptor.Output, "error:{errores_publicos,evidence_refs?}") {
		t.Fatalf("descriptor shutdown debe declarar evidencia en entrada y salida: %+v", descriptor)
	}
}

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
		RequestedBy:   "orquesta-director",
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
		RequestedBy:   "orquesta-director",
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

func TestMCPServerShutdownToolExecutorV0RechazaAgente(t *testing.T) {
	control := &fakeMCPServerShutdownControlV0{
		states: map[string]orquestaruncontrol.RunControlStateV0{},
	}
	executor := NewMCPServerShutdownToolExecutorV0(orquestaservershutdown.ServerShutdownDepsV0{
		QueueReader: fakeMCPServerShutdownQueueV0{
			candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
				RunRef: "run-ref-shutdown-agent-denied-001",
				AppRef: "app-ref-shutdown-001",
				Status: "ready",
			}},
		},
		RunControlReader: control,
		RunControlWriter: control,
	})

	result, err := executor.Execute(context.Background(), MCPServerShutdownToolInputV0{
		RequestID:   "req-server-shutdown-agent-denied-001",
		Forced:      true,
		RequestedBy: "agent-ref-001",
		Reason:      "agente no autorizado",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoErrorV0 ||
		result.Status != orquestaservershutdown.ServerShutdownStatusRequesterDeniedV0 ||
		control.stop.RunRef != "" {
		t.Fatalf("result=%+v stop=%+v", result, control.stop)
	}
}

func TestMCPServerShutdownToolExecutorV0ErrorConservaEvidenciaV0(t *testing.T) {
	result, err := NewMCPServerShutdownToolExecutorV0(orquestaservershutdown.ServerShutdownDepsV0{}).Execute(
		context.Background(),
		MCPServerShutdownToolInputV0{
			RequestedBy:    "agent-ref-not-authorized",
			Reason:         "apagado no autorizado con evidencia",
			EvidenceRefs:   []string{" evidence-ref-shutdown-denied-001 ", "evidence-ref-shutdown-denied-001", ""},
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoErrorV0 ||
		result.Status != "" ||
		len(result.EvidenceRefs) != 1 ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-shutdown-denied-001") ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPPublicMutationIssueIdempotencyKeyRequiredV0 {
		t.Fatalf("error shutdown debe conservar evidencia compacta: %+v", result)
	}
}

func TestMCPServerShutdownToolExecutorV0ExponeGoalsActivos(t *testing.T) {
	control := &fakeMCPServerShutdownControlV0{
		states: map[string]orquestaruncontrol.RunControlStateV0{},
	}
	executor := NewMCPServerShutdownToolExecutorV0(orquestaservershutdown.ServerShutdownDepsV0{
		QueueReader:      fakeMCPServerShutdownQueueV0{},
		RunControlReader: control,
		RunControlWriter: control,
		ActiveWorkReader: fakeMCPServerShutdownActiveWorkV0{
			works: []orquestaservershutdown.ActiveShutdownWorkV0{{
				Kind:            "goal_first",
				RunRef:          "run-ref-goal-active-001",
				WorkRef:         "goal-ref-active-001",
				ExternalWorkRef: "external-goal-ref-active-001",
				Status:          "running",
				EvidenceRefs:    []string{"goal-state-ref-active-001"},
			}},
		},
	})

	result, err := executor.Execute(context.Background(), MCPServerShutdownToolInputV0{
		RequestID:   "req-active-goals-shutdown",
		RequestedBy: "orquesta-director",
		Reason:      "apagado no forzado",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoOKV0 ||
		result.ShutdownReady ||
		result.Status != orquestaservershutdown.ServerShutdownStatusActiveGoalsPresentV0 ||
		result.ActiveWorkCount != 1 ||
		len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].RunRef != "run-ref-goal-active-001" ||
		result.ActiveWorks[0].ExternalWorkRef != "external-goal-ref-active-001" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "goal-state-ref-active-001") ||
		control.stop.RunRef != "" {
		t.Fatalf("result=%+v stop=%+v", result, control.stop)
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
		orquestaruncontrol.RunControlStateNotFoundErrorV0(request)
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

type fakeMCPServerShutdownActiveWorkV0 struct {
	works []orquestaservershutdown.ActiveShutdownWorkV0
}

func (fake fakeMCPServerShutdownActiveWorkV0) ReadActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return orquestaservershutdown.ActiveShutdownWorkResultV0{
		ActiveWorks: append([]orquestaservershutdown.ActiveShutdownWorkV0(nil), fake.works...),
	}, nil
}
