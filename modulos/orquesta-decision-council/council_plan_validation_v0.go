package orquestadecisioncouncil

import "strings"

func normalizeDecisionCouncilPlanInputV0(input DecisionCouncilPlanInputV0) DecisionCouncilPlanInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.DecisionTopicRef = strings.TrimSpace(input.DecisionTopicRef)
	input.BrainstormRequestRef = strings.TrimSpace(input.BrainstormRequestRef)
	input.VoteRequestRef = strings.TrimSpace(input.VoteRequestRef)
	input.MinimumCapacityLevel = strings.TrimSpace(input.MinimumCapacityLevel)
	if input.MinimumCapacityLevel == "" {
		input.MinimumCapacityLevel = CouncilCapacityHighV0
	}
	if input.MinimumAgents == 0 {
		input.MinimumAgents = 3
	}
	if input.MinimumDistinctFamilies == 0 {
		input.MinimumDistinctFamilies = 2
	}
	input.RequiredFamilyRefs = compactCouncilStringsV0(input.RequiredFamilyRefs)
	input.EvidenceRefs = compactCouncilStringsV0(input.EvidenceRefs)
	for index := range input.Candidates {
		input.Candidates[index].AgentRef = strings.TrimSpace(input.Candidates[index].AgentRef)
		input.Candidates[index].FamilyRef = strings.TrimSpace(input.Candidates[index].FamilyRef)
		input.Candidates[index].CapacityLevel = strings.TrimSpace(input.Candidates[index].CapacityLevel)
	}
	return input
}

func validateDecisionCouncilPlanInputV0(input DecisionCouncilPlanInputV0) error {
	switch {
	case input.RunRef == "":
		return councilErrorV0("run_ref")
	case input.DecisionTopicRef == "":
		return councilErrorV0("decision_topic_ref")
	case input.BrainstormRequestRef == "":
		return councilErrorV0("brainstorm_request_ref")
	case input.VoteRequestRef == "":
		return councilErrorV0("vote_request_ref")
	case input.MinimumAgents < 2:
		return councilErrorV0("minimum_agents")
	case input.MinimumDistinctFamilies < 1:
		return councilErrorV0("minimum_distinct_families")
	case !validCouncilCapacityLevelV0(input.MinimumCapacityLevel):
		return councilErrorV0("minimum_capacity_level")
	case len(input.Candidates) == 0:
		return councilErrorV0("candidates")
	case len(input.EvidenceRefs) == 0:
		return councilErrorV0("evidence_refs")
	}
	for _, candidate := range input.Candidates {
		if err := validateCouncilCandidateV0(candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateCouncilCandidateV0(candidate CouncilAgentCandidateV0) error {
	switch {
	case candidate.AgentRef == "":
		return councilErrorV0("candidates.agent_ref")
	case candidate.FamilyRef == "":
		return councilErrorV0("candidates.family_ref")
	case !validCouncilCapacityLevelV0(candidate.CapacityLevel):
		return councilErrorV0("candidates.capacity_level")
	}
	return nil
}

func validateCouncilQuorumV0(
	input DecisionCouncilPlanInputV0,
	selected []CouncilAgentCandidateV0,
) error {
	if len(selected) < input.MinimumAgents {
		return councilErrorV0("minimum_agents")
	}
	families := councilFamiliesV0(selected)
	if len(families) < input.MinimumDistinctFamilies {
		return councilErrorV0("minimum_distinct_families")
	}
	for _, required := range input.RequiredFamilyRefs {
		if !families[required] {
			return councilErrorV0("required_family_refs")
		}
	}
	return nil
}

func eligibleCouncilCandidatesV0(input DecisionCouncilPlanInputV0) []CouncilAgentCandidateV0 {
	seen := map[string]bool{}
	selected := make([]CouncilAgentCandidateV0, 0, len(input.Candidates))
	minRank := councilCapacityRankV0(input.MinimumCapacityLevel)
	for _, candidate := range input.Candidates {
		if !candidate.Active || seen[candidate.AgentRef] {
			continue
		}
		if councilCapacityRankV0(candidate.CapacityLevel) < minRank {
			continue
		}
		seen[candidate.AgentRef] = true
		selected = append(selected, candidate)
	}
	return selected
}
