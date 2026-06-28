package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestServerAutoprogrammingSuperviseHTTPDevuelveAcceptedBackgroundSinColgarV0(t *testing.T) {
	supervisor := newBlockingServerAutoprogrammingSuperviseExecutorV0()
	gateway := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Timeout:       time.Second,
		RunSupervisor: supervisor,
	})
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: gateway,
	})
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}

	body := []byte(`{
		"request_id":"request-ref-server-autop-supervise-background-001",
		"correlation_id":"corr-server-autop-supervise-background-001",
		"queue_ref":"global",
		"idempotency_key":"idem-server-autop-supervise-background-001",
		"max_dispatches":6,
		"max_outbox":12
	}`)
	first, firstElapsed := postServerAutoprogrammingSuperviseForTestV0(t, handler, body)

	if first.Code != http.StatusAccepted || firstElapsed > 3*time.Second {
		t.Fatalf("primera respuesta status=%d elapsed=%s body=%s", first.Code, firstElapsed, first.Body.String())
	}
	if supervisor.callsV0() != 1 {
		t.Fatalf("calls=%d", supervisor.callsV0())
	}
	firstResult := decodeServerAutoprogrammingSuperviseResultForTestV0(t, first)
	if firstResult.StopReason != "accepted_background" ||
		firstResult.Last.Status != "accepted_background" ||
		firstResult.OperationRef == "" ||
		!serverAutoprogrammingSuperviseHasDiagnosticV0(firstResult.Diagnostics, "autoprogramming_supervise_background_accepted") ||
		!serverAutoprogrammingSuperviseStringInSetV0(firstResult.NextActions, "poll_autoprogramming_status") ||
		!serverAutoprogrammingSuperviseStringInSetV0(firstResult.NextActions, "poll_queue_global_status") {
		t.Fatalf("first_result=%+v", firstResult)
	}

	second, secondElapsed := postServerAutoprogrammingSuperviseForTestV0(t, handler, body)
	if second.Code != http.StatusAccepted || secondElapsed > time.Second {
		t.Fatalf("segunda respuesta status=%d elapsed=%s body=%s", second.Code, secondElapsed, second.Body.String())
	}
	if supervisor.callsV0() != 1 {
		t.Fatalf("segunda llamada duplico supervisor calls=%d", supervisor.callsV0())
	}
	secondResult := decodeServerAutoprogrammingSuperviseResultForTestV0(t, second)
	if secondResult.OperationRef != firstResult.OperationRef ||
		!serverAutoprogrammingSuperviseHasDiagnosticV0(secondResult.Diagnostics, "autoprogramming_supervise_operation_already_running") {
		t.Fatalf("second_result=%+v first_operation=%s", secondResult, firstResult.OperationRef)
	}

	supervisor.releaseV0()
	select {
	case <-supervisor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor bloqueante no finalizo tras release")
	}
}

func TestServerAutoprogrammingSuperviseHTTPClienteRealRecibeCuerpoSinColgarV0(t *testing.T) {
	supervisor := newBlockingServerAutoprogrammingSuperviseExecutorV0()
	gateway := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Timeout:       time.Second,
		RunSupervisor: supervisor,
	})
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: gateway,
	})
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{
		"request_id":"request-ref-server-autop-supervise-real-client-001",
		"correlation_id":"corr-server-autop-supervise-real-client-001",
		"queue_ref":"global",
		"idempotency_key":"idem-server-autop-supervise-real-client-001",
		"max_dispatches":6,
		"max_outbox":12
	}`)
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-server-autop-supervise-real-client-001")

	started := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client.Do elapsed=%s err=%v", elapsed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || elapsed > 3*time.Second {
		t.Fatalf("status=%d elapsed=%s", resp.StatusCode, elapsed)
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.StopReason != "accepted_background" ||
		result.Last.Status != "accepted_background" ||
		result.OperationRef == "" ||
		!serverAutoprogrammingSuperviseHasDiagnosticV0(result.Diagnostics, "autoprogramming_supervise_background_accepted") ||
		!serverAutoprogrammingSuperviseStringInSetV0(result.NextActions, "poll_queue_global_status") {
		t.Fatalf("result=%+v", result)
	}

	supervisor.releaseV0()
	select {
	case <-supervisor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor bloqueante no finalizo tras release")
	}
}

func TestServerRunSuperviseHTTPClienteRealRecibeCuerpoSinColgarV0(t *testing.T) {
	supervisor := newBlockingServerAutoprogrammingSuperviseExecutorV0()
	gateway := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Timeout:       time.Second,
		RunSupervisor: supervisor,
	})
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: gateway,
	})
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{
		"request_id":"request-ref-server-run-supervise-real-client-001",
		"correlation_id":"corr-server-run-supervise-real-client-001",
		"run_ref":"run-ref-server-run-supervise-real-client-001",
		"idempotency_key":"idem-server-run-supervise-real-client-001",
		"max_ticks":20
	}`)
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+orquestamcp.MCPRunSupervisorHTTPPathV0,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-server-run-supervise-real-client-001")

	started := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client.Do elapsed=%s err=%v", elapsed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || elapsed > 3*time.Second {
		t.Fatalf("status=%d elapsed=%s", resp.StatusCode, elapsed)
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.StopReason != "accepted_background" ||
		result.Last.Status != "accepted_background" ||
		result.OperationRef == "" ||
		!serverAutoprogrammingSuperviseHasDiagnosticV0(result.Diagnostics, "run_supervisor_background_accepted") ||
		!serverAutoprogrammingSuperviseStringInSetV0(result.NextActions, "poll_queue_global_status") {
		t.Fatalf("result=%+v", result)
	}

	supervisor.releaseV0()
	select {
	case <-supervisor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor bloqueante no finalizo tras release")
	}
}

