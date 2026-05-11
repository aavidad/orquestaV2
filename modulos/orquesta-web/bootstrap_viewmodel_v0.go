package orquestaweb

import (
	"errors"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

const WebBootstrapProyectoSchemaV0 = "web_bootstrap_proyecto.v0"

type WebBootstrapProyectoEstadoV0 string

const (
	WebBootstrapProyectoEstadoRegistrado WebBootstrapProyectoEstadoV0 = "registrado"
	WebBootstrapProyectoEstadoInvalido   WebBootstrapProyectoEstadoV0 = "invalido"
)

type WebBootstrapProyectoViewModelV0 struct {
	SchemaVersion string                       `json:"schema_version"`
	Estado        WebBootstrapProyectoEstadoV0 `json:"estado"`
	RequestID     string                       `json:"request_id,omitempty"`
	CorrelationID string                       `json:"correlation_id,omitempty"`
	ProjectRef    string                       `json:"project_ref,omitempty"`
	AppSpecRef    string                       `json:"app_spec_ref,omitempty"`
	Registro      WebBootstrapRegistroV0       `json:"registro,omitempty"`
	StartRun      WebBootstrapStartRunV0       `json:"start_run,omitempty"`
	Workflow      WebBootstrapWorkflowV0       `json:"workflow,omitempty"`
	Warnings      []string                     `json:"warnings"`
	Errores       []WebBootstrapIssueV0        `json:"errores_publicos"`
}

type WebBootstrapRegistroV0 struct {
	RegistroID       string   `json:"registro_id,omitempty"`
	Estado           string   `json:"estado,omitempty"`
	EventosDominio   int      `json:"eventos_dominio"`
	FasesIniciales   int      `json:"fases_iniciales"`
	Microtareas      int      `json:"microtareas"`
	Contratos        int      `json:"contratos"`
	BootstrapVersion string   `json:"bootstrap_version,omitempty"`
	Warnings         []string `json:"warnings"`
}

type WebBootstrapStartRunV0 struct {
	CommandID   string `json:"command_id,omitempty"`
	CommandType string `json:"command_type,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	OccurredAt  string `json:"occurred_at,omitempty"`
}

type WebBootstrapWorkflowV0 struct {
	Events     []WebBootstrapWorkflowEventV0 `json:"events"`
	Outbox     int                           `json:"outbox"`
	Idempotent bool                          `json:"idempotent,omitempty"`
	NoopReason string                        `json:"noop_reason,omitempty"`
}

type WebBootstrapWorkflowEventV0 struct {
	EventID    string `json:"event_id,omitempty"`
	EventType  string `json:"event_type,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Sequence   int64  `json:"sequence"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type WebBootstrapIssueV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message,omitempty"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func NewWebBootstrapProyectoViewModelV0(
	result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0,
) WebBootstrapProyectoViewModelV0 {
	accepted := result.RegistroAceptado
	return WebBootstrapProyectoViewModelV0{
		SchemaVersion: WebBootstrapProyectoSchemaV0,
		Estado:        WebBootstrapProyectoEstadoRegistrado,
		RequestID:     trimV0(accepted.RequestID),
		CorrelationID: trimV0(accepted.CorrelationID),
		ProjectRef:    trimV0(result.ProjectRef),
		AppSpecRef:    trimV0(result.AppSpecRef),
		Registro:      webBootstrapRegistroV0(accepted),
		StartRun:      webBootstrapStartRunV0(result.StartRunCommand),
		Workflow:      webBootstrapWorkflowV0(result.WorkflowResult),
		Warnings:      compactStringsV0(accepted.Warnings),
		Errores:       []WebBootstrapIssueV0{},
	}
}

func NewWebBootstrapProyectoErrorViewModelV0(err error) (WebBootstrapProyectoViewModelV0, bool) {
	var bootstrapErr orquestadirector.BootstrapProyectoDesdeAppSpecErrorV0
	if !errors.As(err, &bootstrapErr) {
		return WebBootstrapProyectoViewModelV0{}, false
	}
	return WebBootstrapProyectoViewModelV0{
		SchemaVersion: WebBootstrapProyectoSchemaV0,
		Estado:        WebBootstrapProyectoEstadoInvalido,
		CorrelationID: trimV0(bootstrapErr.CorrelationID),
		Warnings:      []string{},
		Errores: []WebBootstrapIssueV0{{
			Code:          trimV0(bootstrapErr.Code),
			Message:       trimV0(bootstrapErr.Message),
			Field:         trimV0(bootstrapErr.Field),
			Retryable:     bootstrapErr.Retryable,
			CorrelationID: trimV0(bootstrapErr.CorrelationID),
		}},
	}, true
}

func webBootstrapRegistroV0(value orquestadirector.BootstrapRegistroAceptadoV0) WebBootstrapRegistroV0 {
	return WebBootstrapRegistroV0{
		RegistroID:       trimV0(value.RegistroID),
		Estado:           trimV0(value.Estado),
		EventosDominio:   value.EventosDominio,
		FasesIniciales:   value.FasesIniciales,
		Microtareas:      value.Microtareas,
		Contratos:        value.Contratos,
		BootstrapVersion: trimV0(value.BootstrapVersion),
		Warnings:         compactStringsV0(value.Warnings),
	}
}

func webBootstrapStartRunV0(value orquestacoreworkflow.OrchestrationCommandV0) WebBootstrapStartRunV0 {
	return WebBootstrapStartRunV0{
		CommandID:   trimV0(value.CommandID),
		CommandType: trimV0(value.CommandType),
		RunID:       trimV0(value.RunID),
		OccurredAt:  trimV0(value.OccurredAt),
	}
}

func webBootstrapWorkflowV0(value orquestacoreworkflow.OrchestrationCommandResultV0) WebBootstrapWorkflowV0 {
	return WebBootstrapWorkflowV0{
		Events:     webBootstrapEventsV0(value.Events),
		Outbox:     len(value.Outbox),
		Idempotent: value.Idempotent,
		NoopReason: trimV0(value.NoopReason),
	}
}

func webBootstrapEventsV0(values []orquestacoreworkflow.OrchestrationEventV0) []WebBootstrapWorkflowEventV0 {
	out := make([]WebBootstrapWorkflowEventV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebBootstrapWorkflowEventV0{
			EventID:    trimV0(value.EventID),
			EventType:  trimV0(value.EventType),
			RunID:      trimV0(value.RunID),
			Sequence:   value.Sequence,
			OccurredAt: trimV0(value.OccurredAt),
		})
	}
	if out == nil {
		return []WebBootstrapWorkflowEventV0{}
	}
	return out
}
