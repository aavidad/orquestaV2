package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorRunStatsV0ResumeRunYAgentes(t *testing.T) {
	run := directorStatsRunForTestV0(t)
	stats := BuildDirectorRunStatsV0(run)

	if stats.SchemaVersion != DirectorRunStatsSchemaVersionV0 ||
		stats.RunRef != run.RunID ||
		stats.CurrentPhase != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		t.Fatalf("stats base=%+v", stats)
	}
	if stats.Counts.TasksTotal != 3 ||
		stats.Counts.TasksClosed != 1 ||
		stats.Counts.TasksOpen != 2 ||
		stats.Counts.AgentsRequested != 3 ||
		stats.Counts.AgentsStarted != 2 ||
		stats.Counts.AgentsInFlight != 1 ||
		stats.Counts.AgentsControlMissing != 0 {
		t.Fatalf("counts=%+v", stats.Counts)
	}
	if len(stats.Refs.OpenTasks) != 2 ||
		!containsNucleoRefV0(stats.Refs.OpenTasks, "task-ref-stats-001") ||
		!containsNucleoRefV0(stats.Refs.OpenTasks, "task-ref-stats-003") {
		t.Fatalf("refs=%+v", stats.Refs)
	}
	if len(stats.Phases) == 0 || len(stats.Agents) != 3 {
		t.Fatalf("phases=%+v agents=%+v", stats.Phases, stats.Agents)
	}
	running := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-stats-001")
	if running.Status != DirectorAgentStatusRunningV0 ||
		running.ControlState != DirectorAgentControlStateNotLoadedV0 ||
		running.NeedsAttention {
		t.Fatalf("running agent=%+v", running)
	}
}

func TestBuildDirectorRunStatsV0ExponeStopControlPendienteYConfirmado(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-stop-control-001")
	run.Agents = []string{"agent-ref-stop-001", "agent-ref-stop-002"}
	run.StartedAgents = []string{"agent-ref-stop-001", "agent-ref-stop-002"}
	run.StoppedAgents = []string{"agent-ref-stop-001", "agent-ref-stop-002"}
	run.ConfirmedStoppedAgents = []string{"agent-ref-stop-001"}

	stats := BuildDirectorRunStatsV0(run)
	if stats.StopControl.Status != DirectorRunStopStatusPendingV0 ||
		!stats.StopControl.Requested ||
		!stats.StopControl.Propagated ||
		!stats.StopControl.Pending ||
		stats.StopControl.Confirmed ||
		stats.StopControl.StopRequestedAgents != 2 ||
		stats.StopControl.StopConfirmedAgents != 1 ||
		stats.StopControl.StopPendingAgents != 1 ||
		!containsNucleoRefV0(stats.StopControl.PendingAgentRefs, "agent-ref-stop-002") {
		t.Fatalf("stop_control pendiente=%+v", stats.StopControl)
	}

	run.ConfirmedStoppedAgents = []string{"agent-ref-stop-001", "agent-ref-stop-002"}
	confirmed := BuildDirectorRunStatsV0(run)
	if confirmed.StopControl.Status != DirectorRunStopStatusConfirmedV0 ||
		!confirmed.StopControl.Confirmed ||
		confirmed.StopControl.Pending ||
		confirmed.StopControl.StopPendingAgents != 0 ||
		len(confirmed.StopControl.PendingAgentRefs) != 0 {
		t.Fatalf("stop_control confirmado=%+v", confirmed.StopControl)
	}
}

