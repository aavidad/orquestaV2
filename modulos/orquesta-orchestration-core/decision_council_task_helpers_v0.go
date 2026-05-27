package orquestacionnucleoapp

import (
	"fmt"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func decisionCouncilTaskTitleV0(role string, ordinal int) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return fmt.Sprintf("Critica cruzada consejo %03d", ordinal)
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return fmt.Sprintf("Voto consejo %03d", ordinal)
	default:
		return fmt.Sprintf("Propuesta independiente consejo %03d", ordinal)
	}
}

func decisionCouncilTaskSummaryV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
) string {
	switch assignment.Role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "Criticar propuesta asignada por refs opacas y entregar artefacto de critica."
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "Votar opcion con evidencia y disenso durable si aplica."
	default:
		return "Proponer solucion independiente con evidencia durable."
	}
}

func decisionCouncilTaskCriteriaV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	round orquestadecisioncouncil.DecisionCouncilOperationalRoundV0,
) []string {
	return compactStringsV0([]string{
		"decision_council_role:" + assignment.Role,
		"assignment_ref:" + assignment.AssignmentRef,
		"expected_artifact:" + assignment.ExpectedArtifact,
		"gate_ref:" + round.GateRef,
		fmt.Sprintf("minimum_artifacts:%d", round.MinimumArtifacts),
		fmt.Sprintf("minimum_distinct_families:%d", round.MinimumDistinctFamilies),
		"context_policy:" + assignment.ContextPolicy,
	})
}

func decisionCouncilTaskContextRefsV0(
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	round orquestadecisioncouncil.DecisionCouncilOperationalRoundV0,
) []string {
	refs := []string{
		decisionCouncilRoleContextRefV0(assignment.Role),
		"decision-council-assignment-" + decisionCouncilRoleScopeV0(assignment.Role),
		round.GateRef,
	}
	refs = append(refs, plan.EvidenceRefs...)
	refs = append(refs, assignment.DependsOnRefs...)
	return compactStringsV0(refs)
}
