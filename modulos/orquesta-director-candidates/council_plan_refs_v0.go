package orquestadirectorcandidates

import (
	"fmt"
	"strings"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type councilDerivedRefsDataV0 struct {
	candidateRef           string
	taskRef                string
	claimRef               string
	capacityRequestRef     string
	agentRequestRef        string
	capacityCommandRef     string
	capacityIdempotencyRef string
	gateCommandRef         string
	gateIdempotencyRef     string
	agentCommandRef        string
	agentIdempotencyRef    string
}

func councilDerivedRefsV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
) councilDerivedRefsDataV0 {
	base := fmt.Sprintf("c:%s:%03d", councilRoleScopeV0(assignment.Role), ordinal)
	return councilDerivedRefsDataV0{
		candidateRef:           base + ":cand",
		taskRef:                base + ":task",
		claimRef:               base + ":claim",
		capacityRequestRef:     base + ":cap",
		agentRequestRef:        base + ":agent",
		capacityCommandRef:     base + ":ccap",
		capacityIdempotencyRef: base + ":icap",
		gateCommandRef:         base + ":cgate",
		gateIdempotencyRef:     base + ":igate",
		agentCommandRef:        base + ":cagent",
		agentIdempotencyRef:    base + ":iagent",
	}
}

func councilReadScopesV0(role string) []string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return []string{councilArtifactScopeV0("p")}
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return []string{
			councilArtifactScopeV0("p"),
			councilArtifactScopeV0("c"),
		}
	default:
		return nil
	}
}

func councilWriteScopeV0(
	assignment orquestadecisioncouncil.CouncilAssignmentV0,
	ordinal int,
) string {
	return councilArtifactScopeV0(councilRoleScopeV0(assignment.Role)) +
		fmt.Sprintf("/%03d", ordinal)
}

func councilArtifactScopeV0(role string) string {
	return "artifacts/council/" + safeCouncilScopePartV0(role)
}

func councilRoleScopeV0(role string) string {
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		return "c"
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		return "v"
	default:
		return "p"
	}
}

func safeCouncilScopePartV0(value string) string {
	replacer := strings.NewReplacer(
		":", "-",
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\\", "-",
		"/", "-",
	)
	value = strings.Trim(replacer.Replace(strings.TrimSpace(value)), "-")
	if value == "" {
		return "unknown"
	}
	return value
}
