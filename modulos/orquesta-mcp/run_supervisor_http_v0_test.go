package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestMCPRunSupervisorHTTPHandlerV0NoCancelaSupervisorPorCierreHTTP(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-supervisor-http-cancel-001",
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-supervisor-http-cancel-001","max_ticks":1}`)
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.ctxErr != nil {
		t.Fatalf("executor recibio contexto cancelado: %v", executor.ctxErr)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0PropagaErrorPublicoDelExecutor(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-error-001",
		CorrelationID: "corr-run-supervisor-http-error-001",
		RunRef:        "run-ref-supervisor-http-error-001",
	}
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: NewMCPRunSupervisorErrorResultV0(
			input,
			"run_supervisor_execute_error",
			"executor",
			"delivery_ack_ingestion_failed",
		),
		err: errors.New("internal stack error must not hide public payload"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_supervisor_execute_error" ||
		result.Errores[0].Message != "delivery_ack_ingestion_failed" {
		t.Fatalf("error publico perdido: %+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0NoPropagaErrorNoCatalogado(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-cause-001",
		CorrelationID: "corr-run-supervisor-http-cause-001",
		RunRef:        "run-ref-supervisor-http-cause-001",
	}
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		err: errors.New("codex_worktree_verification: ack_files_mismatch en /root/Trabajo/orquesta con token=secret123456"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Message != "run_supervisor_error" {
		t.Fatalf("error publico inesperado: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("error filtra datos operativos: %s", rec.Body.String())
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

	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != mcpPublicHTTPAllowHeaderV0(http.MethodPost) {
		t.Fatalf("status=%d allow=%s", rec.Code, rec.Header().Get("Allow"))
	}
}

type fakeMCPRunSupervisorHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	ctxErr error
	result MCPRunSupervisorToolResultV0
	err    error
}

func (executor *fakeMCPRunSupervisorHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	executor.ctxErr = ctx.Err()
	if executor.result.Estado == "" {
		executor.result = MCPRunSupervisorToolResultV0{Estado: MCPRunSupervisorEstadoOKV0}
	}
	return executor.result, executor.err
}
