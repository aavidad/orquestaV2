package orquestaweb

const (
	WebNuevaAppIntakeHandoffSchemaV0 = "web_nueva_app_intake_handoff.v0"
	WebNuevaAppDirectorTargetToolV0  = "orquesta.apps.arrancar_director.v0"
	WebNuevaAppFallbackTargetToolV0  = "orquesta.apps.solicitar_nueva.v0"
)

type WebNuevaAppIntakeHandoffV0 struct {
	SchemaVersion        string                       `json:"schema_version"`
	Ready                bool                         `json:"ready"`
	TargetTool           string                       `json:"target_tool"`
	TargetPath           string                       `json:"target_path,omitempty"`
	FallbackTool         string                       `json:"fallback_tool,omitempty"`
	SessionRef           string                       `json:"session_ref,omitempty"`
	RequestRef           string                       `json:"request_ref,omitempty"`
	CorrelationRef       string                       `json:"correlation_ref,omitempty"`
	PendingQuestions     []string                     `json:"pending_questions"`
	RecommendedQuestions []string                     `json:"recommended_questions,omitempty"`
	ContextRefs          []string                     `json:"context_refs"`
	ContextSummary       []WebNuevaAppIntakeContextV0 `json:"context_summary"`
}

type WebNuevaAppIntakeContextV0 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func webNuevaAppIntakeHandoffV0(session WebNuevaAppIntakeSessionV0) WebNuevaAppIntakeHandoffV0 {
	request := session.AppSpecPartial
	requestRef := firstNuevaAppValueV0(request.RequestID, session.SessionRef, session.SessionID)
	return WebNuevaAppIntakeHandoffV0{
		SchemaVersion:        WebNuevaAppIntakeHandoffSchemaV0,
		Ready:                len(session.PendingQuestions) == 0,
		TargetTool:           WebNuevaAppDirectorTargetToolV0,
		TargetPath:           ArrancarDirectorAppEndpointV0,
		FallbackTool:         WebNuevaAppFallbackTargetToolV0,
		SessionRef:           session.SessionRef,
		RequestRef:           requestRef,
		CorrelationRef:       requestRef,
		PendingQuestions:     append([]string(nil), session.PendingQuestions...),
		RecommendedQuestions: append([]string(nil), session.RecommendedQuestions...),
		ContextRefs:          webNuevaAppIntakeContextRefsV0(session.SessionRef, requestRef),
		ContextSummary:       webNuevaAppIntakeContextSummaryV0(session),
	}
}

func webNuevaAppIntakeContextRefsV0(sessionRef, requestRef string) []string {
	return compactStringsV0([]string{
		"session_ref:" + trimV0(sessionRef),
		"request_ref:" + trimV0(requestRef),
	})
}

func webNuevaAppIntakeContextSummaryV0(session WebNuevaAppIntakeSessionV0) []WebNuevaAppIntakeContextV0 {
	request := session.AppSpecPartial
	values := []WebNuevaAppIntakeContextV0{
		{Key: "locale", Value: request.Locale},
		{Key: "nombre", Value: request.Nombre},
		{Key: "objetivo", Value: request.Objetivo},
		{Key: "tipo_app", Value: request.TipoApp},
	}
	out := make([]WebNuevaAppIntakeContextV0, 0, len(values))
	for _, value := range values {
		if trimV0(value.Value) == "" {
			continue
		}
		out = append(out, WebNuevaAppIntakeContextV0{Key: value.Key, Value: trimV0(value.Value)})
	}
	if out == nil {
		return []WebNuevaAppIntakeContextV0{}
	}
	return out
}
