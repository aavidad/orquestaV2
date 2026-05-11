package orquestamcp

import (
	"errors"
	"strings"

	orquestadirector "orquesta/modulos/orquesta-director"
)

const (
	MCPBootstrapToolNameV0        = "orquesta.director.bootstrap_appspec.v0"
	MCPBootstrapToolVersionV0     = "v0"
	MCPBootstrapPromptNameV0      = "orquesta.bootstrap_proyecto_desde_appspec.v0"
	MCPBootstrapRespuestaV0       = "compacta"
	MCPBootstrapEstadoOKV0        = "ok"
	MCPBootstrapEstadoErrorV0     = "error"
	MCPBootstrapDefaultErrorMsgV0 = "error_publico_bootstrap"
)

type MCPBootstrapToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	PromptName  string   `json:"prompt_name"`
	Invariantes []string `json:"invariantes"`
}

type MCPBootstrapToolInputV0 struct {
	RequestID     string                                                  `json:"request_id,omitempty"`
	CorrelationID string                                                  `json:"correlation_id,omitempty"`
	Respuesta     string                                                  `json:"respuesta,omitempty"`
	Command       orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 `json:"command"`
}

type MCPBootstrapToolResultV0 struct {
	Estado        string                        `json:"estado"`
	RequestID     string                        `json:"request_id,omitempty"`
	CorrelationID string                        `json:"correlation_id,omitempty"`
	Registro      MCPBootstrapRegistroCompactV0 `json:"registro_aceptado,omitempty"`
	ProjectRef    string                        `json:"project_ref,omitempty"`
	AppSpecRef    string                        `json:"app_spec_ref,omitempty"`
	Workflow      MCPBootstrapWorkflowCompactV0 `json:"workflow,omitempty"`
	Errores       []MCPBootstrapPublicErrorV0   `json:"errores_publicos,omitempty"`
}

type MCPBootstrapRegistroCompactV0 struct {
	RegistroID       string   `json:"registro_id,omitempty"`
	ProjectRef       string   `json:"project_ref,omitempty"`
	AppSpecRef       string   `json:"app_spec_ref,omitempty"`
	Estado           string   `json:"estado,omitempty"`
	EventosDominio   int      `json:"eventos_dominio"`
	Warnings         []string `json:"warnings,omitempty"`
	FasesIniciales   int      `json:"fases_iniciales"`
	Microtareas      int      `json:"microtareas"`
	Contratos        int      `json:"contratos"`
	BootstrapVersion string   `json:"bootstrap_version,omitempty"`
}

type MCPBootstrapWorkflowCompactV0 struct {
	CommandID       string   `json:"command_id,omitempty"`
	CommandType     string   `json:"command_type,omitempty"`
	RunID           string   `json:"run_id,omitempty"`
	EventTypes      []string `json:"event_types,omitempty"`
	Events          int      `json:"events"`
	Outbox          int      `json:"outbox"`
	Idempotent      bool     `json:"idempotent,omitempty"`
	NoopReason      string   `json:"noop_reason,omitempty"`
	PayloadVersions []string `json:"payload_versions,omitempty"`
}

type MCPBootstrapPublicErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func MCPBootstrapToolDescriptorV0Value() MCPBootstrapToolDescriptorV0 {
	return MCPBootstrapToolDescriptorV0{
		Name:        MCPBootstrapToolNameV0,
		Version:     MCPBootstrapToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,respuesta?,command:BootstrapProyectoDesdeAppSpecCommandV0}",
		Output:      "ok:{registro_aceptado compacto,project_ref,app_spec_ref,workflow compacto}|error:{errores_publicos}",
		ResourceURI: MCPBootstrapResourceURIV0,
		PromptName:  MCPBootstrapPromptNameV0,
		Invariantes: []string{
			"tool puro testable",
			"invoca solo orquesta-director.BootstrapProyectoDesdeAppSpecV0",
			"sin persistencia, runtime, DB ni filesystem productivo",
		},
	}
}

func ExecuteMCPBootstrapToolV0(input MCPBootstrapToolInputV0) MCPBootstrapToolResultV0 {
	cmd := normalizeMCPBootstrapCommandV0(input)
	result, err := orquestadirector.BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		return NewMCPBootstrapErrorResultV0(cmd.RequestID, cmd.CorrelationID, err)
	}
	return NewMCPBootstrapOKResultV0(result, cmd.RequestID, cmd.CorrelationID)
}

