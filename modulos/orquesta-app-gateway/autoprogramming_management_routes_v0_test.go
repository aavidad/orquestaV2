package orquestaappgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
