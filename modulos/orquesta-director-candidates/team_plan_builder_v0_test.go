package orquestadirectorcandidates

import (
	"testing"

	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func TestBuildDecisionCouncilTeamPlanFromComplexityV0DerivaEquipoAlto(t *testing.T) {
	plan, err := BuildDecisionCouncilTeamPlanFromComplexityV0(validTeamPlanInputV0("high"))
	if err != nil {
		t.Fatalf("BuildDecisionCouncilTeamPlanFromComplexityV0() error = %v", err)
	}
	if len(plan.SelectedAgents) != 4 {
		t.Fatalf("selected_agents=%d", len(plan.SelectedAgents))
	}
	if len(plan.Assignments) != 12 {
		t.Fatalf("assignments=%d", len(plan.Assignments))
	}
	assertCouncilTeamSelectedCapacityV0(t, plan, orquestadecisioncouncil.CouncilCapacityHighV0)
	assertCouncilTeamDistinctFamiliesV0(t, plan, 3)
}

func TestBuildDecisionCouncilTeamPlanFromComplexityV0AceptaAliasCompacto(t *testing.T) {
	plan, err := BuildDecisionCouncilTeamPlanFromComplexityV0(validTeamPlanInputV0("xl"))
	if err != nil {
		t.Fatalf("BuildDecisionCouncilTeamPlanFromComplexityV0() error = %v", err)
	}
	if len(plan.SelectedAgents) != 5 {
		t.Fatalf("selected_agents=%d", len(plan.SelectedAgents))
	}
	assertCouncilTeamSelectedCapacityV0(t, plan, orquestadecisioncouncil.CouncilCapacityXHighV0)
	assertCouncilTeamDistinctFamiliesV0(t, plan, 3)
}

func TestBuildDecisionCouncilTeamPlanFromComplexityV0RechazaComplejidadInvalida(t *testing.T) {
	_, err := BuildDecisionCouncilTeamPlanFromComplexityV0(validTeamPlanInputV0("desconocida"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildDecisionCouncilTeamPlanFromComplexityV0RechazaSinEvidencia(t *testing.T) {
	input := validTeamPlanInputV0("medium")
	input.EvidenceRefs = nil
	_, err := BuildDecisionCouncilTeamPlanFromComplexityV0(input)
	if err == nil {
		t.Fatal("expected error")
	}
}

func validTeamPlanInputV0(complexity string) TeamPlanFromComplexityInputV0 {
	return TeamPlanFromComplexityInputV0{
		RunRef:               "run-team-plan-001",
		DecisionTopicRef:     "decision-team-plan-001",
		BrainstormRequestRef: "brainstorm-team-plan-001",
		VoteRequestRef:       "vote-team-plan-001",
		Complexity:           complexity,
		EvidenceRefs:         []string{"evidence-team-plan-001"},
	}
}

func assertCouncilTeamSelectedCapacityV0(
	t *testing.T,
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	want string,
) {
	t.Helper()
	for _, agent := range plan.SelectedAgents {
		if agent.CapacityLevel != want {
			t.Fatalf("capacity=%s want=%s", agent.CapacityLevel, want)
		}
	}
}

func assertCouncilTeamDistinctFamiliesV0(
	t *testing.T,
	plan orquestadecisioncouncil.DecisionCouncilPlanV0,
	want int,
) {
	t.Helper()
	families := map[string]bool{}
	for _, agent := range plan.SelectedAgents {
		families[agent.FamilyRef] = true
	}
	if len(families) != want {
		t.Fatalf("families=%d want=%d selected=%+v", len(families), want, plan.SelectedAgents)
	}
}
