package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
		len(result.Operator.SafeActions) < 2 {
		t.Fatalf("operator=%+v", result.Operator)
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
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
				},
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
	if result.Operator == nil ||
		!hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "observe_goal", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "run", runRef) ||
		hasMCPAutoprogrammingSafeActionForTestV0(result.Operator.SafeActions, "supervise", "queue", "") {
		t.Fatalf("operator=%+v", result.Operator)
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
						},
					},
					{
						AgentRequestID: "agent-ref-running-live-process-count-002",
						Status:         orquestacionnucleoapp.DirectorAgentStatusRequestedV0,
						Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
							SessionRef: "session-ref-running-live-process-count-002",
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

func TestMCPAutoprogrammingStatusExecutorV0DeclaraSuperviseColaAunqueFaltenStatsDeRunV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
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

func hasStringMCPAutoprogrammingStatusTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
