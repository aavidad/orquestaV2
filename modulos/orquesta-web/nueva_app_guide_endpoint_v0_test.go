package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNuevaAppGuideWebEndpointV0GETSirveGuiaEmbebida(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app/guia", nil)

	NewNuevaAppGuideWebEndpointV0().ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	for _, want := range []string{
		`Guia de opciones de nueva app`,
		`Documento completo de uso y contrato visible para el wizard.`,
		`# Guia de opciones del wizard`,
		`Modo Experto`,
		`clean_architecture`,
		`calidad.accesibilidad`,
		`href="/nueva-app"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("guia no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppGuideWebEndpointV0MetodoNoPermitido(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app/guia", nil)

	NewNuevaAppGuideWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != "GET, OPTIONS" {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}
