package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPQueueGlobalStatusHTTPHandlerV0GetProyectaEstadoCompacto(t *testing.T) {
	executor := &fakeMCPQueueGlobalStatusExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: "global",
			QueueHealth: &MCPAutoprogrammingQueueHealthV0{
				Queued:                    2,
				QueuedNotDispatched:       1,
				RunningLive:               1,
				AgentsLive:                3,
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
				Code:   "stale_running",
				RunRef: "run-ref-global-status-stale-001",
			}},
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code:  "queued_not_dispatched",
				Scope: "queue",
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
		result.Summary.Blocked != 1 ||
		!result.Summary.NeedsAttention ||
		len(result.GoalRunRefs) != 1 ||
		result.GoalRunRefs[0] != "run-ref-global-status-goal-001" ||
		len(result.SafeActions) != 1 ||
		len(result.StaleRunning) != 1 ||
		len(result.Diagnostics) != 1 {
		t.Fatalf("result=%+v", result)
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

type fakeMCPQueueGlobalStatusExecutorV0 struct {
	input  MCPAutoprogrammingStatusToolInputV0
	result MCPAutoprogrammingStatusToolResultV0
	err    error
}

func (executor *fakeMCPQueueGlobalStatusExecutorV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingStatusToolResultV0{
			Estado:   MCPAutoprogrammingStatusEstadoOKV0,
			QueueRef: input.QueueRef,
		}
	}
	return executor.result, executor.err
}
