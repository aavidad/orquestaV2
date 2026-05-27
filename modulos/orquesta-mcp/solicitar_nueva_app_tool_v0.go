package orquestamcp

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	MCPNuevaAppToolNameV0     = "orquesta.apps.solicitar_nueva.v0"
	MCPNuevaAppToolVersionV0  = "v0"
	MCPNuevaAppSourceV0       = "orquesta-mcp"
	MCPNuevaAppResourceURIV0  = "orquesta://contracts/solicitar-nueva-app/v0"
	MCPNuevaAppPromptNameV0   = "orquesta.solicitar_nueva_app.v0"
	MCPNuevaAppRespuestaV0    = "compacta"
	MCPNuevaAppEstadoOKV0     = "ok"
	MCPNuevaAppEstadoErrorV0  = "error"
	MCPNuevaAppDefaultErrorV0 = "validacion_publica"
)

type MCPNuevaAppToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	PromptName  string   `json:"prompt_name"`
	Invariantes []string `json:"invariantes"`
}

type MCPNuevaAppToolInputV0 struct {
	RequestID      string                           `json:"request_id,omitempty"`
	CorrelationID  string                           `json:"correlation_id,omitempty"`
	Respuesta      string                           `json:"respuesta,omitempty"`
	AppSpecRequest orquestafactory.AppSpecRequestV0 `json:"app_spec_request"`
}

type MCPNuevaAppToolResultV0 struct {
	Estado        string                 `json:"estado"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	AppSpec       MCPAppSpecCompactV0    `json:"app_spec,omitempty"`
	Backlog       MCPBacklogCompactV0    `json:"backlog_inicial_propuesto,omitempty"`
	Errores       []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
	Preguntas     []string               `json:"preguntas_abiertas,omitempty"`
	Supuestos     []string               `json:"supuestos,omitempty"`
}

type MCPAppSpecCompactV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	SpecID        string   `json:"spec_id,omitempty"`
	RequestID     string   `json:"request_id,omitempty"`
	Nombre        string   `json:"nombre,omitempty"`
	Slug          string   `json:"slug,omitempty"`
	Objetivo      string   `json:"objetivo,omitempty"`
	TipoApp       string   `json:"tipo_app,omitempty"`
	RequestKind   string   `json:"request_kind,omitempty"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
	Locale        string   `json:"locale,omitempty"`
	I18NLocales   []string `json:"i18n_locales,omitempty"`
	DeployTarget  string   `json:"deploy_target,omitempty"`
}

type MCPBacklogCompactV0 struct {
	SchemaVersion       string   `json:"schema_version,omitempty"`
	SpecID              string   `json:"spec_id,omitempty"`
	Estado              string   `json:"estado,omitempty"`
	FreshnessSourceRef  string   `json:"freshness_source_ref,omitempty"`
	DirectorHandoff     string   `json:"director_handoff,omitempty"`
	DirectorHandoffRef  string   `json:"director_handoff_ref,omitempty"`
	Fases               int      `json:"fases"`
	Microtareas         int      `json:"microtareas"`
	ContratosRequeridos []string `json:"contratos_requeridos,omitempty"`
	Riesgos             []string `json:"riesgos,omitempty"`
}

type MCPValidationIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func MCPNuevaAppDescriptorV0() MCPNuevaAppToolDescriptorV0 {
	return MCPNuevaAppToolDescriptorV0{
		Name:        MCPNuevaAppToolNameV0,
		Version:     MCPNuevaAppToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,respuesta?,app_spec_request:AppSpecRequestV0(request_kind?,execution_mode?)}",
		Output:      "ok:{app_spec compacta,backlog_inicial_propuesto compacto}|error:{errores_publicos}",
		ResourceURI: MCPNuevaAppResourceURIV0,
		PromptName:  MCPNuevaAppPromptNameV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"sin DB, CLI, runtime, filesystem ni servidor MCP real",
			"validacion de negocio delegada en orquesta-factory",
		},
	}
}

func ToAppSpecRequestV0(input MCPNuevaAppToolInputV0) (orquestafactory.AppSpecRequestV0, string) {
	req := input.AppSpecRequest
	req.Source = MCPNuevaAppSourceV0
	req.RequestID = firstNonEmptyMCPV0(req.RequestID, input.RequestID, input.CorrelationID)
	return req, firstNonEmptyMCPV0(input.CorrelationID, req.RequestID)
}

