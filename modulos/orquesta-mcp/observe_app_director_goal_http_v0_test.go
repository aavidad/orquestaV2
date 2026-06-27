package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
)

func TestMCPObserveAppDirectorGoalHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPObserveAppDirectorGoalHTTPExecutorV0{
		result: MCPObserveAppDirectorGoalToolResultV0{
			Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
			RunRef:                "run-ref-goal-http-001",
			RunStatus:             "closed",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			GoalRef:               "goal-ref-http-001",
			GoalStatus:            "complete",
			EvidenceRefs:          []string{"evidence-ref-goal-http-001"},
		},
	}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-http-001",
		RunRef:    "run-ref-goal-http-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-goal-http-001")
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Correlation-ID") != "corr-goal-http-001" {
		t.Fatalf("correlation=%q", rec.Header().Get("X-Correlation-ID"))
	}
	if executor.input.RunRef != "run-ref-goal-http-001" {
		t.Fatalf("input=%+v", executor.input)
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.GoalRef != "goal-ref-http-001" ||
		result.RunStatus != "closed" ||
		result.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0RunRefRequerido(t *testing.T) {
	executor := &fakeMCPObserveAppDirectorGoalHTTPExecutorV0{}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.called != 0 {
		t.Fatalf("executor llamado sin run_ref")
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" ||
		result.Errores[0].Message != "run_ref_requerido" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0ErrorOperativoNoDevuelve500(t *testing.T) {
	executor := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{})
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-http-operational-error-001",
		RunRef:    "run-ref-goal-http-operational-error-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "observe_app_director_goal_state_store_unbound" ||
		result.Errores[0].Field != "goal_state_store" ||
		result.Errores[0].Message != "goal_state_store_not_configured" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0RunRefRequerido(t *testing.T) {
	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{}).Execute(
		context.Background(),
		MCPObserveAppDirectorGoalToolInputV0{},
	)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPObserveAppDirectorGoalHTTPExecutorV0 struct {
	input  MCPObserveAppDirectorGoalToolInputV0
	called int
	result MCPObserveAppDirectorGoalToolResultV0
}

func (executor *fakeMCPObserveAppDirectorGoalHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	executor.called++
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPObserveAppDirectorGoalToolResultV0{
			Estado: MCPObserveAppDirectorGoalEstadoOKV0,
			RunRef: input.RunRef,
		}
	}
	return executor.result, nil
}
