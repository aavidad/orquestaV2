package orquestaappgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingStatusAPIRouteV0(t *testing.T) {
	queue := &recordingAutoprogrammingStatusQueueExecutorV0{}
	stats := &recordingAutoprogrammingStatusRunExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:          time.Second,
		RunQueuePriority: queue,
		DirectorStats:    stats,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/status",
		strings.NewReader(`{"run_ref":"run-ref-app-gateway-autop-status-001"}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if queue.Input.Action != orquestamcp.MCPRunQueuePriorityActionRankV0 ||
		stats.Input.RunRef != "run-ref-app-gateway-autop-status-001" {
		t.Fatalf("queue=%+v stats=%+v", queue.Input, stats.Input)
	}
}

func TestAutoprogrammingStatusAPIRouteV0DevuelveTimeoutJSONSinColgar(t *testing.T) {
	queue := &blockingAutoprogrammingStatusQueueExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:          time.Second,
		RunQueuePriority: queue,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/status",
		strings.NewReader(`{}`),
	)

	started := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(started)

	if rec.Code != http.StatusGatewayTimeout || elapsed > 3*time.Second {
		t.Fatalf("status=%d elapsed=%s body=%s", rec.Code, elapsed, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_status_timeout" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingSuperviseAPIRouteV0(t *testing.T) {
	supervisor := &recordingAutoprogrammingSuperviseExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:       time.Second,
		RunSupervisor: supervisor,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/supervise",
		strings.NewReader(`{"run_ref":"run-ref-app-gateway-autop-supervise-001","max_ticks":1}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if supervisor.Input.RunRef != "run-ref-app-gateway-autop-supervise-001" ||
		supervisor.Input.MaxTicks != 1 {
		t.Fatalf("input=%+v", supervisor.Input)
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != "run-ref-app-gateway-autop-supervise-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingObserveActiveGoalsAPIRouteV0(t *testing.T) {
	executor := &recordingAutoprogrammingObserveActiveGoalsExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:                           time.Second,
		AutoprogrammingObserveActiveGoals: executor,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/goals/observe-active",
		strings.NewReader(`{"request_id":"request-ref-app-gateway-observe-active-goals-001","max_items":3}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingObserveActiveGoalsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if executor.Input.MaxItems != 3 ||
		result.Estado != orquestamcp.MCPAutoprogrammingObserveActiveGoalsEstadoOKV0 {
		t.Fatalf("input=%+v result=%+v", executor.Input, result)
	}
}

func TestAutoprogrammingSuperviseAPIRouteV0DevuelveAcceptedBackgroundSinColgar(t *testing.T) {
	supervisor := newBlockingAutoprogrammingSuperviseExecutorV0()
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:       time.Second,
		RunSupervisor: supervisor,
	})
	body := `{
		"request_id":"request-ref-app-gateway-autop-supervise-background-001",
		"run_ref":"run-ref-app-gateway-autop-supervise-background-001",
		"idempotency_key":"idem-app-gateway-autop-supervise-background-001"
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/supervise",
		strings.NewReader(body),
	)

	started := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(started)

	if rec.Code != http.StatusAccepted || elapsed > 3*time.Second {
		t.Fatalf("status=%d elapsed=%s body=%s", rec.Code, elapsed, rec.Body.String())
	}
	if supervisor.callsV0() != 1 {
		t.Fatalf("calls=%d", supervisor.callsV0())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Last.Status != "accepted_background" ||
		result.OperationRef == "" {
		t.Fatalf("result=%+v", result)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/supervise",
		strings.NewReader(body),
	)
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusAccepted || supervisor.callsV0() != 1 {
		t.Fatalf("segunda llamada status=%d calls=%d body=%s", rec2.Code, supervisor.callsV0(), rec2.Body.String())
	}
	supervisor.releaseV0()
	select {
	case <-supervisor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor bloqueante no finalizo tras release")
	}
}

type recordingAutoprogrammingStatusQueueExecutorV0 struct {
	Input orquestamcp.MCPRunQueuePriorityToolInputV0
}

func (executor *recordingAutoprogrammingStatusQueueExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado: orquestamcp.MCPRunQueuePriorityEstadoOKV0,
		Action: orquestamcp.MCPRunQueuePriorityActionRankV0,
		Count:  0,
	}, nil
}

type blockingAutoprogrammingStatusQueueExecutorV0 struct{}

func (executor *blockingAutoprogrammingStatusQueueExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	<-ctx.Done()
	return orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado:        orquestamcp.MCPRunQueuePriorityEstadoErrorV0,
		CorrelationID: input.CorrelationID,
	}, ctx.Err()
}

type recordingAutoprogrammingStatusRunExecutorV0 struct {
	Input orquestamcp.MCPDirectorStatsToolInputV0
}

func (executor *recordingAutoprogrammingStatusRunExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDirectorStatsToolInputV0,
) (orquestamcp.MCPDirectorStatsToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPDirectorStatsToolResultV0{
		Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

type recordingAutoprogrammingSuperviseExecutorV0 struct {
	Input orquestamcp.MCPRunSupervisorToolInputV0
}

func (executor *recordingAutoprogrammingSuperviseExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRunSupervisorToolResultV0{
		Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef: input.RunRef,
		Ticks:  input.MaxTicks,
	}, nil
}

type recordingAutoprogrammingObserveActiveGoalsExecutorV0 struct {
	Input orquestamcp.MCPAutoprogrammingObserveActiveGoalsToolInputV0
}

func (executor *recordingAutoprogrammingObserveActiveGoalsExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) (orquestamcp.MCPAutoprogrammingObserveActiveGoalsToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPAutoprogrammingObserveActiveGoalsToolResultV0{
		Estado: orquestamcp.MCPAutoprogrammingObserveActiveGoalsEstadoOKV0,
	}, nil
}

type blockingAutoprogrammingSuperviseExecutorV0 struct {
	mu      sync.Mutex
	calls   int
	release chan struct{}
	done    chan struct{}
}

func newBlockingAutoprogrammingSuperviseExecutorV0() *blockingAutoprogrammingSuperviseExecutorV0 {
	return &blockingAutoprogrammingSuperviseExecutorV0{
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (executor *blockingAutoprogrammingSuperviseExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, error) {
	executor.mu.Lock()
	executor.calls++
	executor.mu.Unlock()
	defer close(executor.done)
	<-executor.release
	return orquestamcp.MCPRunSupervisorToolResultV0{
		Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef: input.RunRef,
		Ticks:  1,
	}, nil
}

func (executor *blockingAutoprogrammingSuperviseExecutorV0) callsV0() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.calls
}

func (executor *blockingAutoprogrammingSuperviseExecutorV0) releaseV0() {
	close(executor.release)
}
