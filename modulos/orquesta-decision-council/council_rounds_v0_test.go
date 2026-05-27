package orquestadecisioncouncil

import "testing"

func TestBuildDecisionCouncilOperationalRoundsV0GateaPropuestaCriticaVoto(t *testing.T) {
	plan, err := BuildDecisionCouncilPlanV0(validCouncilPlanInputV0())
	if err != nil {
		t.Fatalf("BuildDecisionCouncilPlanV0: %v", err)
	}
	rounds, err := BuildDecisionCouncilOperationalRoundsV0(plan)
	if err != nil {
		t.Fatalf("BuildDecisionCouncilOperationalRoundsV0: %v", err)
	}
	if len(rounds.Rounds) != 3 {
		t.Fatalf("rounds=%d, want 3", len(rounds.Rounds))
	}
	assertCouncilRoundV0(t, rounds.Rounds[0], CouncilRoleProposalV0, 0)
	assertCouncilRoundV0(t, rounds.Rounds[1], CouncilRoleCritiqueV0, 1)
	assertCouncilRoundV0(t, rounds.Rounds[2], CouncilRoleVoteV0, 1)
	if rounds.Rounds[2].PhaseID != CouncilPhaseVotacionYDecisionV0 {
		t.Fatalf("vote phase=%s", rounds.Rounds[2].PhaseID)
	}
}

func assertCouncilRoundV0(
	t *testing.T,
	round DecisionCouncilOperationalRoundV0,
	role string,
	depends int,
) {
	t.Helper()
	if round.Role != role || round.GateRef == "" || round.WaitCohortRef == "" ||
		round.WaitWaveRef == "" || len(round.AssignmentRefs) != 3 ||
		len(round.DependsOnRoundRefs) != depends ||
		round.MinimumArtifacts != 3 || round.MinimumDistinctFamilies != 3 {
		t.Fatalf("round=%+v", round)
	}
}
