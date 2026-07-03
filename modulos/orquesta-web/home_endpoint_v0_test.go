package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeWebEndpointV0RenderizaConsolaOperativa(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	NewHomeWebEndpointV0().ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK ||
		!strings.Contains(body, "Orquesta") ||
		!strings.Contains(body, `href="/ops"`) ||
		!strings.Contains(body, `href="/ops/kanban"`) ||
		!strings.Contains(body, `href="/nueva-app"`) ||
		!strings.Contains(body, `href="/autoprogramming"`) ||
		!strings.Contains(body, `href="/app-change"`) ||
		!strings.Contains(body, `href="/run-control"`) {
		t.Fatalf("home incompleta status=%d body=%s", rec.Code, body)
	}
	if rec.Header().Get("Content-Type") != WebHTMLContentTypeHeaderV0 {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
}

func TestHomeWebEndpointV0NoCapturaRutasDesconocidas(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ruta-inexistente", nil)

	NewHomeWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHomeWebEndpointV0MetodoNoSoportado(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	NewHomeWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed ||
		rec.Header().Get("Allow") != "GET, OPTIONS" {
		t.Fatalf("status=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}