func TestApplyDirectorRunControlStateV0ExponeStopRequestedSinPropagar(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-stop-control-runcontrol-001")
	run.Agents = []string{"agent-ref-stop-runcontrol-001"}
	run.StartedAgents = []string{"agent-ref-stop-runcontrol-001"}
	stats := BuildDirectorRunStatsV0(run)

	ApplyDirectorRunControlStateV0(
		&stats,
		"stop_requested",
		true,
		true,
		[]string{"evidence-ref-stop-control-runcontrol-001"},
	)

	if stats.StopControl.Status != DirectorRunStopStatusRequestedV0 ||
		!stats.StopControl.Requested ||
		stats.StopControl.Propagated ||
		!stats.StopControl.Pending ||
		stats.StopControl.RunControlStatus != "stop_requested" ||
		!stats.StopControl.CheckpointRecorded ||
		!stats.StopControl.Forced ||
		stats.StopControl.StopPendingAgents != 1 ||
		!containsNucleoRefV0(stats.StopControl.PendingAgentRefs, "agent-ref-stop-runcontrol-001") ||
		!containsNucleoRefV0(stats.StopControl.EvidenceRefs, "evidence-ref-stop-control-runcontrol-001") {
		t.Fatalf("stop_control runcontrol=%+v", stats.StopControl)
	}
}

func TestBuildDirectorRunStatsV0ExponeCierreBloqueadoPorHitosGenericos(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-closure-blocked-001")
	run.Tasks = []string{"task-ref-closure-001"}
	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{}, nil)

	if stats.Closure.Status != DirectorClosureStatusBlockedV0 ||
		!stats.Closure.Blocked ||
		stats.Closure.Ready ||
		stats.Closure.Closed {
		t.Fatalf("closure=%+v", stats.Closure)
	}
	for _, reason := range []string{
		DirectorClosureBlockedByContratosV0,
		DirectorClosureBlockedByProgramacionEntregasV0,
		DirectorClosureBlockedByRevisionFinalV0,
		DirectorClosureBlockedByValidacionFinalV0,
		DirectorClosureBlockedByFaseCierreV0,
	} {
		if !containsNucleoRefV0(stats.Closure.BlockedBy, reason) {
			t.Fatalf("blocked_by=%+v, falta %s", stats.Closure.BlockedBy, reason)
		}
	}
}

func TestBuildDirectorRunStatsV0ExponeCierreReadyYCerrado(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-closure-ready-001")
	run.Tasks = []string{"task-ref-closure-ready-001"}
	run.ClosedTasks = []string{"task-ref-closure-ready-001"}
	run.FunctionContracts = []string{"contract-ref-closure-ready-001"}
	run.Deliveries = []string{"delivery-ref-closure-ready-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-closure-ready-001"}
	run.Validations = []string{"validation-ref-closure-ready-001"}
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseCierreV0
	markDirectorStatsPhaseForTestV0(&run, orquestacoreworkflow.OrchestrationPhaseCierreV0, orquestacoreworkflow.OrchestrationPhaseStatusActiveV0)

	stats := BuildDirectorRunStatsV0(run)
	if stats.Closure.Status != DirectorClosureStatusReadyV0 ||
		!stats.Closure.Ready ||
		stats.Closure.Blocked ||
		len(stats.Closure.BlockedBy) != 0 {
		t.Fatalf("closure ready=%+v", stats.Closure)
	}

	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	run.Closures = []string{"closure-ref-ready-001"}
	closed := BuildDirectorRunStatsV0(run)
	if closed.Closure.Status != DirectorClosureStatusClosedV0 ||
		!closed.Closure.Closed ||
		closed.Closure.Blocked ||
		closed.Closure.Ready {
		t.Fatalf("closure closed=%+v", closed.Closure)
	}
}

