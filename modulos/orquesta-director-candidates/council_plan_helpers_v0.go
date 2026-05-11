package orquestadirectorcandidates

import orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
import "fmt"

func councilReasonCodeV0(role string) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "council_cross_critique"
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "council_vote"
	default:
		return "council_independent_proposal"
	}
}

func councilSummaryV0(role string) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "Critica consejo."
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "Voto consejo."
	default:
		return "Propuesta consejo."
	}
}

func councilAssignmentEvidenceRefsV0(
	input CouncilPlanCandidatesInputV0,
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
) []string {
	refs := make([]string, 0, 2+len(input.EvidenceRefs))
	refs = append(refs, fmt.Sprintf("c:%s:%03d", councilRoleScopeV0(assignment.Role), ordinal))
	refs = append(refs, input.EvidenceRefs...)
	return normalizeRefsV0(refs)
}
