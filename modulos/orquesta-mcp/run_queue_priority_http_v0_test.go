package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestMCPRunQueuePriorityHTTPHandlerV0RankTimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &fakeMCPRunQueuePriorityHTTPExecutorV0{waitForCancel: true}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunQueuePriorityHTTPPathV0, bytes.NewBufferString(`{"queue_ref":"global"}`))
	req.Header.Set("X-Correlation-ID", "corr-run-queue-timeout-001")

	newMCPRunQueuePriorityHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-queue-timeout-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPRunQueuePriorityToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if result.Estado != MCPRunQueuePriorityEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_queue_priority_timeout" ||
		result.Errores[0].Field != "executor" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunQueuePriorityHTTPHandlerV0SetPriorityClienteRealRecibeTimeoutJSON(t *testing.T) {
	executor := &fakeMCPRunQueuePriorityHTTPExecutorV0{waitForCancel: true}
	handler := newMCPRunQueuePriorityHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	server := httptest.NewServer(handler)
	defer server.Close()

	body := bytes.NewBufferString(`{
		"request_id":"request-ref-run-queue-priority-timeout-set-001",
		"correlation_id":"corr-run-queue-priority-timeout-set-001",
		"idempotency_key":"idem-run-queue-priority-timeout-set-001",
		"action":"set_priority",
		"queue_ref":"global",
		"run_ref":"run-ref-run-queue-priority-timeout-set-001",
		"priority_score":80
	}`)
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+MCPRunQueuePriorityHTTPPathV0,
		body,
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-run-queue-priority-timeout-set-001")

	client := &http.Client{Timeout: time.Second}
	started := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client.Do elapsed=%s err=%v", elapsed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGatewayTimeout || elapsed > time.Second {
		t.Fatalf("status=%d elapsed=%s", resp.StatusCode, elapsed)
	}
	var result MCPRunQueuePriorityToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunQueuePriorityEstadoErrorV0 ||
		result.Action != MCPRunQueuePriorityActionSetV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_queue_priority_timeout" ||
		resp.Header.Get("X-Correlation-ID") != "corr-run-queue-priority-timeout-set-001" {
		t.Fatalf("result=%+v headers=%v", result, resp.Header)
	}
}

type fakeMCPRunQueuePriorityHTTPExecutorV0 struct {
	input         MCPRunQueuePriorityToolInputV0
	result        MCPRunQueuePriorityToolResultV0
	waitForCancel bool
}

func (executor *fakeMCPRunQueuePriorityHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	executor.input = input
	if executor.waitForCancel {
		<-ctx.Done()
	}
	return executor.result, nil
}
