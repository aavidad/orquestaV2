package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorRunStatsV0CuentaTrabajoVivoComoProgresoMinimo(t *testing.T) {
	firstTask := "task-ref-live-progress-001"
	secondTask := "task-ref-live-progress-002"
	firstAgent := WorkflowTaskAgentRequestRefV0(firstTask)
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-progress-001")
	run.Tasks = []string{firstTask, secondTask}
	run.Agents = []string{firstAgent}
	run.StartedAgents = []string{firstAgent}

	stats := BuildDirectorRunStatsV0(run)

	if stats.Progress.PercentComplete != 1 ||
		stats.Progress.TasksObserved != 0 ||
		stats.Counts.AgentsInFlight != 1 {
		t.Fatalf("progress=%+v counts=%+v", stats.Progress, stats.Counts)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, firstTask)
	if task.Status != DirectorTaskProgressInProgressV0 ||
		task.AgentRequestID != firstAgent ||
		task.ProgressStatus == "" {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsV0CuentaAgenteVivoOpacoComoProgresoMinimo(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-opaque-001")
	run.Tasks = []string{"task-ref-live-opaque-001", "task-ref-live-opaque-002"}
	run.Agents = []string{"agent-ref-live-opaque-001"}
	run.StartedAgents = []string{"agent-ref-live-opaque-001"}

	stats := BuildDirectorRunStatsV0(run)

	if stats.Progress.PercentComplete != 1 ||
		stats.Progress.TasksObserved != 0 ||
		stats.Counts.AgentsInFlight != 1 ||
		stats.Counts.TasksClosed != 0 {
		t.Fatalf("progress=%+v counts=%+v", stats.Progress, stats.Counts)
	}
}

func TestBuildDirectorRunStatsWithProcessRegistryV0UsaProcesoComoSenalParcial(t *testing.T) {
	taskRef := "task-ref-live-process-001"
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-process-001")
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-live-process-001",
		SessionRef:     "session-ref-live-process-001",
		LaunchRef:      "launch-ref-live-process-001",
		ReadinessRef:   "readiness-ref-live-process-001",
		EvidenceRefs:   []string{"evidence-ref-live-process-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)

	if stats.Progress.PercentComplete != 1 ||
		stats.Progress.TasksObserved != 1 ||
		stats.Counts.TasksClosed != 0 {
		t.Fatalf("progress=%+v counts=%+v", stats.Progress, stats.Counts)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	if task.Status != DirectorTaskProgressInProgressV0 ||
		task.ProgressStatus != DirectorTaskProgressProcessRegisteredV0 ||
		task.AgentRequestID != agentRef {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsWithProcessRegistryV0MarcaMissingCuandoRegistryNoTieneProceso(t *testing.T) {
	taskRef := "task-ref-missing-process-001"
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-missing-process-001")
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(
		context.Background(),
		run,
		NewInMemoryAgentProcessRegistryV0(),
	)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.ControlState != DirectorAgentControlStateMissingV0 ||
		agent.ControlRegistered ||
		!agent.NeedsAttention ||
		agent.CanStop ||
		agent.Process != nil ||
		stats.Counts.AgentsControlMissing != 1 {
		t.Fatalf("agent=%+v counts=%+v", agent, stats.Counts)
	}
}

func TestBuildDirectorRunStatsWithProcessSnapshotsV0PublicaStatusDeProceso(t *testing.T) {
	taskRef := "task-ref-live-process-status-001"
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	processRef := "process-ref-live-process-status-001"
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-process-status-001")
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     processRef,
		SessionRef:     "session-ref-live-process-status-001",
		LaunchRef:      "launch-ref-live-process-status-001",
		ReadinessRef:   "readiness-ref-live-process-status-001",
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessSnapshotsV0(
		context.Background(),
		run,
		registry,
		liveProcessSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				processRef: {
					ProcessRef: processRef,
					Status:     orquestaruntime.ProcessRuntimeRunningV0,
				},
			},
		},
	)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Process == nil ||
		agent.Process.Status != DirectorAgentProcessStatusRunningV0 ||
		agent.Process.ProcessRef != processRef {
		t.Fatalf("agent=%+v", agent)
	}
}

func TestBuildDirectorRunStatsWithProcessSnapshotsV0NoUsaProcesoParadoComoProgreso(t *testing.T) {
	taskRef := "task-ref-stopped-process-status-001"
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	processRef := "process-ref-stopped-process-status-001"
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-stopped-process-status-001")
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     processRef,
		SessionRef:     "session-ref-stopped-process-status-001",
		LaunchRef:      "launch-ref-stopped-process-status-001",
		ReadinessRef:   "readiness-ref-stopped-process-status-001",
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessSnapshotsV0(
		context.Background(),
		run,
		registry,
		liveProcessSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				processRef: {
					ProcessRef: processRef,
					Status:     orquestaruntime.ProcessRuntimeStoppedV0,
				},
			},
		},
	)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	if agent.Process == nil ||
		agent.Process.Status != DirectorAgentProcessStatusStoppedV0 ||
		task.ProgressStatus == DirectorTaskProgressProcessRegisteredV0 ||
		stats.Progress.TasksObserved != 0 {
		t.Fatalf("agent=%+v task=%+v progress=%+v", agent, task, stats.Progress)
	}
}

func TestBuildDirectorRunStatsWithProcessRegistryV0ProyectaProcesoNoReflejadoEnRun(t *testing.T) {
	taskRef := "task-ref-live-process-unreflected-001"
	agentRef := "agent-ref-live-process-unreflected-subrole-001"
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-process-unreflected-001")
	run.Tasks = []string{taskRef}
	run.Agents = nil
	run.StartedAgents = nil
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-live-process-unreflected-001",
		SessionRef:     "session-ref-live-process-unreflected-001",
		LaunchRef:      "launch-ref-live-process-unreflected-001",
		ReadinessRef:   "readiness-ref-live-process-unreflected-001",
		EvidenceRefs:   []string{"evidence-ref-live-process-unreflected-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)

	if stats.Counts.AgentsInFlight != 1 ||
		stats.Counts.AgentsControlRegistered != 1 ||
		stats.Progress.PercentComplete != 1 ||
		stats.Progress.TasksObserved != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Status != DirectorAgentStatusRunningV0 ||
		!agent.Started ||
		!agent.InFlight ||
		!agent.ControlRegistered ||
		agent.Process == nil ||
		agent.Process.ProcessRef != "process-ref-live-process-unreflected-001" {
		t.Fatalf("agent=%+v", agent)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	if task.Status != DirectorTaskProgressInProgressV0 ||
		task.ProgressStatus != DirectorTaskProgressProcessRegisteredV0 ||
		task.AgentRequestID != agentRef {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsWithProcessRegistryV0AsociaProcesoOpacoATareaUnica(t *testing.T) {
	taskRef := "task-ref-live-process-opaque-001"
	agentRef := "agent-ref-live-process-opaque-001"
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-process-opaque-001")
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-live-process-opaque-001",
		SessionRef:     "session-ref-live-process-opaque-001",
		LaunchRef:      "launch-ref-live-process-opaque-001",
		ReadinessRef:   "readiness-ref-live-process-opaque-001",
		EvidenceRefs:   []string{"evidence-ref-live-process-opaque-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)

	if stats.Progress.PercentComplete != 1 ||
		stats.Progress.TasksObserved != 1 ||
		stats.Counts.TasksClosed != 0 {
		t.Fatalf("progress=%+v counts=%+v", stats.Progress, stats.Counts)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	if task.Status != DirectorTaskProgressInProgressV0 ||
		task.ProgressStatus != DirectorTaskProgressProcessRegisteredV0 ||
		task.AgentRequestID != agentRef {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsV0CuentaEntregaSinCierreComoProgreso(t *testing.T) {
	deliveredTask := "task-ref-live-delivered-001"
	openTask := "task-ref-live-delivered-002"
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-live-delivered-001")
	run.Tasks = []string{deliveredTask, openTask}
	run.DeliveredTasks = []string{deliveredTask}
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0

	stats := BuildDirectorRunStatsV0(run)

	if stats.Progress.PercentComplete != 50 ||
		stats.Progress.TasksObserved != 1 ||
		stats.Counts.TasksClosed != 0 {
		t.Fatalf("progress=%+v counts=%+v", stats.Progress, stats.Counts)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, deliveredTask)
	if task.Status != DirectorTaskProgressDeliveredV0 {
		t.Fatalf("task=%+v", task)
	}
}
