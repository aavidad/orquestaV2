package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	NuevaAppIntakeGuidedEndpointSchemaV0    = "nueva_app_intake_guided_endpoint.v0"
	WebNuevaAppIntakeGuidedRequestSchemaV0  = "web_nueva_app_intake_guided_request.v0"
	WebNuevaAppIntakeGuidedResponseSchemaV0 = "web_nueva_app_intake_guided_response.v0"
)

type WebNuevaAppIntakeGuidedRequestV0 struct {
	SchemaVersion string                      `json:"schema_version,omitempty"`
	SessionID     string                      `json:"session_id,omitempty"`
	Locale        string                      `json:"locale,omitempty"`
	Nombre        string                      `json:"nombre,omitempty"`
	Idea          string                      `json:"idea,omitempty"`
	Need          string                      `json:"need,omitempty"`
	ActionID      string                      `json:"action_id,omitempty"`
	ActionIDs     []string                    `json:"action_ids,omitempty"`
	AnswerField   string                      `json:"answer_field,omitempty"`
	Answer        string                      `json:"answer,omitempty"`
	Session       *WebNuevaAppIntakeSessionV0 `json:"session,omitempty"`
}

type WebNuevaAppIntakeGuidedResponseV0 struct {
	SchemaVersion string                        `json:"schema_version"`
	Turn          WebNuevaAppIntakeGuidedTurnV0 `json:"turn"`
	Session       WebNuevaAppIntakeSessionV0    `json:"session"`
}

type WebNuevaAppIntakeAssistantPortV0 interface {
	BuildNuevaAppIntakeGuidedTurnV0(
		context.Context,
		WebNuevaAppIntakeGuidedRequestV0,
		WebNuevaAppIntakeSessionV0,
	) (WebNuevaAppIntakeGuidedTurnV0, error)
}

type NuevaAppIntakeGuidedHTTPHandlerV0 struct {
	Assistant WebNuevaAppIntakeAssistantPortV0
}

func NewNuevaAppIntakeGuidedHTTPHandlerV0() NuevaAppIntakeGuidedHTTPHandlerV0 {
	return NuevaAppIntakeGuidedHTTPHandlerV0{}
}

func NewNuevaAppIntakeGuidedHTTPHandlerWithAssistantV0(
	assistant WebNuevaAppIntakeAssistantPortV0,
) NuevaAppIntakeGuidedHTTPHandlerV0 {
	return NuevaAppIntakeGuidedHTTPHandlerV0{Assistant: assistant}
}

func (handler NuevaAppIntakeGuidedHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := validateWebPublicQueryV0(r); err != nil {
		writeNuevaAppIntakeGuidedErrorV0(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		handler.handlePost(w, r)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodPost)
		writeNuevaAppIntakeGuidedErrorV0(w, http.StatusMethodNotAllowed)
	}
}

func (handler NuevaAppIntakeGuidedHTTPHandlerV0) handlePost(w http.ResponseWriter, r *http.Request) {
	if !webControlContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		writeNuevaAppIntakeGuidedErrorV0(w, http.StatusUnsupportedMediaType)
		return
	}
	var request WebNuevaAppIntakeGuidedRequestV0
	if err := decodeWebControlJSONV0(w, r, &request); err != nil {
		writeNuevaAppIntakeGuidedErrorV0(w, http.StatusBadRequest)
		return
	}
	writeNuevaAppIntakeGuidedJSONV0(w, http.StatusOK, handler.NewResponseV0(r.Context(), request))
}

func NewWebNuevaAppIntakeGuidedResponseV0(
	request WebNuevaAppIntakeGuidedRequestV0,
) WebNuevaAppIntakeGuidedResponseV0 {
	return NuevaAppIntakeGuidedHTTPHandlerV0{}.NewResponseV0(context.Background(), request)
}

func (handler NuevaAppIntakeGuidedHTTPHandlerV0) NewResponseV0(
	ctx context.Context,
	request WebNuevaAppIntakeGuidedRequestV0,
) WebNuevaAppIntakeGuidedResponseV0 {
	session := webNuevaAppIntakeGuidedSessionV0(request)
	turn := WebNuevaAppIntakeGuidedTurnV0{
		SchemaVersion: WebNuevaAppIntakeGuidedTurnSchemaV0,
		Followups:     guidedFollowupActionsV0(),
		Messages:      []string{},
	}
	if assistant := handler.Assistant; assistant != nil {
		if assistedTurn, err := assistant.BuildNuevaAppIntakeGuidedTurnV0(ctx, request, session); err == nil {
			turn = normalizeWebNuevaAppIntakeGuidedTurnV0(assistedTurn, request.Need)
		}
	}
	if need := strings.TrimSpace(request.Need); need != "" && len(turn.Decisions) == 0 {
		turn = NewWebNuevaAppIntakeGuidedTurnV0(need)
	}
	if len(turn.Decisions) > 0 {
		session = ApplyWebNuevaAppIntakeGuidedTurnV0(session, turn)
	}
	session = ApplyWebNuevaAppIntakeGuidedAnswerV0(session, request.AnswerField, request.Answer)
	for _, actionID := range append([]string{request.ActionID}, request.ActionIDs...) {
		session = ApplyWebNuevaAppIntakeGuidedActionV0(session, actionID)
	}
	return WebNuevaAppIntakeGuidedResponseV0{
		SchemaVersion: WebNuevaAppIntakeGuidedResponseSchemaV0,
		Turn:          turn,
		Session:       session,
	}
}

func normalizeWebNuevaAppIntakeGuidedTurnV0(
	turn WebNuevaAppIntakeGuidedTurnV0,
	fallbackNeed string,
) WebNuevaAppIntakeGuidedTurnV0 {
	turn.SchemaVersion = WebNuevaAppIntakeGuidedTurnSchemaV0
	if strings.TrimSpace(turn.Need) == "" {
		turn.Need = strings.TrimSpace(fallbackNeed)
	}
	if turn.Decisions == nil {
		turn.Decisions = []WebNuevaAppIntakeDecisionV0{}
	}
	if len(turn.Followups) == 0 {
		turn.Followups = guidedFollowupActionsV0()
	}
	if turn.Messages == nil {
		turn.Messages = []string{}
	}
	return turn
}

func webNuevaAppIntakeGuidedSessionV0(request WebNuevaAppIntakeGuidedRequestV0) WebNuevaAppIntakeSessionV0 {
	if request.Session != nil {
		session := *request.Session
		if strings.TrimSpace(session.SessionID) == "" {
			session.SessionID = strings.TrimSpace(request.SessionID)
		}
		if strings.TrimSpace(session.Locale) == "" {
			session.Locale = strings.TrimSpace(request.Locale)
		}
		return refreshWebNuevaAppIntakeSessionV0(session)
	}
	idea := firstNuevaAppValueV0(request.Idea, request.Need)
	return NewWebNuevaAppIntakeSessionV0(request.SessionID, request.Locale, request.Nombre, idea)
}

func writeNuevaAppIntakeGuidedErrorV0(w http.ResponseWriter, status int) {
	writeNuevaAppIntakeGuidedJSONV0(w, status, map[string]string{
		"schema_version": NuevaAppIntakeGuidedEndpointSchemaV0,
		"error":          http.StatusText(status),
	})
}

func writeNuevaAppIntakeGuidedJSONV0(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
