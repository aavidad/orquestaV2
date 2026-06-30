package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto(t *testing.T) {
	descriptor := MCPDirectorStatsDescriptorV0()

	if descriptor.Name != MCPDirectorStatsToolNameV0 ||
		descriptor.Version != MCPDirectorStatsToolVersionV0 ||
		descriptor.ResourceURI != MCPDirectorStatsResourceURIV0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if descriptor.InputSchema == "" ||
		descriptor.Output == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor incompleto=%+v", descriptor)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, descriptor, 1400)
}

func TestMCPDirectorStatsToolExecutorV0DevuelveStatsDeRunStore(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-001")
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     "process-ref-mcp-stats-001",
		SessionRef:     "session-ref-mcp-stats-001",
		LaunchRef:      "launch-ref-mcp-stats-001",
		ReadinessRef:   "readiness-ref-mcp-stats-001",
		EvidenceRefs:   []string{"evidence-ref-mcp-stats-process-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProcessRegistry: registry,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:          "request-ref-mcp-director-stats-001",
		CorrelationID:      "corr-mcp-director-stats-001",
		RunRef:             run.RunID,
		IncludeProcessRefs: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.Stats == nil ||
		result.DecisionContext == nil ||
		result.OpsSnapshot == nil {
		t.Fatalf("result=%+v", result)
	}
	if result.Stats.SchemaVersion != orquestacionnucleoapp.DirectorRunStatsSchemaVersionV0 ||
		result.Stats.Counts.TasksTotal != 3 ||
		result.Stats.Counts.AgentsStarted != 1 ||
		result.Stats.Counts.AgentsControlRegistered != 1 ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 {
		t.Fatalf("stats=%+v", result.Stats)
	}
	if !containsStringMCPTestV0(result.Stats.Closure.BlockedBy, orquestacionnucleoapp.DirectorClosureBlockedByContratosV0) ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, orquestacionnucleoapp.DirectorClosureBlockedByValidacionFinalV0) {
		t.Fatalf("closure=%+v", result.Stats.Closure)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if agent.Process == nil || agent.Process.SessionRef != "session-ref-mcp-stats-001" {
		t.Fatalf("agent=%+v", agent)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(*result.DecisionContext); err != nil {
		t.Fatalf("decision context invalido: %v\n%+v", err, result.DecisionContext)
	}
	if result.DecisionContext.Lifecycle.AgentsRunning != 1 ||
		result.DecisionContext.Lifecycle.AgentsFailed != 0 ||
		len(result.DecisionContext.Agents) != 2 ||
		result.DecisionContext.Agents[0].SessionRef == "" ||
		len(result.DecisionContext.Blockers) == 0 ||
		len(result.DecisionContext.Activity) == 0 {
		t.Fatalf("decision context incompleto=%+v", result.DecisionContext)
	}
	if result.OpsSnapshot.SchemaVersion != orquestaobservability.DirectorAutonomousOpsSnapshotSchemaVersionV0 ||
		len(result.OpsSnapshot.Runs) != 1 ||
		len(result.OpsSnapshot.Agents) == 0 ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot incompleto=%+v", result.OpsSnapshot)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, result, 13000)
}

func TestMCPDirectorStatsToolExecutorV0PublicaStatusDeProcesoDesdeSnapshot(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-process-status-001")
	processRef := "process-ref-mcp-stats-process-status-001"
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     processRef,
		SessionRef:     "session-ref-mcp-stats-process-status-001",
		LaunchRef:      "launch-ref-mcp-stats-process-status-001",
		ReadinessRef:   "readiness-ref-mcp-stats-process-status-001",
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProcessRegistry: registry,
		ProcessSnapshot: mcpDirectorStatsSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				processRef: {
					ProcessRef: processRef,
					Status:     orquestaruntime.ProcessRuntimeRunningV0,
				},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RunRef:             run.RunID,
		IncludeProcessRefs: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if agent.Process == nil ||
		agent.Process.Status != orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0 ||
		agent.Process.ProcessRef != processRef {
		t.Fatalf("agent=%+v", agent)
	}
}

func TestMCPDirectorStatsToolExecutorV0ExponeGoalFirstSiExisteEstado(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-001")

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: mcpDirectorGoalStateForTestV0(run.RunID)},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-goal-001",
		CorrelationID: "corr-mcp-director-stats-goal-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		result.Goal.DirectorExecutionMode != "goal_first" ||
		result.Goal.RunRef != run.RunID ||
		result.Goal.GoalRef != "goal-ref-mcp-director-stats-001" ||
		result.Goal.ExternalGoalRef != "thread-ref-mcp-director-stats-001" ||
		result.Goal.Status != "running" ||
		!containsStringMCPTestV0(result.Goal.EvidenceRefs, "evidence-ref-mcp-director-stats-goal-state") ||
		!containsStringMCPTestV0(result.Goal.EvidenceRefs, "evidence-ref-mcp-director-stats-goal-launch") ||
		result.Stats == nil ||
		result.Stats.Status != orquestagoal.GoalStatusRunningV0 ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 ||
		result.Stats.Closure.Ready ||
		!result.Stats.Closure.Blocked ||
		result.Stats.Closure.Closed {
		t.Fatalf("goal=%+v result=%+v", result.Goal, result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, result, 13000)
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaMetadataOperacionalV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-provider-timeout-001")
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Spec.ContextRefs = append(state.Spec.ContextRefs,
		orquestagoal.GoalContextRefV0{Ref: "provider_timeout=true", Purpose: "timeout proveedor"},
		orquestagoal.GoalContextRefV0{Ref: "domain_counters={\"segments_pending\":7}", Purpose: "contadores audio"},
	)

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-goal-provider-timeout-001",
		CorrelationID: "corr-mcp-director-stats-goal-provider-timeout-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		result.Goal.CurrentPhase != "tts" ||
		result.Goal.RetryFromPhase != "tts" ||
		result.Goal.OperationalReason != "provider_timeout" ||
		result.Goal.DomainCounters["segments_pending"] != 7 ||
		result.Stats == nil ||
		result.Stats.Status != orquestagoal.GoalStatusBlockedV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, "provider_timeout") {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstMarkerSinStatePublicaRepairGoalState(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-marker-missing-state-001")
	markers := newMCPGoalRunMarkerStoreForStatusTestV0()
	if err := markers.SaveGoalWorkRunMarkerV0(context.Background(), orquestagoal.GoalWorkRunMarkerV0{
		RunRef:          run.RunID,
		GoalRef:         "goal-ref-mcp-director-stats-marker-missing-state-001",
		ExternalGoalRef: "thread-ref-mcp-director-stats-marker-missing-state-001",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		EvidenceRefs:    []string{"evidence-ref-mcp-director-stats-marker-missing-state-001"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:         orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource:  mcpDirectorGoalStateSourceForTestV0{},
		GoalMarkerSource: markers,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-goal-marker-missing-state-001",
		CorrelationID: "corr-mcp-director-stats-goal-marker-missing-state-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		result.Goal.DirectorExecutionMode != "goal_first" ||
		result.Goal.Status != "goal_first_state_missing" ||
		result.Goal.ClosureStatus != orquestagoal.GoalStatusBlockedV0 ||
		!result.Goal.ClosureNeedsRework ||
		result.Stats == nil ||
		result.Stats.Status != "goal_first_state_missing" ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 ||
		!result.Stats.Closure.Blocked ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, "goal_first_state_missing") {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionRepairGoalStateV0 ||
		result.OpsSnapshot.Decision.RunRef != run.RunID ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_state_missing" ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstAceptadoCierraStats(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-accepted-001")
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusCompleteV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		EvidenceRefs:  []string{"evidence-ref-mcp-director-stats-goal-result"},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: []string{"evidence-ref-mcp-director-stats-goal-closure"},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!result.Goal.ClosureAccepted ||
		result.Goal.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		result.Stats == nil ||
		result.Stats.Status != "closed" ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusClosedV0 ||
		!result.Stats.Closure.Closed ||
		result.Stats.Closure.Ready ||
		result.Stats.Closure.Blocked {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstBloqueadoSinTareasNoDaCienV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-timeout-no-artifacts-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusBlockedV0,
		GoalRef:       state.GoalRef,
		Summary:       "codex_app_server_goal_active_timeout",
		EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Progress.TasksTotal != 0 ||
		result.Stats.Progress.PercentComplete != 0 ||
		result.Stats.Counts.Deliveries != 0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, "blocked_no_artifacts_timeout") ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			"goal_first_blocked_no_artifacts",
		) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstBloqueadoConReceiptParcialNoPierdeEntregaV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-timeout-partial-delivery-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:     orquestagoal.GoalWorkResultSchemaV0,
		Status:            orquestagoal.GoalStatusBlockedV0,
		GoalRef:           state.GoalRef,
		Summary:           "codex_app_server_goal_active_timeout",
		ArtifactRefs:      []string{"artifact-ref-work-delivery-json-001"},
		DomainReceiptRefs: []string{"domain-receipt-ref-work-delivery-001"},
		EvidenceRefs:      []string{"evidence-ref-work-delivery-detected"},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		len(result.Goal.DomainReceiptRefs) != 1 ||
		result.Stats == nil ||
		result.Stats.Progress.TasksTotal != 0 ||
		result.Stats.Progress.PercentComplete != 0 ||
		result.Stats.Counts.Deliveries != 2 ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "domain-receipt-ref-work-delivery-001") ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, "blocked_with_partial_delivery") ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			"goal_first_blocked_with_partial_delivery",
		) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0CalculaControlSinExponerProcessRefs(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-control-redacted-001")
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     "process-ref-mcp-stats-control-redacted-001",
		SessionRef:     "session-ref-mcp-stats-control-redacted-001",
		LaunchRef:      "launch-ref-mcp-stats-control-redacted-001",
		ReadinessRef:   "readiness-ref-mcp-stats-control-redacted-001",
		EvidenceRefs:   []string{"evidence-ref-mcp-stats-control-redacted-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProcessRegistry: registry,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-control-redacted-001",
		CorrelationID: "corr-mcp-director-stats-control-redacted-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if !agent.ControlRegistered || agent.ControlState == "not_loaded" || !agent.CanStop {
		t.Fatalf("control no calculado sin process refs: %+v", agent)
	}
	if agent.Process != nil {
		t.Fatalf("process refs expuestos sin opt-in: %+v", agent.Process)
	}
	if result.DecisionContext != nil && len(result.DecisionContext.Agents) > 0 &&
		result.DecisionContext.Agents[0].SessionRef != "" {
		t.Fatalf("decision context expuso session ref sin opt-in: %+v", result.DecisionContext.Agents[0])
	}
	if result.OpsSnapshot == nil || len(result.OpsSnapshot.Agents) == 0 {
		t.Fatalf("ops_snapshot ausente: %+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0IncluyeRunControlStopRequested(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-stop-control-001")
	run.StoppedAgents = nil
	run.AgentStopRequests = nil
	run.ConfirmedStoppedAgents = nil
	control := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := control.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:         run.RunID,
		RequestedBy:    "operator",
		Reason:         "parada de prueba",
		Forced:         true,
		IdempotencyKey: "idem-mcp-director-stats-stop-control-001",
		EvidenceRefs:   []string{"evidence-ref-mcp-director-stats-stop-control-001"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		RunControl: control,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-stop-control-001",
		CorrelationID: "corr-mcp-director-stats-stop-control-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.StopControl.Status != orquestacionnucleoapp.DirectorRunStopStatusRequestedV0 ||
		result.Stats.StopControl.RunControlStatus != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		!result.Stats.StopControl.Requested ||
		result.Stats.StopControl.Propagated ||
		!result.Stats.StopControl.Pending ||
		!result.Stats.StopControl.Forced ||
		!containsStringMCPTestV0(result.Stats.StopControl.EvidenceRefs, "evidence-ref-mcp-director-stats-stop-control-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0DiagnosticaExternalWorkAgenteSolicitadoNoArrancado(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-agent-not-started-001")
	run.ProjectRef = "opes"
	run.AppSpecRef = "app-spec-external-work-opes-qa-visual"
	run.Agents = []string{"agent-ref-qa-visual-001"}
	run.StartedAgents = nil
	run.FailedAgents = nil
	run.LostAgents = nil
	run.StoppedAgents = nil
	run.ConfirmedStoppedAgents = nil
	run.Deliveries = nil

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-agent-not-started-001",
		CorrelationID: "corr-mcp-director-stats-agent-not-started-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Counts.AgentsRequested != 1 ||
		result.Stats.Counts.AgentsStarted != 0 ||
		result.Stats.Counts.AgentsInFlight != 0 ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0,
		) {
		t.Fatalf("result=%+v", result)
	}
	if !strings.Contains(result.Stats.Progress.Issues[0].Message, "cause=unknown") ||
		!strings.Contains(result.Stats.Progress.Issues[0].Message, "check_capacity_auth_runtime_queue_outbox_policy") {
		t.Fatalf("issues=%+v", result.Stats.Progress.Issues)
	}
}

func TestMCPDirectorStatsToolExecutorV0NoClasificaOPESDirectoComoExternalWorkV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-opes-direct-agent-not-started-001")
	run.ProjectRef = "opes-revision-tcae"
	run.AppSpecRef = "app-spec-opes-qa-visual"
	run.Agents = []string{"agent-ref-qa-visual-direct-001"}
	run.StartedAgents = nil
	run.FailedAgents = nil
	run.LostAgents = nil
	run.StoppedAgents = nil
	run.ConfirmedStoppedAgents = nil
	run.Deliveries = nil

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-opes-direct-agent-not-started-001",
		CorrelationID: "corr-mcp-director-stats-opes-direct-agent-not-started-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			mcpDirectorStatsAgentRequestedNotStartedV0,
		) ||
		mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0,
		) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0DiagnosticaExternalWorkStoppedSinEntrega(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-opes-psicologo-rework-visual-019-030-20260626")
	run.ProjectRef = "opes"
	run.AppSpecRef = "app-spec-external-work-opes-rework-visual"
	run.Status = "stopped"
	run.Agents = nil
	run.StartedAgents = nil
	run.FailedAgents = nil
	run.LostAgents = nil
	run.StoppedAgents = nil
	run.ConfirmedStoppedAgents = nil
	run.Deliveries = nil
	run.DeliveredAgents = nil

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-stopped-no-delivery-001",
		CorrelationID: "corr-mcp-director-stats-stopped-no-delivery-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			mcpDirectorStatsExternalWorkStoppedNoDeliveryV0,
		) {
		t.Fatalf("result=%+v", result)
	}
	if !strings.Contains(result.Stats.Progress.Issues[0].Message, "relaunch_or_replan_external_work_with_causal_error") {
		t.Fatalf("issues=%+v", result.Stats.Progress.Issues)
	}
}

