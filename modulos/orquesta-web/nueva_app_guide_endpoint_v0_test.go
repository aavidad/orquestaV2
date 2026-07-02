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
		`<article><h2 id="guia-de-opciones-del-wizard-nueva-app-2026-06-25">Guia de opciones del wizard`,
		`Modo Experto`,
		`clean_architecture`,
		`id="preferencias-tecnicas-arquitectura"`,
		`id="modo-basico-datos-db-required"`,
		`id="deploy-target"`,
		`calidad.accesibilidad`,
		`<table>`,
		`<th>Patron</th>`,
		`<td><code>hexagonal</code></td>`,
		`Campos Contractuales Visibles Que No Deben Perderse`,
		`El HTML actual muestra seis filas`,
		`El HTML actual muestra seis preferencias de almacenamiento`,
		`hasta seis integraciones visibles`,
		`href="/nueva-app"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("guia no contiene %q\n%s", want, body)
		}
	}
	for _, stale := range []string{
		`Campo contractual no visible en HTML actual.`,
		`Campos Contractuales No Visibles En HTML Basico`,
		`hasta cuatro integraciones visibles`,
		`El HTML actual muestra cuatro filas`,
		`El HTML actual muestra cuatro preferencias de almacenamiento`,
	} {
		if strings.Contains(body, stale) {
			t.Fatalf("guia conserva texto stale %q\n%s", stale, body)
		}
	}
	if strings.Contains(body, `<pre># Guia de opciones del wizard`) {
		t.Fatalf("guia vuelve a servir markdown completo en pre\n%s", body)
	}
}

func TestNuevaAppGuideMarkdownToHTMLV0RenderizaTablasV0(t *testing.T) {
	html := string(nuevaAppGuideMarkdownToHTMLV0(`
| Campo | Uso |
| --- | --- |
| ` + "`api`" + ` | Integracion |
`))

	for _, want := range []string{
		`<table>`,
		`<thead><tr><th>Campo</th><th>Uso</th></tr></thead>`,
		`<tbody>`,
		`<tr><td><code>api</code></td><td>Integracion</td></tr>`,
		`</tbody>`,
		`</table>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("tabla markdown no renderizada como HTML semantico; falta %q en %s", want, html)
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
