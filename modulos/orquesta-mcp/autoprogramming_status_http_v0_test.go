package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPAutoprogrammingStatusHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:        MCPAutoprogrammingStatusEstadoOKV0,
			CorrelationID: "corr-autop-status-http-001",
			RunRef:        "run-ref-autop-status-http-001",
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-autop-status-http-001"}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-autop-status-http-001" ||
		rec.Header().Get("X-Correlation-ID") != "corr-autop-status-http-001" {
		t.Fatalf("input=%+v headers=%v", executor.input, rec.Header())
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeMCPAutoprogrammingStatusHTTPExecutorV0 struct {
	input  MCPAutoprogrammingStatusToolInputV0
	result MCPAutoprogrammingStatusToolResultV0
}

func (executor *fakeMCPAutoprogrammingStatusHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0ErrorPublico(t *testing.T) {
	result := MCPAutoprogrammingStatusToolResultV0{
		Estado: MCPAutoprogrammingStatusEstadoErrorV0,
		Errores: []MCPValidationIssueV0{{
			Code:    "autoprogramming_status_no_disponible",
			Field:   "ports",
			Message: "estado de autoprogramacion no disponible",
		}},
	}
	encoded, _ := json.Marshal(result)
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{result: result}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewReader(encoded))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
