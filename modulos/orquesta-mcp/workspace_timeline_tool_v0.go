package orquestamcp

import (
	"context"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	MCPWorkspaceTimelineToolNameV0    = "orquesta.observability.workspace_timeline.query.v0"
	MCPWorkspaceTimelineToolVersionV0 = "v0"
	MCPWorkspaceTimelineEstadoOKV0    = "ok"
	MCPWorkspaceTimelineEstadoErrorV0 = "error"
)

type MCPWorkspaceTimelineToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPWorkspaceTimelineToolResultV0 struct {
	Estado        string                                     `json:"estado"`
	RequestID     string                                     `json:"request_id,omitempty"`
	CorrelationID string                                     `json:"correlation_id,omitempty"`
	Timeline      *orquestaobservability.WorkspaceTimelineV0 `json:"timeline,omitempty"`
	Errores       []MCPValidationIssueV0                     `json:"errores_publicos,omitempty"`
}

type MCPWorkspaceTimelineToolInputV0 = orquestaobservability.WorkspaceTimelineQueryV0

type MCPWorkspaceTimelineToolExecutorV0 struct {
	Source orquestaobservability.WorkspaceTimelineSourcePortV0
}

func MCPWorkspaceTimelineToolDescriptorV0Value() MCPWorkspaceTimelineToolDescriptorV0 {
	return MCPWorkspaceTimelineToolDescriptorV0{
		Name:        MCPWorkspaceTimelineToolNameV0,
		Version:     MCPWorkspaceTimelineToolVersionV0,
		InputSchema: "workspace_timeline_query.v0:{request_id,correlation_id,consumer,locale,scope,agent_ref?,project_ref?,task_ref?,time_window,page,sources}",
		Output:      "ok:{timeline{sources,items,privacy}}|error:{errores_publicos}",
		ResourceURI: MCPWorkspaceTimelineResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"lee solo WorkspaceTimelineSourcePortV0 inyectado",
			"contrato cubre agent_ref project_ref task_ref",
			"sin prompts transcripts crudos HOME tokens DB runtime ni shell",
		},
	}
}

func (executor MCPWorkspaceTimelineToolExecutorV0) Execute(
	ctx context.Context,
	input orquestaobservability.WorkspaceTimelineQueryV0,
) (MCPWorkspaceTimelineToolResultV0, error) {
	if err := orquestaobservability.ValidateWorkspaceTimelineQueryV0(input); err != nil {
		return newMCPWorkspaceTimelineErrorV0(input, orquestaobservability.ErrWorkspaceTimelineQueryInvalidaV0, "query", err.Error()), nil
	}
	if executor.Source == nil {
		return newMCPWorkspaceTimelineErrorV0(input, orquestaobservability.ErrWorkspaceTimelineNoDisponibleV0, "source", "source requerido"), nil
	}
	timeline, err := executor.Source.QueryWorkspaceTimelineV0(ctx, input)
	if err != nil {
		return newMCPWorkspaceTimelineErrorV0(
			input,
			orquestaobservability.ErrWorkspaceTimelineNoDisponibleV0,
			"source",
			publicMCPExecutorErrorMessageFromErrorV0(orquestaobservability.ErrWorkspaceTimelineNoDisponibleV0, err),
		), nil
	}
	return MCPWorkspaceTimelineToolResultV0{
		Estado:        MCPWorkspaceTimelineEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Timeline:      &timeline,
		Errores:       []MCPValidationIssueV0{},
	}, nil
}

func newMCPWorkspaceTimelineErrorV0(
	input orquestaobservability.WorkspaceTimelineQueryV0,
	code string,
	field string,
	message string,
) MCPWorkspaceTimelineToolResultV0 {
	return MCPWorkspaceTimelineToolResultV0{
		Estado:        MCPWorkspaceTimelineEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}
