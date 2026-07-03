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
	RequestID             string                                                           `json:"request_id,omitempty"`
	CorrelationID         string                                                           `json:"correlation_id,omitempty"`
	DirectorExecutionMode string                                                           `json:"director_execution_mode,omitempty"`
	AutoPrepareRun        bool                                                             `json:"auto_prepare_run,omitempty"`
	OperatorAdvice        []MCPAutoprogrammingOperatorAdviceV0                             `json:"operator_advice,omitempty"`
	Proposal              orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 `json:"proposal"`
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
	PreparedRun            *MCPAutoprogrammingPrepareRunToolResultV0         `json:"prepared_run,omitempty"`
	OperatorAdvice         []MCPAutoprogrammingOperatorAdviceV0              `json:"operator_advice,omitempty"`
	NextActions            []string                                          `json:"next_actions,omitempty"`
	Errores                []MCPValidationIssueV0                            `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingSelfImprovementToolExecutorV0 struct {
	PrepareRun MCPTransportAutoprogrammingPrepareRunExecutorV0
}

func NewMCPAutoprogrammingSelfImprovementToolExecutorV0(
	prepareRun MCPTransportAutoprogrammingPrepareRunExecutorV0,
) MCPAutoprogrammingSelfImprovementToolExecutorV0 {
	return MCPAutoprogrammingSelfImprovementToolExecutorV0{PrepareRun: prepareRun}
}

func MCPAutoprogrammingSelfImprovementDescriptorV0() MCPAutoprogrammingSelfImprovementToolDescriptorV0 {
	return MCPAutoprogrammingSelfImprovementToolDescriptorV0{
		Name:        MCPAutoprogrammingSelfImprovementToolNameV0,
		Version:     MCPAutoprogrammingSelfImprovementToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,director_execution_mode?:goal_first|legacy_director_loop,auto_prepare_run?,operator_advice?,proposal:AutoprogrammingSelfImprovementProposalV0}",
		Output:      "ok:{accepted,background,autoprogramming_request,priority_score,prepare_run,prepared_run?,operator_advice?,next_actions}|error:{accepted:false,background?,errores_publicos,operator_advice?,next_actions}",
		ResourceURI: MCPAutoprogrammingSelfImprovementResourceURIV0,
		Invariantes: []string{
			"convierte fallos observados en automejora secundaria",
			"no sustituye la ruta goal-first ni supervision de estado",
			"solo prepara run si auto_prepare_run y executor inyectado existen",
			"prioridad baja por defecto para no bloquear trabajo principal",
			"operator_advice es observacion no bloqueante y no decide runtime",
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
	operatorAdvice := normalizeMCPAutoprogrammingOperatorAdviceV0(
		input.OperatorAdvice,
		firstNonEmptyMCPV0(proposal.RequestRef, input.RequestID, input.CorrelationID),
	)
	proposal = withMCPAutoprogrammingOperatorAdviceV0(proposal, operatorAdvice)
	result := orquestaautoprogramming.BuildAutoprogrammingSelfImprovementRequestV0(proposal)
	out := MCPAutoprogrammingSelfImprovementToolResultV0{
		Estado:         MCPAutoprogrammingSelfImprovementEstadoOKV0,
		RequestID:      strings.TrimSpace(result.Request.RequestRef),
		CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, proposal.RequestRef),
		Accepted:       result.Accepted,
		Background:     result.Background,
		PriorityScore:  result.PriorityScore,
		OperatorAdvice: operatorAdvice,
		NextActions:    compactStringsMCPV0(result.NextActions),
	}
	if len(out.OperatorAdvice) > 0 {
		out.NextActions = compactStringsMCPV0(append(out.NextActions, "operator_advice_recorded_non_blocking"))
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
		DirectorExecutionMode:  strings.TrimSpace(input.DirectorExecutionMode),
		AutoprogrammingRequest: result.Request,
		PriorityScore:          result.PriorityScore,
	}
	executor.maybePrepareSelfImprovementRunV0(ctx, input, &out)
	return out, nil
}

func (executor MCPAutoprogrammingSelfImprovementToolExecutorV0) maybePrepareSelfImprovementRunV0(
	ctx context.Context,
	input MCPAutoprogrammingSelfImprovementToolInputV0,
	out *MCPAutoprogrammingSelfImprovementToolResultV0,
) {
	if !input.AutoPrepareRun || out == nil || out.PrepareRun == nil {
		return
	}
	if executor.PrepareRun == nil {
		out.NextActions = compactStringsMCPV0(append(out.NextActions,
			"repair_configure_autoprogramming_prepare_run_executor",
			"post_/api/v0/autoprogramming/prepare-run_with_prepare_run",
		))
		out.Errores = append(out.Errores, MCPValidationIssueV0{
			Code:    "autoprogramming_prepare_run_port_unavailable",
			Field:   "auto_prepare_run",
			Message: "prepare-run executor no configurado",
		})
		return
	}
	prepared, err := executor.PrepareRun.Execute(ctx, *out.PrepareRun)
	if err != nil {
		prepared = NewMCPAutoprogrammingPrepareRunErrorResultV0(
			*out.PrepareRun,
			"autoprogramming_prepare_run_error",
			"auto_prepare_run",
			err.Error(),
		)
	}
	out.PreparedRun = &prepared
	if prepared.Estado == MCPAutoprogrammingPrepareRunEstadoErrorV0 {
		out.NextActions = compactStringsMCPV0(append(out.NextActions,
			"repair_autoprogramming_prepare_run_result",
		))
		return
	}
	out.NextActions = compactStringsMCPV0(append(
		out.NextActions,
		selfImprovementPreparedRunNextActionsMCPV0(prepared)...,
	))
}

func selfImprovementPreparedRunNextActionsMCPV0(
	prepared MCPAutoprogrammingPrepareRunToolResultV0,
) []string {
	if prepared.Goal != nil || len(prepared.Goals) > 0 {
		return []string{"observe_autoprogramming_goal"}
	}
	if len(prepared.GoalSpecs) > 0 {
		return []string{"handoff_goal_first_specs_to_internal_backend"}
	}
	if strings.TrimSpace(prepared.RunRef) != "" {
		return []string{"supervise_prepared_run_by_run_ref"}
	}
	return []string{"inspect_autoprogramming_prepare_run_result"}
}
