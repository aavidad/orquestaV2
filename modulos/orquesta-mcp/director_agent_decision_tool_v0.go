package orquestamcp

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

const (
	MCPDirectorAgentDecisionToolNameV0    = "orquesta.director_agent.apply_decision.v0"
	MCPDirectorAgentDecisionToolVersionV0 = "v0"
	MCPDirectorAgentDecisionResourceURIV0 = "orquesta://contracts/director-agent-decision/v0"
	MCPDirectorAgentDecisionEstadoOKV0    = "ok"
	MCPDirectorAgentDecisionEstadoErrorV0 = "error"
)

type MCPDirectorAgentDecisionToolDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	InputSchema string `json:"input_schema"`
	Output      string `json:"output"`
	ResourceURI string `json:"resource_uri"`
}

type MCPDirectorAgentDecisionToolInputV0 struct {
	RequestID     string                                        `json:"request_id,omitempty"`
	CorrelationID string                                        `json:"correlation_id,omitempty"`
	OccurredAt    string                                        `json:"occurred_at"`
	Decision      orquestadirectoragent.DirectorAgentDecisionV0 `json:"decision"`
}

type MCPDirectorAgentDecisionToolResultV0 struct {
	Estado        string                            `json:"estado"`
	RequestID     string                            `json:"request_id,omitempty"`
	CorrelationID string                            `json:"correlation_id,omitempty"`
	RunRef        string                            `json:"run_ref,omitempty"`
	CurrentPhase  string                            `json:"current_phase,omitempty"`
	Command       MCPCoreWorkflowCommandCompactV0   `json:"command,omitempty"`
	EventsCount   int                               `json:"events_count,omitempty"`
	Idempotent    bool                              `json:"idempotent,omitempty"`
	Errores       []MCPDirectorAgentDecisionIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPDirectorAgentDecisionIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func MCPDirectorAgentDecisionDescriptorV0() MCPDirectorAgentDecisionToolDescriptorV0 {
	return MCPDirectorAgentDecisionToolDescriptorV0{
		Name:        MCPDirectorAgentDecisionToolNameV0,
		Version:     MCPDirectorAgentDecisionToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,occurred_at,decision:DirectorAgentDecisionV0}",
		Output:      "ok:{run_ref,current_phase,command,events_count}|error:{errores_publicos}",
		ResourceURI: MCPDirectorAgentDecisionResourceURIV0,
	}
}

func ToApplyDirectorAgentDecisionRequestV0(
	input MCPDirectorAgentDecisionToolInputV0,
) orquestadirectoragentworkflow.ApplyDirectorAgentDecisionRequestV0 {
	return orquestadirectoragentworkflow.ApplyDirectorAgentDecisionRequestV0{
		Decision:      input.Decision,
		OccurredAt:    strings.TrimSpace(input.OccurredAt),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
		RequestedBy:   "orquesta-mcp",
	}
}

func NewMCPDirectorAgentDecisionResultV0(
	input MCPDirectorAgentDecisionToolInputV0,
	result orquestadirectoragentworkflow.ApplyDirectorAgentDecisionResultV0,
) MCPDirectorAgentDecisionToolResultV0 {
	if len(result.Issues) > 0 {
		return MCPDirectorAgentDecisionToolResultV0{
			Estado:        MCPDirectorAgentDecisionEstadoErrorV0,
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Decision.DecisionRef),
			Errores:       directorAgentDecisionIssuesMCPV0(result.Issues),
		}
	}
	return MCPDirectorAgentDecisionToolResultV0{
		Estado:        MCPDirectorAgentDecisionEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Decision.DecisionRef),
		RunRef:        strings.TrimSpace(result.Run.RunID),
		CurrentPhase:  strings.TrimSpace(string(result.Run.CurrentPhase)),
		Command:       compactCoreWorkflowCommandMCPV0(result.Command),
		EventsCount:   result.EventsCount,
		Idempotent:    result.Idempotent,
	}
}

func NewMCPDirectorAgentDecisionErrorV0(
	input MCPDirectorAgentDecisionToolInputV0,
	err error,
) MCPDirectorAgentDecisionToolResultV0 {
	return MCPDirectorAgentDecisionToolResultV0{
		Estado:        MCPDirectorAgentDecisionEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Decision.DecisionRef),
		Errores: []MCPDirectorAgentDecisionIssueV0{{
			Code: strings.TrimSpace(err.Error()),
		}},
	}
}

func directorAgentDecisionIssuesMCPV0(
	issues []orquestadirectoragentworkflow.DirectorAgentWorkflowIssueV0,
) []MCPDirectorAgentDecisionIssueV0 {
	out := make([]MCPDirectorAgentDecisionIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPDirectorAgentDecisionIssueV0{
			Code:  strings.TrimSpace(issue.Code),
			Field: strings.TrimSpace(issue.Field),
		})
	}
	return out
}
