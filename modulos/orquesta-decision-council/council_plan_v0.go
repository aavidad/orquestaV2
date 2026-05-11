package orquestadecisioncouncil

import "fmt"

func BuildDecisionCouncilPlanV0(input DecisionCouncilPlanInputV0) (DecisionCouncilPlanV0, error) {
	input = normalizeDecisionCouncilPlanInputV0(input)
	if err := validateDecisionCouncilPlanInputV0(input); err != nil {
		return DecisionCouncilPlanV0{}, err
	}
	selected := eligibleCouncilCandidatesV0(input)
	if err := validateCouncilQuorumV0(input, selected); err != nil {
		return DecisionCouncilPlanV0{}, err
	}
	return DecisionCouncilPlanV0{
		RunRef:           input.RunRef,
		DecisionTopicRef: input.DecisionTopicRef,
		Assignments:      buildCouncilAssignmentsV0(input, selected),
		Gates:            buildCouncilGatesV0(input),
		SelectedAgents:   selected,
		EvidenceRefs:     compactCouncilStringsV0(input.EvidenceRefs),
	}, nil
}

func buildCouncilAssignmentsV0(
	input DecisionCouncilPlanInputV0,
	selected []CouncilAgentCandidateV0,
) []CouncilAssignmentV0 {
	assignments := make([]CouncilAssignmentV0, 0, len(selected)*3)
	for _, candidate := range selected {
		assignments = append(assignments, councilProposalAssignmentV0(input, candidate))
	}
	for index, candidate := range selected {
		target := critiqueTargetCandidateV0(selected, index)
		assignments = append(assignments, councilCritiqueAssignmentV0(input, candidate, target))
	}
	for _, candidate := range selected {
		assignments = append(assignments, councilVoteAssignmentV0(input, candidate))
	}
	return assignments
}

func councilProposalAssignmentV0(
	input DecisionCouncilPlanInputV0,
	candidate CouncilAgentCandidateV0,
) CouncilAssignmentV0 {
	return CouncilAssignmentV0{
		AssignmentRef:    fmt.Sprintf("%s:proposal:%s", input.DecisionTopicRef, candidate.AgentRef),
		Role:             CouncilRoleProposalV0,
		AgentRef:         candidate.AgentRef,
		FamilyRef:        candidate.FamilyRef,
		DependsOnRefs:    []string{input.BrainstormRequestRef},
		ExpectedArtifact: CouncilArtifactProposalV0,
		ContextPolicy:    "small_isolated_refs_only",
	}
}

func councilCritiqueAssignmentV0(
	input DecisionCouncilPlanInputV0,
	candidate CouncilAgentCandidateV0,
	target CouncilAgentCandidateV0,
) CouncilAssignmentV0 {
	return CouncilAssignmentV0{
		AssignmentRef: fmt.Sprintf(
			"%s:critique:%s:%s",
			input.DecisionTopicRef,
			candidate.AgentRef,
			target.AgentRef,
		),
		Role:             CouncilRoleCritiqueV0,
		AgentRef:         candidate.AgentRef,
		FamilyRef:        candidate.FamilyRef,
		DependsOnRefs:    []string{fmt.Sprintf("%s:proposal:%s", input.DecisionTopicRef, target.AgentRef)},
		ExpectedArtifact: CouncilArtifactCritiqueV0,
		ContextPolicy:    "small_cross_review_refs_only",
	}
}

func councilVoteAssignmentV0(
	input DecisionCouncilPlanInputV0,
	candidate CouncilAgentCandidateV0,
) CouncilAssignmentV0 {
	return CouncilAssignmentV0{
		AssignmentRef:    fmt.Sprintf("%s:vote:%s", input.DecisionTopicRef, candidate.AgentRef),
		Role:             CouncilRoleVoteV0,
		AgentRef:         candidate.AgentRef,
		FamilyRef:        candidate.FamilyRef,
		DependsOnRefs:    []string{input.VoteRequestRef, fmt.Sprintf("%s:gate:critiques", input.DecisionTopicRef)},
		ExpectedArtifact: CouncilArtifactVoteV0,
		ContextPolicy:    "small_vote_refs_with_dissent_required",
	}
}

func buildCouncilGatesV0(input DecisionCouncilPlanInputV0) []CouncilSynchronizationGateV0 {
	return []CouncilSynchronizationGateV0{
		councilGateV0(input, "proposals", CouncilRoleProposalV0),
		councilGateV0(input, "critiques", CouncilRoleCritiqueV0),
		councilGateV0(input, "votes", CouncilRoleVoteV0),
	}
}

func councilGateV0(
	input DecisionCouncilPlanInputV0,
	suffix string,
	role string,
) CouncilSynchronizationGateV0 {
	return CouncilSynchronizationGateV0{
		GateRef:                 fmt.Sprintf("%s:gate:%s", input.DecisionTopicRef, suffix),
		WaitForRole:             role,
		MinimumArtifacts:        input.MinimumAgents,
		MinimumDistinctFamilies: input.MinimumDistinctFamilies,
		OnFailure:               "ask_director_or_spawn_research_microtask",
	}
}