func TestMCPDirectorStatsToolExecutorV0DiagnosticaExternalWorkDoneSinAgenteMaterializado(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-opes-tractorista-tema-001-done-sin-agentes")
	run.ProjectRef = "opes"
	run.AppSpecRef = "app-spec-external-work-opes-tractorista"
	run.Status = "done"
	run.Agents = nil
	run.StartedAgents = nil
	run.FailedAgents = nil
	run.LostAgents = nil
	run.StoppedAgents = nil
	run.ConfirmedStoppedAgents = nil
	run.Deliveries = nil
	run.DeliveredAgents = nil

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-mcp-director-stats-done-no-agent-001",
		CorrelationID: "corr-mcp-director-stats-done-no-agent-001",
		RunRef:        run.RunID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		!mcpDirectorStatsProgressIssueExistsV0(
			result.Stats.Progress.Issues,
			mcpDirectorStatsExternalWorkNoAgentMaterializedV0,
		) {
		t.Fatalf("result=%+v", result)
	}
	if !strings.Contains(result.Stats.Progress.Issues[0].Message, "without agents or delivery") ||
		!strings.Contains(result.Stats.Progress.Issues[0].Message, "relaunch_or_replan_external_work_with_causal_error") {
		t.Fatalf("issues=%+v", result.Stats.Progress.Issues)
	}
}

