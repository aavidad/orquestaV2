package orquestadirectorcandidates

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildSchedulableWorkCandidatesFromCouncilPlanV0Propuestas(t *testing.T) {
	candidates, err := BuildSchedulableWorkCandidatesFromCouncilPlanV0(validCouncilCandidatesInputV0(
		orquestadecisioncouncil.CouncilRoleProposalV0,
		string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
	))
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidatesFromCouncilPlanV0() error = %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidates=%d, want 3", len(candidates))
	}
	for _, candidate := range candidates {
		if candidate.AgentCandidate.Payload.Role != orquestadecisioncouncil.CouncilRoleProposalV0 {
			t.Fatalf("role=%s", candidate.AgentCandidate.Payload.Role)
		}
		if len(candidate.Claims) != 1 || len(candidate.Claims[0].WriteSet) != 1 {
			t.Fatalf("claims=%+v", candidate.Claims)
		}
	}
}

func TestBuildSchedulableWorkCandidatesFromCouncilPlanV0ValidaScheduler(t *testing.T) {
	candidates, err := BuildSchedulableWorkCandidatesFromCouncilPlanV0(validCouncilCandidatesInputV0(
		orquestadecisioncouncil.CouncilRoleCritiqueV0,
		string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
	))
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidatesFromCouncilPlanV0() error = %v", err)
	}
	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(
		councilSchedulerInputV0(candidates),
	)
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0() error = %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
		t.Fatalf("status=%q", plan.Status)
	}
	if len(plan.Commands) != 3 {
		t.Fatalf("commands=%d, want 3", len(plan.Commands))
	}
}

func TestBuildSchedulableWorkCandidatesFromCouncilPlanV0VotoEnFaseDecision(t *testing.T) {
	candidates, err := BuildSchedulableWorkCandidatesFromCouncilPlanV0(validCouncilCandidatesInputV0(
		orquestadecisioncouncil.CouncilRoleVoteV0,
		string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
	))
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidatesFromCouncilPlanV0() error = %v", err)
	}
	for _, candidate := range candidates {
		if got := candidate.CapacityCandidate.Payload.PhaseID; got != string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0) {
			t.Fatalf("phase=%s", got)
		}
		if candidate.AgentCandidate.Payload.Role != orquestadecisioncouncil.CouncilRoleVoteV0 {
			t.Fatalf("role=%s", candidate.AgentCandidate.Payload.Role)
		}
	}
}

func TestBuildSchedulableWorkCandidatesFromCouncilPlanV0RechazaRunDistinto(t *testing.T) {
	input := validCouncilCandidatesInputV0(
		orquestadecisioncouncil.CouncilRoleProposalV0,
		string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
	)
	input.Plan.RunRef = "run-distinto"

	_, err := BuildSchedulableWorkCandidatesFromCouncilPlanV0(input)
	assertCandidateFieldErrorV0(t, err, "plan.run_ref")
}

func validCouncilCandidatesInputV0(role string, phaseID string) CouncilPlanCandidatesInputV0 {
	plan, err := orquestadecisioncouncil.BuildDecisionCouncilPlanV0(orquestadecisioncouncil.DecisionCouncilPlanInputV0{
		RunRef:                  "run-council-001",
		DecisionTopicRef:        "topic-agenda",
		BrainstormRequestRef:    "brainstorm-agenda",
		VoteRequestRef:          "vote-agenda",
		MinimumCapacityLevel:    orquestadecisioncouncil.CouncilCapacityHighV0,
		MinimumAgents:           3,
		MinimumDistinctFamilies: 3,
		RequiredFamilyRefs:      []string{"family-a", "family-b", "family-c"},
		EvidenceRefs:            []string{"evidence-council-plan-001"},
		Candidates: []orquestadecisioncouncil.CouncilAgentCandidateV0{
			{AgentRef: "agent-a-1", FamilyRef: "family-a", CapacityLevel: orquestadecisioncouncil.CouncilCapacityXHighV0, Active: true},
			{AgentRef: "agent-b-1", FamilyRef: "family-b", CapacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, Active: true},
			{AgentRef: "agent-c-1", FamilyRef: "family-c", CapacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, Active: true},
		},
	})
	if err != nil {
		panic(err)
	}
	return CouncilPlanCandidatesInputV0{
		RunRef:        "run-council-001",
		PhaseID:       phaseID,
		Role:          role,
		OccurredAt:    "2026-05-07T12:00:00Z",
		CorrelationID: "corr-council-001",
		RequestedBy:   "director-council-test",
		Plan:          plan,
		EvidenceRefs:  []string{"evidence-council-001"},
	}
}

func councilSchedulerInputV0(
	candidates []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	return orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:    "tick-council-001",
		RunRef:     "run-council-001",
		OccurredAt: "2026-05-07T12:00:00Z",
		Snapshot: orquestadirectorscheduler.RunSchedulingSnapshotV0{
			RunRef:         "run-council-001",
			CurrentPhaseID: candidates[0].CapacityCandidate.Payload.PhaseID,
		},
		WorkCandidates: candidates,
		EvidenceRefs:   []string{"evidence-council-tick-001"},
	}
}
