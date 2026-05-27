package orquestadecisioncouncil

import (
	"fmt"
	"strings"
)

const (
	CouncilRoundRefProposalsV0 = "proposals"
	CouncilRoundRefCritiquesV0 = "critiques"
	CouncilRoundRefVotesV0     = "votes"
)

const (
	CouncilPhaseBrainstormingArquitecturaV0 = "brainstorming_arquitectura"
	CouncilPhaseVotacionYDecisionV0         = "votacion_y_decision"
)

type DecisionCouncilOperationalRoundsV0 struct {
	RunRef           string                              `json:"run_ref"`
	DecisionTopicRef string                              `json:"decision_topic_ref"`
	Rounds           []DecisionCouncilOperationalRoundV0 `json:"rounds"`
	EvidenceRefs     []string                            `json:"evidence_refs,omitempty"`
}

type DecisionCouncilOperationalRoundV0 struct {
	RoundRef                string   `json:"round_ref"`
	Role                    string   `json:"role"`
	PhaseID                 string   `json:"phase_id"`
	GateRef                 string   `json:"gate_ref"`
	WaitCohortRef           string   `json:"wait_cohort_ref"`
	WaitWaveRef             string   `json:"wait_wave_ref"`
	AssignmentRefs          []string `json:"assignment_refs"`
	DependsOnRoundRefs      []string `json:"depends_on_round_refs,omitempty"`
	MinimumArtifacts        int      `json:"minimum_artifacts"`
	MinimumDistinctFamilies int      `json:"minimum_distinct_families"`
	EvidenceRefs            []string `json:"evidence_refs,omitempty"`
}

func BuildDecisionCouncilOperationalRoundsV0(
	plan DecisionCouncilPlanV0,
) (DecisionCouncilOperationalRoundsV0, error) {
	plan.RunRef = strings.TrimSpace(plan.RunRef)
	plan.DecisionTopicRef = strings.TrimSpace(plan.DecisionTopicRef)
	if plan.RunRef == "" {
		return DecisionCouncilOperationalRoundsV0{}, councilErrorV0("run_ref")
	}
	if plan.DecisionTopicRef == "" {
		return DecisionCouncilOperationalRoundsV0{}, councilErrorV0("decision_topic_ref")
	}
	if len(plan.Assignments) == 0 {
		return DecisionCouncilOperationalRoundsV0{}, councilErrorV0("assignments")
	}
	if len(plan.Gates) == 0 {
		return DecisionCouncilOperationalRoundsV0{}, councilErrorV0("gates")
	}
	rounds := []DecisionCouncilOperationalRoundV0{
		councilOperationalRoundV0(plan, CouncilRoundRefProposalsV0, CouncilRoleProposalV0, nil),
		councilOperationalRoundV0(plan, CouncilRoundRefCritiquesV0, CouncilRoleCritiqueV0, []string{CouncilRoundRefProposalsV0}),
		councilOperationalRoundV0(plan, CouncilRoundRefVotesV0, CouncilRoleVoteV0, []string{CouncilRoundRefCritiquesV0}),
	}
	for _, round := range rounds {
		if len(round.AssignmentRefs) == 0 || round.GateRef == "" {
			return DecisionCouncilOperationalRoundsV0{}, councilErrorV0("rounds")
		}
	}
	return DecisionCouncilOperationalRoundsV0{
		RunRef:           plan.RunRef,
		DecisionTopicRef: plan.DecisionTopicRef,
		Rounds:           rounds,
		EvidenceRefs:     compactCouncilStringsV0(plan.EvidenceRefs),
	}, nil
}

func councilOperationalRoundV0(
	plan DecisionCouncilPlanV0,
	roundSuffix string,
	role string,
	dependsOn []string,
) DecisionCouncilOperationalRoundV0 {
	gate := councilGateForRoleV0(plan, role)
	roundRef := fmt.Sprintf("%s:round:%s", plan.DecisionTopicRef, roundSuffix)
	return DecisionCouncilOperationalRoundV0{
		RoundRef:                roundRef,
		Role:                    role,
		PhaseID:                 councilRoundPhaseV0(role),
		GateRef:                 gate.GateRef,
		WaitCohortRef:           fmt.Sprintf("%s:cohort", roundRef),
		WaitWaveRef:             fmt.Sprintf("%s:wave", roundRef),
		AssignmentRefs:          councilAssignmentRefsForRoleV0(plan, role),
		DependsOnRoundRefs:      append([]string(nil), dependsOn...),
		MinimumArtifacts:        gate.MinimumArtifacts,
		MinimumDistinctFamilies: gate.MinimumDistinctFamilies,
		EvidenceRefs:            compactCouncilStringsV0(append(plan.EvidenceRefs, gate.GateRef)),
	}
}

func councilGateForRoleV0(
	plan DecisionCouncilPlanV0,
	role string,
) CouncilSynchronizationGateV0 {
	for _, gate := range plan.Gates {
		if gate.WaitForRole == role {
			return gate
		}
	}
	return CouncilSynchronizationGateV0{}
}

func councilAssignmentRefsForRoleV0(
	plan DecisionCouncilPlanV0,
	role string,
) []string {
	refs := make([]string, 0)
	for _, assignment := range plan.Assignments {
		if assignment.Role == role {
			refs = append(refs, assignment.AssignmentRef)
		}
	}
	return compactCouncilStringsV0(refs)
}

func councilRoundPhaseV0(role string) string {
	if role == CouncilRoleVoteV0 {
		return CouncilPhaseVotacionYDecisionV0
	}
	return CouncilPhaseBrainstormingArquitecturaV0
}
