package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
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
	for _, want := range []string{
		"external_job?{status,status_reason?,external_goal_ref?,issue_refs?,evidence_refs?,diagnostics?}",
		"goal?{goal_ref,external_goal_ref?,status,closure_status?,context_budget_total_bytes?,static_prompt_bytes?,dynamic_context_bytes?,code_context_cache_status?,artifact_refs?,domain_receipt_refs?,expected_terminal_receipt_refs?,issue_codes?,evidence_refs?}",
	} {
		if !strings.Contains(descriptor.Output, want) {
			t.Fatalf("descriptor director.stats no declara evidencia accionable %q: %s", want, descriptor.Output)
		}
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

func TestMCPDirectorStatsToolExecutorV0ProyectaContextBudgetGoalFirstV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-context-budget-001")
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.ContextBudget = orquestagoal.GoalContextBudgetV0{
		ContextBudgetTotalBytes:  8192,
		StaticPromptBytes:        2048,
		QueriedContextBytes:      384,
		MaterializedContextBytes: 768,
		DynamicContextBytes:      6144,
		CodeContextCacheStatus:   "hit",
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Goal == nil ||
		result.Goal.ContextBudgetTotalBytes != 8192 ||
		result.Goal.StaticPromptBytes != 2048 ||
		result.Goal.DynamicContextBytes != 6144 ||
		result.Goal.CodeContextCacheStatus != "hit" {
		t.Fatalf("goal context budget no proyectado: %+v", result.Goal)
	}
}

func TestMCPDirectorStatsToolExecutorV0DerivaVidaDesdeProyeccionV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-estado-vivo-001")
	run.Tasks = []string{"task-ref-stats-estado-vivo-001"}
	run.ClosedTasks = []string{"task-ref-stats-estado-vivo-001"}
	run.DeliveredTasks = []string{"task-ref-stats-estado-vivo-001"}
	run.Agents = nil
	run.StartedAgents = nil
	run.Deliveries = []string{"delivery-ref-stats-estado-vivo-001"}
	estadoVivo := &fakeMCPAutoprogrammingEstadoVivoSourceV0{
		evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
			RunRef:       run.RunID,
			Fuente:       "process_snapshot",
			ProcesoVivo:  true,
			ObservadoEn:  "2026-07-03T09:59:00Z",
			EvidenceRefs: []string{"evidence-ref-estado-vivo-process-live-001"},
		}},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:         orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EstadoVivoSource: estadoVivo,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RunRef:     run.RunID,
		OccurredAt: "2026-07-03T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Status != mcpDirectorStatsEstadoVivoProcesoVivoV0 ||
		result.Stats.Progress.PercentComplete >= 100 ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 ||
		!result.Stats.Closure.Blocked ||
		result.Stats.Closure.Closed ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, mcpDirectorStatsEstadoVivoProcesoVivoV0) ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockerRefs, "evidence-ref-estado-vivo-process-live-001") {
		t.Fatalf("stats=%+v", result.Stats)
	}
	if len(estadoVivo.inputs) != 1 ||
		estadoVivo.inputs[0].RunRef != run.RunID ||
		estadoVivo.inputs[0].Limit != mcpDirectorStatsEstadoVivoEvidenceLimitV0 {
		t.Fatalf("estado vivo inputs=%+v", estadoVivo.inputs)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPDirectorStatsToolExecutorV0ConflictoEstadoVivoNuncaProyectaVerdeV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-estado-vivo-conflict-001")
	run.Tasks = []string{"task-ref-stats-estado-vivo-conflict-001"}
	run.ClosedTasks = []string{"task-ref-stats-estado-vivo-conflict-001"}
	run.DeliveredTasks = []string{"task-ref-stats-estado-vivo-conflict-001"}
	run.Agents = nil
	run.StartedAgents = nil
	run.Deliveries = []string{"delivery-ref-stats-estado-vivo-conflict-001"}
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusCompleteV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       state.GoalRef,
		EvidenceRefs:  []string{"evidence-ref-goal-result-accepted-before-conflict"},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: []string{"evidence-ref-goal-closure-accepted-before-conflict"},
	}
	estadoVivo := &fakeMCPAutoprogrammingEstadoVivoSourceV0{
		evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
			RunRef:       run.RunID,
			Fuente:       "process_snapshot",
			ProcesoVivo:  true,
			ObservadoEn:  "2026-07-03T09:59:00Z",
			EvidenceRefs: []string{"evidence-ref-estado-vivo-process-live-conflict"},
		}, {
			RunRef:       run.RunID,
			Fuente:       "receipt",
			Terminal:     true,
			Aceptado:     true,
			ObservadoEn:  "2026-07-03T09:58:00Z",
			EvidenceRefs: []string{"evidence-ref-estado-vivo-terminal-accepted-conflict"},
		}},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:         orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource:  mcpDirectorGoalStateSourceForTestV0{State: state},
		EstadoVivoSource: estadoVivo,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RunRef:     run.RunID,
		OccurredAt: "2026-07-03T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Status != mcpDirectorStatsEstadoVivoConflictoV0 ||
		result.Stats.Progress.PercentComplete >= 100 ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 ||
		!result.Stats.Closure.Blocked ||
		result.Stats.Closure.Closed ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, mcpDirectorStatsEstadoVivoConflictoV0) ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, mcpDirectorStatsEstadoVivoConflictoV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		result.OpsSnapshot.Decision.ReasonCode != "attention_required" {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
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
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "codex_app_server_goal_active_timeout",
		}},
		EvidenceRefs: []string{"evidence-ref-codex-app-server-goal-active-timeout"},
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
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "codex_app_server_goal_active_timeout",
		}},
		EvidenceRefs: []string{"evidence-ref-work-delivery-detected"},
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

