package orquestamcp

import (
	"context"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
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
	RequestID        string                                                     `json:"request_id,omitempty"`
	CorrelationID    string                                                     `json:"correlation_id,omitempty"`
	WorkIntake       orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0 `json:"work_intake"`
	WorktreeIsolated bool                                                       `json:"worktree_isolated,omitempty"`
}

type MCPHumanDirectorWorkReviewPlanToolResultV0 struct {
	Estado                 string                                                  `json:"estado"`
	RequestID              string                                                  `json:"request_id,omitempty"`
	CorrelationID          string                                                  `json:"correlation_id,omitempty"`
	Accepted               bool                                                    `json:"accepted"`
	Plan                   orquestaappdirectorintake.HumanDirectorReviewablePlanV0 `json:"plan,omitempty"`
	AutoprogrammingRequest *orquestaautoprogramming.AutoprogrammingRequestV0       `json:"autoprogramming_request,omitempty"`
	NextActions            []string                                                `json:"next_actions,omitempty"`
	Errores                []MCPValidationIssueV0                                  `json:"errores_publicos,omitempty"`
}

type MCPHumanDirectorWorkReviewPlanToolExecutorV0 struct{}

func MCPHumanDirectorWorkReviewPlanDescriptorV0() MCPHumanDirectorWorkReviewPlanToolDescriptorV0 {
	return MCPHumanDirectorWorkReviewPlanToolDescriptorV0{
		Name:        MCPHumanDirectorWorkReviewPlanToolNameV0,
		Version:     MCPHumanDirectorWorkReviewPlanToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,worktree_isolated?,work_intake:HumanDirectorWorkIntakeRequestV0}",
		Output:      "ok:{plan,next_actions,autoprogramming_request?}|error:{errores_publicos,plan?}",
		ResourceURI: MCPHumanDirectorWorkReviewPlanResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino para orden humana amplia",
			"construye plan revisable; no arranca agentes ni ejecuta runtime",
			"proyecta prepare-run solo cuando hay execute_now y worktree aislada declarada",
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
	return MCPHumanDirectorWorkReviewPlanToolResultV0{
		Estado:                 MCPHumanDirectorWorkReviewPlanEstadoOKV0,
		RequestID:              strings.TrimSpace(result.Plan.RequestRef),
		CorrelationID:          firstNonEmptyMCPV0(input.CorrelationID, request.CorrelationID, input.RequestID, request.RequestRef),
		Accepted:               true,
		Plan:                   result.Plan,
		AutoprogrammingRequest: autoprogrammingRequest,
		NextActions:            compactStringsMCPV0(next),
	}, nil
}

func humanDirectorWorkIntakeFromMCPV0(
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
) orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0 {
	request := input.WorkIntake
	if strings.TrimSpace(request.SchemaVersion) == "" {
		request.SchemaVersion = orquestaappdirectorintake.HumanDirectorWorkIntakeSchemaVersionV0
	}
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, request.RequestRef)
	}
	return request
}

func humanDirectorPlanIssuesMCPV0(
	issues []orquestaappdirectorintake.HumanDirectorPlanIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(issue.Detail, issue.Code)),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}

func humanDirectorAutoprogrammingRequestMCPV0(
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
) (*orquestaautoprogramming.AutoprogrammingRequestV0, []string) {
	executeSteps := humanDirectorExecutableStepsMCPV0(plan)
	if len(executeSteps) == 0 {
		return nil, humanDirectorReviewNextActionsMCPV0(plan, "study_before_prepare_run")
	}
	if !input.WorktreeIsolated {
		return nil, humanDirectorReviewNextActionsMCPV0(plan, "declare_worktree_isolated_before_prepare_run")
	}
	if strings.TrimSpace(plan.WorktreeRef) == "" || strings.TrimSpace(plan.BranchRef) == "" {
		return nil, humanDirectorReviewNextActionsMCPV0(plan, "complete_worktree_ref_and_branch_ref_before_prepare_run")
	}
	request := orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:         strings.TrimSpace(plan.RequestRef),
		ProjectRef:         strings.TrimSpace(plan.ProjectRef),
		WorktreeRef:        strings.TrimSpace(plan.WorktreeRef),
		WorktreeIsolated:   true,
		BranchRef:          strings.TrimSpace(plan.BranchRef),
		MaxTaskRefs:        positiveOrDefaultMCPHumanWorkV0(plan.Limits.MaxAgents, len(executeSteps)),
		MaxAreas:           positiveOrDefaultMCPHumanWorkV0(plan.Limits.MaxFanout, len(executeSteps)),
		MaxWriteSetEntries: positiveOrDefaultMCPHumanWorkV0(plan.Limits.MaxWriteSetEntries, len(executeSteps)*4),
	}
	for _, step := range executeSteps {
		request.Tasks = append(request.Tasks, humanDirectorTaskCandidateMCPV0(plan, step))
		request.WriteSet = append(request.WriteSet, step.WriteSet...)
		request.RequiredTests = append(request.RequiredTests, step.RequiredTests...)
	}
	request.WriteSet = compactStringsMCPV0(request.WriteSet)
	request.RequiredTests = compactStringsMCPV0(request.RequiredTests)
	return &request, humanDirectorReviewNextActionsMCPV0(
		plan,
		"review_plan",
		"post_/api/v0/autoprogramming/prepare-run_with_autoprogramming_request",
	)
}

func humanDirectorExecutableStepsMCPV0(
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
) []orquestaappdirectorintake.HumanDirectorPlanStepV0 {
	var steps []orquestaappdirectorintake.HumanDirectorPlanStepV0
	for _, step := range plan.Steps {
		if strings.TrimSpace(step.Action) == orquestaappdirectorintake.HumanDirectorPlanActionExecuteNowV0 {
			steps = append(steps, step)
		}
	}
	return steps
}

func humanDirectorTaskCandidateMCPV0(
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
	step orquestaappdirectorintake.HumanDirectorPlanStepV0,
) orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	return orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
		TaskRef:            strings.TrimSpace(step.StepRef),
		Area:               firstNonEmptyMCPV0(step.Area, "programacion"),
		Title:              strings.TrimSpace(step.Title),
		Objective:          strings.TrimSpace(step.Objective),
		Context:            compactStringsMCPV0([]string{step.Reason}),
		ContextRefs:        compactStringsMCPV0(append(append([]string(nil), plan.ContextRefs...), step.ContextRefs...)),
		AcceptanceCriteria: compactStringsMCPV0(step.AcceptanceCriteria),
		RequiredTests:      compactStringsMCPV0(step.RequiredTests),
		CompactRules:       compactStringsMCPV0(plan.Rules),
	}
}

func positiveOrDefaultMCPHumanWorkV0(value int, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func humanDirectorReviewNextActionsMCPV0(
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
	actions ...string,
) []string {
	out := append([]string(nil), actions...)
	if humanDirectorPlanHasActionMCPV0(plan, orquestaappdirectorintake.HumanDirectorPlanActionRequestReviewV0) {
		out = append(out, "use_orquesta.operator.directed_query.v0_for_human_bridge_if_needed")
	}
	if humanDirectorPlanHasActionMCPV0(plan, orquestaappdirectorintake.HumanDirectorPlanActionPostponeOverlapV0) {
		out = append(out, "postpone_overlapped_write_set_instead_of_overwriting")
	}
	return compactStringsMCPV0(out)
}

func humanDirectorPlanHasActionMCPV0(
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
	action string,
) bool {
	for _, step := range plan.Steps {
		if strings.TrimSpace(step.Action) == action {
			return true
		}
	}
	return false
}
