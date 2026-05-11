package orquestaweb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNuevaAppWebEndpointV0POSTErrorClienteDevuelveErrorPublicoLocalizado(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		err: WebNuevaAppClientErrorV0{Code: WebNuevaAppErrTransporteV0, StatusCode: http.StatusBadGateway},
	}
	endpoint := NewNuevaAppWebEndpointV0(client)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(WebNuevaAppFormV0{
		RequestID: "req-error-006",
		Locale:    "en-US",
		Nombre:    "Agenda",
		Objetivo:  "Coordinate rehearsals",
		TipoApp:   "web",
	}); err != nil {
		t.Fatalf("encode form: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", body)
	req.Header.Set("Content-Type", "application/json")

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	if page.ViewModel.Estado != WebNuevaAppEstadoError ||
		page.ViewModel.ErroresPublicos[0].Code != WebNuevaAppErrTransporteV0 ||
		page.Textos.ErroresPublicos[WebNuevaAppErrTransporteV0] != "The communication could not be completed." {
		t.Fatalf("error publico inesperado: %+v", page)
	}
}

func TestNuevaAppWebEndpointV0MetodoNoSoportadoNoDelega(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	endpoint := NewNuevaAppWebEndpointV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/nueva-app?locale=es", nil)

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
	if client.calls != 0 {
		t.Fatalf("metodo no soportado no debe delegar: calls=%d", client.calls)
	}
	if page.ViewModel.Estado != WebNuevaAppEstadoError ||
		page.ViewModel.ErroresPublicos[0].Code != WebNuevaAppErrMetodoNoSoportadoV0 {
		t.Fatalf("page metodo no soportado: %+v", page)
	}
}
