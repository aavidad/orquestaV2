package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutoprogrammingWebEndpointV0RenderizaPantallaOperativa(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/autoprogramming", nil)

	NewAutoprogrammingWebEndpointV0().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, required := range []string{
		"Autoprogramacion",
		"/api/v0/autoprogramming/prepare-run",
		"/api/v0/autoprogramming/status",
		"/api/v0/runs/supervise",
		`href="/ops"`,
		`name="project_ref"`,
		`name="required_tests"`,
	} {
		if rec.Code != http.StatusOK || !strings.Contains(body, required) {
			t.Fatalf("autoprogramming incompleto status=%d falta=%q body=%s", rec.Code, required, body)
		}
	}
	if strings.Contains(strings.ToLower(body), "local_path") {
		t.Fatalf("pantalla no debe pedir paths locales: %s", body)
	}
}

func TestAutoprogrammingWebEndpointV0MetodoNoSoportado(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/autoprogramming", nil)

	NewAutoprogrammingWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed ||
		rec.Header().Get("Allow") != "GET, OPTIONS" {
		t.Fatalf("status=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}
