package orquestaappgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingPrepareRunAPIRouteV0(t *testing.T) {
	executor := &recordingAutoprogrammingPrepareRunExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:                   time.Second,
		AutoprogrammingPrepareRun: executor,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/prepare-run",
		strings.NewReader(`{"request_id":"request-prepare-run-route-001"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.RequestID != "request-prepare-run-route-001" {
		t.Fatalf("input=%+v", executor.Input)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, `"goal_specs"`) || strings.Contains(raw, `"write_set"`) {
		t.Fatalf("respuesta publica filtra specs completos: %s", raw)
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0 ||
		!result.Accepted ||
		result.RunRef != "run-prepare-route-001" ||
		len(result.GoalSpecs) != 0 ||
		len(result.GoalSpecSummaries) != 1 ||
		result.GoalSpecSummaries[0].RunRef != "run-prepare-route-001" ||
		result.GoalSpecSummaries[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		result.GoalSpecSummaries[0].SpecHash == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingObserveGoalAPIRouteV0(t *testing.T) {
	executor := &recordingAutoprogrammingObserveGoalExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:                    time.Second,
		AutoprogrammingObserveGoal: executor,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/goal/observe",
		strings.NewReader(`{"request_id":"request-observe-goal-route-001","run_ref":"run-observe-goal-route-001"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.RunRef != "run-observe-goal-route-001" {
		t.Fatalf("input=%+v", executor.Input)
	}
	var result orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0 ||
		result.RunRef != "run-observe-goal-route-001" ||
		result.GoalRef != "goal-ref-observe-goal-route-001" {
		t.Fatalf("result=%+v", result)
	}
}

type recordingAutoprogrammingPrepareRunExecutorV0 struct {
	Input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0
}

func (executor *recordingAutoprogrammingPrepareRunExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) (orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0, error) {
	_ = ctx
	executor.Input = input
	return orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:   orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
		Accepted: true,
		RunRef:   "run-prepare-route-001",
		GoalSpecs: []orquestagoal.GoalWorkSpecV0{{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-prepare-route-001",
			RunRef:        "run-prepare-route-001",
			Objective:     "validar passthrough app gateway de specs internas",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-app-gateway"}},
		}},
	}, nil
}

type recordingAutoprogrammingObserveGoalExecutorV0 struct {
	Input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0
}

func (executor *recordingAutoprogrammingObserveGoalExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
) (orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0, error) {
	_ = ctx
	executor.Input = input
	return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{
		Estado:     orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0,
		RunRef:     input.RunRef,
		GoalRef:    "goal-ref-observe-goal-route-001",
		GoalStatus: orquestagoal.GoalStatusRunningV0,
	}, nil
}
