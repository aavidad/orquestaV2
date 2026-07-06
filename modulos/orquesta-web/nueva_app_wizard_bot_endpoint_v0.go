package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	NuevaAppWizardBotEndpointSchemaV0    = "nueva_app_wizard_bot_endpoint.v0"
	WebNuevaAppWizardBotRequestSchemaV0  = "web_nueva_app_wizard_bot_request.v0"
	WebNuevaAppWizardBotResponseSchemaV0 = "web_nueva_app_wizard_bot_response.v0"
)

type WebNuevaAppWizardBotRequestV0 struct {
	SchemaVersion string                      `json:"schema_version,omitempty"`
	SessionID     string                      `json:"session_id,omitempty"`
	SessionRef    string                      `json:"session_ref,omitempty"`
	UserText      string                      `json:"user_text"`
	Locale        string                      `json:"locale,omitempty"`
	Session       *WebNuevaAppIntakeSessionV0 `json:"session,omitempty"`
}

type WebNuevaAppWizardBotResponseV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	Session       WebNuevaAppIntakeSessionV0 `json:"session"`
	Reply         WizardBotReplyV0           `json:"reply"`
}

type NuevaAppWizardBotHTTPHandlerV0 struct {
	Assistant WizardBotLLMAssistPortV0
}

func NewNuevaAppWizardBotHTTPHandlerV0() NuevaAppWizardBotHTTPHandlerV0 {
	return NuevaAppWizardBotHTTPHandlerV0{}
}

func NewNuevaAppWizardBotHTTPHandlerWithAssistantV0(
	assistant WizardBotLLMAssistPortV0,
) NuevaAppWizardBotHTTPHandlerV0 {
	return NuevaAppWizardBotHTTPHandlerV0{Assistant: assistant}
}

func (handler NuevaAppWizardBotHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := validateWebPublicQueryV0(r); err != nil {
		writeNuevaAppWizardBotErrorV0(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		handler.handlePost(w, r)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodPost)
		writeNuevaAppWizardBotErrorV0(w, http.StatusMethodNotAllowed)
	}
}

func (handler NuevaAppWizardBotHTTPHandlerV0) handlePost(w http.ResponseWriter, r *http.Request) {
	if !webControlContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		writeNuevaAppWizardBotErrorV0(w, http.StatusUnsupportedMediaType)
		return
	}
	var request WebNuevaAppWizardBotRequestV0
	if err := decodeWebControlJSONV0(w, r, &request); err != nil {
		writeNuevaAppWizardBotErrorV0(w, http.StatusBadRequest)
		return
	}
	writeNuevaAppWizardBotJSONV0(w, http.StatusOK, handler.NewResponseV0(r.Context(), request))
}

func NewWebNuevaAppWizardBotResponseV0(
	request WebNuevaAppWizardBotRequestV0,
) WebNuevaAppWizardBotResponseV0 {
	return NuevaAppWizardBotHTTPHandlerV0{}.NewResponseV0(context.Background(), request)
}

func (handler NuevaAppWizardBotHTTPHandlerV0) NewResponseV0(
	ctx context.Context,
	request WebNuevaAppWizardBotRequestV0,
) WebNuevaAppWizardBotResponseV0 {
	session := webNuevaAppWizardBotSessionV0(request)
	session, reply := NewWebNuevaAppWizardBotReplyWithLLMV0(ctx, session, WizardBotTurnV0{
		SessionRef: firstNuevaAppValueV0(request.SessionRef, request.SessionID),
		UserText:   request.UserText,
		Locale:     request.Locale,
	}, handler.Assistant)
	return WebNuevaAppWizardBotResponseV0{
		SchemaVersion: WebNuevaAppWizardBotResponseSchemaV0,
		Session:       session,
		Reply:         reply,
	}
}

func webNuevaAppWizardBotSessionV0(request WebNuevaAppWizardBotRequestV0) WebNuevaAppIntakeSessionV0 {
	sessionRef := strings.TrimSpace(firstNuevaAppValueV0(request.SessionRef, request.SessionID))
	if request.Session != nil {
		session := *request.Session
		if strings.TrimSpace(session.SessionRef) == "" {
			session.SessionRef = sessionRef
		}
		if strings.TrimSpace(session.SessionID) == "" {
			session.SessionID = sessionRef
		}
		if strings.TrimSpace(session.Locale) == "" {
			session.Locale = strings.TrimSpace(request.Locale)
		}
		if strings.TrimSpace(request.Locale) != "" {
			session.Locale = strings.TrimSpace(request.Locale)
			session.Form.Locale = strings.TrimSpace(request.Locale)
		}
		return refreshWebNuevaAppIntakeSessionV0(session)
	}
	return NewWebNuevaAppIntakeSessionV0(sessionRef, request.Locale, "", "")
}

func writeNuevaAppWizardBotErrorV0(w http.ResponseWriter, status int) {
	code, message := nuevaAppIntakeGuidedErrorMessageV0(status)
	writeNuevaAppWizardBotJSONV0(w, status, map[string]string{
		"schema_version": NuevaAppWizardBotEndpointSchemaV0,
		"code":           code,
		"error":          message,
	})
}

func writeNuevaAppWizardBotJSONV0(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