func TestServerRunQueuePriorityHTTPSetPriorityClienteRealRecibeTimeoutJSONV0(t *testing.T) {
	queue := newBlockingServerRunQueuePriorityExecutorV0()
	gateway := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Timeout:          time.Second,
		RunQueuePriority: queue,
	})
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: gateway,
	})
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{
		"request_id":"request-ref-server-run-queue-priority-timeout-001",
		"correlation_id":"corr-server-run-queue-priority-timeout-001",
		"idempotency_key":"idem-server-run-queue-priority-timeout-001",
		"action":"set_priority",
		"queue_ref":"global",
		"run_ref":"run-ref-server-run-queue-priority-timeout-001",
		"priority_score":90
	}`)
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+orquestamcp.MCPRunQueuePriorityHTTPPathV0,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-server-run-queue-priority-timeout-001")

	started := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client.Do elapsed=%s err=%v", elapsed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGatewayTimeout || elapsed > 3*time.Second {
		t.Fatalf("status=%d elapsed=%s", resp.StatusCode, elapsed)
	}
	var result orquestamcp.MCPRunQueuePriorityToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunQueuePriorityEstadoErrorV0 ||
		result.Action != orquestamcp.MCPRunQueuePriorityActionSetV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_queue_priority_timeout" ||
		resp.Header.Get("X-Correlation-ID") != "corr-server-run-queue-priority-timeout-001" {
		t.Fatalf("result=%+v headers=%v", result, resp.Header)
	}
	if queue.callsV0() != 1 {
		t.Fatalf("calls=%d", queue.callsV0())
	}
	select {
	case <-queue.done:
	case <-time.After(time.Second):
		t.Fatalf("queue executor no recibio cancelacion tras timeout HTTP")
	}
}

func postServerAutoprogrammingSuperviseForTestV0(
	t *testing.T,
	handler http.Handler,
	body []byte,
) (*httptest.ResponseRecorder, time.Duration) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0,
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-server-autop-supervise-background-001")
	started := time.Now()
	handler.ServeHTTP(rec, req)
	return rec, time.Since(started)
}

func decodeServerAutoprogrammingSuperviseResultForTestV0(
	t *testing.T,
	rec *httptest.ResponseRecorder,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode supervise: %v body=%s", err, rec.Body.String())
	}
	return result
}

type blockingServerAutoprogrammingSuperviseExecutorV0 struct {
	mu      sync.Mutex
	calls   int
	release chan struct{}
	done    chan struct{}
}

type blockingServerRunQueuePriorityExecutorV0 struct {
	mu    sync.Mutex
	once  sync.Once
	calls int
	done  chan struct{}
}

func newBlockingServerAutoprogrammingSuperviseExecutorV0() *blockingServerAutoprogrammingSuperviseExecutorV0 {
	return &blockingServerAutoprogrammingSuperviseExecutorV0{
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func newBlockingServerRunQueuePriorityExecutorV0() *blockingServerRunQueuePriorityExecutorV0 {
	return &blockingServerRunQueuePriorityExecutorV0{
		done: make(chan struct{}),
	}
}

func (executor *blockingServerAutoprogrammingSuperviseExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, error) {
	executor.mu.Lock()
	executor.calls++
	executor.mu.Unlock()
	defer close(executor.done)
	<-executor.release
	return orquestamcp.MCPRunSupervisorToolResultV0{
		Estado:     orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef:     strings.TrimSpace(input.RunRef),
		StopReason: "released",
		Ticks:      1,
	}, nil
}

func (executor *blockingServerRunQueuePriorityExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	executor.mu.Lock()
	executor.calls++
	executor.mu.Unlock()
	<-ctx.Done()
	executor.once.Do(func() { close(executor.done) })
	return orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado: orquestamcp.MCPRunQueuePriorityEstadoOKV0,
		Action: strings.TrimSpace(input.Action),
	}, nil
}

func (executor *blockingServerAutoprogrammingSuperviseExecutorV0) callsV0() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.calls
}

func (executor *blockingServerRunQueuePriorityExecutorV0) callsV0() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.calls
}

func (executor *blockingServerAutoprogrammingSuperviseExecutorV0) releaseV0() {
	close(executor.release)
}

func serverAutoprogrammingSuperviseHasDiagnosticV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == code {
			return true
		}
	}
	return false
}

func serverAutoprogrammingSuperviseStringInSetV0(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
