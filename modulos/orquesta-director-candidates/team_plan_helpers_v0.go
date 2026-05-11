package orquestadirectorcandidates

import (
	"strings"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type teamShapeV0 struct {
	capacityLevel string
	agents        int
	families      int
}

func normalizeTeamComplexityV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "l", "simple":
		return TeamComplexityLowV0
	case "medium", "m", "standard", "normal":
		return TeamComplexityMediumV0
	case "high", "h", "complex":
		return TeamComplexityHighV0
	case "xhigh", "xl", "critical":
		return TeamComplexityXHighV0
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func validTeamComplexityV0(value string) bool {
	switch value {
	case TeamComplexityLowV0,
		TeamComplexityMediumV0,
		TeamComplexityHighV0,
		TeamComplexityXHighV0:
		return true
	default:
		return false
	}
}

func teamShapeForComplexityV0(complexity string) teamShapeV0 {
	switch complexity {
	case TeamComplexityLowV0:
		return teamShapeV0{capacityLevel: orquestadecisioncouncil.CouncilCapacityLowV0, agents: 2, families: 2}
	case TeamComplexityMediumV0:
		return teamShapeV0{capacityLevel: orquestadecisioncouncil.CouncilCapacityMediumV0, agents: 3, families: 2}
	case TeamComplexityXHighV0:
		return teamShapeV0{capacityLevel: orquestadecisioncouncil.CouncilCapacityXHighV0, agents: 5, families: 3}
	default:
		return teamShapeV0{capacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, agents: 4, families: 3}
	}
}
