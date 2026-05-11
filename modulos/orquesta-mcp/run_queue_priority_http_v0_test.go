package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRunQueuePriorityHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPRunQueuePriorityHTTPExecutorV0{
		result: MCPRunQueuePriorityToolResultV0{
			Estado:        MCPRunQueuePriorityEstadoOKV0,
			CorrelationID: "corr-run-queue-http-result-001",
			Action:        "rank",
			Count:         1,
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunQueuePriorityToolInputV0{
		Action:        "rank",
		QueueRef:      "global",
		CorrelationID: "corr-run-queue-http-input-001",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunQueuePriorityHTTPPathV0, body)

	NewMCPRunQueuePriorityHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-queue-http-result-001" ||
		executor.input.QueueRef != "global" {
		t.Fatalf("status=%d headers=%v input=%+v body=%s", rec.Code, rec.Header(), executor.input, rec.Body.String())
	}
}

func TestMCPRunQueuePriorityHTTPHandlerV0ExecutorNil(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunQueuePriorityHTTPPathV0, bytes.NewBufferString(`{}`))
	req.Header.Set("X-Correlation-ID", "corr-run-queue-http-nil-001")

	NewMCPRunQueuePriorityHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-queue-http-nil-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

type fakeMCPRunQueuePriorityHTTPExecutorV0 struct {
	input  MCPRunQueuePriorityToolInputV0
	result MCPRunQueuePriorityToolResultV0
}

func (executor *fakeMCPRunQueuePriorityHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
