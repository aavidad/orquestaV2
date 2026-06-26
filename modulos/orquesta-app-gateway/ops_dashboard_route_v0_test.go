package orquestaappgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppGatewayOpsDashboardRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ops", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Orquesta Ops") ||
		!strings.Contains(body, "/api/v0/autoprogramming/status") ||
		!strings.Contains(body, "/api/v0/server/resources") {
		t.Fatalf("ops html incompleto: %s", body)
	}
}

func TestAppGatewayHomeRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, required := range []string{
		"Orquesta",
		`href="/ops"`,
		`href="/nueva-app"`,
		`href="/autoprogramming"`,
		`href="/app-change"`,
		`href="/run-control"`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("home sin %q: %s", required, body)
		}
	}
}

func TestAppGatewayAutoprogrammingPageRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/autoprogramming", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, required := range []string{
		"Autoprogramacion",
		"/api/v0/autoprogramming/prepare-run",
		"/api/v0/autoprogramming/status",
		"/api/v0/autoprogramming/goal/observe",
		"/api/v0/autoprogramming/supervise",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("autoprogramming sin %q: %s", required, body)
		}
	}
}

func TestAppGatewayOpsAgentRuntimeDetailRouteInyectadaV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{
		OpsAgentRuntimeDetail: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v0/ops/agent-runtime-detail" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"estado":"ok"}`))
		}),
		Timeout: time.Second,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/ops/agent-runtime-detail", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"estado":"ok"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
