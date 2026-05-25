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

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:        MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted:      true,
			RunRef:        "run-autoprogramming-001",
			WaitAgentRefs: []string{"agent-request-001"},
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-autoprogramming-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RequestID != "request-autoprogramming-001" {
		t.Fatalf("input=%+v", executor.input)
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.RunRef != "run-autoprogramming-001" || len(result.WaitAgentRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0SoloPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPAutoprogrammingPrepareRunHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(&fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0ExponeCausaSanitizadaSiExecutorFalla(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		err: errors.New("prepare failed at /root/Trabajo/orquesta token=secret123456 request-ref-prepare-error-001"),
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{"request_id":"request-ref-prepare-error-001"}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0 ||
		!strings.Contains(result.Errores[0].Message, "prepare failed") ||
		!strings.Contains(result.Errores[0].Message, "request-ref-prepare-error-001") {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", rec.Body.String())
	}
}

func TestMCPAutoprogrammingPrepareRunTransportV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		err: errors.New("prepare failed at /root/Trabajo/orquesta token=secret123456 request-ref-prepare-transport-error-001"),
	}
	raw, err := json.Marshal(MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-ref-prepare-transport-error-001",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	output, err := mcpAutoprogrammingPrepareRunTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("transport no debe romper JSON-RPC: %v", err)
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_prepare_run_executor_error" ||
		!strings.Contains(result.Errores[0].Message, "prepare failed") ||
		!strings.Contains(result.Errores[0].Message, "request-ref-prepare-transport-error-001") {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(string(output), "/root/Trabajo") ||
		strings.Contains(string(output), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", string(output))
	}
}

type fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0 struct {
	input  MCPAutoprogrammingPrepareRunToolInputV0
	result MCPAutoprogrammingPrepareRunToolResultV0
	err    error
}

func (executor *fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingPrepareRunToolInputV0,
) (MCPAutoprogrammingPrepareRunToolResultV0, error) {
	_ = ctx
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingPrepareRunToolResultV0{Estado: MCPAutoprogrammingPrepareRunEstadoOKV0}
	}
	return executor.result, executor.err
}