func TestMCPDirectorStatsToolExecutorV0GoalFirstReconcilaWorkDeliveryMaterializadoV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-timeout-materialized-delivery-001")
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
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "codex_app_server_goal_active_timeout",
		}},
		EvidenceRefs: []string{"evidence-ref-codex-app-server-goal-active-timeout"},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource:            mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceForTestV0{},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.DomainReceiptRefs, "domain-receipt-ref-materialized-work-delivery-001") ||
		!containsStringMCPTestV0(result.Goal.EvidenceRefs, "evidence-ref-goal-materialized-work-delivery-detected") ||
		result.Stats == nil ||
		result.Stats.Progress.PercentComplete != 0 ||
		result.Stats.Counts.Deliveries != 1 ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "domain-receipt-ref-materialized-work-delivery-001") ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, "blocked_with_partial_delivery") ||
		mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, "goal_first_blocked_no_artifacts") ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, "goal_first_blocked_with_partial_delivery") {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaQAFailedPublicTextV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-qa-failed-public-text-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-qa-failed-public-text-001"},
				EvidenceRefs: []string{"evidence-ref-goal-materialized-qa-failed-public-text-001"},
				IssueCodes:   []string{MCPGoalFirstQAFailedPublicTextV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstQAFailedPublicTextV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstQAFailedPublicTextV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstQAFailedPublicTextV0) ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "artifact-ref-materialized-qa-failed-public-text-001") ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstQAFailedPublicTextV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0NoUsaSummaryComoIssueCodeV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-summary-not-issue-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusRunningV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		Status:  orquestagoal.GoalStatusRunningV0,
		GoalRef: state.GoalRef,
		Summary: "checkpoint_started; implementacion pendiente",
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: "checkpoint_started",
		}},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, "checkpoint_started") ||
		containsStringMCPTestV0(result.Goal.IssueCodes, "checkpoint_started; implementacion pendiente") {
		t.Fatalf("issue_codes=%+v", result.Goal)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathsOmitidosV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-artifact-paths-omitted-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-omitted-path-001"},
				EvidenceRefs: []string{
					"evidence-ref-goal-materialized-artifact-paths-omitted",
					"evidence-ref-goal-materialized-artifact-paths-omitted:run:tema_001",
				},
				IssueCodes: []string{MCPGoalFirstArtifactPathsOmittedMaterializedV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstArtifactPathsOmittedMaterializedV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstArtifactPathsOmittedMaterializedV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstArtifactPathsOmittedMaterializedV0) ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstArtifactPathsOmittedMaterializedV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaOutOfScopeMaterializedV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-out-of-scope-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-out-of-scope-001"},
				EvidenceRefs: []string{
					"evidence-ref-goal-materialized-out-of-scope-artifacts",
					"evidence-ref-goal-materialized-out-of-scope-artifacts:run:tema_003_html_index",
				},
				IssueCodes: []string{MCPGoalFirstOutOfScopeMaterializedArtifactsV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstOutOfScopeMaterializedArtifactsV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstOutOfScopeMaterializedArtifactsV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstOutOfScopeMaterializedArtifactsV0) ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstOutOfScopeMaterializedArtifactsV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaRuntimeWriteSetViolationV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-runtime-write-set-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-runtime-write-set-outside-fuera-md"},
				EvidenceRefs: []string{"evidence-ref-codex-app-server-runtime-write-set-violation"},
				IssueCodes:   []string{mcpAutoprogrammingActionRuntimeWriteSetViolationV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, mcpAutoprogrammingActionRuntimeWriteSetViolationV0) ||
		result.Stats == nil ||
		result.Stats.Status != mcpAutoprogrammingActionRuntimeWriteSetViolationV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, mcpAutoprogrammingActionRuntimeWriteSetViolationV0) ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, mcpAutoprogrammingActionRuntimeWriteSetViolationV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtefactosParcialesV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-partial-artifacts-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-partial-001"},
				EvidenceRefs: []string{"evidence-ref-goal-materialized-partial-artifacts-written"},
				IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstPartialArtifactsWrittenV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstPartialArtifactsWrittenV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstPartialArtifactsWrittenV0) ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "artifact-ref-materialized-partial-001") ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstPartialArtifactsWrittenV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaPhase0NoPublicableV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-phase0-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-phase0-001"},
				EvidenceRefs: []string{"evidence-ref-goal-materialized-phase0-complete-non-publishable"},
				IssueCodes:   []string{MCPGoalFirstPhase0CompleteNonPublishableV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstPhase0CompleteNonPublishableV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstPhase0CompleteNonPublishableV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstPhase0CompleteNonPublishableV0) ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "artifact-ref-materialized-phase0-001") ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstPhase0CompleteNonPublishableV0) {
		t.Fatalf("goal=%+v stats=%+v", result.Goal, result.Stats)
	}
}

func TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaRequiredTestEvidenceAusenteV0(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-goal-required-test-evidence-001")
	run.Tasks = nil
	run.ClosedTasks = nil
	run.DeliveredTasks = nil
	run.Deliveries = nil
	run.Agents = nil
	run.StartedAgents = nil
	state := mcpDirectorGoalStateForTestV0(run.RunID)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		GoalStateSource: mcpDirectorGoalStateSourceForTestV0{State: state},
		GoalMaterializedRefsSource: mcpDirectorMaterializedRefsSourceStaticForTestV0{
			Resolved: MCPDirectorGoalMaterializedRefsV0{
				ArtifactRefs: []string{"artifact-ref-materialized-required-test-evidence-001"},
				EvidenceRefs: []string{"evidence-ref-goal-materialized-required-test-evidence-missing"},
				IssueCodes:   []string{MCPGoalFirstRequiredTestEvidenceMissingV0},
			},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{RunRef: run.RunID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Goal == nil ||
		!containsStringMCPTestV0(result.Goal.IssueCodes, MCPGoalFirstRequiredTestEvidenceMissingV0) ||
		result.Stats == nil ||
		result.Stats.Status != MCPGoalFirstRequiredTestEvidenceMissingV0 ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, MCPGoalFirstRequiredTestEvidenceMissingV0) ||
		!containsStringMCPTestV0(result.Stats.Refs.Deliveries, "artifact-ref-materialized-required-test-evidence-001") ||
		!mcpDirectorStatsProgressIssueExistsV0(result.Stats.Progress.Issues, MCPGoalFirstRequiredTestEvidenceMissingV0) {
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

type mcpDirectorMaterializedRefsSourceForTestV0 struct{}

type mcpDirectorMaterializedRefsSourceStaticForTestV0 struct {
	Resolved MCPDirectorGoalMaterializedRefsV0
}

func (mcpDirectorMaterializedRefsSourceForTestV0) ResolveDirectorGoalMaterializedRefsV0(
	context.Context,
	orquestagoal.GoalWorkStateV0,
) (MCPDirectorGoalMaterializedRefsV0, bool, error) {
	return MCPDirectorGoalMaterializedRefsV0{
		DomainReceiptRefs: []string{"domain-receipt-ref-materialized-work-delivery-001"},
		EvidenceRefs:      []string{"evidence-ref-goal-materialized-work-delivery-detected"},
		IssueCodes:        []string{"goal_first_materialized_work_delivery_detected"},
	}, true, nil
}

func (source mcpDirectorMaterializedRefsSourceStaticForTestV0) ResolveDirectorGoalMaterializedRefsV0(
	context.Context,
	orquestagoal.GoalWorkStateV0,
) (MCPDirectorGoalMaterializedRefsV0, bool, error) {
	return source.Resolved, true, nil
}

func (source mcpDirectorStatsSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return source.snapshots[strings.TrimSpace(processRef)], nil
}
