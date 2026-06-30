package orquestaweb

import orquestafactory "orquesta/modulos/orquesta-factory"

const WebNuevaAppIntakeSessionSchemaV0 = "web_nueva_app_intake_session.v0"

type WebNuevaAppIntakeEstadoV0 string

const (
	WebNuevaAppIntakeEstadoRequiereDatos WebNuevaAppIntakeEstadoV0 = "requiere_datos"
	WebNuevaAppIntakeEstadoLista         WebNuevaAppIntakeEstadoV0 = "lista_para_solicitar"
)

type WebNuevaAppIntakeSessionV0 struct {
	SchemaVersion    string                           `json:"schema_version"`
	SessionID        string                           `json:"session_id"`
	SessionRef       string                           `json:"session_ref,omitempty"`
	Locale           string                           `json:"locale,omitempty"`
	Estado           WebNuevaAppIntakeEstadoV0        `json:"estado"`
	Form             WebNuevaAppFormV0                `json:"form"`
	Sections         []WebNuevaAppIntakeSectionV0     `json:"sections"`
	FieldIndex       []WebNuevaAppIntakeFieldIndexV0  `json:"field_index"`
	Questions        []WebNuevaAppIntakeQuestionV0    `json:"questions"`
	PendingQuestions []string                         `json:"pending_questions"`
	Decisions        []WebNuevaAppIntakeDecisionV0    `json:"decisions"`
	AppSpecPartial   orquestafactory.AppSpecRequestV0 `json:"app_spec_partial"`
	Handoff          WebNuevaAppIntakeHandoffV0       `json:"handoff"`
}

type WebNuevaAppIntakeSectionV0 struct {
	ID             string   `json:"id"`
	Status         string   `json:"status"`
	RequiredFields []string `json:"required_fields"`
	CapturedFields []string `json:"captured_fields"`
	PendingFields  []string `json:"pending_fields"`
}

type WebNuevaAppIntakeFieldIndexV0 struct {
	Field     string `json:"field"`
	SectionID string `json:"section_id"`
}

