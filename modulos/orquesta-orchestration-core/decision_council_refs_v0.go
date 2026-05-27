package orquestacionnucleoapp

import (
	"fmt"
	"strings"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

const decisionCouncilContextRolePrefixV0 = "decision-council-role-"

func decisionCouncilTaskRefV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
) string {
	return fmt.Sprintf(
		"task-council-%s-%03d",
		decisionCouncilRoleScopeV0(assignment.Role),
		ordinal,
	)
}

func decisionCouncilRoleScopeV0(role string) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "c"
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "v"
	default:
		return "p"
	}
}

func decisionCouncilRoleContextRefV0(role string) string {
	return decisionCouncilContextRolePrefixV0 + decisionCouncilRoleScopeV0(role)
}

func decisionCouncilRoleFromContextRefsV0(refs []string) string {
	for _, ref := range refs {
		switch strings.TrimSpace(ref) {
		case decisionCouncilRoleContextRefV0(orquestadecisioncouncil.CouncilRoleProposalV0):
			return orquestadecisioncouncil.CouncilRoleProposalV0
		case decisionCouncilRoleContextRefV0(orquestadecisioncouncil.CouncilRoleCritiqueV0):
			return orquestadecisioncouncil.CouncilRoleCritiqueV0
		case decisionCouncilRoleContextRefV0(orquestadecisioncouncil.CouncilRoleVoteV0):
			return orquestadecisioncouncil.CouncilRoleVoteV0
		}
	}
	return ""
}

func decisionCouncilPhaseForRoleV0(role string) string {
	if role == orquestadecisioncouncil.CouncilRoleVoteV0 {
		return orquestadecisioncouncil.CouncilPhaseVotacionYDecisionV0
	}
	return orquestadecisioncouncil.CouncilPhaseBrainstormingArquitecturaV0
}