func normalizeMCPBootstrapCommandV0(input MCPBootstrapToolInputV0) orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 {
	cmd := input.Command
	cmd.RequestID = firstNonEmptyMCPV0(cmd.RequestID, input.RequestID, cmd.AppSpec.RequestID)
	cmd.CorrelationID = firstNonEmptyMCPV0(input.CorrelationID, cmd.CorrelationID)
	cmd.RequestedBy = firstNonEmptyMCPV0(cmd.RequestedBy, "director")
	return cmd
}

func NewMCPBootstrapOKResultV0(
	result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0,
	requestID string,
	correlationID string,
) MCPBootstrapToolResultV0 {
	registro := compactBootstrapRegistroV0(result.RegistroAceptado)
	return MCPBootstrapToolResultV0{
		Estado:        MCPBootstrapEstadoOKV0,
		RequestID:     firstNonEmptyMCPV0(requestID, result.RegistroAceptado.RequestID),
		CorrelationID: firstNonEmptyMCPV0(correlationID, result.RegistroAceptado.CorrelationID),
		Registro:      registro,
		ProjectRef:    strings.TrimSpace(result.ProjectRef),
		AppSpecRef:    strings.TrimSpace(result.AppSpecRef),
		Workflow:      compactBootstrapWorkflowV0(result),
		Errores:       []MCPBootstrapPublicErrorV0{},
	}
}

func NewMCPBootstrapErrorResultV0(requestID string, correlationID string, err error) MCPBootstrapToolResultV0 {
	return MCPBootstrapToolResultV0{
		Estado:        MCPBootstrapEstadoErrorV0,
		RequestID:     strings.TrimSpace(requestID),
		CorrelationID: strings.TrimSpace(correlationID),
		Errores:       []MCPBootstrapPublicErrorV0{publicBootstrapErrorV0(err, correlationID)},
	}
}

func compactBootstrapRegistroV0(value orquestadirector.BootstrapRegistroAceptadoV0) MCPBootstrapRegistroCompactV0 {
	return MCPBootstrapRegistroCompactV0{
		RegistroID:       strings.TrimSpace(value.RegistroID),
		ProjectRef:       strings.TrimSpace(value.ProjectRef),
		AppSpecRef:       strings.TrimSpace(value.AppSpecRef),
		Estado:           strings.TrimSpace(value.Estado),
		EventosDominio:   value.EventosDominio,
		Warnings:         compactStringsMCPV0(value.Warnings),
		FasesIniciales:   value.FasesIniciales,
		Microtareas:      value.Microtareas,
		Contratos:        value.Contratos,
		BootstrapVersion: strings.TrimSpace(value.BootstrapVersion),
	}
}

func compactBootstrapWorkflowV0(result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0) MCPBootstrapWorkflowCompactV0 {
	eventTypes := make([]string, 0, len(result.WorkflowResult.Events))
	payloadVersions := []string{result.StartRunCommand.PayloadVersion}
	for _, event := range result.WorkflowResult.Events {
		eventTypes = append(eventTypes, event.EventType)
		payloadVersions = append(payloadVersions, event.PayloadVersion)
	}
	return MCPBootstrapWorkflowCompactV0{
		CommandID:       strings.TrimSpace(result.StartRunCommand.CommandID),
		CommandType:     strings.TrimSpace(result.StartRunCommand.CommandType),
		RunID:           strings.TrimSpace(result.StartRunCommand.RunID),
		EventTypes:      compactStringsMCPV0(eventTypes),
		Events:          len(result.WorkflowResult.Events),
		Outbox:          len(result.WorkflowResult.Outbox),
		Idempotent:      result.WorkflowResult.Idempotent,
		NoopReason:      strings.TrimSpace(result.WorkflowResult.NoopReason),
		PayloadVersions: compactStringsMCPV0(payloadVersions),
	}
}

func publicBootstrapErrorV0(err error, correlationID string) MCPBootstrapPublicErrorV0 {
	var directorErr orquestadirector.BootstrapProyectoDesdeAppSpecErrorV0
	if errors.As(err, &directorErr) {
		return MCPBootstrapPublicErrorV0{
			Code:          strings.TrimSpace(directorErr.Code),
			Message:       firstNonEmptyMCPV0(directorErr.Message, MCPBootstrapDefaultErrorMsgV0),
			Field:         strings.TrimSpace(directorErr.Field),
			Retryable:     directorErr.Retryable,
			CorrelationID: firstNonEmptyMCPV0(directorErr.CorrelationID, correlationID),
		}
	}
	return MCPBootstrapPublicErrorV0{
		Code:          firstNonEmptyMCPV0(err.Error(), MCPBootstrapDefaultErrorMsgV0),
		Message:       MCPBootstrapDefaultErrorMsgV0,
		Retryable:     false,
		CorrelationID: strings.TrimSpace(correlationID),
	}
}
