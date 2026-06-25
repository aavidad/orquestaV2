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
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0 ||
		!result.Accepted ||
		result.RunRef != "run-prepare-route-001" ||
		len(result.GoalSpecs) != 1 ||
		result.GoalSpecs[0].RunRef != "run-prepare-route-001" ||
		result.GoalSpecs[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 {
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
			Objective:     "validar passthrough app gateway de goal_specs",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-app-gateway"}},
		}},
	}, nil
}
