package orquestaappgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAppGatewayBrowserMutationIntentGuardV0BloqueaOriginCruzado(t *testing.T) {
	control := &recordingRunControlExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunControl: control,
		Timeout:    time.Second,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{
		"request_id":"request-ref-origin-cross-001",
		"action":"pause",
		"run_ref":"run-origin-cross-001"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || control.Input.Action != "" {
		t.Fatalf("status=%d input=%+v body=%s", rec.Code, control.Input, rec.Body.String())
	}
}

func TestAppGatewayBrowserMutationIntentGuardV0BloqueaSupervisorConOriginCruzado(t *testing.T) {
	supervisor := &recordingRunSupervisorExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunSupervisor: supervisor,
		Timeout:       time.Second,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", strings.NewReader(`{
		"request_id":"request-ref-origin-cross-supervise-001",
		"run_ref":"run-origin-cross-supervise-001",
		"max_ticks":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || supervisor.Input.RunRef != "" {
		t.Fatalf("status=%d input=%+v body=%s", rec.Code, supervisor.Input, rec.Body.String())
	}
}

func TestAppGatewayBrowserMutationIntentGuardV0PermiteOriginSameOrigin(t *testing.T) {
	control := &recordingRunControlExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunControl: control,
		Timeout:    time.Second,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{
		"request_id":"request-ref-origin-same-001",
		"action":"pause",
		"run_ref":"run-origin-same-001"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		control.Input.Action != "pause" ||
		rec.Header().Get("X-Orquesta-Browser-Intent-Decision") != "origin_allowed" {
		t.Fatalf("status=%d input=%+v headers=%v body=%s", rec.Code, control.Input, rec.Header(), rec.Body.String())
	}
}

var _ orquestamcp.MCPTransportRunControlExecutorV0 = (*recordingRunControlExecutorV0)(nil)
var _ orquestamcp.MCPTransportRunSupervisorExecutorV0 = (*recordingRunSupervisorExecutorV0)(nil)
