package orquestamcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAutoprogrammingStatusDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingStatusToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingStatusResourceURIV0 ||
		len(descriptor.Invariantes) == 0 ||
		!strings.Contains(descriptor.Output, "efficiency_summary") {
		t.Fatalf("descriptor incompleto: %+v", descriptor)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DelegaEnColaYRun(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: queue,
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RequestID:     "request-ref-autop-status-001",
		CorrelationID: "corr-autop-status-001",
		RunRef:        "run-ref-autop-status-001",
		QueueRef:      "queue-ref-autop-status-001",
		QueueLimit:    3,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.Queue == nil ||
		result.Run == nil ||
		result.Operator == nil ||
		result.OpsSnapshot == nil ||
		result.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("result=%+v", result)
	}
	if !result.Operator.QueueLive ||
		len(result.Operator.ActiveRuns) != 1 ||
		len(result.Operator.AgentsInFlight) != 1 ||
		len(result.Operator.ClosureBlockers) != 1 ||
		len(result.Operator.SafeActions) != 0 {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "legacy_supervisor_actions_disabled") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if len(result.Projects) != 1 ||
		result.Projects[0].QueueCount != 1 ||
		result.Projects[0].AgentsInFlight != 1 ||
		len(result.Tasks) != 1 ||
		result.Tasks[0].TaskRef != "task-ref-autop-status-001" ||
		result.Tasks[0].DeliveryRef != "delivery-ref-autop-status-001" ||
		!result.Tasks[0].DecisionRequired ||
		len(result.Tasks[0].EvidenceRefs) != 1 ||
		len(result.Agents) != 1 ||
		result.Agents[0].TaskRef != "task-ref-autop-status-001" ||
		result.Agents[0].ProcessRef != "process-ref-autop-status-001" ||
		result.Agents[0].RuntimeKind != "cli" ||
		result.Agents[0].CapacityLevel != "medium" ||
		result.Agents[0].TotalTokens != 123 ||
		len(result.Agents[0].EvidenceRefs) != 3 {
		t.Fatalf("projects=%+v tasks=%+v agents=%+v", result.Projects, result.Tasks, result.Agents)
	}
	if result.OpsSnapshot.Queue.Count != 1 ||
		len(result.OpsSnapshot.Runs) != 1 ||
		result.OpsSnapshot.Decision.Action != "review_replan" ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.SchemaVersion != MCPAutoprogrammingEfficiencySummarySchemaVersionV0 ||
		result.EfficiencySummary.State != "live" ||
		result.EfficiencySummary.OperationalHealthPercentage != 100 ||
		result.EfficiencySummary.AlivePercentage != 100 ||
		result.EfficiencySummary.AliveStuckStatus != "observed" ||
		result.EfficiencySummary.AliveStuckSampleSize != 1 ||
		result.EfficiencySummary.QueueCandidates != 1 ||
		result.EfficiencySummary.AgentsInFlight != 1 {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
	if queue.input.Action != MCPRunQueuePriorityActionRankV0 ||
		queue.input.Limit != 3 ||
		stats.input.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("queue=%+v stats=%+v", queue.input, stats.input)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0PublicaDiagnosticosConfigurados(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		StatusDiagnostics: []MCPAutoprogrammingDiagnosticV0{
			{
				Code:         " codex_goal_backend_degraded ",
				Scope:        " app_goal ",
				Message:      " codex goal backend degradado: codex_app_server_auth_missing ",
				EvidenceRefs: []string{"", " evidence-ref-server-codex-goal-backend-degraded-app_goal "},
			},
			{Code: "   ", Scope: "ignored"},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "codex_goal_backend_degraded") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	var found MCPAutoprogrammingDiagnosticV0
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "codex_goal_backend_degraded" {
			found = diagnostic
			break
		}
	}
	if found.Scope != "app_goal" ||
		!strings.Contains(found.Message, "codex_app_server_auth_missing") ||
		len(found.EvidenceRefs) != 1 ||
		found.EvidenceRefs[0] != "evidence-ref-server-codex-goal-backend-degraded-app_goal" {
		t.Fatalf("diagnostico=%+v diagnostics=%+v", found, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0RecomiendaObserveGoalParaRunGoalFirst(t *testing.T) {
	runRef := "run-ref-autop-status-goal-first-001"
	goalRef := "goal-ref-autop-status-goal-first-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Terminar app goal-first sin loop legacy.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{"evidence-ref-goal-first-status-test"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        runRef,
				AppRef:        "app-ref-goal-first",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:     runRef,
				ProjectRef: "app-ref-goal-first",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:     1,
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					Tasks: []orquestacionnucleoapp.DirectorTaskProgressV0{{
						TaskRef:        "task-ref-goal-first-legacy-residual-001",
						Status:         "in_progress",
						AgentRequestID: "agent-ref-goal-first-legacy-residual-001",
					}},
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID:    "agent-ref-goal-first-legacy-residual-001",
					Status:            orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:          true,
					ControlRegistered: true,
					ControlState:      orquestacionnucleoapp.DirectorAgentControlStateRegisteredV0,
				}},
			},
		},
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: runRef,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_observe_required") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
	if len(result.Tasks) != 0 || len(result.Agents) != 0 {
		t.Fatalf("goal-first no debe exponer proyecciones legacy: tasks=%+v agents=%+v", result.Tasks, result.Agents)
	}
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != "observe_goal" ||
		result.OpsSnapshot.Decision.RunRef != runRef ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_observe_required" {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ListaGoalCompletePendingClosure(t *testing.T) {
	runRef := "run-ref-autop-status-goal-complete-pending-001"
	goalRef := "goal-ref-autop-status-goal-complete-pending-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusCompleteV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Validar cierre pendiente de goal-first.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{"evidence-ref-goal-complete-pending-status-test"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-complete-pending-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        runRef,
				AppRef:        "app-ref-goal-complete-pending",
				Status:        "running",
				PriorityScore: 80,
			}},
		},
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_observe_required") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.GoalClosurePending != 1 ||
		result.QueueHealth.QueueRuns != 1 ||
		result.QueueHealth.ObservedRuns != 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != "observe_goal" ||
		result.OpsSnapshot.Decision.RunRef != runRef ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_observe_required" {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalFirstMarkerSinStateNoSupervisaLegacy(t *testing.T) {
	runRef := "run-ref-autop-status-goal-marker-missing-state-001"
	goalRef := "goal-ref-autop-status-goal-marker-missing-state-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	goalMarkers := newMCPGoalRunMarkerStoreForStatusTestV0()
	if err := goalMarkers.SaveGoalWorkRunMarkerV0(context.Background(), orquestagoal.GoalWorkRunMarkerV0{
		RunRef:       runRef,
		GoalRef:      goalRef,
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:       orquestagoal.GoalStatusRunningV0,
		EvidenceRefs: []string{"evidence-ref-goal-marker-missing-state-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        runRef,
				AppRef:        "app-ref-goal-marker-missing-state",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:     runRef,
				ProjectRef: "app-ref-goal-marker-missing-state",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:     1,
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					Tasks: []orquestacionnucleoapp.DirectorTaskProgressV0{{
						TaskRef:        "task-ref-goal-marker-legacy-residual-001",
						Status:         "in_progress",
						AgentRequestID: "agent-ref-goal-marker-legacy-residual-001",
					}},
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-goal-marker-legacy-residual-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
				}},
			},
		},
		GoalStateStore:               goalStates,
		GoalRunMarkerStore:           goalMarkers,
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: runRef,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_state_missing") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Blocked != 1 ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != mcpAutoprogrammingActionGoalFirstStateMissingV0 ||
		result.StaleRunning[0].RecommendedAction != "repair_goal_state_before_legacy_supervision" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
	if len(result.Tasks) != 0 || len(result.Agents) != 0 {
		t.Fatalf("goal-first marker no debe exponer proyecciones legacy: tasks=%+v agents=%+v", result.Tasks, result.Agents)
	}
	if result.Operator == nil ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionRepairGoalStateV0 ||
		result.OpsSnapshot.Decision.RunRef != runRef ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_state_missing" ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ListaGoalActivoAunqueColaNoVisible(t *testing.T) {
	runRef := "run-ref-autop-status-goal-listed-001"
	goalRef := "goal-ref-autop-status-goal-listed-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Observar goal aunque no aparezca en cola.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{"evidence-ref-goal-listed-status-test"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-listed-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		len(result.Errores) != 0 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_observe_required") ||
		result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) {
		t.Fatalf("result=%+v diagnostics=%+v operator=%+v", result, result.Diagnostics, result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ListaGoalBloqueadoAunqueColaNoVisible(t *testing.T) {
	runRef := "run-ref-autop-status-goal-blocked-listed-001"
	goalRef := "goal-ref-autop-status-goal-blocked-listed-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusInvalidV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reparar un fallo durable goal-first sin ocultarlo por cola vacia.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusInvalidV0,
			EvidenceRefs: []string{"evidence-ref-goal-blocked-listed-status-test"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-listed-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore:               goalStates,
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		len(result.Errores) != 0 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_blocked") ||
		hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_observe_required") {
		t.Fatalf("result=%+v diagnostics=%+v", result, result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Blocked != 1 ||
		result.QueueHealth.ObservedRuns != 1 ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.GoalClosurePending != 0 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != mcpAutoprogrammingActionGoalFirstBlockedV0 ||
		result.StaleRunning[0].RunRef != runRef ||
		result.StaleRunning[0].RecommendedAction != "review_replan_goal_first" {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	if result.Operator == nil ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		result.OpsSnapshot.Decision.RunRef != runRef ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_blocked" ||
		!result.OpsSnapshot.Decision.Attention {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0WriteSetReadOnlyPideConfigurarSandboxV0(t *testing.T) {
	runRef := "run-ref-autop-status-write-set-read-only-001"
	goalRef := "goal-ref-autop-status-write-set-read-only-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusInvalidV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "No elevar sandbox read-only cuando el goal necesita write-set.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusInvalidV0,
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0,
			}},
			EvidenceRefs: []string{"evidence-ref-codex-app-server-write-set-requires-workspace-write"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-write-set-read-only-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0 ||
		action.RecommendedAction != mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceWriteSetRequiresWorkspaceWriteV0) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-codex-app-server-write-set-requires-workspace-write") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalBloqueadoConBackendActivoPublicaSnapshotAccionable(t *testing.T) {
	runRef := "run-ref-autop-status-goal-backend-active-001"
	goalRef := "goal-ref-autop-status-goal-backend-active-001"
	externalGoalRef := "thread-ref-autop-status-goal-backend-active-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "No ocultar goal Codex activo tras cola local stopped.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status:            orquestagoal.GoalStatusRunningV0,
			GoalRef:           goalRef,
			ExternalGoalRef:   externalGoalRef,
			ArtifactRefs:      []string{"artifact-ref-backend-active-001"},
			DomainReceiptRefs: []string{"receipt-ref-backend-active-001"},
			EvidenceRefs:      []string{"result-ref-backend-active-001"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: []string{"closure-ref-backend-active-001"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-backend-active-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 4321,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-backend-active-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DeliveryRef: "delivery-ref-backend-active-001",
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T10:15:00Z",
					},
				},
				Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
					ProcessRef: "process-ref-backend-active-001",
					SessionRef: "session-ref-backend-active-001",
					Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionActiveNoCheckpointYetV0 ||
		action.Severity != "info" ||
		action.GoalRef != goalRef ||
		action.ExternalGoalRef != externalGoalRef ||
		action.GoalStatus != "active" ||
		action.RunStatus != "stopped" ||
		action.ResultRef != "result-ref-backend-active-001" ||
		action.ClosureStatus != orquestagoal.GoalStatusBlockedV0 ||
		!action.ClosureNeedsRework ||
		action.RecommendedAction != "observe_goal_backend_wait_for_checkpoint" ||
		!strings.Contains(action.Reason, "goal_backend_active_no_checkpoint_yet") ||
		action.ProcessAliveCount != 1 ||
		action.TokensUsed != 4321 ||
		action.LastOutputAt != "2026-07-01T10:15:00Z" ||
		action.LastArtifactAt != "2026-07-01T10:15:00Z" {
		t.Fatalf("action=%+v", action)
	}
	if !hasStringMCPAutoprogrammingStatusTestV0(action.ProcessRefs, "process-ref-backend-active-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-backend-active-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.DomainReceiptRefs, "receipt-ref-backend-active-001") {
		t.Fatalf("action refs=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyHighConsumptionEsBloqueante(t *testing.T) {
	runRef := "run-ref-autop-status-goal-checkpoint-only-high-tokens-001"
	goalRef := "goal-ref-autop-status-goal-checkpoint-only-high-tokens-001"
	externalGoalRef := "thread-ref-autop-status-goal-checkpoint-only-high-tokens-001"
	checkpointRef := "artifact-ref-checkpoint:run-ref-autop-status-goal-checkpoint-only-high-tokens-001:checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Detectar alto consumo con solo checkpoint.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			ArtifactRefs:    []string{checkpointRef},
			EvidenceRefs:    []string{"result-ref-checkpoint-only-high-tokens-001"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-checkpoint-only-high-tokens-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 150000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-checkpoint-only-high-tokens-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:10:00Z",
					},
				},
				Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
					ProcessRef: "process-ref-checkpoint-only-high-tokens-001",
					Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{checkpointRef},
			EvidenceRefs:    []string{"evidence-ref-checkpoint-only-high-tokens-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 150000 ||
		action.RecommendedAction != "replan_narrow_context" ||
		!strings.Contains(action.Reason, "checkpoint_only_high_consumption") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, checkpointRef) ||
		len(action.DomainReceiptRefs) != 0 ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0SinCheckpointConsumoMedioAvisaAntesDeUmbralAlto(t *testing.T) {
	runRef := "run-ref-autop-status-goal-no-checkpoint-warning-tokens-001"
	goalRef := "goal-ref-autop-status-goal-no-checkpoint-warning-tokens-001"
	externalGoalRef := "thread-ref-autop-status-goal-no-checkpoint-warning-tokens-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Avisar antes de alto consumo sin checkpoint.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-no-checkpoint-warning-tokens-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 60000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-no-checkpoint-warning-tokens-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:15:00Z",
					},
				},
				Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
					ProcessRef: "process-ref-no-checkpoint-warning-tokens-001",
					Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			EvidenceRefs:    []string{"evidence-ref-no-checkpoint-warning-tokens-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionNoCheckpointConsumptionWarningV0 ||
		action.Severity != "warning" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 60000 ||
		action.RecommendedAction != "observe_goal_backend_require_checkpoint" ||
		!strings.Contains(action.Reason, "goal_active_no_checkpoint_consumption_warning") ||
		len(action.ArtifactRefs) != 0 ||
		len(action.DomainReceiptRefs) != 0 ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceNoCheckpointConsumptionWarningV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0SinCheckpointWarningEstancadoPromueveReplanV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-no-checkpoint-warning-stale-001"
	goalRef := "goal-ref-autop-status-goal-no-checkpoint-warning-stale-001"
	externalGoalRef := "thread-ref-autop-status-goal-no-checkpoint-warning-stale-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Promocionar warning sin checkpoint estancado.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-no-checkpoint-warning-stale-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 60000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-no-checkpoint-warning-stale-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:00:00Z",
					},
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			EvidenceRefs:    []string{"evidence-ref-no-checkpoint-warning-stale-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
		GoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 100000,
			NoCheckpointWarningMaxWaitSeconds:   600,
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef:     runRef,
		OccurredAt: "2026-07-01T18:20:01Z",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionNoCheckpointHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 60000 ||
		action.RecommendedAction != "replan_narrow_context" ||
		action.LastOutputAt != "2026-07-01T18:00:00Z" ||
		!strings.Contains(action.Reason, "max warning wait") ||
		len(action.ArtifactRefs) != 0 ||
		len(action.DomainReceiptRefs) != 0 ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0SinCheckpointHighConsumptionEsBloqueante(t *testing.T) {
	runRef := "run-ref-autop-status-goal-no-checkpoint-high-tokens-001"
	goalRef := "goal-ref-autop-status-goal-no-checkpoint-high-tokens-001"
	externalGoalRef := "thread-ref-autop-status-goal-no-checkpoint-high-tokens-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Detectar alto consumo sin checkpoint ni artefactos.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-no-checkpoint-high-tokens-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 175000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-no-checkpoint-high-tokens-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:20:00Z",
					},
				},
				Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
					ProcessRef: "process-ref-no-checkpoint-high-tokens-001",
					Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			EvidenceRefs:    []string{"evidence-ref-no-checkpoint-high-tokens-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionNoCheckpointHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 175000 ||
		action.RecommendedAction != "replan_narrow_context" ||
		!strings.Contains(action.Reason, "goal_active_no_checkpoint_high_consumption") ||
		len(action.ArtifactRefs) != 0 ||
		len(action.DomainReceiptRefs) != 0 ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0BackendActivoConSoloTokensHistoricosSigueBloqueado(t *testing.T) {
	runRef := "run-ref-autop-status-goal-backend-active-historical-tokens-001"
	goalRef := "goal-ref-autop-status-goal-backend-active-historical-tokens-001"
	externalGoalRef := "thread-ref-autop-status-goal-backend-active-historical-tokens-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Tokens acumulados sin actividad no deben ocultar stale real.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-backend-historical-tokens-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 9876,
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code == mcpAutoprogrammingActionActiveNoCheckpointYetV0 ||
		action.Code != mcpAutoprogrammingActionGoalFirstBlockedV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 9876 ||
		action.LastOutputAt != "" ||
		action.LastArtifactAt != "" ||
		action.ProcessAliveCount != 0 ||
		action.RecommendedAction != "review_replan_goal_first" ||
		!strings.Contains(action.Reason, "goal_backend_state_unreconciled") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConBackendActivoEsReplanAccionable(t *testing.T) {
	runRef := "run-ref-autop-status-goal-active-timeout-backend-active-001"
	goalRef := "goal-ref-autop-status-goal-active-timeout-backend-active-001"
	externalGoalRef := "thread-ref-autop-status-goal-active-timeout-backend-active-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Goal local bloqueado por timeout pero backend sigue activo.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       goalRef,
			Summary:       "codex_app_server_goal_active_timeout",
			EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-timeout-backend-active-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 338171,
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{"artifact-ref-checkpoint:run-ref-autop-status-goal-active-timeout-backend-active-001:tema-001-checkpoint-started-txt"},
			EvidenceRefs:    []string{"evidence-ref-backend-active-timeout-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 338171 ||
		action.RecommendedAction != "replan_narrow_context" ||
		!strings.Contains(action.Reason, "checkpoint_only_high_consumption") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-checkpoint:run-ref-autop-status-goal-active-timeout-backend-active-001:tema-001-checkpoint-started-txt") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-backend-active-timeout-observed") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyConsumoMedioAvisaAntesDeUmbralAlto(t *testing.T) {
	runRef := "run-ref-autop-status-goal-checkpoint-warning-tokens-001"
	goalRef := "goal-ref-autop-status-goal-checkpoint-warning-tokens-001"
	externalGoalRef := "thread-ref-autop-status-goal-checkpoint-warning-tokens-001"
	checkpointRef := "artifact-ref-checkpoint:run-ref-autop-status-goal-checkpoint-warning-tokens-001:tema-001-checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Avisar si un goal solo tiene checkpoint y consumo creciente.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-checkpoint-warning-tokens-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "running",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 60000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-checkpoint-warning-tokens-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DeliveryRef: "delivery-ref-checkpoint-warning-tokens-001",
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:12:00Z",
					},
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{checkpointRef},
			EvidenceRefs:    []string{"evidence-ref-checkpoint-warning-tokens-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
		GoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 100000,
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionCheckpointOnlyConsumptionWarningV0 ||
		action.Severity != "warning" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 60000 ||
		action.RecommendedAction != "observe_goal_backend_require_next_artifact" ||
		!strings.Contains(action.Reason, "checkpoint_only_consumption_warning") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, checkpointRef) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyConsumptionWarningV0) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-checkpoint-warning-tokens-observed") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRecienteNoReplanificaAun(t *testing.T) {
	runRef := "run-ref-autop-status-goal-active-timeout-checkpoint-recent-001"
	goalRef := "goal-ref-autop-status-goal-active-timeout-checkpoint-recent-001"
	externalGoalRef := "thread-ref-autop-status-goal-active-timeout-checkpoint-recent-001"
	checkpointRef := "artifact-ref-checkpoint:run-ref-autop-status-goal-active-timeout-checkpoint-recent-001:tema-001-checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "No cortar un checkpoint reciente por timeout local.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       goalRef,
			Summary:       "codex_app_server_goal_active_timeout",
			ArtifactRefs:  []string{checkpointRef},
			EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
		EvidenceRefs: []string{"evidence-ref-goal-timeout-checkpoint-recent-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 23000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-timeout-checkpoint-recent-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:12:00Z",
					},
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{checkpointRef},
			EvidenceRefs:    []string{"evidence-ref-backend-active-timeout-checkpoint-recent-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionActiveTimeoutCheckpointRecentV0 ||
		action.Severity != "info" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 23000 ||
		action.RecommendedAction != "observe_goal_backend_wait_for_checkpoint" ||
		!strings.Contains(action.Reason, "active_timeout_checkpoint_recent") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, checkpointRef) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceActiveTimeoutCheckpointRecentV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointEstancadoPromueveReplanV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-active-timeout-checkpoint-stale-001"
	goalRef := "goal-ref-autop-status-goal-active-timeout-checkpoint-stale-001"
	externalGoalRef := "thread-ref-autop-status-goal-active-timeout-checkpoint-stale-001"
	checkpointRef := "artifact-ref-checkpoint:run-ref-autop-status-goal-active-timeout-checkpoint-stale-001:tema-001-checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Promocionar checkpoint estancado tras timeout local.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       goalRef,
			Summary:       "codex_app_server_goal_active_timeout",
			ArtifactRefs:  []string{checkpointRef},
			EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 23000,
			},
			Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
				AgentRequestID: "agent-ref-timeout-checkpoint-stale-001",
				Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
				InFlight:       true,
				LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
					DeliveryRef: "delivery-ref-timeout-checkpoint-stale-001",
					DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
						LastActivityAt: "2026-07-01T18:00:00Z",
					},
				},
			}},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{checkpointRef},
			EvidenceRefs:    []string{"evidence-ref-backend-active-timeout-checkpoint-stale-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
		GoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 100000,
			CheckpointOnlyMaxWaitSeconds:        600,
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef:     runRef,
		OccurredAt: "2026-07-01T18:20:01Z",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 23000 ||
		action.RecommendedAction != "replan_narrow_context" ||
		action.LastArtifactAt != "2026-07-01T18:00:00Z" ||
		!strings.Contains(action.Reason, "max checkpoint wait") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, checkpointRef) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRespetaUmbralConfiguradoV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-active-timeout-checkpoint-policy-001"
	goalRef := "goal-ref-autop-status-goal-active-timeout-checkpoint-policy-001"
	externalGoalRef := "thread-ref-autop-status-goal-active-timeout-checkpoint-policy-001"
	checkpointRef := "artifact-ref-checkpoint:run-ref-autop-status-goal-active-timeout-checkpoint-policy-001:tema-001-checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Aplicar politica parametrizable de alto consumo con checkpoint.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       goalRef,
			Summary:       "codex_app_server_goal_active_timeout",
			ArtifactRefs:  []string{checkpointRef},
			EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
		},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 23000,
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          "active",
			ArtifactRefs:    []string{checkpointRef},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
		GoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 20000,
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0 ||
		action.Severity != "blocked" ||
		action.GoalStatus != "active" ||
		action.TokensUsed != 23000 ||
		action.RecommendedAction != "replan_narrow_context" ||
		!strings.Contains(action.Reason, "checkpoint_only_high_consumption") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, checkpointRef) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalRunningSinBackendActivoPideReconciliarCleanupExternoV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-running-backend-missing-001"
	goalRef := "goal-ref-autop-status-goal-running-backend-missing-001"
	externalGoalRef := "thread-ref-autop-status-goal-running-backend-missing-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reconciliar estado running tras cleanup externo del backend.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-running-before-external-cleanup-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			runRef: {
				RunRef: runRef,
				Status: "stopped",
				UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
					TotalTokens: 151000,
				},
			},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue:          &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionGoalBackendMissingAfterExternalCleanupV0 ||
		action.Severity != "blocked" ||
		action.GoalRef != goalRef ||
		action.ExternalGoalRef != externalGoalRef ||
		action.TokensUsed != 151000 ||
		action.RecommendedAction != "run_control_reconcile_external_cleanup" ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) ||
		!strings.Contains(action.Reason, "goal_backend_missing_after_external_cleanup") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalBloqueadoConBackendCompleteSaleDeStaleRunning(t *testing.T) {
	runRef := "run-ref-autop-status-goal-backend-complete-001"
	goalRef := "goal-ref-autop-status-goal-backend-complete-001"
	externalGoalRef := "thread-ref-autop-status-goal-backend-complete-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reconciliar goal Codex complete aunque la cola local quedara stopped.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status:            orquestagoal.GoalStatusCompleteV0,
			GoalRef:           goalRef,
			ExternalGoalRef:   externalGoalRef,
			ArtifactRefs:      []string{"artifact-ref-backend-complete-001"},
			DomainReceiptRefs: []string{"receipt-ref-backend-complete-001"},
			EvidenceRefs:      []string{"result-ref-backend-complete-001"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-backend-complete-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
			UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
				TotalTokens: 9876,
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:            runRef,
			GoalRef:           goalRef,
			ExternalGoalRef:   externalGoalRef,
			Status:            orquestagoal.GoalStatusCompleteV0,
			ArtifactRefs:      []string{"artifact-ref-backend-complete-001"},
			DomainReceiptRefs: []string{"receipt-ref-backend-complete-001"},
			EvidenceRefs:      []string{"evidence-ref-backend-complete-observed"},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(stats.inputs) != 1 || stats.inputs[0].RunRef != runRef {
		t.Fatalf("stats inputs=%+v", stats.inputs)
	}
	if len(result.StaleRunning) != 0 {
		t.Fatalf("stale_running no debe mantener blocked con backend complete: %+v", result.StaleRunning)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Blocked != 0 ||
		result.QueueHealth.Completed != 1 ||
		result.QueueHealth.ObservedRuns != 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if len(result.ResolvedRuns) != 1 {
		t.Fatalf("resolved_runs=%+v", result.ResolvedRuns)
	}
	action := result.ResolvedRuns[0]
	if action.Code != "goal_first_terminal_reconciled" ||
		action.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		action.RunStatus != "stopped" ||
		action.RecommendedAction != "reconcile_goal_terminal" ||
		!strings.Contains(action.Reason, "goal_backend_terminal_reconciled") ||
		action.TokensUsed != 9876 {
		t.Fatalf("action=%+v", action)
	}
	if !hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-backend-complete-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.DomainReceiptRefs, "receipt-ref-backend-complete-001") {
		t.Fatalf("action refs=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ResultadoDurableTerminalConBackendActivoNoQuedaOpaco(t *testing.T) {
	runRef := "run-ref-autop-status-goal-terminal-pending-001"
	goalRef := "goal-ref-autop-status-goal-terminal-pending-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Resultado durable terminal no debe quedar como backend unreconciled opaco.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			Status:       orquestagoal.GoalStatusCompleteV0,
			GoalRef:      goalRef,
			ArtifactRefs: []string{"artifact-ref-terminal-pending-001"},
			EvidenceRefs: []string{"result-ref-terminal-pending-001"},
		},
		EvidenceRefs: []string{"evidence-ref-terminal-pending-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "stopped",
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:  runRef,
			GoalRef: goalRef,
			Status:  "active",
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef: runRef,
				AppRef: "opes",
				Status: "stopped",
			}},
		},
		Stats:          stats,
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.ResolvedRuns) != 0 ||
		len(result.StaleRunning) != 1 {
		t.Fatalf("resolved=%+v stale=%+v", result.ResolvedRuns, result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.GoalStatus != "active" ||
		action.RecommendedAction != "reconcile_goal_terminal" ||
		!strings.Contains(action.Reason, "goal_terminal_reconcile_pending") ||
		strings.Contains(action.Reason, "goal_backend_state_unreconciled") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoDaCienConColaVaciaYGoalBloqueado(t *testing.T) {
	runRef := "run-ref-autop-status-goal-blocked-empty-queue-001"
	goalRef := "goal-ref-autop-status-goal-blocked-empty-queue-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cierre OPES bloqueado no debe parecer eficiencia 100 con cola vacia.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "external/opes"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusBlockedV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-empty-queue-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue:          &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Blocked != 1 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != mcpAutoprogrammingActionGoalFirstBlockedV0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "attention_required" ||
		result.EfficiencySummary.OperationalHealthPercentage >= 100 ||
		result.EfficiencySummary.OverallPercentage >= 100 ||
		!hasStringMCPAutoprogrammingStatusTestV0(result.EfficiencySummary.Reasons, "goal_first_blocked") {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalFirstBloqueadoProyectaMetadataOPESAudioV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-blocked-opes-audio-001"
	goalRef := "goal-ref-autop-status-goal-blocked-opes-audio-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			RunRef:        runRef,
			GoalRef:       goalRef,
			DomainRef:     "opes",
			WorkKind:      "generate_audio_asset",
			Objective:     "Continuar audio OPES bloqueado con metadata durable.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{
					Kind:    "input_field_value",
					Ref:     "input-field-current-phase-value-test",
					Purpose: `Campo input_fields.current_phase inlineado de forma acotada: {"name":"current_phase","value":"prepare"}`,
				},
				{
					Kind:    "input_field_value",
					Ref:     "input-field-retry-from-phase-value-test",
					Purpose: `Campo input_fields.retry_from_phase inlineado de forma acotada: {"name":"retry_from_phase","value":"prepare"}`,
				},
				{
					Kind:    "input_field_value",
					Ref:     "input-field-audio-counters-value-test",
					Purpose: `Campo input_fields.audio_counters inlineado de forma acotada: {"name":"audio_counters","value_json":{"audio_manifest_refs":1,"text_hash_refs":2}}`,
				},
			},
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/audio"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusBlockedV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-opes-audio-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionGoalFirstBlockedV0 ||
		action.RecommendedAction != "retry_from_phase" ||
		action.CurrentPhase != "prepare" ||
		action.DomainCounters["audio_manifest_refs"] != 1 ||
		action.DomainCounters["text_hash_refs"] != 2 {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalFirstBloqueadoCierraSupersededPorEvidenciaLocalV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-superseded-001"
	goalRef := "goal-ref-autop-status-goal-superseded-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			RunRef:        runRef,
			GoalRef:       goalRef,
			DomainRef:     "opes",
			WorkKind:      "generate_audio_asset",
			Objective:     "Cerrar bloqueo reemplazado por evidencia local durable.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			EvidenceRefs: []string{
				"superseded_by_local_evidence=true",
				"evidence-ref-opes-local-audio-manifest-current",
			},
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/audio"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusBlockedV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion:     orquestagoal.GoalWorkResultSchemaV0,
			Status:            orquestagoal.GoalStatusBlockedV0,
			GoalRef:           goalRef,
			DomainReceiptRefs: []string{"domain-receipt-ref-opes-local-evidence-current"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-superseded-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].RecommendedAction != "close_superseded_by_local_evidence" {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalFirstCierraSupersededDesdeContextRefsV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-superseded-context-001"
	goalRef := "goal-ref-autop-status-goal-superseded-context-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			RunRef:        runRef,
			GoalRef:       goalRef,
			DomainRef:     "opes",
			WorkKind:      "generate_audio_asset",
			Objective:     "Cerrar bloqueo reemplazado por evidencia local durable.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{
					Kind:    "input_field_value",
					Ref:     "input-field-superseded-by-local-evidence-value-test",
					Purpose: `Campo input_fields.superseded_by_local_evidence inlineado de forma acotada: {"name":"superseded_by_local_evidence","value":"true"}`,
				},
				{
					Kind:    "input_field_value",
					Ref:     "input-field-local-evidence-ref-value-test",
					Purpose: `Campo input_fields.local_evidence_ref inlineado de forma acotada: {"name":"local_evidence_ref","value":"evidence-ref-opes-local-audio-manifest-current"}`,
				},
			},
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/audio"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusBlockedV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-superseded-context-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].RecommendedAction != "close_superseded_by_local_evidence" {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalFirstSupersededSinEvidenciaLocalNoCierraV0(t *testing.T) {
	runRef := "run-ref-autop-status-goal-superseded-sin-evidencia-001"
	goalRef := "goal-ref-autop-status-goal-superseded-sin-evidencia-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusBlockedV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			RunRef:        runRef,
			GoalRef:       goalRef,
			DomainRef:     "opes",
			WorkKind:      "generate_audio_asset",
			Objective:     "No cerrar sin evidencia local durable.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			EvidenceRefs:  []string{"superseded_by_local_evidence=true"},
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/audio"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusBlockedV0,
		},
		EvidenceRefs: []string{"evidence-ref-goal-blocked-superseded-sin-local-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].RecommendedAction == "close_superseded_by_local_evidence" {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0(t *testing.T) {
	runRef := "run-ref-autop-status-qa-failed-public-text-001"
	goalRef := "goal-ref-autop-status-qa-failed-public-text-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstQAFailedPublicTextV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Status:    orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstQAFailedPublicTextV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstQAFailedPublicTextV0,
					Field: "goal_first.qa_public_text",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-qa-failed-public-text-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-qa-failed-public-text-001"},
			IssueCodes:   []string{MCPGoalFirstQAFailedPublicTextV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstQAFailedPublicTextV0 ||
		action.Severity != "blocked" ||
		action.RecommendedAction != MCPGoalFirstReworkPublicTextActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-qa-failed-public-text-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-qa-failed-public-text") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-goal-materialized-qa-failed-public-text-001") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ArtifactPathsOmitidosPideRepairReceiptV0(t *testing.T) {
	runRef := "run-ref-autop-status-artifact-paths-omitted-001"
	goalRef := "goal-ref-autop-status-artifact-paths-omitted-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstArtifactPathsOmittedMaterializedV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Status:    orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstArtifactPathsOmittedMaterializedV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstArtifactPathsOmittedMaterializedV0,
					Field: "goal_first.artifact_paths",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-omitted-path-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-artifact-paths-omitted"},
			IssueCodes:   []string{MCPGoalFirstArtifactPathsOmittedMaterializedV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstArtifactPathsOmittedMaterializedV0 ||
		action.RecommendedAction != MCPGoalFirstRepairReceiptActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-omitted-path-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-artifact-paths-omitted-materialized") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0OutOfScopeMaterializedPideReworkV0(t *testing.T) {
	runRef := "run-ref-autop-status-out-of-scope-001"
	goalRef := "goal-ref-autop-status-out-of-scope-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstOutOfScopeMaterializedArtifactsV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
					Field: "goal_first.write_set",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-out-of-scope-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-out-of-scope-artifacts"},
			IssueCodes:   []string{MCPGoalFirstOutOfScopeMaterializedArtifactsV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstOutOfScopeMaterializedArtifactsV0 ||
		action.RecommendedAction != MCPGoalFirstReworkWriteSetViolationActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-out-of-scope-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-out-of-scope-materialized-artifacts") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ArtefactosParcialesPideRevisionV0(t *testing.T) {
	runRef := "run-ref-autop-status-partial-artifacts-001"
	goalRef := "goal-ref-autop-status-partial-artifacts-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstPartialArtifactsWrittenV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Status:    orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstPartialArtifactsWrittenV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstPartialArtifactsWrittenV0,
					Field: "goal_first.partial_artifacts",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-partial-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-partial-artifacts-written"},
			IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstPartialArtifactsWrittenV0 ||
		action.RecommendedAction != MCPGoalFirstReviewPartialArtifactsActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-partial-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-partial-artifacts-written") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0(t *testing.T) {
	runRef := "run-ref-autop-status-thread-output-sanitized-001"
	goalRef := "goal-ref-autop-status-thread-output-sanitized-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: "running",
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:  runRef,
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0,
			},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != mcpAutoprogrammingActionThreadOutputSanitizedV0 ||
		action.Severity != "warning" ||
		action.RecommendedAction != "replan_narrow_context" ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceThreadOutputSanitizedV0) ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0) {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0Phase0NoPublicablePideContinuarV0(t *testing.T) {
	runRef := "run-ref-autop-status-phase0-001"
	goalRef := "goal-ref-autop-status-phase0-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstPhase0CompleteNonPublishableV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Status:    orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstPhase0CompleteNonPublishableV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstPhase0CompleteNonPublishableV0,
					Field: "goal_first.phase0",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-phase0-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-phase0-complete-non-publishable"},
			IssueCodes:   []string{MCPGoalFirstPhase0CompleteNonPublishableV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstPhase0CompleteNonPublishableV0 ||
		action.RecommendedAction != MCPGoalFirstContinueFromPhase0ActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-phase0-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-phase0-complete-non-publishable") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-goal-materialized-phase0-complete-non-publishable") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0RequiredTestEvidenceAusentePideRepairReceiptV0(t *testing.T) {
	runRef := "run-ref-autop-status-required-test-evidence-001"
	goalRef := "goal-ref-autop-status-required-test-evidence-001"
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			RunRef: runRef,
			Status: MCPGoalFirstRequiredTestEvidenceMissingV0,
			Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
				Status:    orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
				Blocked:   true,
				BlockedBy: []string{MCPGoalFirstRequiredTestEvidenceMissingV0},
			},
			Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
				Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
					Code:  MCPGoalFirstRequiredTestEvidenceMissingV0,
					Field: "goal_first.required_tests",
				}},
			},
		},
		goal: &MCPDirectorGoalStatsV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusBlockedV0,
			ArtifactRefs: []string{"artifact-ref-materialized-required-test-evidence-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-required-test-evidence-missing"},
			IssueCodes:   []string{MCPGoalFirstRequiredTestEvidenceMissingV0},
		},
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 {
		t.Fatalf("stale_running=%+v", result.StaleRunning)
	}
	action := result.StaleRunning[0]
	if action.Code != MCPGoalFirstRequiredTestEvidenceMissingV0 ||
		action.RecommendedAction != MCPGoalFirstRepairReceiptActionV0 ||
		action.GoalRef != goalRef ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.ArtifactRefs, "artifact-ref-materialized-required-test-evidence-001") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-autoprogramming-status-required-test-evidence-missing") ||
		!hasStringMCPAutoprogrammingStatusTestV0(action.EvidenceRefs, "evidence-ref-goal-materialized-required-test-evidence-missing") {
		t.Fatalf("action=%+v", action)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ListaGoalMarkerSinStateAunqueColaNoVisible(t *testing.T) {
	runRef := "run-ref-autop-status-goal-marker-listed-001"
	goalRef := "goal-ref-autop-status-goal-marker-listed-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	goalMarkers := newMCPGoalRunMarkerStoreForStatusTestV0()
	if err := goalMarkers.SaveGoalWorkRunMarkerV0(context.Background(), orquestagoal.GoalWorkRunMarkerV0{
		RunRef:       runRef,
		GoalRef:      goalRef,
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:       orquestagoal.GoalStatusRunningV0,
		EvidenceRefs: []string{"evidence-ref-goal-marker-listed-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore:               goalStates,
		GoalRunMarkerStore:           goalMarkers,
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		len(result.Errores) != 0 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_state_missing") ||
		len(result.Tasks) != 0 ||
		len(result.Agents) != 0 ||
		result.Operator == nil ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("result=%+v diagnostics=%+v operator=%+v", result, result.Diagnostics, result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0PublicaAccionBatchParaGoalsActivos(t *testing.T) {
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	for _, pair := range []struct {
		runRef  string
		goalRef string
	}{
		{"run-ref-autop-status-goal-batch-001", "goal-ref-autop-status-goal-batch-001"},
		{"run-ref-autop-status-goal-batch-002", "goal-ref-autop-status-goal-batch-002"},
	} {
		if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
			RunRef:  pair.runRef,
			GoalRef: pair.goalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
			Spec: orquestagoal.GoalWorkSpecV0{
				RunRef:       pair.runRef,
				GoalRef:      pair.goalRef,
				Objective:    "Observar goals activos en lote.",
				DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
			},
			LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
				GoalRef: pair.goalRef,
				Status:  orquestagoal.GoalStatusRunningV0,
			},
		}); err != nil {
			t.Fatalf("SaveGoalWorkStateV0: %v", err)
		}
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_active_goals", "goals", "") ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", "run-ref-autop-status-goal-batch-001") ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", "run-ref-autop-status-goal-batch-002") {
		t.Fatalf("operator=%+v", result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ColaMixtaGoalFirstNoSupervisaColaGlobal(t *testing.T) {
	goalRunRef := "run-ref-autop-status-goal-first-queue-001"
	legacyRunRef := "run-ref-autop-status-legacy-queue-001"
	goalRef := "goal-ref-autop-status-goal-first-queue-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  goalRunRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       goalRunRef,
			GoalRef:      goalRef,
			Objective:    "Observar goal sin supervisor legacy de cola.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/goal"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:      goalRef,
			Status:       orquestagoal.GoalStatusRunningV0,
			EvidenceRefs: []string{"evidence-ref-goal-first-queue-status-test"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-queue-status-test"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{
				{
					Rank:          1,
					RunRef:        goalRunRef,
					AppRef:        "app-ref-goal-first",
					Status:        "running",
					PriorityScore: 90,
				},
				{
					Rank:          2,
					RunRef:        legacyRunRef,
					AppRef:        "app-ref-legacy",
					Status:        "ready",
					PriorityScore: 80,
				},
			},
		},
		Stats:                        &fakeMCPAutoprogrammingRunStatusV0{},
		GoalStateStore:               goalStates,
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_goal_first_observe_required") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Queued != 1 ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		result.QueueHealth.RunningStale != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", goalRunRef) ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", legacyRunRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", goalRunRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if !hasMCPAutoprogrammingSafeActionPayloadForTestV0(
		result.Operator.SafeActions,
		"supervise",
		"run",
		legacyRunRef,
		"director_execution_mode",
		"legacy_director_loop",
	) {
		t.Fatalf("legacy supervise payload sin modo explicito: %+v", result.Operator.SafeActions)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != "observe_goal" ||
		result.OpsSnapshot.Decision.RunRef != goalRunRef ||
		result.OpsSnapshot.Decision.ReasonCode != "goal_first_observe_required" {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ExponeQueueHealthSeparada(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{
				{Rank: 1, RunRef: "run-ref-health-ready", AppRef: "app", Status: "ready"},
				{Rank: 2, RunRef: "run-ref-health-live", AppRef: "app", Status: "running"},
				{Rank: 3, RunRef: "run-ref-health-blocked", AppRef: "app", Status: "stopped"},
				{Rank: 4, RunRef: "run-ref-health-completed", AppRef: "app", Status: "closed"},
				{Rank: 5, RunRef: "run-ref-health-failed", AppRef: "app", Status: "canceled"},
			},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef: "run-ref-health-live",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					ProgressingAgents: 1,
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-health-live",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Queued != 1 ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.Blocked != 1 ||
		result.QueueHealth.Completed != 1 ||
		result.QueueHealth.Failed != 1 ||
		result.QueueHealth.Lost != 0 ||
		result.QueueHealth.ObservedRuns != 5 ||
		result.QueueHealth.QueueRuns != 5 ||
		result.QueueHealth.StatsRuns != 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if len(result.StaleRunning) != 0 {
		t.Fatalf("run con stats live no debe aparecer como stale_running: %+v", result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoMarcaStaleSinStatsV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-stale-running-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
				EvidenceRefs:  []string{"evidence-ref-test-stale-running"},
			}},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_without_recent_stats" ||
		result.StaleRunning[0].Severity != "info" ||
		result.StaleRunning[0].RecommendedAction != "observe_run_ref_with_process_refs_before_reconcile" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "running_without_recent_stats") ||
		result.QueueHealth == nil ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 1 {
		t.Fatalf("stale_running=%+v diagnostics=%+v queue_health=%+v", result.StaleRunning, result.Diagnostics, result.QueueHealth)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaMuestraRunningLimitadaV0(t *testing.T) {
	ranked := make([]MCPRunQueueRankedCandidateCompactV0, 0, mcpAutoprogrammingRunningStatsMaxV0+2)
	for index := 0; index < mcpAutoprogrammingRunningStatsMaxV0+2; index++ {
		ranked = append(ranked, MCPRunQueueRankedCandidateCompactV0{
			Rank:          index + 1,
			RunRef:        "run-ref-running-sample-" + string(rune('a'+index)),
			AppRef:        "app-ref-running-sample",
			Status:        "running",
			PriorityScore: 100 - index,
		})
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{ranked: ranked},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(stats.inputs) != mcpAutoprogrammingRunningStatsMaxV0 {
		t.Fatalf("stats inputs=%d want=%d inputs=%+v", len(stats.inputs), mcpAutoprogrammingRunningStatsMaxV0, stats.inputs)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "running_stats_sample_limited") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.QueueRuns != len(ranked) ||
		result.QueueHealth.ObservedRuns == 0 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ObservaRunningConProcesoVivoV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-live-process-001": {
				RunRef: "run-ref-running-live-process-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 1,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-live-process-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-running-live-process-001",
						SessionRef: "session-ref-running-live-process-001",
						Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
					},
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-live-process-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(stats.inputs) != 1 ||
		stats.inputs[0].RunRef != "run-ref-running-live-process-001" ||
		!stats.inputs[0].IncludeProcessRefs ||
		!stats.inputs[0].IncludeAgentProgress {
		t.Fatalf("stats inputs=%+v", stats.inputs)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoMarcaStaleConDirectorStatsSnapshotRunningV0(t *testing.T) {
	runRef := "run-ref-autop-status-snapshot-running-001"
	processRef := "process-ref-autop-status-snapshot-running-001"
	run := mcpDirectorStatsRunForTestV0(t, runRef)
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     processRef,
		SessionRef:     "session-ref-autop-status-snapshot-running-001",
		LaunchRef:      "launch-ref-autop-status-snapshot-running-001",
		ReadinessRef:   "readiness-ref-autop-status-snapshot-running-001",
		EvidenceRefs:   []string{"evidence-ref-autop-status-snapshot-running-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        runRef,
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: MCPDirectorStatsToolExecutorV0{
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
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
	if hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "running_stale_no_process") ||
		hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "running_without_recent_stats") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ProcesoSinStatusQuedaSinStatsRecientesV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-process-unknown-001": {
				RunRef: "run-ref-running-process-unknown-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 1,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-process-unknown-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-running-process-unknown-001",
					},
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-process-unknown-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.AgentsLive != 0 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 1 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_without_recent_stats" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0AgenteSinProcesoNoDeclaraNoProcessV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-agent-without-process-001": {
				RunRef: "run-ref-running-agent-without-process-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents:     1,
					ProgressingAgents: 0,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-agent-without-process-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-agent-without-process-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 1 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_without_recent_stats" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0RegistrySinProcesoEsRunningStaleNoProcessV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-registry-missing-process-001": {
				RunRef: "run-ref-running-registry-missing-process-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight:       1,
					AgentsControlMissing: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents:     1,
					ProgressingAgents: 0,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-registry-missing-process-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					NeedsAttention: true,
					ControlState:   orquestacionnucleoapp.DirectorAgentControlStateMissingV0,
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-registry-missing-process-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.AgentsLive != 0 ||
		result.QueueHealth.RunningStale != 1 ||
		result.QueueHealth.RunningStaleNoProcess != 1 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_stale_no_process" ||
		result.StaleRunning[0].Severity != "warning" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ProcesoParadoEsRunningStaleNoProcessV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-process-stopped-001": {
				RunRef: "run-ref-running-process-stopped-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 1,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-process-stopped-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-running-process-stopped-001",
						Status:     orquestacionnucleoapp.DirectorAgentProcessStatusStoppedV0,
					},
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-process-stopped-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 0 ||
		result.QueueHealth.AgentsLive != 0 ||
		result.QueueHealth.RunningStale != 1 ||
		result.QueueHealth.RunningStaleNoProcess != 1 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_stale_no_process" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ProcesoParadoConAckCleanupEsperaACKV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-process-stopped-ack-cleanup-001": {
				RunRef: "run-ref-running-process-stopped-ack-cleanup-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 1,
					Tasks: []orquestacionnucleoapp.DirectorTaskProgressV0{{
						TaskRef:        "task-ref-running-process-stopped-ack-cleanup-001",
						AgentRequestID: "agent-ref-running-process-stopped-ack-cleanup-001",
						DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
							Classification: orquestacionnucleoapp.DirectorProgressClassificationAckCleanupV0,
							LastActivityAt: "2026-06-25T20:11:00Z",
							LastAckAt:      "2026-06-25T20:10:00Z",
						},
					}},
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-running-process-stopped-ack-cleanup-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
						TaskRef:     "task-ref-running-process-stopped-ack-cleanup-001",
						DeliveryRef: "delivery-ref-running-process-stopped-ack-cleanup-001",
						DirectorProgressTemporalV0: orquestacionnucleoapp.DirectorProgressTemporalV0{
							Classification: orquestacionnucleoapp.DirectorProgressClassificationAckCleanupV0,
							LastActivityAt: "2026-06-25T20:12:00Z",
							LastAckAt:      "2026-06-25T20:10:00Z",
						},
					},
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-running-process-stopped-ack-cleanup-001",
						Status:     orquestacionnucleoapp.DirectorAgentProcessStatusStoppedV0,
					},
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-process-stopped-ack-cleanup-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningStale != 1 ||
		result.QueueHealth.RunningStaleNoProcess != 1 ||
		len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "running_stale_no_process" ||
		result.StaleRunning[0].Reason != "running_stale_no_live_process_pending_ack" ||
		result.StaleRunning[0].RecommendedAction != "wait_for_ack_before_reconcile" ||
		result.StaleRunning[0].ProcessAliveCount != 0 ||
		!result.StaleRunning[0].AckDetected ||
		result.StaleRunning[0].LastAckAt != "2026-06-25T20:10:00Z" ||
		result.StaleRunning[0].LastOutputAt != "2026-06-25T20:12:00Z" ||
		result.StaleRunning[0].LastArtifactAt != "2026-06-25T20:12:00Z" {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0EnriqueceRunExplicitoParaQueueHealthV0(t *testing.T) {
	runRef := "run-ref-running-live-explicit-001"
	publicStats := &orquestacionnucleoapp.DirectorRunStatsV0{
		RunRef: runRef,
		Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
			AgentsInFlight: 1,
		},
		Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
			StalledAgents: 1,
		},
	}
	liveStats := &orquestacionnucleoapp.DirectorRunStatsV0{
		RunRef: runRef,
		Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
			AgentsInFlight: 1,
		},
		Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
			StalledAgents: 1,
		},
		Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
			AgentRequestID: "agent-ref-running-live-explicit-001",
			Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
			InFlight:       true,
			Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
				ProcessRef: "process-ref-running-live-explicit-001",
				Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
			},
		}},
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsForInput: func(input MCPDirectorStatsToolInputV0) *orquestacionnucleoapp.DirectorRunStatsV0 {
			if input.IncludeProcessRefs && input.IncludeAgentProgress {
				return liveStats
			}
			return publicStats
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        runRef,
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: runRef,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(stats.inputs) != 2 ||
		stats.inputs[0].IncludeProcessRefs ||
		stats.inputs[0].IncludeAgentProgress ||
		!stats.inputs[1].IncludeProcessRefs ||
		!stats.inputs[1].IncludeAgentProgress {
		t.Fatalf("stats inputs=%+v", stats.inputs)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoMarcaStaleConProcesoVivoAunqueFalteContadorV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-live-process-no-count-001": {
				RunRef: "run-ref-running-live-process-no-count-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 0,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 1,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID:    "agent-ref-running-live-process-no-count-001",
					Status:            orquestacionnucleoapp.DirectorAgentStatusRequestedV0,
					ControlRegistered: true,
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-running-live-process-no-count-001",
						SessionRef: "session-ref-running-live-process-no-count-001",
						Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
					},
				}},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-live-process-no-count-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(stats.inputs) != 1 ||
		stats.inputs[0].RunRef != "run-ref-running-live-process-no-count-001" ||
		!stats.inputs[0].IncludeProcessRefs ||
		!stats.inputs[0].IncludeAgentProgress {
		t.Fatalf("stats inputs=%+v", stats.inputs)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0CuentaAgentesVivosPorProcesoAunqueNoProgresenV0(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			"run-ref-running-live-process-count-001": {
				RunRef: "run-ref-running-live-process-count-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsInFlight: 2,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					StalledAgents: 2,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{
					{
						AgentRequestID: "agent-ref-running-live-process-count-001",
						Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
						InFlight:       true,
						Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
							ProcessRef: "process-ref-running-live-process-count-001",
							Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
						},
					},
					{
						AgentRequestID: "agent-ref-running-live-process-count-002",
						Status:         orquestacionnucleoapp.DirectorAgentStatusRequestedV0,
						Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
							SessionRef: "session-ref-running-live-process-count-002",
							Status:     orquestacionnucleoapp.DirectorAgentProcessStatusStoppingV0,
						},
					},
					{
						AgentRequestID: "agent-ref-running-live-process-count-003",
						Status:         orquestacionnucleoapp.DirectorAgentStatusFailedV0,
						Failed:         true,
						Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
							ProcessRef: "process-ref-running-live-process-count-003",
						},
					},
				},
			},
		},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-running-live-process-count-001",
				AppRef:        "opes",
				Status:        "running",
				PriorityScore: 90,
			}},
		},
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 2 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v", result.QueueHealth, result.StaleRunning)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0ExponeUsageLimitReconciliadoV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef:       "run-ref-usage-limit-001",
				AppRef:       "opes",
				Status:       "stopped",
				RescueReason: "provider_usage_limit_retry_after",
				EvidenceRefs: []string{
					"evidence-ref-run-queue-running-stale-no-live-process-reconciled",
					"evidence-ref-provider-usage-limit-retry-after",
					"evidence-ref-codex-usage-quota-exhausted",
				},
			}},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != "provider_usage_limit_retry_after" ||
		result.StaleRunning[0].Severity != "blocked" ||
		result.StaleRunning[0].Status != "stopped" ||
		result.StaleRunning[0].RecommendedAction != "wait_for_quota_and_relaunch_idempotently" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "provider_usage_limit_retry_after") {
		t.Fatalf("stale_running=%+v diagnostics=%+v", result.StaleRunning, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0PropagaDiagnosticoAgenteSolicitadoNoArrancadoV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:     "run-ref-autop-agent-not-started-001",
				ProjectRef: "opes",
				AppSpecRef: "app-spec-external-work-opes-qa-visual",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:      1,
					TasksOpen:       1,
					AgentsRequested: 1,
					AgentsStarted:   0,
					AgentsInFlight:  0,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					Issues: []orquestacionnucleoapp.DirectorProgressIssueV0{{
						Code:    mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0,
						Field:   "agents",
						Message: "cause=unknown action=retry_materialization_or_check_capacity_auth_runtime_queue_outbox_policy",
					}},
				},
			},
		},
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-agent-not-started-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(
		result.Diagnostics,
		mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0,
	) {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.Operator == nil ||
		len(result.Operator.SafeActions) == 0 ||
		result.Operator.SafeActions[0].Action != "supervise" {
		t.Fatalf("operator=%+v", result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaExternalWorkStoppedSinEntregaV0(t *testing.T) {
	runRef := "run-opes-psicologo-rework-visual-019-030-20260626"
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef:        runRef,
				AppRef:        "opes",
				Status:        "stopped",
				PriorityScore: 1,
				EvidenceRefs: []string{
					mcpAutoprogrammingEvidenceExternalWorkRunStartedV0,
					mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0,
					mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0,
				},
			}},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:     runRef,
				ProjectRef: "opes",
				AppSpecRef: "app-spec-opes-rework-visual",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:      1,
					TasksOpen:       1,
					AgentsRequested: 0,
					AgentsStarted:   0,
					AgentsInFlight:  0,
					Deliveries:      0,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: runRef,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0 ||
		result.StaleRunning[0].Severity != "blocked" ||
		result.StaleRunning[0].RecommendedAction != "relaunch_or_replan_external_work_with_causal_error" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0) {
		t.Fatalf("stale_running=%+v diagnostics=%+v", result.StaleRunning, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaExternalWorkDoneSinAgenteMaterializadoV0(t *testing.T) {
	runRef := "run-opes-tractorista-tema-001-done-sin-agentes"
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			empty: true,
			terminal: []MCPRunQueueRankedCandidateCompactV0{{
				RunRef:        runRef,
				AppRef:        "opes",
				Status:        "done",
				PriorityScore: 1,
				EvidenceRefs: []string{
					mcpAutoprogrammingEvidenceExternalWorkRunStartedV0,
					mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0,
					mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0,
				},
			}},
		},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:     runRef,
				Status:     "done",
				ProjectRef: "opes",
				AppSpecRef: "app-spec-opes-tractorista",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:      1,
					TasksClosed:     1,
					AgentsRequested: 0,
					AgentsStarted:   0,
					AgentsInFlight:  0,
					Deliveries:      0,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: runRef,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.StaleRunning) != 1 ||
		result.StaleRunning[0].Code != mcpAutoprogrammingActionExternalWorkNoAgentMaterializedV0 ||
		result.StaleRunning[0].Severity != "blocked" ||
		result.StaleRunning[0].RecommendedAction != "relaunch_or_replan_external_work_with_causal_error" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, mcpAutoprogrammingActionExternalWorkNoAgentMaterializedV0) {
		t.Fatalf("stale_running=%+v diagnostics=%+v", result.StaleRunning, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaAutoprogrammingSuprimidaPorSesionDominio(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{
		empty: true,
		terminal: []MCPRunQueueRankedCandidateCompactV0{{
			RunRef:       "request-ref-autoprogramming-backlog-scanner-15eeecb9",
			AppRef:       "app-ref-autoprogramming",
			Status:       "stopped",
			RescueReason: mcpAutoprogrammingDomainSessionSuppressedReasonV0,
			EvidenceRefs: []string{mcpAutoprogrammingDomainSessionSuppressedEvidenceV0},
		}},
	}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: queue,
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !queue.input.IncludeNonExecutable {
		t.Fatalf("autoprogramming status debe pedir terminales para diagnostico")
	}
	if result.Queue == nil ||
		len(result.Queue.Ranked) != 0 ||
		len(result.Queue.Terminal) != 1 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, mcpAutoprogrammingDomainSessionSuppressedReasonV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0OverallNoOcultaProgresoBajo(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:       "run-ref-autop-status-low-progress-001",
				Status:       "activa",
				CurrentPhase: "programacion",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:      10,
					TasksOpen:       10,
					AgentsRequested: 5,
					AgentsStarted:   5,
					AgentsInFlight:  5,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					PercentComplete:   1,
					TasksTotal:        10,
					TasksClosed:       0,
					ProgressingAgents: 5,
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status:  orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
					Blocked: true,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-status-low-progress-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "live" ||
		result.EfficiencySummary.CompletionPercentage != 1 ||
		result.EfficiencySummary.OperationalHealthPercentage != 100 ||
		result.EfficiencySummary.OverallPercentage != 31 {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaPuertosNoConfigurados(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{}).Execute(
		context.Background(),
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-missing-001"},
	)

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Diagnostics) < 2 ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DeclaraSuperviseColaLegacyConOptInV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue:                        &fakeMCPAutoprogrammingQueueStatusV0{},
		AllowLegacySupervisorActions: true,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action != "supervise_queue" ||
		result.OpsSnapshot.Queue.Count != 1 {
		t.Fatalf("ops_snapshot cola=%+v", result.OpsSnapshot)
	}
	foundSuperviseQueue := false
	for _, action := range result.Operator.SafeActions {
		if action.Action == "supervise" && action.Scope == "queue" {
			foundSuperviseQueue = true
			if action.Payload["resident_mode"] != true ||
				action.Payload["max_runs_per_tick"] != 70 ||
				action.Payload["max_executions"] != 70 ||
				action.Payload["max_dispatches_per_wait"] != 70 {
				t.Fatalf("payload de supervision de cola insuficiente: %+v", action)
			}
		}
	}
	if !foundSuperviseQueue {
		t.Fatalf("debe permitir supervision de cola con validacion posterior: %+v", result.Operator.SafeActions)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_stats_required_for_safe_supervision") {
		t.Fatalf("el aviso informativo debe conservarse: %+v", result.Diagnostics)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "queued" ||
		result.EfficiencySummary.QueueCandidates != 1 ||
		result.EfficiencySummary.OperationalHealthPercentage != 85 ||
		result.EfficiencySummary.AliveStuckStatus != "no_data" ||
		result.EfficiencySummary.Confidence != "partial" {
		t.Fatalf("efficiency_summary cola=%+v", result.EfficiencySummary)
	}
	for _, item := range result.Operator.SupervisorErrors {
		if item.Code == "run_stats_required_for_safe_supervision" {
			t.Fatalf("el aviso informativo no debe bloquear como error de supervisor: %+v", result.Operator.SupervisorErrors)
		}
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaQueuedNotDispatchedV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{
			ranked: []MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-queued-not-dispatched-001",
				AppRef:        "app-ref-queued-not-dispatched-001",
				Status:        "ready",
				PriorityScore: 90,
				EvidenceRefs:  []string{"evidence-ref-external-work-run-queued"},
			}},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != mcpAutoprogrammingQueuedNotDispatchedV0 {
			continue
		}
		found = true
		if diagnostic.Scope != "run:run-ref-queued-not-dispatched-001" ||
			!hasStringMCPAutoprogrammingStatusTestV0(diagnostic.EvidenceRefs, "evidence-ref-run-queue-queued-not-dispatched") ||
			!hasStringMCPAutoprogrammingStatusTestV0(diagnostic.EvidenceRefs, "evidence-ref-external-work-run-queued") {
			t.Fatalf("diagnostic=%+v", diagnostic)
		}
	}
	if !found {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.Queued != 1 ||
		result.QueueHealth.QueuedNotDispatched != 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoPublicaSuperviseLegacyPorDefectoV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-status-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	for _, action := range result.Operator.SafeActions {
		if action.Endpoint == MCPAutoprogrammingSuperviseHTTPPathV0 {
			t.Fatalf("no debe publicar supervision legacy sin opt-in: %+v", result.Operator.SafeActions)
		}
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "legacy_supervisor_actions_disabled") {
		t.Fatalf("falta diagnostico de opt-in legacy: %+v", result.Diagnostics)
	}
	if result.OpsSnapshot == nil ||
		result.OpsSnapshot.Decision.Action == "supervise_queue" {
		t.Fatalf("ops_snapshot=%+v", result.OpsSnapshot)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoDeclaraSuperviseSeguroConColaVacia(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	for _, action := range result.Operator.SafeActions {
		if action.Action == "supervise" || action.Action == "retry" {
			t.Fatalf("no debe recomendar supervision sobre cola vacia/no visible: %+v", result.Operator.SafeActions)
		}
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "queue_empty_or_not_visible") {
		t.Fatalf("falta diagnostico de cola vacia/no visible: %+v", result.Diagnostics)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "idle" ||
		result.EfficiencySummary.QueueCandidates != 0 ||
		result.EfficiencySummary.QueueVisibilityPercentage != 100 ||
		result.EfficiencySummary.OperationalHealthPercentage != 100 ||
		result.EfficiencySummary.AliveStuckStatus != "no_data" ||
		result.EfficiencySummary.Confidence != "medium" {
		t.Fatalf("efficiency_summary cola vacia=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0EfficiencyNoDeclaraIdleConGoalVivoYColaVacia(t *testing.T) {
	runRef := "run-ref-autop-status-live-goal-empty-queue-001"
	goalRef := "goal-ref-autop-status-live-goal-empty-queue-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  runRef,
		GoalRef: goalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Mantener visible un goal vivo aunque la cola legacy este vacia.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-live-goal-empty-queue"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue:          &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.ObservedRuns != 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "live" ||
		result.EfficiencySummary.ActiveRuns != 1 ||
		result.EfficiencySummary.AlivePercentage != 100 ||
		result.EfficiencySummary.AliveStuckStatus != "observed" ||
		result.EfficiencySummary.OverallPercentage >= 100 ||
		!hasStringMCPAutoprogrammingStatusTestV0(result.EfficiencySummary.Reasons, "running_live=1") {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
	if result.EfficiencySummary.RecommendedAction != "observe_goal:run:"+runRef {
		t.Fatalf("recommended_action=%q operator=%+v", result.EfficiencySummary.RecommendedAction, result.Operator)
	}
	if result.Operator == nil ||
		len(result.Operator.ActiveRuns) != 1 ||
		result.Operator.ActiveRuns[0].RunRef != runRef ||
		result.Operator.ActiveRuns[0].Status != orquestagoal.GoalStatusRunningV0 ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0GoalsCompletosNoOcultanRunningPorMaxItems(t *testing.T) {
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	for index := 0; index < mcpAutoprogrammingRunningStatsMaxV0; index++ {
		runRef := fmt.Sprintf("run-ref-autop-status-complete-%03d", index)
		goalRef := fmt.Sprintf("goal-ref-autop-status-complete-%03d", index)
		if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
			RunRef:  runRef,
			GoalRef: goalRef,
			Status:  orquestagoal.GoalStatusCompleteV0,
			Spec: orquestagoal.GoalWorkSpecV0{
				RunRef:       runRef,
				GoalRef:      goalRef,
				Objective:    "Goal complete pendiente de cierre.",
				DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
			},
			LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
				GoalRef: goalRef,
				Status:  orquestagoal.GoalStatusRunningV0,
			},
			EvidenceRefs: []string{"evidence-ref-complete-goal-max-items"},
		}); err != nil {
			t.Fatalf("SaveGoalWorkStateV0 complete %d: %v", index, err)
		}
	}
	liveRunRef := "run-ref-z-autop-status-running-no-starve-001"
	liveGoalRef := "goal-ref-z-autop-status-running-no-starve-001"
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  liveRunRef,
		GoalRef: liveGoalRef,
		Status:  orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       liveRunRef,
			GoalRef:      liveGoalRef,
			Objective:    "Goal running que no puede desaparecer por completados previos.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: liveGoalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
		},
		LastResult: &orquestagoal.GoalWorkResultV0{
			GoalRef: liveGoalRef,
			Status:  orquestagoal.GoalStatusRunningV0,
			Summary: "codex_app_server_thread_status_active",
		},
		EvidenceRefs: []string{"evidence-ref-running-goal-max-items"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 running: %v", err)
	}

	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue:          &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		GoalStateStore: goalStates,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.ObservedRuns < 1 {
		t.Fatalf("queue_health=%+v", result.QueueHealth)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "live" ||
		!hasStringMCPAutoprogrammingStatusTestV0(result.EfficiencySummary.Reasons, "running_live=1") {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", liveRunRef) {
		t.Fatalf("operator=%+v", result.Operator)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoRecomiendaSupervisarBucleDeReplan(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:       "run-ref-autop-status-loop-001",
				Status:       "activa",
				CurrentPhase: "programacion",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsStarted:       107,
					AgentsStopRequested: 106,
					AgentsStopConfirmed: 105,
					AgentsInFlight:      1,
					Deliveries:          0,
					Reviews:             0,
					ReplanDecisions:     106,
					Closures:            0,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-status-loop-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	for _, action := range result.Operator.SafeActions {
		if action.RequiresPost {
			t.Fatalf("no debe recomendar acciones POST que puedan amplificar supervision en bucle: %+v", result.Operator.SafeActions)
		}
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "supervisor_replan_amplification_blocked") {
		t.Fatalf("falta diagnostico de bucle: %+v", result.Diagnostics)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "hung" ||
		result.EfficiencySummary.OperationalHealthPercentage != 30 ||
		!hasStringMCPAutoprogrammingStatusTestV0(result.EfficiencySummary.Reasons, "supervisor_replan_amplification_blocked") {
		t.Fatalf("efficiency_summary bucle=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0EfficiencySummaryNoConfundeStalledConHung(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:       "run-ref-autop-status-stalled-001",
				Status:       "activa",
				CurrentPhase: "programacion",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:     1,
					TasksOpen:      1,
					AgentsStarted:  1,
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					TasksTotal:        1,
					StalledAgents:     1,
					ProgressingAgents: 1,
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status:  orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
					Blocked: true,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-autop-status-stalled-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
				}},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-status-stalled-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "live" ||
		result.EfficiencySummary.AgentsStalled != 1 ||
		result.EfficiencySummary.AgentsStuck != 0 ||
		result.EfficiencySummary.StuckPercentage != 0 {
		t.Fatalf("efficiency_summary stalled=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-transport-001"},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 || result.Run == nil || result.Queue == nil {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusTransportV0AceptaOperatorAdviceTexto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingStatusToolNameV0, map[string]any{
		"queue_limit":     1,
		"telemetry_flags": "summary",
		"operator_advice": "revisar trabajos sin borrar datos validos",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result mcpAutoprogrammingStatusTransportResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].Message != "revisar trabajos sin borrar datos validos" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("result=%+v advice=%+v diagnostics=%+v", result.MCPAutoprogrammingStatusToolResultV0, result.OperatorAdvice, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusTransportV0AceptaIncludesFlexibles(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    stats,
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	_, err = transport.CallToolV0(context.Background(), MCPAutoprogrammingStatusToolNameV0, map[string]any{
		"run_ref":                "run-ref-autop-status-flex-001",
		"include_process_refs":   []string{"process_refs"},
		"include_agent_progress": "yes",
		"include_agent_usage":    1,
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !stats.input.IncludeProcessRefs ||
		!stats.input.IncludeAgentProgress ||
		!stats.input.IncludeAgentUsage {
		t.Fatalf("stats input no normalizado: %+v", stats.input)
	}
}

type mcpGoalRunMarkerStoreForStatusTestV0 struct {
	markers map[string]orquestagoal.GoalWorkRunMarkerV0
}

func newMCPGoalRunMarkerStoreForStatusTestV0() *mcpGoalRunMarkerStoreForStatusTestV0 {
	return &mcpGoalRunMarkerStoreForStatusTestV0{
		markers: map[string]orquestagoal.GoalWorkRunMarkerV0{},
	}
}

func (store *mcpGoalRunMarkerStoreForStatusTestV0) SaveGoalWorkRunMarkerV0(
	_ context.Context,
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil {
		return err
	}
	store.markers[normalized.RunRef] = normalized
	return nil
}

func (store *mcpGoalRunMarkerStoreForStatusTestV0) LoadGoalWorkRunMarkerV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	marker, ok := store.markers[strings.TrimSpace(runRef)]
	if !ok {
		return orquestagoal.GoalWorkRunMarkerV0{}, context.Canceled
	}
	return marker, nil
}

func (store *mcpGoalRunMarkerStoreForStatusTestV0) ListGoalWorkRunMarkersV0(
	_ context.Context,
	request orquestagoal.GoalWorkRunMarkerListRequestV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	request = orquestagoal.NormalizeGoalWorkRunMarkerListRequestV0(request)
	out := make([]orquestagoal.GoalWorkRunMarkerV0, 0, len(store.markers))
	for _, marker := range store.markers {
		if !orquestagoal.GoalWorkRunMarkerMatchesListRequestV0(marker, request) {
			continue
		}
		out = append(out, marker)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.TrimSpace(out[i].RunRef) < strings.TrimSpace(out[j].RunRef)
	})
	if request.MaxItems > 0 && len(out) > request.MaxItems {
		out = out[:request.MaxItems]
	}
	if out == nil {
		return []orquestagoal.GoalWorkRunMarkerV0{}, nil
	}
	return out, nil
}

func hasStringMCPAutoprogrammingStatusTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (store *mcpGoalStateStoreForTestV0) ListGoalWorkStatesV0(
	_ context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	request = orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(store.states))
	for _, state := range store.states {
		if !orquestagoal.GoalWorkStateMatchesListRequestV0(state, request) {
			continue
		}
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.TrimSpace(out[i].RunRef) < strings.TrimSpace(out[j].RunRef)
	})
	if request.MaxItems > 0 && len(out) > request.MaxItems {
		out = out[:request.MaxItems]
	}
	if out == nil {
		return []orquestagoal.GoalWorkStateV0{}, nil
	}
	return out, nil
}

func hasMCPAutoprogrammingSafeActionForTestV0(
	actions []MCPAutoprogrammingSafeActionV0,
	action string,
	scope string,
	runRef string,
) bool {
	for _, value := range actions {
		if value.Action == action &&
			value.Scope == scope &&
			value.RunRef == runRef {
			return true
		}
	}
	return false
}

func hasMCPAutoprogrammingSafeActionPayloadForTestV0(
	actions []MCPAutoprogrammingSafeActionV0,
	action string,
	scope string,
	runRef string,
	key string,
	want string,
) bool {
	for _, value := range actions {
		if value.Action != action ||
			value.Scope != scope ||
			value.RunRef != runRef {
			continue
		}
		if got, ok := value.Payload[key].(string); ok && got == want {
			return true
		}
	}
	return false
}
