package orquestadecisioncouncil

import "testing"

func TestBuildDecisionCouncilPlanV0TresFamiliasOpacas(t *testing.T) {
	plan, err := BuildDecisionCouncilPlanV0(validCouncilPlanInputV0())
	if err != nil {
		t.Fatalf("BuildDecisionCouncilPlanV0: %v", err)
	}
	if len(plan.SelectedAgents) != 3 {
		t.Fatalf("selected=%d, want 3", len(plan.SelectedAgents))
	}
	if len(plan.Assignments) != 9 {
		t.Fatalf("assignments=%d, want 9", len(plan.Assignments))
	}
	assertCouncilRoleCountV0(t, plan, CouncilRoleProposalV0, 3)
	assertCouncilRoleCountV0(t, plan, CouncilRoleCritiqueV0, 3)
	assertCouncilRoleCountV0(t, plan, CouncilRoleVoteV0, 3)
	if len(plan.Gates) != 3 {
		t.Fatalf("gates=%d, want 3", len(plan.Gates))
	}
	if len(plan.EvidenceRefs) != 2 {
		t.Fatalf("evidence_refs=%v, want 2 refs", plan.EvidenceRefs)
	}
}

func TestBuildDecisionCouncilPlanV0RechazaFaltaDiversidad(t *testing.T) {
	input := validCouncilPlanInputV0()
	input.Candidates[1].FamilyRef = "family-a"
	input.Candidates[2].FamilyRef = "family-a"

	_, err := BuildDecisionCouncilPlanV0(input)
	assertCouncilErrorFieldV0(t, err, "minimum_distinct_families")
}

func TestBuildDecisionCouncilPlanV0CriticaNoEsAutoRevision(t *testing.T) {
	plan, err := BuildDecisionCouncilPlanV0(validCouncilPlanInputV0())
	if err != nil {
		t.Fatalf("BuildDecisionCouncilPlanV0: %v", err)
	}
	for _, assignment := range plan.Assignments {
		if assignment.Role != CouncilRoleCritiqueV0 {
			continue
		}
		selfProposal := "topic-agenda:proposal:" + assignment.AgentRef
		if len(assignment.DependsOnRefs) != 1 || assignment.DependsOnRefs[0] == selfProposal {
			t.Fatalf("critique assignment self review: %+v", assignment)
		}
	}
}

func TestBuildDecisionCouncilPlanV0ExigeFamiliasRequeridas(t *testing.T) {
	input := validCouncilPlanInputV0()
	input.RequiredFamilyRefs = []string{"family-a", "family-unknown"}

	_, err := BuildDecisionCouncilPlanV0(input)
	assertCouncilErrorFieldV0(t, err, "required_family_refs")
}

func TestBuildDecisionCouncilPlanV0RechazaBrainstormingSinEvidencia(t *testing.T) {
	input := validCouncilPlanInputV0()
	input.EvidenceRefs = nil

	_, err := BuildDecisionCouncilPlanV0(input)
	assertCouncilErrorFieldV0(t, err, "evidence_refs")
}
