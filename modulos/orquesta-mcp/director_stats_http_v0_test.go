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

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPDirectorStatsHTTPHandlerV0OKConExecutorFake(t *testing.T) {
	executor := mcpDirectorStatsHTTPFakeExecutorV0{
		Result: MCPDirectorStatsToolResultV0{
			Estado:        MCPDirectorStatsEstadoOKV0,
			RequestID:     "request-ref-director-stats-http-001",
			CorrelationID: "corr-director-stats-http-result-001",
			RunRef:        "run-ref-director-stats-http-001",
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-director-stats-http-001",
		CorrelationID: "corr-director-stats-http-input-001",
		RunRef:        "run-ref-director-stats-http-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-director-stats-http-header-001")

	NewMCPDirectorStatsHTTPHandlerV0(&executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Correlation-ID"); got != "corr-director-stats-http-result-001" {
		t.Fatalf("correlation header=%q", got)
	}
	if executor.Input.RunRef != "run-ref-director-stats-http-001" ||
		executor.Input.CorrelationID != "corr-director-stats-http-input-001" {
		t.Fatalf("input recibido=%+v", executor.Input)
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != "run-ref-director-stats-http-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsHTTPHandlerV0OKConExecutorRealInMemory(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-ref-director-stats-http-real-001")
	executor := MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-director-stats-http-real-001",
		CorrelationID: "corr-director-stats-http-real-001",
		RunRef:        run.RunID,
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, body)

	NewMCPDirectorStatsHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Correlation-ID"); got != "corr-director-stats-http-real-001" {
		t.Fatalf("correlation header=%q", got)
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.Stats == nil ||
		result.Stats.Counts.TasksTotal != 3 ||
		!result.Stats.Closure.Blocked {
		t.Fatalf("result=%+v", result)
	}
	if !containsStringMCPTestV0(result.Stats.Closure.BlockedBy, orquestacionnucleoapp.DirectorClosureBlockedByRevisionFinalV0) {
		t.Fatalf("closure no expone bloqueo de cierre para web: %+v", result.Stats.Closure)
	}
}

func TestMCPDirectorStatsHTTPHandlerV0MetodoIncorrecto(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, MCPDirectorStatsHTTPPathV0, nil)
	req.Header.Set("X-Correlation-ID", "corr-director-stats-http-method-001")

	NewMCPDirectorStatsHTTPHandlerV0(&mcpDirectorStatsHTTPFakeExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed ||
		rec.Header().Get("Allow") != http.MethodPost ||
		rec.Header().Get("X-Correlation-ID") != "corr-director-stats-http-method-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestMCPDirectorStatsHTTPHandlerV0ExecutorNil(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, bytes.NewBufferString(`{}`))
	req.Header.Set("X-Correlation-ID", "corr-director-stats-http-nil-001")

	NewMCPDirectorStatsHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable ||
		rec.Header().Get("X-Correlation-ID") != "corr-director-stats-http-nil-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestMCPDirectorStatsHTTPHandlerV0ExponeCausaSanitizadaSiExecutorFalla(t *testing.T) {
	executor := mcpDirectorStatsHTTPFakeExecutorV0{
		Err: errors.New("director stats failed at /root/Trabajo/orquesta token=secret123456 run-ref-director-error-001"),
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-director-error-001"}`)
	req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPDirectorStatsHTTPHandlerV0(&executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		!strings.Contains(result.Errores[0].Message, "director_stats_executor_error") ||
		!strings.Contains(result.Errores[0].Message, "run-ref-director-error-001") {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", rec.Body.String())
	}
}

func TestMCPDirectorStatsTransportV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	executor := &mcpDirectorStatsHTTPFakeExecutorV0{
		Err: errors.New("director stats failed at /root/Trabajo/orquesta token=secret123456 run-ref-director-transport-error-001"),
	}
	raw, err := json.Marshal(MCPDirectorStatsToolInputV0{RunRef: "run-ref-director-transport-error-001"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	output, err := mcpDirectorStatsTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("transport no debe romper JSON-RPC: %v", err)
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.Estado != MCPDirectorStatsEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		!strings.Contains(result.Errores[0].Message, "director_stats_executor_error") ||
		!strings.Contains(result.Errores[0].Message, "run-ref-director-transport-error-001") {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(string(output), "/root/Trabajo") ||
		strings.Contains(string(output), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", string(output))
	}
}

func TestMCPDirectorStatsHTTPHandlerV0ErrorValidacionRunRef(t *testing.T) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPDirectorStatsToolInputV0{
		RequestID:     "request-ref-director-stats-http-invalid-001",
		CorrelationID: "corr-director-stats-http-invalid-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, body)

	NewMCPDirectorStatsHTTPHandlerV0(MCPDirectorStatsToolExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" ||
		rec.Header().Get("X-Correlation-ID") != "corr-director-stats-http-invalid-001" {
		t.Fatalf("result=%+v headers=%v", result, rec.Header())
	}
}

func TestMCPDirectorStatsHTTPHandlerV0BodyInvalidoYPathIncorrecto(t *testing.T) {
	t.Run("body invalido", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, MCPDirectorStatsHTTPPathV0, bytes.NewBufferString(`{`))
		req.Header.Set("X-Correlation-ID", "corr-director-stats-http-body-001")

		NewMCPDirectorStatsHTTPHandlerV0(&mcpDirectorStatsHTTPFakeExecutorV0{}).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest ||
			rec.Header().Get("X-Correlation-ID") != "corr-director-stats-http-body-001" {
			t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
		}
	})

	t.Run("path incorrecto", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v0/director/stats/nope", nil)
		req.Header.Set("X-Correlation-ID", "corr-director-stats-http-path-001")

		NewMCPDirectorStatsHTTPHandlerV0(&mcpDirectorStatsHTTPFakeExecutorV0{}).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound ||
			rec.Header().Get("X-Correlation-ID") != "corr-director-stats-http-path-001" {
			t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
		}
	})
}

type mcpDirectorStatsHTTPFakeExecutorV0 struct {
	Input  MCPDirectorStatsToolInputV0
	Result MCPDirectorStatsToolResultV0
	Err    error
}

func (executor *mcpDirectorStatsHTTPFakeExecutorV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	executor.Input = input
	return executor.Result, executor.Err
}
