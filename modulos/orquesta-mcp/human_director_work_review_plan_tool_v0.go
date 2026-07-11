package orquestamcp

import (
	"context"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

const (
	MCPHumanDirectorWorkReviewPlanToolNameV0    = "orquesta.director.human_work.review_plan.v0"
	MCPHumanDirectorWorkReviewPlanToolVersionV0 = "v0"
	MCPHumanDirectorWorkReviewPlanResourceURIV0 = "orquesta://contracts/director-human-work-review-plan/v0"
	MCPHumanDirectorWorkReviewPlanEstadoOKV0    = "ok"
	MCPHumanDirectorWorkReviewPlanEstadoErrorV0 = "error"
)

type MCPHumanDirectorWorkReviewPlanToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPHumanDirectorWorkReviewPlanToolInputV0 struct {
	RequestID             string                                                     `json:"request_id,omitempty"`
	CorrelationID         string                                                     `json:"correlation_id,omitempty"`
	WorkIntake            orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0 `json:"work_intake"`
	WorktreeIsolated      bool                                                       `json:"worktree_isolated,omitempty"`
	RaiseOperatorQuestion bool                                                       `json:"raise_operator_question,omitempty"`
	OperatorQuery         operator.OperatorDirectedQueryV0                           `json:"operator_query,omitempty"`
}

type MCPHumanDirectorWorkReviewPlanToolResultV0 struct {
	Estado                 string                                                  `json:"estado"`
	RequestID              string                                                  `json:"request_id,omitempty"`
	CorrelationID          string                                                  `json:"correlation_id,omitempty"`
	Accepted               bool                                                    `json:"accepted"`
	Plan                   orquestaappdirectorintake.HumanDirectorReviewablePlanV0 `json:"plan,omitempty"`
	AutoprogrammingRequest *orquestaautoprogramming.AutoprogrammingRequestV0       `json:"autoprogramming_request,omitempty"`
	OperatorQuestion       *operator.OperatorMCPDirectedQueryResultV0              `json:"operator_question,omitempty"`
	NextActions            []string                                                `json:"next_actions,omitempty"`
	Errores                []MCPValidationIssueV0                                  `json:"errores_publicos,omitempty"`
}

type MCPHumanDirectorWorkReviewPlanToolExecutorV0 struct {
	OperatorQuery operator.OperatorMCPDirectedQueryPortV0
}

func MCPHumanDirectorWorkReviewPlanDescriptorV0() MCPHumanDirectorWorkReviewPlanToolDescriptorV0 {
	return MCPHumanDirectorWorkReviewPlanToolDescriptorV0{
		Name:        MCPHumanDirectorWorkReviewPlanToolNameV0,
		Version:     MCPHumanDirectorWorkReviewPlanToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,worktree_isolated?,raise_operator_question?,operator_query?,work_intake:{request:{acceptance_checks?:[{criterion_ref,description?,command}]},...}}",
		Output:      "ok:{plan,next_actions,autoprogramming_request?,operator_question?}|error:{errores_publicos,plan?}",
		ResourceURI: MCPHumanDirectorWorkReviewPlanResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino para orden humana amplia",
			"construye plan revisable; no arranca agentes ni ejecuta runtime",
			"proyecta prepare-run solo cuando hay execute_now y worktree aislada declarada",
			"solo eleva pregunta humana si se solicita y existe puerto operador inyectado",
			"refs opacas; sin adaptadores concretos en el contrato",
		},
	}
}

func (executor MCPHumanDirectorWorkReviewPlanToolExecutorV0) Execute(
	ctx context.Context,
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
) (MCPHumanDirectorWorkReviewPlanToolResultV0, error) {
	_ = ctx
	request := humanDirectorWorkIntakeFromMCPV0(input)
	result := orquestaappdirectorintake.BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		return MCPHumanDirectorWorkReviewPlanToolResultV0{
			Estado:        MCPHumanDirectorWorkReviewPlanEstadoErrorV0,
			RequestID:     strings.TrimSpace(request.RequestRef),
			CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, request.CorrelationID, input.RequestID, request.RequestRef),
			Accepted:      false,
			Plan:          result.Plan,
			Errores:       humanDirectorPlanIssuesMCPV0(result.Issues),
		}, nil
	}
	autoprogrammingRequest, next := humanDirectorAutoprogrammingRequestMCPV0(input, result.Plan)
	out := MCPHumanDirectorWorkReviewPlanToolResultV0{
		Estado:                 MCPHumanDirectorWorkReviewPlanEstadoOKV0,
		RequestID:              strings.TrimSpace(result.Plan.RequestRef),
		CorrelationID:          firstNonEmptyMCPV0(input.CorrelationID, request.CorrelationID, input.RequestID, request.RequestRef),
		Accepted:               true,
		Plan:                   result.Plan,
		AutoprogrammingRequest: autoprogrammingRequest,
		NextActions:            compactStringsMCPV0(next),
	}
	executor.maybeRaiseHumanDirectorOperatorQuestionV0(input, &out)
	return out, nil
}

func (executor MCPHumanDirectorWorkReviewPlanToolExecutorV0) maybeRaiseHumanDirectorOperatorQuestionV0(
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
	out *MCPHumanDirectorWorkReviewPlanToolResultV0,
) {
	if !input.RaiseOperatorQuestion || out == nil {
		return
	}
	query := humanDirectorOperatorQueryMCPV0(input, out.Plan)
	if executor.OperatorQuery == nil {
		out.NextActions = compactStringsMCPV0(append(out.NextActions,
			"repair_configure_operator_directed_query_port",
			"use_orquesta.operator.directed_query.v0_for_human_bridge_if_needed",
		))
		out.Errores = append(out.Errores, MCPValidationIssueV0{
			Code:    operator.ErrOperatorMCPPortUnavailableV0,
			Field:   "operator_query",
			Message: "operator directed query port no configurado",
		})
		return
	}
	result := ExecuteMCPOperatorDirectedQueryToolV0(executor.OperatorQuery, query)
	if result.Estado == MCPOperatorToolEstadoOKV0 && result.DirectedQuery != nil {
		out.OperatorQuestion = result.DirectedQuery
		out.NextActions = compactStringsMCPV0(append(out.NextActions, "await_operator_answer_ref"))
		return
	}
	out.NextActions = compactStringsMCPV0(append(out.NextActions, "repair_operator_directed_query_request"))
	out.Errores = append(out.Errores, operatorIssuesAsMCPV0(result)...)
}
