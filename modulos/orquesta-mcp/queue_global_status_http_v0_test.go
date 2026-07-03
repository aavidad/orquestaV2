package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPQueueGlobalStatusHTTPHandlerV0GetProyectaEstadoCompacto(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:         1,
					RunRef:       "run-ref-global-status-stale-001",
					AppRef:       "app-ref-global-status-001",
					Status:       "running",
					EvidenceRefs: []string{"evidence-ref-queue-stale-001"},
				}, {
					Rank:         2,
					RunRef:       "run-ref-global-status-queued-001",
					AppRef:       "app-ref-global-status-001",
					Status:       "ready",
					EvidenceRefs: []string{"evidence-ref-queue-ready-001"},
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Queued:                    2,
				QueuedNotDispatched:       1,
				RunningLive:               1,
				AgentsLive:                3,
				RunningStale:              1,
				RunningStaleNoProcess:     1,
				RunningWithoutRecentStats: 1,
				Blocked:                   1,
			},
			Operator: &MCPAutoprogrammingOperatorV0{
				ActiveRuns: []MCPAutoprogrammingActiveRunV0{{
					RunRef: "run-ref-global-status-active-001",
					Status: "running",
				}},
				SafeActions: []MCPAutoprogrammingSafeActionV0{{
					Action:   "observe_goal",
					Scope:    "run",
					RunRef:   "run-ref-global-status-goal-001",
					Method:   http.MethodPost,
					Endpoint: MCPAutoprogrammingObserveGoalHTTPPathV0,
				}},
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              "stale_running",
				RunRef:            "run-ref-global-status-stale-001",
				RecommendedAction: "observe_run_ref_with_process_refs_before_reconcile",
			}},
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code:  "queued_not_dispatched",
				Scope: "run:run-ref-global-status-queued-001",
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.QueueRef != "global" ||
		executor.input.QueueLimit != defaultMCPQueueGlobalStatusQueueLimitV0 ||
		!bool(executor.input.IncludeAgentProgress) ||
		!bool(executor.input.IncludeAgentUsage) {
		t.Fatalf("input=%+v", executor.input)
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.SchemaVersion != MCPQueueGlobalStatusSchemaVersionV0 ||
		result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.QueueRef != "global" ||
		result.Summary.Queued != 2 ||
		result.Summary.QueuedNotDispatched != 1 ||
		result.Summary.RunningLive != 1 ||
		result.Summary.AgentsLive != 3 ||
		result.Summary.RunningStale != 1 ||
		result.Summary.RunningStaleNoProcess != 1 ||
		result.Summary.RunningWithoutRecentStats != 1 ||
		result.Summary.Blocked != 1 ||
		!result.Summary.NeedsAttention ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		len(result.GoalRunRefs) != 1 ||
		result.GoalRunRefs[0] != "run-ref-global-status-goal-001" ||
		len(result.Items) != 4 ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-stale-001", "stale_running", true, "inspect_liveness") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-queued-001", "queued_not_dispatched", true, "reencolar") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-active-001", "running_live", false, "") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-goal-001", "observer_required", true, "restart_observer") ||
		len(result.SafeActions) != 1 ||
		len(result.StaleRunning) != 1 ||
		len(result.Diagnostics) != 1 {
		t.Fatalf("result=%+v", result)
	}
	goalItem := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-goal-001")
	if len(goalItem.EvidenceRefs) == 0 {
		t.Fatalf("safe action sin evidencia fallback: %+v", goalItem)
	}
	liveItem := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-active-001")
	if liveItem.NoActionReason != "running_live_wait_processes" {
		t.Fatalf("run live sin razon de no accion: %+v", liveItem)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0WillFinishAloneSinAccionPendiente(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:         1,
					RunRef:       "run-ref-global-status-live-001",
					AppRef:       "app-ref-global-status-live-001",
					Status:       "running",
					EvidenceRefs: []string{"evidence-ref-live-queue-001"},
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				RunningLive: 1,
				AgentsLive:  1,
			},
			Operator: &MCPAutoprogrammingOperatorV0{
				ActiveRuns: []MCPAutoprogrammingActiveRunV0{{
					RunRef: "run-ref-global-status-live-001",
					AppRef: "app-ref-global-status-live-001",
					Status: "running",
				}},
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.NeedsAttention ||
		result.Summary.NeedsAction ||
		!result.Summary.WillFinishAlone ||
		len(result.Items) != 1 ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-live-001", "running_live", false, "") {
		t.Fatalf("result=%+v", result)
	}
	item := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-live-001")
	if item.NoActionReason != "running_live_wait_processes" {
		t.Fatalf("item sin razon estable de no accion: %+v", item)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0DerivaVidaDesdeProyeccionV0(t *testing.T) {
	runRef := "run-ref-global-status-estado-vivo-001"
	queue := &fakeMCPAutoprogrammingQueueStatusV0{
		ranked: []MCPRunQueueRankedCandidateCompactV0{{
			Rank:         1,
			RunRef:       runRef,
			AppRef:       "app-ref-global-status-estado-vivo-001",
			Status:       "running",
			EvidenceRefs: []string{"evidence-ref-queue-estado-vivo-001"},
		}},
	}
	stats := &fakeMCPAutoprogrammingRunStatusV0{
		statsByRun: map[string]*orquestacionnucleoapp.DirectorRunStatsV0{
			runRef: {
				RunRef:     runRef,
				ProjectRef: "app-ref-global-status-estado-vivo-001",
				Status:     "running",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksTotal:     1,
					AgentsInFlight: 1,
				},
				Progress: orquestacionnucleoapp.DirectorProgressStatsV0{
					TasksTotal:        1,
					PercentComplete:   50,
					ProgressingAgents: 1,
				},
				Closure: orquestacionnucleoapp.DirectorClosureStatsV0{
					Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
				},
				Agents: []orquestacionnucleoapp.DirectorAgentStatsV0{{
					AgentRequestID: "agent-ref-global-status-estado-vivo-001",
					Status:         orquestacionnucleoapp.DirectorAgentStatusRunningV0,
					InFlight:       true,
					LastProgress: &orquestacionnucleoapp.DirectorAgentProgressV0{
						Status: "progressing",
					},
					Process: &orquestacionnucleoapp.DirectorAgentProcessStatsV0{
						ProcessRef: "process-ref-global-status-estado-vivo-001",
						Status:     orquestacionnucleoapp.DirectorAgentProcessStatusRunningV0,
					},
				}},
			},
		},
	}
	estadoVivo := &fakeMCPAutoprogrammingEstadoVivoSourceV0{
		evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
			RunRef:       runRef,
			Fuente:       "process_snapshot",
			ProcesoVivo:  true,
			ObservadoEn:  "2026-07-03T09:59:00Z",
			EvidenceRefs: []string{"evidence-ref-estado-vivo-running-live-001"},
		}},
	}
	executor := MCPAutoprogrammingStatusToolExecutorV0{
		Queue:            queue,
		Stats:            stats,
		EstadoVivoSource: estadoVivo,
	}
	req := httptest.NewRequest(
		http.MethodGet,
		MCPQueueGlobalStatusHTTPPathV0+"?occurred_at=2026-07-03T10:00:00Z",
		nil,
	)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.SchemaVersion != MCPQueueGlobalStatusSchemaVersionV0 ||
		result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.Summary.RunningLive != 1 ||
		result.Summary.NeedsAttention ||
		result.Summary.NeedsAction ||
		!result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, runRef, "running_live", false, "") {
		t.Fatalf("result=%+v", result)
	}
	item := mcpQueueGlobalStatusItemForTestV0(result.Items, runRef)
	if item.NoActionReason != "running_live_wait_processes" {
		t.Fatalf("item=%+v", item)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0CadaRunVisibleTieneAccionORazon(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:   1,
					RunRef: "run-ref-global-status-completed-001",
					Status: "completed",
				}, {
					Rank:   2,
					RunRef: "run-ref-global-status-failed-001",
					Status: "failed",
				}, {
					Rank:   3,
					RunRef: "run-ref-global-status-queued-healthy-001",
					Status: "pending",
				}},
				Terminal: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:   4,
					RunRef: "run-ref-global-status-accepted-terminal-001",
					Status: "accepted",
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Queued:    1,
				Completed: 1,
				Failed:    1,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-failed-001", "failed", true, "repair_runtime") {
		t.Fatalf("result=%+v", result)
	}
	completed := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-completed-001")
	queued := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-queued-healthy-001")
	accepted := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-accepted-terminal-001")
	if completed.NeedsAction ||
		completed.NoActionReason != "terminal_completed" ||
		queued.NeedsAction ||
		queued.NoActionReason != "queued_waiting_scheduler" ||
		accepted.NeedsAction ||
		accepted.NoActionReason != "terminal_completed" {
		t.Fatalf("completed=%+v queued=%+v accepted=%+v", completed, queued, accepted)
	}
	for _, item := range result.Items {
		if item.NeedsAction == (item.RecommendedAction == "") {
			t.Fatalf("item sin contrato accion/razon: %+v", item)
		}
		if !item.NeedsAction && item.NoActionReason == "" {
			t.Fatalf("item sin razon de no accion: %+v", item)
		}
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0MezclaGoalFirstYLivenessAccionORazon(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:   1,
					RunRef: "run-ref-global-status-running-live-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "running",
				}, {
					Rank:   2,
					RunRef: "run-ref-global-status-running-stale-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "running",
				}, {
					Rank:   3,
					RunRef: "run-ref-global-status-goal-missing-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "running",
				}, {
					Rank:   4,
					RunRef: "run-ref-global-status-goal-observe-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "running",
				}, {
					Rank:   5,
					RunRef: "run-ref-global-status-ready-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "ready",
				}},
				Terminal: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:   6,
					RunRef: "run-ref-global-status-complete-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "completed",
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Queued:                    1,
				RunningLive:               2,
				RunningWithoutRecentStats: 1,
				Blocked:                   1,
				Completed:                 1,
			},
			Operator: &MCPAutoprogrammingOperatorV0{
				ActiveRuns: []MCPAutoprogrammingActiveRunV0{{
					RunRef: "run-ref-global-status-running-live-001",
					AppRef: "app-ref-global-status-mixed",
					Status: "running",
				}},
				SafeActions: []MCPAutoprogrammingSafeActionV0{{
					Action:   "observe_goal",
					Scope:    "run",
					RunRef:   "run-ref-global-status-goal-observe-001",
					Method:   http.MethodPost,
					Endpoint: MCPAutoprogrammingObserveGoalHTTPPathV0,
				}},
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
				RunRef:            "run-ref-global-status-running-stale-001",
				AppRef:            "app-ref-global-status-mixed",
				RecommendedAction: "observe_run_ref_with_process_refs_before_reconcile",
				EvidenceRefs:      []string{"evidence-ref-global-status-running-stale"},
			}, {
				Code:              mcpAutoprogrammingActionGoalFirstStateMissingV0,
				RunRef:            "run-ref-global-status-goal-missing-001",
				AppRef:            "app-ref-global-status-mixed",
				RecommendedAction: "repair_goal_state_before_legacy_supervision",
				EvidenceRefs:      []string{"evidence-ref-global-status-goal-missing"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Items) != 6 ||
		!result.Summary.NeedsAttention ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-running-live-001", "running_live", false, "") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-running-stale-001", mcpAutoprogrammingActionRunningWithoutRecentStatsV0, true, "inspect_liveness") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-goal-missing-001", mcpAutoprogrammingActionGoalFirstStateMissingV0, true, "repair_goal_state") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-goal-observe-001", "observer_required", true, "restart_observer") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-ready-001", "ready", false, "") ||
		!hasMCPQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-complete-001", mcpAutoprogrammingHealthCompletedV0, false, "") {
		t.Fatalf("result=%+v", result)
	}
	assertMCPQueueGlobalStatusItemsHaveActionOrReasonForTestV0(t, result.Items)
}

func TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstStateMissingRecomiendaRepararState(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstStateMissingV0,
				RunRef:            "run-ref-global-status-goal-state-missing-001",
				RecommendedAction: "repair_goal_state_before_legacy_supervision",
				EvidenceRefs:      []string{"evidence-ref-goal-state-missing-global-status"},
			}},
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code:         "autoprogramming_goal_first_state_missing",
				Scope:        "run:run-ref-global-status-goal-state-missing-001",
				EvidenceRefs: []string{"evidence-ref-goal-state-missing-diagnostic"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Summary.NeedsAttention ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(
			result.Items,
			"run-ref-global-status-goal-state-missing-001",
			mcpAutoprogrammingActionGoalFirstStateMissingV0,
			true,
			"repair_goal_state",
		) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0GoalBackendAusenteConservaRunControlReconcile(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalBackendMissingAfterExternalCleanupV0,
				RunRef:            "run-ref-global-status-goal-cleanup-001",
				RecommendedAction: mcpQueueGlobalStatusActionRunControlReconcileCleanupV0,
				EvidenceRefs:      []string{"evidence-ref-goal-cleanup-global-status"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Summary.NeedsAttention ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(
			result.Items,
			"run-ref-global-status-goal-cleanup-001",
			mcpAutoprogrammingActionGoalBackendMissingAfterExternalCleanupV0,
			true,
			mcpQueueGlobalStatusActionRunControlReconcileCleanupV0,
		) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstBlockedConservaReviewReplan(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
				RunRef:            "run-ref-global-status-goal-blocked-001",
				RecommendedAction: "review_replan_goal_first",
				EvidenceRefs:      []string{"evidence-ref-goal-blocked-global-status"},
			}},
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code:         "autoprogramming_goal_first_blocked",
				Scope:        "run:run-ref-global-status-goal-blocked-001",
				EvidenceRefs: []string{"evidence-ref-goal-blocked-diagnostic"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Summary.NeedsAttention ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusItemForTestV0(
			result.Items,
			"run-ref-global-status-goal-blocked-001",
			mcpAutoprogrammingActionGoalFirstBlockedV0,
			true,
			mcpQueueGlobalStatusActionReviewReplanGoalFirstV0,
		) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstBlockedDesdeDiagnosticoNoReparaRuntime(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code:         "autoprogramming_goal_first_blocked",
				Scope:        "run:run-ref-global-status-goal-blocked-diagnostic-001",
				EvidenceRefs: []string{"evidence-ref-goal-blocked-diagnostic"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-goal-blocked-diagnostic-001",
		mcpAutoprogrammingActionGoalFirstBlockedV0,
		true,
		mcpQueueGlobalStatusActionReviewReplanGoalFirstV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaObserveGoalBackendNextArtifact(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				RunningStale: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              "checkpoint_only_consumption_warning",
				RunRef:            "run-ref-global-status-checkpoint-next-artifact-001",
				AppRef:            "app-ref-global-status-checkpoint",
				RecommendedAction: mcpQueueGlobalStatusActionObserveGoalRequireNextArtifactV0,
				EvidenceRefs:      []string{"artifact-ref-checkpoint"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-checkpoint-next-artifact-001",
		"checkpoint_only_consumption_warning",
		true,
		mcpQueueGlobalStatusActionObserveGoalRequireNextArtifactV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaReplanNarrowContext(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			GoalProgressPolicy: &MCPAutoprogrammingGoalProgressPolicyV0{
				CheckpointOnlyHighConsumptionTokens: 64000,
				CheckpointOnlyMaxWaitSeconds:        321,
				NoCheckpointWarningMaxWaitSeconds:   654,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0,
				RunRef:            "run-ref-global-status-checkpoint-high-001",
				AppRef:            "app-ref-global-status-checkpoint",
				RecommendedAction: mcpQueueGlobalStatusActionReplanNarrowContextV0,
				EvidenceRefs:      []string{"evidence-ref-checkpoint-high"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-checkpoint-high-001",
		mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0,
		true,
		mcpQueueGlobalStatusActionReplanNarrowContextV0,
	) {
		t.Fatalf("result=%+v", result)
	}
	if result.GoalProgressPolicy == nil ||
		result.GoalProgressPolicy.CheckpointOnlyHighConsumptionTokens != 64000 ||
		result.GoalProgressPolicy.CheckpointOnlyMaxWaitSeconds != 321 ||
		result.GoalProgressPolicy.NoCheckpointWarningMaxWaitSeconds != 654 {
		t.Fatalf("goal_progress_policy no conservada en global-status: %+v", result.GoalProgressPolicy)
	}
}

func TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas(t *testing.T) {
	for _, action := range []string{
		"replan",
		mcpQueueGlobalStatusActionReviewReplanGoalFirstV0,
		mcpQueueGlobalStatusActionRetryFromPhaseV0,
		mcpQueueGlobalStatusActionCloseSupersededByLocalEvidenceV0,
		mcpQueueGlobalStatusActionRunControlReconcileCleanupV0,
		mcpQueueGlobalStatusActionObserveGoalRequireCheckpointV0,
		mcpQueueGlobalStatusActionObserveGoalRequireNextArtifactV0,
		mcpQueueGlobalStatusActionObserveGoalWaitForCheckpointV0,
		mcpQueueGlobalStatusActionReplanNarrowContextV0,
		mcpQueueGlobalStatusActionReplanGoalAfterActiveTimeoutV0,
		mcpQueueGlobalStatusActionReconcileGoalTerminalV0,
		mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0,
		mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0,
		MCPGoalFirstRepairReceiptActionV0,
		MCPGoalFirstReworkWriteSetViolationActionV0,
		MCPGoalFirstReworkPublicTextActionV0,
		MCPGoalFirstReviewPartialArtifactsActionV0,
		MCPGoalFirstContinueFromPhase0ActionV0,
	} {
		t.Run(action, func(t *testing.T) {
			if got := mcpQueueGlobalStatusNormalizeRecommendedActionV0(action, "repair_runtime"); got != action {
				t.Fatalf("action=%q, want %q", got, action)
			}
		})
	}
}

func TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstPorRun(t *testing.T) {
	runRef := "run-ref-global-status-goal-first-action-001"
	for _, action := range []string{
		mcpQueueGlobalStatusActionReplanNarrowContextV0,
		mcpQueueGlobalStatusActionReplanGoalAfterActiveTimeoutV0,
		mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0,
		mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0,
		MCPGoalFirstRepairReceiptActionV0,
		MCPGoalFirstReworkWriteSetViolationActionV0,
		MCPGoalFirstReworkPublicTextActionV0,
		MCPGoalFirstReviewPartialArtifactsActionV0,
		MCPGoalFirstContinueFromPhase0ActionV0,
	} {
		want := action + ":run:" + runRef
		t.Run(want, func(t *testing.T) {
			if got := mcpQueueGlobalStatusNormalizeRecommendedActionV0(want, "repair_runtime"); got != want {
				t.Fatalf("action=%q, want %q", got, want)
			}
		})
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaWriteSetGuardContract(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0,
				RunRef:            "run-ref-global-status-write-set-guard-mismatch-001",
				RecommendedAction: mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0,
				EvidenceRefs: []string{
					mcpAutoprogrammingEvidenceWriteSetGuardAllowedWriteSetMismatchV0,
					"evidence-ref-codex-app-server-write-set-guard-allowed-write-set-mismatch",
				},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-write-set-guard-mismatch-001",
		mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0,
		true,
		mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaWriteSetRequiresWorkspaceWrite(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0,
				RunRef:            "run-ref-global-status-write-set-read-only-001",
				RecommendedAction: mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0,
				EvidenceRefs: []string{
					mcpAutoprogrammingEvidenceWriteSetRequiresWorkspaceWriteV0,
					"evidence-ref-codex-app-server-write-set-requires-workspace-write",
				},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-write-set-read-only-001",
		mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0,
		true,
		mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0PropagaFaseYCountersOPESRetryFromPhase(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
				RunRef:            "run-ref-global-status-opes-audio-prepare-001",
				RecommendedAction: mcpQueueGlobalStatusActionRetryFromPhaseV0,
				CurrentPhase:      "prepare",
				DomainCounters: map[string]int{
					"audio_manifest_refs": 1,
					"text_hash_refs":      2,
				},
				EvidenceRefs: []string{"evidence-ref-opes-audio-prepare-stale"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	item := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-opes-audio-prepare-001")
	if item.Status != mcpAutoprogrammingActionGoalFirstBlockedV0 ||
		!item.NeedsAction ||
		item.RecommendedAction != mcpQueueGlobalStatusActionRetryFromPhaseV0 ||
		item.CurrentPhase != "prepare" ||
		item.DomainCounters["audio_manifest_refs"] != 1 ||
		item.DomainCounters["text_hash_refs"] != 2 {
		t.Fatalf("item=%+v result=%+v", item, result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaOutputSaneadoGoalFirst(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				RunningLive: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionThreadOutputSanitizedV0,
				RunRef:            "run-ref-global-status-thread-output-sanitized-001",
				AppRef:            "app-ref-global-status-thread-output",
				RecommendedAction: mcpQueueGlobalStatusActionReplanNarrowContextV0,
				EvidenceRefs: []string{
					mcpAutoprogrammingEvidenceThreadOutputSanitizedV0,
					mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0,
				},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-thread-output-sanitized-001",
		mcpAutoprogrammingActionThreadOutputSanitizedV0,
		true,
		mcpQueueGlobalStatusActionReplanNarrowContextV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ConservaRepairReceiptGoalFirst(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Blocked: 1,
			},
			StaleRunning: []MCPAutoprogrammingActionableRunV0{{
				Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
				RunRef:            "run-ref-global-status-repair-receipt-001",
				RecommendedAction: MCPGoalFirstRepairReceiptActionV0,
				EvidenceRefs:      []string{"evidence-ref-repair-receipt"},
			}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasMCPQueueGlobalStatusItemForTestV0(
		result.Items,
		"run-ref-global-status-repair-receipt-001",
		mcpAutoprogrammingActionGoalFirstBlockedV0,
		true,
		MCPGoalFirstRepairReceiptActionV0,
	) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0AcceptedNoRequiereAccion(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			Queue: &MCPRunQueuePriorityToolResultV0{
				Estado:   MCPRunQueuePriorityEstadoOKV0,
				QueueRef: "global",
				Ranked: []MCPRunQueueRankedCandidateCompactV0{{
					Rank:   1,
					RunRef: "run-ref-global-status-accepted-ranked-001",
					Status: "accepted",
				}},
			},
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Completed: 1,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	item := mcpQueueGlobalStatusItemForTestV0(result.Items, "run-ref-global-status-accepted-ranked-001")
	if item.Status != mcpAutoprogrammingHealthCompletedV0 ||
		item.NeedsAction ||
		item.RecommendedAction != "" ||
		item.NoActionReason != "terminal_completed" {
		t.Fatalf("item=%+v result=%+v", item, result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPQueueGlobalStatusHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Message != "queue_global_status_no_configurado" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPQueueGlobalStatusHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{waitForCancel: true}
	req := httptest.NewRequest(http.MethodGet, MCPQueueGlobalStatusHTTPPathV0, nil)
	req.Header.Set("X-Correlation-ID", "corr-queue-global-timeout-001")
	rec := httptest.NewRecorder()

	newMCPQueueGlobalStatusHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-queue-global-timeout-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPQueueGlobalStatusResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "queue_global_status_timeout" ||
		!result.Summary.NeedsAction ||
		result.Summary.WillFinishAlone ||
		!hasMCPQueueGlobalStatusDiagnosticCodeForTestV0(result.Diagnostics, "queue_global_status_timeout") {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPQueueGlobalStatusExecutorV0 struct {
	input         MCPAutoprogrammingStatusToolInputV0
	result        MCPAutoprogrammingStatusToolResultV0
	err           error
	waitForCancel bool
}

func (executor *fakeMCPQueueGlobalStatusExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	executor.input = input
	if executor.waitForCancel {
		<-ctx.Done()
		return MCPAutoprogrammingStatusToolResultV0{}, ctx.Err()
	}
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: input.QueueRef,
		}
	}
	return executor.result, executor.err
}

func mcpQueueGlobalStatusItemForTestV0(
	items []MCPQueueGlobalStatusItemV0,
	runRef string,
) MCPQueueGlobalStatusItemV0 {
	for _, item := range items {
		if item.RunRef == runRef {
			return item
		}
	}
	return MCPQueueGlobalStatusItemV0{}
}

func hasMCPQueueGlobalStatusItemForTestV0(
	items []MCPQueueGlobalStatusItemV0,
	runRef string,
	status string,
	needsAction bool,
	recommendedAction string,
) bool {
	for _, item := range items {
		if item.RunRef == runRef &&
			item.Status == status &&
			item.NeedsAction == needsAction &&
			item.RecommendedAction == recommendedAction {
			return true
		}
	}
	return false
}

func hasMCPQueueGlobalStatusDiagnosticCodeForTestV0(
	diagnostics []MCPAutoprogrammingDiagnosticV0,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func assertMCPQueueGlobalStatusItemsHaveActionOrReasonForTestV0(
	t *testing.T,
	items []MCPQueueGlobalStatusItemV0,
) {
	t.Helper()
	for _, item := range items {
		hasAction := item.NeedsAction && item.RecommendedAction != ""
		hasReason := !item.NeedsAction && item.NoActionReason != ""
		if hasAction == hasReason {
			t.Fatalf("item debe tener exactamente accion o razon de no accion: %+v", item)
		}
	}
}
