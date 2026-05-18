package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRunSupervisorHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado:        MCPRunSupervisorEstadoOKV0,
			CorrelationID: "corr-run-supervisor-http-001",
			RunRef:        "run-ref-supervisor-http-001",
			StopReason:    "max_ticks",
			Ticks:         2,
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-001",
		CorrelationID: "corr-run-supervisor-http-001",
		RunRef:        "run-ref-supervisor-http-001",
		MaxTicks:      2,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-supervisor-http-001" || executor.input.MaxTicks != 2 {
		t.Fatalf("input=%+v", executor.input)
	}
	if rec.Header().Get("X-Correlation-ID") != "corr-run-supervisor-http-001" {
		t.Fatalf("correlation header=%s", rec.Header().Get("X-Correlation-ID"))
	}
}

func TestMCPRunSupervisorHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPRunSupervisorHTTPHandlerV0SoloPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPRunSupervisorHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(&fakeMCPRunSupervisorHTTPExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("status=%d allow=%s", rec.Code, rec.Header().Get("Allow"))
	}
}

type fakeMCPRunSupervisorHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	result MCPRunSupervisorToolResultV0
}

func (executor *fakeMCPRunSupervisorHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPRunSupervisorToolResultV0{Estado: MCPRunSupervisorEstadoOKV0}
	}
	return executor.result, nil
}
