package orquestadecisioncouncil

import "strings"

func councilErrorV0(field string) DecisionCouncilErrorV0 {
	return DecisionCouncilErrorV0{
		Code:    ErrDecisionCouncilInvalidV0,
		Field:   field,
		Message: field,
	}
}

func compactCouncilStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func stringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}

func validCouncilCapacityLevelV0(level string) bool {
	return councilCapacityRankV0(level) > 0
}

func councilCapacityRankV0(level string) int {
	switch level {
	case CouncilCapacityLowV0:
		return 1
	case CouncilCapacityMediumV0:
		return 2
	case CouncilCapacityHighV0:
		return 3
	case CouncilCapacityXHighV0:
		return 4
	default:
		return 0
	}
}

func councilFamiliesV0(candidates []CouncilAgentCandidateV0) map[string]bool {
	out := map[string]bool{}
	for _, candidate := range candidates {
		out[candidate.FamilyRef] = true
	}
	return out
}

func councilVoteFamiliesV0(votes []CouncilVoteV0) map[string]bool {
	out := map[string]bool{}
	for _, vote := range votes {
		out[vote.FamilyRef] = true
	}
	return out
}

func critiqueTargetCandidateV0(
	selected []CouncilAgentCandidateV0,
	index int,
) CouncilAgentCandidateV0 {
	current := selected[index]
	for offset := 1; offset < len(selected); offset++ {
		candidate := selected[(index+offset)%len(selected)]
		if candidate.FamilyRef != current.FamilyRef {
			return candidate
		}
	}
	return selected[(index+1)%len(selected)]
}