func TestBuildDirectorRunStatsWithProcessRegistryV0EnriqueceControl(t *testing.T) {
	run := directorStatsRunForTestV0(t)
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     "process-ref-stats-001",
		SessionRef:     "session-ref-stats-001",
		LaunchRef:      "launch-ref-stats-001",
		ReadinessRef:   "readiness-ref-stats-001",
		EvidenceRefs:   []string{"evidence-ref-stats-process-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)
	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-stats-001")
	if !agent.ControlRegistered ||
		agent.Process == nil ||
		agent.Process.SessionRef != "session-ref-stats-001" ||
		!agent.CanStop ||
		stats.Counts.AgentsControlMissing != 0 ||
		stats.Counts.AgentsControlRegistered != 1 {
		t.Fatalf("agent=%+v counts=%+v", agent, stats.Counts)
	}
}

func TestBuildDirectorRunStatsWithObservationsV0ExponeProgresoPorAgenteYTarea(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-001")
	run.Tasks = []string{"task-ref-progress-001", "task-ref-progress-002", "task-ref-progress-003"}
	run.ClosedTasks = []string{"task-ref-progress-003"}
	run.Agents = []string{"agent-ref-progress-001", "agent-ref-progress-002", "agent-ref-progress-003"}
	run.StartedAgents = []string{"agent-ref-progress-001", "agent-ref-progress-002", "agent-ref-progress-003"}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{
		directorStatsProgressObservationForTestV0(
			run.RunID,
			"agent-ref-progress-001",
			"task-ref-progress-001",
			orquestaruntime.AgentProgressingV0,
		),
		directorStatsProgressObservationForTestV0(
			run.RunID,
			"agent-ref-progress-002",
			"task-ref-progress-002",
			orquestaruntime.AgentStalledV0,
		),
	}, nil)

	if stats.Progress.SourceStatus != DirectorProgressSourceLoadedV0 ||
		stats.Progress.PercentComplete != 33 ||
		stats.Progress.TasksObserved != 2 ||
		stats.Progress.ProgressingAgents != 1 ||
		stats.Progress.StalledAgents != 1 ||
		!containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-progress-003") {
		t.Fatalf("progress=%+v", stats.Progress)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-progress-002")
	if agent.LastProgress == nil ||
		agent.LastProgress.Status != string(orquestaruntime.AgentStalledV0) ||
		agent.NeedsAttention {
		t.Fatalf("agent=%+v", agent)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, "task-ref-progress-002")
	if task.Status != DirectorTaskProgressStalledV0 ||
		task.AgentRequestID != "agent-ref-progress-002" {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsWithObservationsV0NoMarcaSinSenalAgentesConEntrega(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-reflected-001")
	run.Tasks = []string{"task-ref-reflected-001", "task-ref-reflected-002", "task-ref-reflected-003"}
	run.Agents = []string{
		"agent-ref-reflected-phase-001",
		"agent-ref-reflected-delivery-001",
		"agent-ref-reflected-missing-001",
	}
	run.StartedAgents = append([]string{}, run.Agents...)
	run.PhaseArtifacts = []string{
		"artifact-reflected-001#phase:brainstorming_arquitectura#agent:agent-ref-reflected-phase-001",
	}
	run.Deliveries = []string{
		"ack-ref-app-stack-agent-ref-reflected-delivery-001",
	}
	run.DeliveredAgents = []string{"agent-ref-reflected-delivery-001"}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{}, nil)

	if stats.Counts.AgentsInFlight != 1 ||
		stats.Progress.ObservedAgents != 0 ||
		!containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-reflected-missing-001") ||
		containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-reflected-phase-001") ||
		containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-reflected-delivery-001") {
		t.Fatalf("counts=%+v progress=%+v", stats.Counts, stats.Progress)
	}
	phaseAgent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-reflected-phase-001")
	if phaseAgent.Status != DirectorAgentStatusCompletedV0 || phaseAgent.InFlight {
		t.Fatalf("phase_agent=%+v", phaseAgent)
	}
	deliveryAgent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-reflected-delivery-001")
	if deliveryAgent.Status != DirectorAgentStatusCompletedV0 || deliveryAgent.InFlight {
		t.Fatalf("delivery_agent=%+v", deliveryAgent)
	}
}

