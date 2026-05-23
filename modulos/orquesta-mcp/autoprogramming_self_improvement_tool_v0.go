package orquestamcp

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const (
	MCPAutoprogrammingSelfImprovementToolNameV0    = "orquesta.autoprogramming.self_improvement.propose.v0"
	MCPAutoprogrammingSelfImprovementToolVersionV0 = "v0"
	MCPAutoprogrammingSelfImprovementResourceURIV0 = "orquesta://contracts/autoprogramming-self-improvement/v0"
	MCPAutoprogrammingSelfImprovementEstadoOKV0    = "ok"
	MCPAutoprogrammingSelfImprovementEstadoErrorV0 = "error"
)

type MCPAutoprogrammingSelfImprovementToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingSelfImprovementToolInputV0 struct {
	RequestID     string                                                           `json:"request_id,omitempty"`
	CorrelationID string                                                           `json:"correlation_id,omitempty"`
	Proposal      orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 `json:"proposal"`
}

type MCPAutoprogrammingSelfImprovementToolResultV0 struct {
	Estado                 string                                            `json:"estado"`
	RequestID              string                                            `json:"request_id,omitempty"`
	CorrelationID          string                                            `json:"correlation_id,omitempty"`
	Accepted               bool                                              `json:"accepted"`
	Background             bool                                              `json:"background"`
	PriorityScore          int                                               `json:"priority_score"`
	AutoprogrammingRequest *orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request,omitempty"`
	PrepareRun             *MCPAutoprogrammingPrepareRunToolInputV0          `json:"prepare_run,omitempty"`
	NextActions            []string                                          `json:"next_actions,omitempty"`
	Errores                []MCPValidationIssueV0                            `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingSelfImprovementToolExecutorV0 struct{}

func MCPAutoprogrammingSelfImprovementDescriptorV0() MCPAutoprogrammingSelfImprovementToolDescriptorV0 {
	return MCPAutoprogrammingSelfImprovementToolDescriptorV0{
		Name:        MCPAutoprogrammingSelfImprovementToolNameV0,
		Version:     MCPAutoprogrammingSelfImprovementToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,proposal:AutoprogrammingSelfImprovementProposalV0}",
		Output:      "ok:{autoprogramming_request,priority_score,prepare_run,next_actions}|error:{errores_publicos,next_actions}",
		ResourceURI: MCPAutoprogrammingSelfImprovementResourceURIV0,
		Invariantes: []string{
			"convierte fallos observados en automejora secundaria",
			"no ejecuta agentes ni encola por si mismo; prepare-run queda como siguiente paso",
			"prioridad baja por defecto para no bloquear trabajo principal",
			"hexagonal: solo refs opacas, evidencia y write-set propio",
		},
	}
}

func (executor MCPAutoprogrammingSelfImprovementToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingSelfImprovementToolInputV0,
) (MCPAutoprogrammingSelfImprovementToolResultV0, error) {
	_ = ctx
	proposal := input.Proposal
	if strings.TrimSpace(proposal.RequestRef) == "" {
		proposal.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	result := orquestaautoprogramming.BuildAutoprogrammingSelfImprovementRequestV0(proposal)
	out := MCPAutoprogrammingSelfImprovementToolResultV0{
		Estado:        MCPAutoprogrammingSelfImprovementEstadoOKV0,
		RequestID:     strings.TrimSpace(result.Request.RequestRef),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, proposal.RequestRef),
		Accepted:      result.Accepted,
		Background:    result.Background,
		PriorityScore: result.PriorityScore,
		NextActions:   compactStringsMCPV0(result.NextActions),
	}
	if !result.Accepted {
		out.Estado = MCPAutoprogrammingSelfImprovementEstadoErrorV0
		out.RequestID = strings.TrimSpace(proposal.RequestRef)
		out.Errores = selfImprovementIssuesMCPV0(result.Issues)
		return out, nil
	}
	out.AutoprogrammingRequest = &result.Request
	out.PrepareRun = &MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              strings.TrimSpace(result.Request.RequestRef),
		CorrelationID:          out.CorrelationID,
		RequestedBy:            firstNonEmptyMCPV0(proposal.ObservedBy, "orquesta-autoprogramming-self-improvement"),
		AutoprogrammingRequest: result.Request,
		PriorityScore:          result.PriorityScore,
	}
	return out, nil
}

func selfImprovementIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
