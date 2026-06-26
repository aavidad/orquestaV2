package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{
		result: MCPAutoprogrammingObserveGoalToolResultV0{
			Estado:     MCPAutoprogrammingObserveGoalEstadoOKV0,
			RunRef:     "run-ref-autoprogramming-goal-http-001",
			GoalRef:    "goal-ref-autoprogramming-goal-http-001",
			GoalStatus: "running",
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID: "request-ref-autoprogramming-observe-goal-http-001",
		RunRef:    "run-ref-autoprogramming-goal-http-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if executor.input.RunRef != "run-ref-autoprogramming-goal-http-001" ||
		result.GoalRef != "goal-ref-autoprogramming-goal-http-001" {
		t.Fatalf("input=%+v result=%+v", executor.input, result)
	}
}

func TestMCPAutoprogrammingObserveGoalHTTPHandlerV0RunRefRequerido(t *testing.T) {
	executor := &fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0{}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingObserveGoalHTTPPathV0, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0 struct {
	input  MCPAutoprogrammingObserveGoalToolInputV0
	result MCPAutoprogrammingObserveGoalToolResultV0
	err    error
}

func (executor *fakeMCPAutoprogrammingObserveGoalHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error) {
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingObserveGoalToolResultV0{
			Estado: MCPAutoprogrammingObserveGoalEstadoOKV0,
			RunRef: input.RunRef,
		}
	}
	return executor.result, executor.err
}
