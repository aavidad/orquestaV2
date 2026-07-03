package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsKanbanWebEndpointV0RenderizaPanelReadOnly(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, WebOpsKanbanPageEndpointV0, nil)

	NewOpsKanbanWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Kanban de observacion",
		"proyeccion read-only",
		"no_new_source_of_truth",
		"/api/v0/autoprogramming/status",
		"/api/v0/queue/global-status",
		"safe_actions_read_only",
		"lane-ready",
		"lane-running",
		"lane-waiting",
		"lane-attention",
		"lane-closure",
		"lane-done",
		"function buildCards",
		"function deriveLane",
		"Esta pantalla es solo una proyeccion read-only",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("html no contiene %q", want)
		}
	}
	if strings.Contains(body, "drag") ||
		strings.Contains(body, "drop") ||
		strings.Contains(body, "localStorage") ||
		strings.Contains(body, "PUT ") ||
		strings.Contains(body, "PATCH ") ||
		strings.Contains(body, "DELETE ") {
		t.Fatalf("kanban no debe exponer mutacion ni persistencia cliente: %s", body)
	}
}

func TestOpsKanbanWebEndpointV0NoCapturaRutasDesconocidas(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ops/kanban/otra", nil)

	NewOpsKanbanWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpsKanbanWebEndpointV0MetodoNoSoportado(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, WebOpsKanbanPageEndpointV0, nil)

	NewOpsKanbanWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed ||
		rec.Header().Get("Allow") != "GET, OPTIONS" {
		t.Fatalf("status=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}
