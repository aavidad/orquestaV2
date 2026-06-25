package orquestaweb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNuevaAppIntakeGuidedHTTPHandlerV0POSTGeneraSesionGuiada(t *testing.T) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-http-1",
		Locale:    "es",
		Need:      "Quiero una app para movil que muestre pisos en alquiler cercanos con OpenStreetMap",
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppIntakeGuidedHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.SchemaVersion != WebNuevaAppIntakeGuidedResponseSchemaV0 ||
		out.Turn.SchemaVersion != WebNuevaAppIntakeGuidedTurnSchemaV0 ||
		out.Session.SchemaVersion != WebNuevaAppIntakeSessionSchemaV0 {
		t.Fatalf("schemas invalidas: %+v", out)
	}
	if out.Session.Form.TipoApp != "mobile" ||
		!stringSliceHasV0(out.Session.Form.Plataformas, "mobile") ||
		len(out.Session.Form.Datos.TiposDetallados) == 0 ||
		len(out.Session.Form.Datos.Storage) == 0 ||
		len(out.Session.Form.Integraciones) == 0 {
		t.Fatalf("sesion guiada incompleta: %+v", out.Session.Form)
	}
}

func TestNuevaAppIntakeGuidedHTTPHandlerV0POSTAplicaAccionSobreSesion(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-http-2", "es", "Portal", "Publicar viviendas")
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppIntakeGuidedRequestV0{
		ActionID: "architecture_event",
		Session:  &session,
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppIntakeGuidedHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Session.Form.PreferenciasTecnicas.Arquitectura != "event_driven" {
		t.Fatalf("arquitectura=%q", out.Session.Form.PreferenciasTecnicas.Arquitectura)
	}
}

func TestNuevaAppIntakeGuidedHTTPHandlerV0RechazaMetodoYContentType(t *testing.T) {
	handler := NewNuevaAppIntakeGuidedHTTPHandlerV0()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v0/apps/intake/guided-turn", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", bytes.NewBufferString("need=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form status=%d", rec.Code)
	}
}
