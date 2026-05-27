package orquestacionnucleoapp

import (
	"fmt"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func decisionCouncilWorkflowTaskV0(
	request DecisionCouncilPlanMaterializeRequestV0,
	rounds orquestadecisioncouncil.DecisionCouncilOperationalRoundsV0,
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
	assignmentTaskRefs map[string]string,
	critiqueTaskRefs []string,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	round, ok := decisionCouncilRoundForRoleV0(rounds, assignment.Role)
	if !ok {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "decision_council.round", "round ausente")
	}
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:               assignmentTaskRefs[assignment.AssignmentRef],
		RunID:                request.Plan.RunRef,
		PhaseID:              orquestacoreworkflow.OrchestrationPhaseIDV0(decisionCouncilPhaseForRoleV0(assignment.Role)),
		WorkProfileKind:      decisionCouncilWorkProfileKindV0(assignment.Role),
		Title:                decisionCouncilTaskTitleV0(assignment.Role, ordinal),
		Summary:              decisionCouncilTaskSummaryV0(assignment),
		WriteSet:             []string{fmt.Sprintf("artifacts/council/%s/%03d", decisionCouncilRoleScopeV0(assignment.Role), ordinal)},
		AcceptanceCriteria:   decisionCouncilTaskCriteriaV0(assignment, round),
		DependsOn:            decisionCouncilTaskDependsOnV0(assignment, assignmentTaskRefs, critiqueTaskRefs),
		ContextRefs:          decisionCouncilTaskContextRefsV0(request.Plan, assignment, round),
		CohortRef:            round.WaitCohortRef,
		WaveRef:              round.WaitWaveRef,
		FunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.FunctionContractRefs...),
	}
	return orquestacoreworkflow.NewWorkflowTaskV0(task)
}

func decisionCouncilAssignmentTaskRefsV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
) map[string]string {
	refs := map[string]string{}
	roleOrdinals := map[string]int{}
	for _, assignment := range plan.Assignments {
		roleOrdinals[assignment.Role]++
		refs[assignment.AssignmentRef] = decisionCouncilTaskRefV0(assignment, roleOrdinals[assignment.Role])
	}
	return refs
}

func decisionCouncilTaskRefsForRoleV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	refs map[string]string,
	role string,
) []string {
	out := make([]string, 0)
	for _, assignment := range plan.Assignments {
		if assignment.Role == role {
			out = append(out, refs[assignment.AssignmentRef])
		}
	}
	return compactStringsV0(out)
}

func decisionCouncilRoundForRoleV0(
	rounds orquestadecisioncouncil.DecisionCouncilOperationalRoundsV0,
	role string,
) (orquestadecisioncouncil.DecisionCouncilOperationalRoundV0, bool) {
	for _, round := range rounds.Rounds {
		if round.Role == role {
			return round, true
		}
	}
	return orquestadecisioncouncil.DecisionCouncilOperationalRoundV0{}, false
}

func decisionCouncilWorkProfileKindV0(role string) orquestacoreworkflow.WorkProfileKindV0 {
	if role == orquestadecisioncouncil.CouncilRoleVoteV0 {
		return orquestacoreworkflow.WorkProfileReviewV0
	}
	return orquestacoreworkflow.WorkProfileCodeStudyV0
}

func decisionCouncilTaskDependsOnV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	refs map[string]string,
	critiqueTaskRefs []string,
) []string {
	out := make([]string, 0, len(assignment.DependsOnRefs)+len(critiqueTaskRefs))
	for _, ref := range assignment.DependsOnRefs {
		if taskRef := refs[ref]; taskRef != "" {
			out = append(out, taskRef)
		}
	}
	if assignment.Role == orquestadecisioncouncil.CouncilRoleVoteV0 {
		out = append(out, critiqueTaskRefs...)
	}
	return compactStringsV0(out)
}
