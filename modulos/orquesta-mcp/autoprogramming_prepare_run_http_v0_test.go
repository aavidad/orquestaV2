package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

type fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0 struct {
	input  MCPAutoprogrammingPrepareRunToolInputV0
	result MCPAutoprogrammingPrepareRunToolResultV0
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
	return executor.result, nil
}