func TestMCPDirectorStatsToolExecutorV0DevuelveIssuesPublicos(t *testing.T) {
	result, err := (MCPDirectorStatsToolExecutorV0{}).Execute(
		context.Background(),
		MCPDirectorStatsToolInputV0{RequestID: "request-ref-mcp-director-stats-invalid-001"},
	)
	if err != nil {
		t.Fatalf("Execute invalid: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0ResuelveRunPorJobExterno(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-job-001")
	source := mcpDirectorExternalJobStatsSourceForTestV0{
		Stats: MCPDirectorExternalJobStatsV0{
			AppRef:       "opes",
			JobRef:       "job-ref-opes-001",
			WorkKind:     "draft_content_block",
			ChangeRef:    "opes-job-job-ref-opes-001",
			RunRef:       run.RunID,
			TaskRef:      "task-ref-opes-001",
			AgentRef:     "agent-ref-opes-001",
			Status:       "running",
			StatusReason: "parent_integration_pending",
			DeliveryRefs: []string{"delivery-ref-opes-001"},
			IssueRefs:    []string{"issue-ref-parent-integration-pending"},
			Diagnostics: []MCPDirectorExternalJobDiagnosticV0{{
				Code:         "external_job_parent_integration_pending",
				Scope:        "job-ref-opes-001",
				EvidenceRefs: []string{"task-ref-opes-001"},
			}},
		},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ExternalJobSource: source,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:      "request-ref-mcp-director-stats-job-001",
		CorrelationID:  "corr-mcp-director-stats-job-001",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-001",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.ExternalJob == nil ||
		result.ExternalJob.JobRef != "job-ref-opes-001" ||
		result.ExternalJob.TaskRef != "task-ref-opes-001" ||
		result.ExternalJob.StatusReason != "parent_integration_pending" ||
		len(result.ExternalJob.IssueRefs) != 1 ||
		len(result.ExternalJob.Diagnostics) != 1 ||
		result.Stats == nil {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, result, 13000)
}

func TestMCPDirectorStatsToolExecutorV0RecuperaRunCanonicoSiRunRefObsoleto(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-job-canonico-001")
	source := mcpDirectorExternalJobStatsSourceForTestV0{
		Stats: MCPDirectorExternalJobStatsV0{
			AppRef:   "opes",
			JobRef:   "job-ref-opes-canonico-001",
			WorkKind: "draft_content_block",
			RunRef:   run.RunID,
			TaskRef:  "task-ref-opes-canonico-001",
			AgentRef: "agent-ref-opes-canonico-001",
			Status:   "running",
		},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ExternalJobSource: source,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:      "request-ref-mcp-director-stats-job-canonico-001",
		CorrelationID:  "corr-mcp-director-stats-job-canonico-001",
		RunRef:         "run-ref-obsoleto",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-canonico-001",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.ExternalJob == nil ||
		result.ExternalJob.RunRef != run.RunID {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0IncluyeUsoDeAgentesOptIn(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-usage-001")

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		AgentUsageSource: mcpDirectorStatsUsageSourceForTestV0{
			Observations: []orquestacionnucleoapp.AgentUsageStatsObservationV0{{
				AgentRequestID: "agent-ref-stats-001",
				RuntimeKind:    "codex",
				ConnectorRef:   "connector-ref-codex-001",
				ProfileRef:     "profile-ref-codex-001",
				CapacityLevel:  "xhigh",
				QuotaStatus:    orquestacionnucleoapp.DirectorAgentUsageQuotaAvailableV0,
				QuotaRemaining: 90,
				QuotaLimit:     100,
				TotalTokens:    1500,
			}},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:         "request-ref-mcp-director-stats-usage-001",
		CorrelationID:     "corr-mcp-director-stats-usage-001",
		RunRef:            run.RunID,
		IncludeAgentUsage: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if agent.Usage == nil ||
		agent.Usage.QuotaRemaining != 90 ||
		agent.Usage.TotalTokens != 1500 {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func TestMCPDirectorStatsToolExecutorV0DecisionContextCompletoParaDirector(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-context-001")
	run.Agents = []string{"agent-ref-stats-001", "agent-ref-stats-002", "agent-ref-stats-003"}
	run.StartedAgents = []string{"agent-ref-stats-001", "agent-ref-stats-002", "agent-ref-stats-003"}
	run.FailedAgents = []string{"agent-ref-stats-002"}
	run.StoppedAgents = []string{"agent-ref-stats-003"}
	run.AgentStopRequests = []string{
		orquestacoreworkflow.AgentStopRequestProjectionRefV0(orquestacoreworkflow.AgentStopRequestedPayloadV0{
			AgentRequestID: "agent-ref-stats-003",
			ReasonCode:     "run_stop_requested",
			Summary:        "Parada solicitada por control de run.",
		}),
	}
	run.ConfirmedStoppedAgents = []string{"agent-ref-stats-003"}
	run.ReworkRequests = []string{"rework-ref-mcp-director-stats-001"}
	run.ReplanDecisions = []string{"replan-ref-mcp-director-stats-001"}
	for index := range run.Phases {
		if run.Phases[index].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			run.Phases[index].OpenedAt = "2026-05-10T08:00:00Z"
		}
	}
	observation := mcpDirectorStatsProgressObservationForTestV0(run.RunID)
	observation.Report.Status = orquestaruntime.AgentStalledV0
	observation.Report.NoProgressTicks = 4

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProgressSource: mcpDirectorStatsProgressSourceForTestV0{
			Observations: []orquestacionnucleoapp.AgentProgressObservationV0{observation},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:            "request-ref-mcp-director-stats-context-001",
		CorrelationID:        "corr-mcp-director-stats-context-001",
		RunRef:               run.RunID,
		OccurredAt:           "2026-05-10T09:00:00Z",
		IncludeAgentProgress: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	context := result.DecisionContext
	if context == nil {
		t.Fatalf("sin decision_context: %+v", result)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(*context); err != nil {
		t.Fatalf("decision context invalido: %v\n%+v", err, context)
	}
	if context.Lifecycle.AgentsRunning != 1 ||
		context.Lifecycle.AgentsFailed != 1 ||
		context.Lifecycle.AgentsStopped != 1 ||
		context.Quietness.MaxNoProgressTicks != 4 ||
		context.ReworkReplan.ReworkRequests != 1 ||
		context.ReworkReplan.ReplanDecisions != 1 ||
		!context.Closure.Blocked {
		t.Fatalf("decision context incompleto=%+v", context)
	}
	if !mcpDirectorContextHasCurrentPhaseDurationV0(context, "programacion", 3600) {
		t.Fatalf("phases sin duracion actual: %+v", context.Phases)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-003")
	if agent.StopReasonCode != "run_stop_requested" ||
		agent.StopReasonSource != orquestacionnucleoapp.DirectorAgentStopReasonSourceStopRequestV0 {
		t.Fatalf("agent stop reason=%+v", agent)
	}
	contextAgent := mcpDirectorContextAgentForTestV0(t, context, "agent-ref-stats-003")
	if contextAgent.StopReasonCode != "run_stop_requested" ||
		contextAgent.StopReasonSource != orquestacionnucleoapp.DirectorAgentStopReasonSourceStopRequestV0 {
		t.Fatalf("context agent stop reason=%+v", contextAgent)
	}
}

type mcpDirectorStatsSnapshotSourceForTestV0 struct {
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
}

func (source mcpDirectorStatsSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return source.snapshots[strings.TrimSpace(processRef)], nil
}
