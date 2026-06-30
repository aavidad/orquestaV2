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
	form := session.Form
	values := []WebNuevaAppIntakeContextV0{
		{Key: "locale", Value: request.Locale},
		{Key: "nombre", Value: request.Nombre},
		{Key: "objetivo", Value: request.Objetivo},
		{Key: "tipo_app", Value: request.TipoApp},
		{Key: "plataformas", Value: stringsFromNuevaAppValuesV0(form.Plataformas)},
		{Key: "arquitectura", Value: form.PreferenciasTecnicas.Arquitectura},
		{Key: "lenguaje", Value: form.PreferenciasTecnicas.Lenguaje},
		{Key: "framework", Value: form.PreferenciasTecnicas.Framework},
		{Key: "integraciones", Value: webNuevaAppIntakeIntegrationSummaryV0(form.Integraciones)},
		{Key: "datos", Value: webNuevaAppIntakeDataSummaryV0(form.Datos)},
		{Key: "storage", Value: webNuevaAppIntakeStorageSummaryV0(form.Datos.Storage)},
		{Key: "deploy", Value: form.Deploy.Target},
		{Key: "pruebas", Value: form.Calidad.Pruebas},
		{Key: "accesibilidad", Value: form.Calidad.Accesibilidad},
		{Key: "observabilidad", Value: webNuevaAppIntakeBoolSummaryV0(form.Calidad.Observabilidad)},
		{Key: "documentacion", Value: webNuevaAppIntakeDocumentationSummaryV0(form.Documentacion)},
		{Key: "i18n", Value: webNuevaAppIntakeI18NSummaryV0(form.I18N)},
		{Key: "autonomia_agentes", Value: form.Agentes.Autonomia},
		{Key: "revision_humana", Value: webNuevaAppIntakeBoolSummaryV0(form.Agentes.RevisionHumana)},
		{Key: "project_source", Value: webNuevaAppIntakeProjectSourceSummaryV0(form.ProjectSource)},
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

func webNuevaAppIntakeIntegrationSummaryV0(values []WebNuevaAppConnectorFormV0) string {
	types := make([]string, 0, len(values))
	for _, value := range values {
		if value.Tipo != "" {
			types = append(types, value.Tipo)
		}
	}
	return stringsFromNuevaAppValuesV0(types)
}

func webNuevaAppIntakeDataSummaryV0(value WebNuevaAppDatosFormV0) string {
	parts := []string{}
	if value.DBRequired {
		parts = append(parts, "db_required")
	}
	if value.NecesidadFuncional != "" {
		parts = append(parts, value.NecesidadFuncional)
	}
	parts = append(parts, value.TiposDatos...)
	for _, item := range value.TiposDetallados {
		if item.Nombre != "" {
			parts = append(parts, item.Nombre)
		}
	}
	return stringsFromNuevaAppValuesV0(parts)
}

func webNuevaAppIntakeStorageSummaryV0(values []WebNuevaAppDataStorageFormV0) string {
	types := make([]string, 0, len(values))
	for _, value := range values {
		if value.Tipo != "" {
			types = append(types, value.Tipo)
		}
	}
	return stringsFromNuevaAppValuesV0(types)
}

func webNuevaAppIntakeDocumentationSummaryV0(value WebNuevaAppDocumentacionFormV0) string {
	parts := []string{}
	if value.Usuario != nil {
		parts = append(parts, "usuario:"+webNuevaAppIntakeBoolSummaryV0(value.Usuario))
	}
	if value.Desarrollo != nil {
		parts = append(parts, "desarrollo:"+webNuevaAppIntakeBoolSummaryV0(value.Desarrollo))
	}
	if value.Sistemas != nil {
		parts = append(parts, "sistemas:"+webNuevaAppIntakeBoolSummaryV0(value.Sistemas))
	}
	if value.Profundidad != "" {
		parts = append(parts, value.Profundidad)
	}
	parts = append(parts, value.Locales...)
	return stringsFromNuevaAppValuesV0(parts)
}

func webNuevaAppIntakeI18NSummaryV0(value WebNuevaAppI18NFormV0) string {
	parts := []string{}
	if value.Enabled != nil {
		parts = append(parts, "enabled:"+webNuevaAppIntakeBoolSummaryV0(value.Enabled))
	}
	if value.DefaultLocale != "" {
		parts = append(parts, "default:"+value.DefaultLocale)
	}
	parts = append(parts, value.Locales...)
	return stringsFromNuevaAppValuesV0(parts)
}

func webNuevaAppIntakeProjectSourceSummaryV0(value WebNuevaAppProjectSourceFormV0) string {
	return stringsFromNuevaAppValuesV0([]string{
		value.Kind,
		value.ProjectRef,
		value.Branch,
	})
}

func webNuevaAppIntakeBoolSummaryV0(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}