type WebNuevaAppIntakeDecisionV0 struct {
	Field  string   `json:"field"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

func NewWebNuevaAppIntakeSessionV0(sessionID, locale, nombre, idea string) WebNuevaAppIntakeSessionV0 {
	session := WebNuevaAppIntakeSessionV0{
		SchemaVersion: WebNuevaAppIntakeSessionSchemaV0,
		SessionID:     trimV0(sessionID),
		Locale:        trimV0(locale),
		Form: WebNuevaAppFormV0{
			RequestID: trimV0(sessionID),
			Locale:    trimV0(locale),
			Nombre:    trimV0(nombre),
			Objetivo:  trimV0(idea),
		},
		Decisions: []WebNuevaAppIntakeDecisionV0{},
	}
	return refreshWebNuevaAppIntakeSessionV0(session)
}

func (session WebNuevaAppIntakeSessionV0) ApplyDecisionV0(
	decision WebNuevaAppIntakeDecisionV0,
) WebNuevaAppIntakeSessionV0 {
	decision = normalizeWebNuevaAppIntakeDecisionV0(decision)
	if decision.Field == "" {
		return refreshWebNuevaAppIntakeSessionV0(session)
	}
	session.Form = applyWebNuevaAppIntakeDecisionToFormV0(session.Form, decision)
	session.Locale = session.Form.Locale
	session.Decisions = append(session.Decisions, decision)
	return refreshWebNuevaAppIntakeSessionV0(session)
}

func refreshWebNuevaAppIntakeSessionV0(session WebNuevaAppIntakeSessionV0) WebNuevaAppIntakeSessionV0 {
	session.SchemaVersion = WebNuevaAppIntakeSessionSchemaV0
	session.SessionID = trimV0(session.SessionID)
	session.SessionRef = firstNuevaAppValueV0(session.SessionRef, session.SessionID)
	session.Locale = trimV0(session.Locale)
	session.Form.RequestID = firstNuevaAppValueV0(session.Form.RequestID, session.SessionID)
	session.Form.Locale = firstNuevaAppValueV0(session.Form.Locale, session.Locale)
	session.PendingQuestions = webNuevaAppIntakePendingFieldsV0(session.Form)
	session.Questions = webNuevaAppIntakeQuestionsV0(session.PendingQuestions)
	session.Sections = webNuevaAppIntakeSectionsV0(session.Form)
	session.FieldIndex = webNuevaAppIntakeFieldIndexV0()
	session.AppSpecPartial = session.Form.ToAppSpecRequestV0()
	if len(session.PendingQuestions) == 0 {
		session.Estado = WebNuevaAppIntakeEstadoLista
	} else {
		session.Estado = WebNuevaAppIntakeEstadoRequiereDatos
	}
	if session.Decisions == nil {
		session.Decisions = []WebNuevaAppIntakeDecisionV0{}
	}
	session.Handoff = webNuevaAppIntakeHandoffV0(session)
	return session
}

func webNuevaAppIntakePendingFieldsV0(form WebNuevaAppFormV0) []string {
	required := []struct {
		field string
		value string
	}{
		{field: "locale", value: form.Locale},
		{field: "nombre", value: form.Nombre},
		{field: "objetivo", value: form.Objetivo},
		{field: "tipo_app", value: form.TipoApp},
	}
	out := make([]string, 0, len(required))
	for _, item := range required {
		if trimV0(item.value) == "" {
			out = append(out, item.field)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func webNuevaAppIntakeSectionsV0(form WebNuevaAppFormV0) []WebNuevaAppIntakeSectionV0 {
	return []WebNuevaAppIntakeSectionV0{
		webNuevaAppIntakeSectionV0("identidad", []string{"locale", "nombre", "objetivo", "tipo_app"}, form),
		webNuevaAppIntakeSectionV0("politica", []string{"request_kind", "execution_mode"}, form),
		webNuevaAppIntakeSectionV0("alcance", []string{"descripcion", "usuarios_objetivo", "restricciones"}, form),
		webNuevaAppIntakeSectionV0("interfaces", []string{"plataformas", "integraciones"}, form),
		webNuevaAppIntakeSectionV0("datos_calidad", []string{"datos", "deploy", "calidad", "documentacion", "i18n"}, form),
		webNuevaAppIntakeSectionV0("gobierno", []string{"agentes", "project_source"}, form),
	}
}

func webNuevaAppIntakeSectionV0(id string, fields []string, form WebNuevaAppFormV0) WebNuevaAppIntakeSectionV0 {
	captured := make([]string, 0, len(fields))
	pendingRequired := make([]string, 0, len(fields))
	for _, field := range fields {
		if webNuevaAppIntakeFieldCapturedV0(form, field) {
			captured = append(captured, field)
			continue
		}
		if webNuevaAppIntakeFieldRequiredV0(field) {
			pendingRequired = append(pendingRequired, field)
		}
	}
	status := "empty"
	if len(captured) > 0 {
		status = "partial"
	}
	if len(pendingRequired) == 0 && len(captured) > 0 {
		status = "complete"
	}
	return WebNuevaAppIntakeSectionV0{
		ID:             id,
		Status:         status,
		RequiredFields: webNuevaAppIntakeRequiredFieldsV0(fields),
		CapturedFields: captured,
		PendingFields:  pendingRequired,
	}
}

func webNuevaAppIntakeRequiredFieldsV0(fields []string) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if webNuevaAppIntakeFieldRequiredV0(field) {
			out = append(out, field)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func webNuevaAppIntakeFieldRequiredV0(field string) bool {
	switch field {
	case "locale", "nombre", "objetivo", "tipo_app":
		return true
	default:
		return false
	}
}

func webNuevaAppIntakeFieldCapturedV0(form WebNuevaAppFormV0, field string) bool {
	switch field {
	case "locale":
		return trimV0(form.Locale) != ""
	case "nombre":
		return trimV0(form.Nombre) != ""
	case "objetivo":
		return trimV0(form.Objetivo) != ""
	case "tipo_app":
		return trimV0(form.TipoApp) != ""
	case "request_kind":
		return trimV0(form.RequestKind) != ""
	case "execution_mode":
		return trimV0(form.ExecutionMode) != ""
	case "descripcion":
		return trimV0(form.Descripcion) != ""
	case "usuarios_objetivo":
		return len(compactStringsV0(form.UsuariosObjetivo)) > 0
	case "restricciones":
		return len(compactStringsV0(form.Restricciones)) > 0
	case "plataformas":
		return len(compactStringsV0(form.Plataformas)) > 0
	case "integraciones":
		return len(form.Integraciones) > 0
	case "datos":
		return form.Datos.DBRequired ||
			trimV0(form.Datos.NecesidadFuncional) != "" ||
			len(compactStringsV0(form.Datos.TiposDatos)) > 0 ||
			len(form.Datos.TiposDetallados) > 0 ||
			len(form.Datos.Storage) > 0
	case "deploy":
		return trimV0(form.Deploy.Target) != "" || len(compactStringsV0(form.Deploy.Restricciones)) > 0
	case "calidad":
		return trimV0(form.Calidad.Pruebas) != "" ||
			trimV0(form.Calidad.Accesibilidad) != "" ||
			len(compactStringsV0(form.Calidad.AccesibilidadOpciones)) > 0 ||
			len(compactStringsV0(form.Calidad.Compliance)) > 0
	case "documentacion":
		return form.Documentacion.Usuario != nil ||
			form.Documentacion.Desarrollo != nil ||
			form.Documentacion.Sistemas != nil ||
			trimV0(form.Documentacion.Profundidad) != "" ||
			len(compactStringsV0(form.Documentacion.Locales)) > 0
	case "i18n":
		return form.I18N.Enabled != nil || trimV0(form.I18N.DefaultLocale) != "" || len(compactStringsV0(form.I18N.Locales)) > 0
	case "agentes":
		return form.Agentes.RevisionHumana != nil || trimV0(form.Agentes.Autonomia) != "" || len(compactStringsV0(form.Agentes.Preferencias)) > 0
	case "project_source":
		return trimV0(form.ProjectSource.Kind) != "" ||
			trimV0(form.ProjectSource.ProjectRef) != "" ||
			trimV0(form.ProjectSource.GitURL) != "" ||
			trimV0(form.ProjectSource.Branch) != "" ||
			trimV0(form.ProjectSource.LocalPath) != ""
	default:
		return false
	}
}

func webNuevaAppIntakeFieldIndexV0() []WebNuevaAppIntakeFieldIndexV0 {
	sections := map[string][]string{
		"identidad":     {"locale", "nombre", "objetivo", "tipo_app"},
		"politica":      {"request_kind", "execution_mode"},
		"alcance":       {"descripcion", "usuarios_objetivo", "restricciones"},
		"interfaces":    {"plataformas", "integraciones"},
		"datos_calidad": {"datos", "deploy", "calidad", "documentacion", "i18n"},
		"gobierno":      {"agentes", "project_source"},
	}
	order := []string{"identidad", "politica", "alcance", "interfaces", "datos_calidad", "gobierno"}
	out := []WebNuevaAppIntakeFieldIndexV0{}
	for _, sectionID := range order {
		for _, field := range sections[sectionID] {
			out = append(out, WebNuevaAppIntakeFieldIndexV0{Field: field, SectionID: sectionID})
		}
	}
	return out
}
