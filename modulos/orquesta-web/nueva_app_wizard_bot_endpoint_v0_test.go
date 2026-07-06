package orquestaweb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNuevaAppWizardBotHTTPHandlerV0POSTRespondeYDevuelveSesion(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-http-wizard-bot-1", "quiero una app para una agenda")
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppWizardBotRequestV0{
		SessionRef: session.SessionRef,
		Locale:     "es-ES",
		UserText:   "que es CalDAV?",
		Session:    &session,
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/wizard-bot", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppWizardBotHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppWizardBotResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.SchemaVersion != WebNuevaAppWizardBotResponseSchemaV0 ||
		out.Reply.SchemaVersion != WizardBotReplySchemaV0 ||
		out.Session.SchemaVersion != WebNuevaAppIntakeSessionSchemaV0 ||
		out.Session.SessionRef != session.SessionRef ||
		!strings.Contains(out.Reply.Say, "CalDAV") ||
		len(out.Reply.GroundingRefs) == 0 {
		t.Fatalf("respuesta bot incompleta: %+v", out)
	}
}

func TestNuevaAppWizardBotHTTPHandlerV0AplicaSlotFilling(t *testing.T) {
	session := wizardSessionWithObjectiveV0("session-http-wizard-bot-slot", "quiero una app para una agenda")
	session.Form.Nombre = "Agenda"
	session = refreshWebNuevaAppIntakeSessionV0(session)
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(WebNuevaAppWizardBotRequestV0{
		SessionRef: session.SessionRef,
		Locale:     "es-ES",
		UserText:   "1",
		Session:    &session,
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/wizard-bot", &body)
	req.Header.Set("Content-Type", "application/json")

	NewNuevaAppWizardBotHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out WebNuevaAppWizardBotResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Reply.FilledAnswers) != 1 ||
		out.Reply.FilledAnswers[0].QuestionRef != "wizard-q-tipo-app" ||
		out.Session.Form.TipoApp == "" {
		t.Fatalf("slot filling no aplicado: %+v", out)
	}
}

func TestNuevaAppWizardBotHTTPHandlerV0RechazaMetodoYContentType(t *testing.T) {
	handler := NewNuevaAppWizardBotHTTPHandlerV0()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v0/apps/intake/wizard-bot", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/intake/wizard-bot", bytes.NewBufferString("user_text=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content-type status=%d body=%s", rec.Code, rec.Body.String())
	}
}
