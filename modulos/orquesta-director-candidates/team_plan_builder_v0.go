package orquestadirectorcandidates

import (
	"fmt"
	"strings"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func BuildDecisionCouncilTeamPlanFromComplexityV0(
	input TeamPlanFromComplexityInputV0,
) (orquestadecisioncouncil.DecisionCouncilPlanV0, error) {
	normalized := normalizeTeamPlanFromComplexityInputV0(input)
	if err := validateTeamPlanFromComplexityInputV0(normalized); err != nil {
		return orquestadecisioncouncil.DecisionCouncilPlanV0{}, err
	}
	team := teamCapacityPlanForComplexityV0(normalized)
	return orquestadecisioncouncil.BuildDecisionCouncilPlanV0(
		orquestadecisioncouncil.DecisionCouncilPlanInputV0{
			RunRef:                  normalized.RunRef,
			DecisionTopicRef:        normalized.DecisionTopicRef,
			BrainstormRequestRef:    normalized.BrainstormRequestRef,
			VoteRequestRef:          normalized.VoteRequestRef,
			MinimumCapacityLevel:    team.MinimumCapacityLevel,
			MinimumAgents:           team.MinimumAgents,
			MinimumDistinctFamilies: team.MinimumDistinctFamilies,
			Candidates:              team.Candidates,
			EvidenceRefs:            normalized.EvidenceRefs,
		},
	)
}

func teamCapacityPlanForComplexityV0(input TeamPlanFromComplexityInputV0) TeamCapacityPlanV0 {
	shape := teamShapeForComplexityV0(input.Complexity)
	return TeamCapacityPlanV0{
		Complexity:              input.Complexity,
		MinimumCapacityLevel:    shape.capacityLevel,
		MinimumAgents:           shape.agents,
		MinimumDistinctFamilies: shape.families,
		Candidates:              teamCandidatesV0(input.DecisionTopicRef, shape),
	}
}

func teamCandidatesV0(topicRef string, shape teamShapeV0) []orquestadecisioncouncil.CouncilAgentCandidateV0 {
	base := safeCouncilScopePartV0(topicRef)
	candidates := make([]orquestadecisioncouncil.CouncilAgentCandidateV0, 0, shape.agents)
	for index := 1; index <= shape.agents; index++ {
		familyOrdinal := ((index - 1) % shape.families) + 1
		candidates = append(candidates, orquestadecisioncouncil.CouncilAgentCandidateV0{
			AgentRef:      fmt.Sprintf("slot:%s:%03d", base, index),
			FamilyRef:     fmt.Sprintf("family:%s:%03d", base, familyOrdinal),
			CapacityLevel: shape.capacityLevel,
			Active:        true,
		})
	}
	return candidates
}

func normalizeTeamPlanFromComplexityInputV0(
	input TeamPlanFromComplexityInputV0,
) TeamPlanFromComplexityInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.DecisionTopicRef = strings.TrimSpace(input.DecisionTopicRef)
	input.BrainstormRequestRef = strings.TrimSpace(input.BrainstormRequestRef)
	input.VoteRequestRef = strings.TrimSpace(input.VoteRequestRef)
	input.Complexity = normalizeTeamComplexityV0(input.Complexity)
	input.EvidenceRefs = normalizeRefsV0(input.EvidenceRefs)
	return input
}

func validateTeamPlanFromComplexityInputV0(input TeamPlanFromComplexityInputV0) error {
	fields := []requiredFieldV0{
		{"run_ref", input.RunRef},
		{"decision_topic_ref", input.DecisionTopicRef},
		{"brainstorm_request_ref", input.BrainstormRequestRef},
		{"vote_request_ref", input.VoteRequestRef},
		{"complexity", input.Complexity},
	}
	for _, field := range fields {
		if field.value == "" {
			return candidateErrorV0(field.name)
		}
	}
	if !validTeamComplexityV0(input.Complexity) {
		return candidateErrorV0("complexity")
	}
	if len(input.EvidenceRefs) == 0 {
		return candidateErrorV0("evidence_refs")
	}
	return nil
}
