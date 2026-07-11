package orquestamcp

import (
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

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

func humanDirectorOperatorQueryMCPV0(
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
	plan orquestaappdirectorintake.HumanDirectorReviewablePlanV0,
) operator.OperatorDirectedQueryV0 {
	query := input.OperatorQuery
	query.QueryRef = firstNonEmptyMCPV0(query.QueryRef, "query-ref-"+plan.RequestRef)
	query.TargetRef = firstNonEmptyMCPV0(query.TargetRef, plan.ProjectRef, plan.RequestRef)
	query.Question = firstNonEmptyMCPV0(query.Question, "Revisar plan humano "+plan.RequestRef+" y responder con decision operativa.")
	query.EvidenceRefs = compactStringsMCPV0(append(query.EvidenceRefs, plan.ContextRefs...))
	return query
}

func operatorIssuesAsMCPV0(result MCPOperatorToolResultV0) []MCPValidationIssueV0 {
	if len(result.Issues) == 0 && strings.TrimSpace(result.ErrorCode) != "" {
		return []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(result.ErrorCode),
			Field:   "operator_query",
			Message: strings.TrimSpace(result.ErrorCode),
		}}
	}
	out := make([]MCPValidationIssueV0, 0, len(result.Issues))
	for _, issue := range result.Issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Code),
		})
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
		AcceptanceChecks:   humanDirectorAcceptanceChecksMCPV0(step.AcceptanceChecks),
		RequiredTests:      compactStringsMCPV0(step.RequiredTests),
		CompactRules:       compactStringsMCPV0(plan.Rules),
	}
}

func humanDirectorAcceptanceChecksMCPV0(
	checks []orquestaappdirectorintake.HumanDirectorAcceptanceCheckV0,
) []orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0 {
	if checks == nil {
		return nil
	}
	out := make([]orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0, len(checks))
	for index, check := range checks {
		out[index] = orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0{
			CriterionRef: check.CriterionRef,
			Description:  check.Description,
			Command:      check.Command,
		}
	}
	return out
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