func TestBuildDirectorRunStatsWithObservationsV0IgnoraProgresoObsoletoDeAgenteCompletado(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-reflected-stale-001")
	agentRef := "agent-ref-reflected-stale-001"
	taskRef := "task-ref-reflected-stale-001"
	run.Tasks = nil
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.PhaseArtifacts = []string{
		"artifact-reflected-stale-001#phase:brainstorming_arquitectura#agent:" + agentRef,
	}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{
		directorStatsProgressObservationForTestV0(
			run.RunID,
			agentRef,
			taskRef,
			orquestaruntime.AgentStalledV0,
		),
	}, nil)

	if stats.Progress.ObservedAgents != 0 ||
		stats.Progress.StalledAgents != 0 ||
		stats.Progress.TasksObserved != 0 ||
		len(stats.Progress.Tasks) != 0 {
		t.Fatalf("progress=%+v", stats.Progress)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Status != DirectorAgentStatusCompletedV0 ||
		agent.InFlight ||
		agent.NeedsAttention ||
		agent.LastProgress != nil {
		t.Fatalf("agent=%+v", agent)
	}
}

func TestBuildDirectorRunStatsV0IgnoraAssessmentObsoletoTrasDelivery(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-delivered-001")
	taskRef := "task-ref-delivered-001"
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	run.Tasks = []string{taskRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{"ack-ref-app-stack-" + agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-delivered-stale-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: agentRef,
			TaskRef:        taskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{}, nil)

	if stats.Counts.TasksDelivered != 1 ||
		stats.Progress.PercentComplete != 100 ||
		stats.Progress.ObservedAgents != 0 ||
		stats.Progress.StalledAgents != 0 ||
		stats.Progress.TasksObserved != 1 {
		t.Fatalf("counts=%+v progress=%+v", stats.Counts, stats.Progress)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.Status != DirectorAgentStatusCompletedV0 ||
		agent.InFlight ||
		agent.NeedsAttention ||
		agent.LastProgress != nil {
		t.Fatalf("agent=%+v", agent)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	if task.Status != DirectorTaskProgressDeliveredV0 ||
		task.DecisionRequired ||
		task.LastReportRef != "" {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildDirectorRunStatsV0CuentaEntregasComoProgresoSinCerrarTarea(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-delivery-percent-001")
	run.Tasks = []string{
		"task-ref-delivery-percent-001",
		"task-ref-delivery-percent-002",
		"task-ref-delivery-percent-003",
	}
	run.DeliveredTasks = []string{"task-ref-delivery-percent-001"}
	run.ClosedTasks = []string{"task-ref-delivery-percent-002"}

	stats := BuildDirectorRunStatsV0(run)

	if stats.Progress.PercentComplete != 66 ||
		stats.Progress.TasksClosed != 1 ||
		stats.Counts.TasksDelivered != 1 ||
		stats.Counts.TasksClosed != 1 {
		t.Fatalf("counts=%+v progress=%+v", stats.Counts, stats.Progress)
	}
}

func TestBuildDirectorRunStatsWithObservationsV0NoMarcaSinSenalAgenteStopRequested(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-stop-requested-001")
	run.Tasks = []string{"task-ref-stop-requested-001"}
	run.Agents = []string{"agent-ref-stop-requested-001"}
	run.StartedAgents = []string{"agent-ref-stop-requested-001"}
	run.StoppedAgents = []string{"agent-ref-stop-requested-001"}
	run.AgentStopRequests = []string{
		orquestacoreworkflow.AgentStopRequestProjectionRefV0(orquestacoreworkflow.AgentStopRequestedPayloadV0{
			AgentRequestID: "agent-ref-stop-requested-001",
			ReasonCode:     "loop_detected",
			Summary:        "Detener agente por bucle detectado.",
		}),
	}
	run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-stop-requested-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: "agent-ref-stop-requested-001",
			TaskRef:        "task-ref-stop-requested-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{}, nil)

	if stats.Counts.AgentAssessments != 1 ||
		stats.Counts.AgentsStopRequested != 1 ||
		containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-stop-requested-001") {
		t.Fatalf("counts=%+v progress=%+v", stats.Counts, stats.Progress)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-stop-requested-001")
	if agent.Status != DirectorAgentStatusStopRequestedV0 ||
		agent.StopReasonCode != "loop_detected" ||
		agent.StopReasonSource != DirectorAgentStopReasonSourceStopRequestV0 ||
		agent.LastProgress == nil ||
		agent.LastProgress.ReportRef != "assessment-ref-stop-requested-001" ||
		agent.LastProgress.Status != string(orquestaruntime.AgentLoopDetectedV0) {
		t.Fatalf("agent=%+v", agent)
	}
}

func TestBuildDirectorRunStatsV0UsaAssessmentComoMotivoLegacyDeParada(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-stop-reason-legacy-001")
	run.Tasks = []string{"task-ref-stop-reason-legacy-001"}
	run.Agents = []string{"agent-ref-stop-reason-legacy-001"}
	run.StartedAgents = []string{"agent-ref-stop-reason-legacy-001"}
	run.StoppedAgents = []string{"agent-ref-stop-reason-legacy-001"}
	run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-stop-reason-legacy-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: "agent-ref-stop-reason-legacy-001",
			TaskRef:        "task-ref-stop-reason-legacy-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		}),
	}

	stats := BuildDirectorRunStatsV0(run)
	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-stop-reason-legacy-001")

	if agent.StopReasonCode != orquestacoreworkflow.AgentAssessmentVerdictGarbageV0 ||
		agent.StopReasonSource != DirectorAgentStopReasonSourceAssessmentV0 ||
		agent.StopReasonRef != "assessment-ref-stop-reason-legacy-001" {
		t.Fatalf("agent=%+v", agent)
	}
}

func TestBuildDirectorRunStatsV0ExponeProgresoCompactoDesdeAssessment(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-progress-assessment-001")
	run.Tasks = []string{"task-ref-assessment-001"}
	run.Agents = []string{"agent-ref-assessment-001"}
	run.StartedAgents = []string{"agent-ref-assessment-001"}
	run.AgentAssessments = []string{
		orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-progress-compact-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: "agent-ref-assessment-001",
			TaskRef:        "task-ref-assessment-001",
			DeliveryRef:    "delivery-ref-assessment-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityMediumV0,
		}),
	}

	stats := BuildDirectorRunStatsV0(run)

	if stats.Progress.ObservedAgents != 1 ||
		stats.Progress.StalledAgents != 1 ||
		stats.Progress.TasksObserved != 1 ||
		containsNucleoRefV0(stats.Progress.NoSignalAgentRefs, "agent-ref-assessment-001") {
		t.Fatalf("progress=%+v", stats.Progress)
	}
	agent := findDirectorAgentStatsForTestV0(t, stats, "agent-ref-assessment-001")
	if agent.LastProgress == nil ||
		agent.LastProgress.ReportRef != "assessment-ref-progress-compact-001" ||
		agent.LastProgress.TaskRef != "task-ref-assessment-001" ||
		agent.LastProgress.Status != string(orquestaruntime.AgentStalledV0) ||
		!agent.LastProgress.DecisionRequired {
		t.Fatalf("agent=%+v", agent)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, "task-ref-assessment-001")
	if task.Status != DirectorTaskProgressStalledV0 ||
		task.LastReportRef != "assessment-ref-progress-compact-001" ||
		task.AgentRequestID != "agent-ref-assessment-001" ||
		task.DeliveryRef != "delivery-ref-assessment-001" {
		t.Fatalf("task=%+v", task)
	}
}

func TestBuildAutonomousDirectorLoopStatsV0IncluyeDecisionYLoop(t *testing.T) {
	run := directorStatsRunForTestV0(t)
	decision := AutonomousDirectorDecisionV0{
		TeamSize:             3,
		MaxParallelAgents:    2,
		RecommendedCapacity:  orquestacoreworkflow.OrchestrationCapacityHighV0,
		MaxCommandsPerCycle:  6,
		MaxOutboxPerCycle:    2,
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
	}
	stats := BuildAutonomousDirectorLoopStatsV0(decision, ProgressiveLoopResultV0{
		Status:             ProgressiveLoopStatusWaitExternalV0,
		Run:                run,
		TotalExecutedSteps: 2,
		PendingOutboxCount: 1,
	})
	if stats.SchemaVersion != DirectorLoopStatsSchemaVersionV0 ||
		stats.Decision.TeamSize != 3 ||
		stats.Loop.Status != string(ProgressiveLoopStatusWaitExternalV0) ||
		stats.Loop.TotalExecutedSteps != 2 ||
		stats.Run.Counts.AgentsInFlight != 1 {
		t.Fatalf("stats=%+v", stats)
	}
}

func directorStatsProgressObservationForTestV0(
	runRef string,
	agentRef string,
	taskRef string,
	status orquestaruntime.AgentProgressStatusV0,
) AgentProgressObservationV0 {
	return AgentProgressObservationV0{
		Report: orquestaruntime.AgentProgressReportV0{
			ReportID:            "agent-progress-report-ref-" + agentRef,
			RunID:               runRef,
			AgentRequestID:      agentRef,
			Status:              status,
			NoProgressTicks:     2,
			RepeatedActionCount: 1,
			Summary:             "Progreso compacto observado.",
			EvidenceRefs:        []string{"evidence-ref-progress-" + agentRef},
		},
		TaskRef:      taskRef,
		EvidenceRefs: []string{"evidence-ref-observation-" + agentRef},
	}
}

func directorStatsRunForTestV0(
	t *testing.T,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustActiveProgrammingRunV0(t, "run-nucleo-director-stats-001")
	run.Tasks = []string{"task-ref-stats-001", "task-ref-stats-002", "task-ref-stats-003"}
	run.ClosedTasks = []string{"task-ref-stats-002"}
	run.CapacityRequests = []string{"capacity-ref-stats-001"}
	run.CapacityDecisions = []string{"capacity-ref-stats-001"}
	run.Agents = []string{"agent-ref-stats-001", "agent-ref-stats-002", "agent-ref-stats-003"}
	run.StartedAgents = []string{"agent-ref-stats-001", "agent-ref-stats-002"}
	run.FailedAgents = []string{"agent-ref-stats-003"}
	run.StoppedAgents = []string{"agent-ref-stats-002"}
	run.ConfirmedStoppedAgents = []string{"agent-ref-stats-002"}
	run.Deliveries = []string{"delivery-ref-stats-001"}
	run.Reviews = []string{"review-ref-stats-001"}
	run.ReplanDecisions = []string{"replan-ref-stats-001"}
	run.DirectorQuestions = []string{"question-ref-stats-001"}
	return run
}

func findDirectorAgentStatsForTestV0(
	t *testing.T,
	stats DirectorRunStatsV0,
	agentRef string,
) DirectorAgentStatsV0 {
	t.Helper()
	for _, agent := range stats.Agents {
		if agent.AgentRequestID == agentRef {
			return agent
		}
	}
	t.Fatalf("agent no encontrado: %s en %+v", agentRef, stats.Agents)
	return DirectorAgentStatsV0{}
}

func findDirectorTaskProgressForTestV0(
	t *testing.T,
	progress DirectorProgressStatsV0,
	taskRef string,
) DirectorTaskProgressV0 {
	t.Helper()
	for _, task := range progress.Tasks {
		if task.TaskRef == taskRef {
			return task
		}
	}
	t.Fatalf("task no encontrada: %s en %+v", taskRef, progress.Tasks)
	return DirectorTaskProgressV0{}
}

func markDirectorStatsPhaseForTestV0(
	run *orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
	status orquestacoreworkflow.OrchestrationPhaseStatusV0,
) {
	for index := range run.Phases {
		if run.Phases[index].ID == phaseID {
			run.Phases[index].Status = status
			return
		}
	}
	run.Phases = append(run.Phases, orquestacoreworkflow.OrchestrationPhaseV0{
		ID:     phaseID,
		Status: status,
	})
}
