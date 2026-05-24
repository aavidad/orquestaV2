package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, WebOpsDashboardPageEndpointV0, nil)

	NewOpsDashboardWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Orquesta Ops",
		"/api/v0/server/status",
		"/api/v0/server/resources",
		"/api/v0/autoprogramming/status",
		"/api/v0/director/stats",
		"refreshMs = 1000",
		"Proyectos / runs",
		"Agentes",
		"Cola",
		"Memoria proceso",
		"Disco peor uso",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("html no contiene %q", want)
		}
	}
}

func TestOpsDashboardWebEndpointV0SoloGET(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, WebOpsDashboardPageEndpointV0, nil)

	NewOpsDashboardWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}