func NewMCPNuevaAppOKResultV0(spec orquestafactory.AppSpecV0, backlog orquestafactory.BacklogInicialPropuestoV0, correlationID string) MCPNuevaAppToolResultV0 {
	return MCPNuevaAppToolResultV0{
		Estado:        MCPNuevaAppEstadoOKV0,
		RequestID:     spec.RequestID,
		CorrelationID: firstNonEmptyMCPV0(correlationID, spec.RequestID),
		AppSpec:       compactAppSpecV0(spec),
		Backlog:       compactBacklogV0(backlog),
		Preguntas:     compactStringsMCPV0(append(spec.Scope.PreguntasAbiertas, backlog.PreguntasAbiertas...)),
		Supuestos:     compactStringsMCPV0(spec.Scope.Supuestos),
	}
}

func NewMCPNuevaAppErrorResultV0(requestID, correlationID string, issues []orquestafactory.ValidationIssue) MCPNuevaAppToolResultV0 {
	return MCPNuevaAppToolResultV0{
		Estado:        MCPNuevaAppEstadoErrorV0,
		RequestID:     strings.TrimSpace(requestID),
		CorrelationID: firstNonEmptyMCPV0(correlationID, requestID),
		Errores:       publicIssuesMCPV0(issues),
		Preguntas:     []string{},
		Supuestos:     []string{},
	}
}

func compactAppSpecV0(spec orquestafactory.AppSpecV0) MCPAppSpecCompactV0 {
	return MCPAppSpecCompactV0{
		SchemaVersion: strings.TrimSpace(spec.SchemaVersion),
		SpecID:        strings.TrimSpace(spec.SpecID),
		RequestID:     strings.TrimSpace(spec.RequestID),
		Nombre:        strings.TrimSpace(spec.App.Nombre),
		Slug:          strings.TrimSpace(spec.App.Slug),
		Objetivo:      strings.TrimSpace(spec.App.Objetivo),
		TipoApp:       strings.TrimSpace(spec.App.TipoApp),
		RequestKind:   strings.TrimSpace(spec.RequestKind),
		ExecutionMode: strings.TrimSpace(spec.ExecutionMode),
		Locale:        strings.TrimSpace(spec.Locale),
		I18NLocales:   compactStringsMCPV0(spec.I18N.Locales),
		DeployTarget:  strings.TrimSpace(spec.Deploy.Target),
	}
}

func compactBacklogV0(backlog orquestafactory.BacklogInicialPropuestoV0) MCPBacklogCompactV0 {
	return MCPBacklogCompactV0{
		SchemaVersion:       strings.TrimSpace(backlog.SchemaVersion),
		SpecID:              strings.TrimSpace(backlog.SpecID),
		Estado:              strings.TrimSpace(backlog.Estado),
		FreshnessSourceRef:  strings.TrimSpace(backlog.Freshness.SourceRef),
		DirectorHandoff:     strings.TrimSpace(backlog.DirectorHandoff.RequiredContract),
		DirectorHandoffRef:  strings.TrimSpace(backlog.DirectorHandoff.RequiredInputRef),
		Fases:               len(backlog.Fases),
		Microtareas:         len(backlog.Microtareas),
		ContratosRequeridos: compactStringsMCPV0(backlog.ContratosRequeridos),
		Riesgos:             compactStringsMCPV0(backlog.Riesgos),
	}
}

func publicIssuesMCPV0(values []orquestafactory.ValidationIssue) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if !isKnownFactoryIssueMCPV0(code) {
			code = orquestafactory.ErrAppSpecInvalida
		}
		message := strings.TrimSpace(value.Message)
		if message == "" {
			message = MCPNuevaAppDefaultErrorV0
		}
		out = append(out, MCPValidationIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(value.Field),
			Message: message,
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}

func isKnownFactoryIssueMCPV0(code string) bool {
	switch strings.TrimSpace(code) {
	case orquestafactory.ErrAppSpecInvalida,
		orquestafactory.ErrOpcionIncompatible,
		orquestafactory.ErrTargetNoSoportado,
		orquestafactory.ErrIdiomaInvalido,
		orquestafactory.ErrConectorRequeridoNoDisponible:
		return true
	default:
		return false
	}
}

func firstNonEmptyMCPV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactStringsMCPV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}
