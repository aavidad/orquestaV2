package orquestamcp

import (
	"context"
	"errors"
	"strings"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

const (
	MCPDirectorSupervisorBriefingToolNameV0    = "orquesta.director_supervisor.briefing.v0"
	MCPDirectorSupervisorBriefingToolVersionV0 = "v0"
	MCPDirectorSupervisorBriefingResourceURIV0 = "orquesta://contracts/director-supervisor-briefing/v0"
	MCPDirectorSupervisorBriefingEstadoOKV0    = "ok"
	MCPDirectorSupervisorBriefingEstadoErrorV0 = "error"
)

type MCPDirectorSupervisorBriefingToolDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	InputSchema string `json:"input_schema"`
	Output      string `json:"output"`
	ResourceURI string `json:"resource_uri"`
}

type MCPDirectorSupervisorBriefingToolInputV0 struct {
	RequestID     string                                                       `json:"request_id,omitempty"`
	CorrelationID string                                                       `json:"correlation_id,omitempty"`
	BriefingInput orquestadirectorsupervisor.DirectorSupervisorBriefingInputV0 `json:"briefing_input"`
}

type MCPDirectorSupervisorBriefingToolResultV0 struct {
	Estado        string                                                   `json:"estado"`
	RequestID     string                                                   `json:"request_id,omitempty"`
	CorrelationID string                                                   `json:"correlation_id,omitempty"`
	RunRef        string                                                   `json:"run_ref,omitempty"`
	Briefing      *orquestadirectorsupervisor.DirectorSupervisorBriefingV0 `json:"briefing,omitempty"`
	Errores       []MCPDirectorSupervisorBriefingIssueV0                   `json:"errores_publicos,omitempty"`
}

type MCPDirectorSupervisorBriefingIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type MCPDirectorSupervisorBriefingToolExecutorV0 struct{}

func MCPDirectorSupervisorBriefingDescriptorV0() MCPDirectorSupervisorBriefingToolDescriptorV0 {
	return MCPDirectorSupervisorBriefingToolDescriptorV0{
		Name:        MCPDirectorSupervisorBriefingToolNameV0,
		Version:     MCPDirectorSupervisorBriefingToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,briefing_input}",
		Output:      "ok:{run_ref,briefing}|error:{errores_publicos}",
		ResourceURI: MCPDirectorSupervisorBriefingResourceURIV0,
	}
}

func (executor MCPDirectorSupervisorBriefingToolExecutorV0) Execute(
	_ context.Context,
	input MCPDirectorSupervisorBriefingToolInputV0,
) (MCPDirectorSupervisorBriefingToolResultV0, error) {
	briefing, err := orquestadirectorsupervisor.BuildDirectorSupervisorBriefingV0(input.BriefingInput)
	if err != nil {
		return newMCPDirectorSupervisorBriefingErrorV0(input, err), nil
	}
	return MCPDirectorSupervisorBriefingToolResultV0{
		Estado:        MCPDirectorSupervisorBriefingEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.BriefingInput.Decision.RunRef),
		RunRef:        briefing.RunRef,
		Briefing:      &briefing,
	}, nil
}

func newMCPDirectorSupervisorBriefingErrorV0(
	input MCPDirectorSupervisorBriefingToolInputV0,
	err error,
) MCPDirectorSupervisorBriefingToolResultV0 {
	issue := MCPDirectorSupervisorBriefingIssueV0{
		Code:    strings.TrimSpace(err.Error()),
		Message: strings.TrimSpace(err.Error()),
	}
	var supervisorErr orquestadirectorsupervisor.DirectorSupervisorErrorV0
	if errors.As(err, &supervisorErr) {
		issue.Code = strings.TrimSpace(supervisorErr.Code)
		issue.Field = strings.TrimSpace(supervisorErr.Field)
		issue.Message = strings.TrimSpace(supervisorErr.Message)
	}
	return MCPDirectorSupervisorBriefingToolResultV0{
		Estado:        MCPDirectorSupervisorBriefingEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.BriefingInput.Decision.RunRef),
		RunRef:        strings.TrimSpace(input.BriefingInput.Decision.RunRef),
		Errores:       []MCPDirectorSupervisorBriefingIssueV0{issue},
	}
}
