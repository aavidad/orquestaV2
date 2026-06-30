package orquestaweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		len(out.Session.Form.Integraciones) == 0 ||
		out.AssistantStatus != "" ||
		len(out.Warnings) != 0 {
		t.Fatalf("sesion guiada incompleta: %+v", out)
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

func TestNuevaAppIntakeGuidedHTTPHandlerV0POSTAplicaRespuestaLibreAPreguntaPendiente(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-http-answer", "es", "Portal", "Publicar viviendas")
	if len(session.PendingQuestions) != 6 || session.PendingQuestions[0] != "tipo_app" {
		t.Fatalf("sesion inicial pending=%+v", session.PendingQuestions)
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppIntakeGuidedRequestV0{
		Session: &session,
		Answer:  "web",
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
	if out.Session.Form.TipoApp != "web" ||
		len(out.Session.PendingQuestions) == 0 ||
		out.Session.Estado != WebNuevaAppIntakeEstadoRequiereDatos ||
		out.Session.Handoff.Ready {
		t.Fatalf("respuesta libre no aplicada: %+v", out.Session)
	}
}

func TestNuevaAppIntakeGuidedHTTPHandlerV0UsaAsistenteInyectadoV0(t *testing.T) {
	assistant := fakeNuevaAppIntakeAssistantV0{
		turn: WebNuevaAppIntakeGuidedTurnV0{
			Decisions: []WebNuevaAppIntakeDecisionV0{
				{Field: "nombre", Value: "Inventario clinico"},
				{Field: "tipo_app", Value: "web"},
				{Field: "integraciones.0.tipo", Value: "messaging"},
			},
			Messages: []string{"nueva_app.wizard.guided_msg_analyzed"},
		},
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-http-assistant",
		Locale:    "es",
		Need:      "Necesito coordinar avisos entre equipos clinicos",
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppIntakeGuidedHTTPHandlerWithAssistantV0(assistant).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Session.Form.Nombre != "Inventario clinico" ||
		out.Session.Form.TipoApp != "web" ||
		len(out.Session.Form.Integraciones) == 0 ||
		out.Session.Form.Integraciones[0].Tipo != "messaging" ||
		out.AssistantStatus != WebNuevaAppIntakeAssistantStatusOKV0 ||
		len(out.Warnings) != 0 ||
		out.Turn.Need != "Necesito coordinar avisos entre equipos clinicos" ||
		len(out.Turn.Followups) == 0 {
		t.Fatalf("respuesta asistida inesperada: %+v", out)
	}
}

func TestNuevaAppIntakeGuidedHTTPHandlerV0FallbackSiAsistenteFallaV0(t *testing.T) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-http-assistant-fallback",
		Locale:    "es",
		Need:      "Quiero una app web con mapas",
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppIntakeGuidedHTTPHandlerWithAssistantV0(fakeNuevaAppIntakeAssistantV0{
		err: errors.New("assistant unavailable"),
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !stringSliceHasV0(out.Session.Form.Plataformas, "web") ||
		len(out.Session.Form.Integraciones) == 0 ||
		out.Session.Form.Integraciones[0].Tipo != "maps" ||
		out.AssistantStatus != WebNuevaAppIntakeAssistantStatusFallbackLocalV0 ||
		len(out.Warnings) != 1 ||
		out.Warnings[0].Code != WebNuevaAppIntakeWarningAssistantFallbackLocalV0 ||
		out.Warnings[0].Status != WebNuevaAppIntakeAssistantStatusFallbackLocalV0 {
		t.Fatalf("fallback local no aplicado: %+v", out)
	}
	if strings.Contains(rec.Body.String(), "assistant unavailable") {
		t.Fatalf("fallback filtra error interno: %s", rec.Body.String())
	}
}

func TestNuevaAppIntakeGuidedHTTPHandlerV0RechazaMetodoYContentType(t *testing.T) {
	handler := NewNuevaAppIntakeGuidedHTTPHandlerV0()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v0/apps/intake/guided-turn", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", rec.Code)
	}
	var methodError map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&methodError); err != nil ||
		methodError["error"] != "Metodo no permitido." ||
		methodError["error"] == http.StatusText(http.StatusMethodNotAllowed) {
		t.Fatalf("GET error no localizado: body=%+v err=%v", methodError, err)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/guided-turn", bytes.NewBufferString("need=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form status=%d", rec.Code)
	}
	var contentTypeError map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&contentTypeError); err != nil ||
		contentTypeError["error"] != "El contenido debe enviarse como JSON." ||
		contentTypeError["error"] == http.StatusText(http.StatusUnsupportedMediaType) {
		t.Fatalf("content-type error no localizado: body=%+v err=%v", contentTypeError, err)
	}
}

type fakeNuevaAppIntakeAssistantV0 struct {
	turn WebNuevaAppIntakeGuidedTurnV0
	err  error
}

func (fake fakeNuevaAppIntakeAssistantV0) BuildNuevaAppIntakeGuidedTurnV0(
	context.Context,
	WebNuevaAppIntakeGuidedRequestV0,
	WebNuevaAppIntakeSessionV0,
) (WebNuevaAppIntakeGuidedTurnV0, error) {
	return fake.turn, fake.err
}
