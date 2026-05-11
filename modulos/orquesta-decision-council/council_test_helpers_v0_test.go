package orquestadecisioncouncil

import "testing"

func validCouncilPlanInputV0() DecisionCouncilPlanInputV0 {
	return DecisionCouncilPlanInputV0{
		RunRef:                  "run-agenda",
		DecisionTopicRef:        "topic-agenda",
		BrainstormRequestRef:    "brainstorm-agenda",
		VoteRequestRef:          "vote-agenda",
		MinimumCapacityLevel:    CouncilCapacityHighV0,
		MinimumAgents:           3,
		MinimumDistinctFamilies: 3,
		RequiredFamilyRefs:      []string{"family-a", "family-b", "family-c"},
		EvidenceRefs:            []string{"source-topic-1", "source-topic-2"},
		Candidates: []CouncilAgentCandidateV0{
			{AgentRef: "agent-a-1", FamilyRef: "family-a", CapacityLevel: CouncilCapacityXHighV0, Active: true},
			{AgentRef: "agent-b-1", FamilyRef: "family-b", CapacityLevel: CouncilCapacityHighV0, Active: true},
			{AgentRef: "agent-c-1", FamilyRef: "family-c", CapacityLevel: CouncilCapacityHighV0, Active: true},
		},
	}
}

func validCouncilVoteInputV0() DecisionCouncilVoteInputV0 {
	return DecisionCouncilVoteInputV0{
		DecisionTopicRef:        "topic-agenda",
		AuthorAgentRef:          "agent-a-1",
		OptionRefs:              []string{"option-rest-hex", "option-grpc-first"},
		MinimumVotes:            3,
		MinimumNonAuthorVotes:   2,
		MinimumDistinctFamilies: 3,
		ApprovalThresholdPct:    67,
		Votes: []CouncilVoteV0{
			{VoteRef: "vote-a-1", VoterRef: "agent-a-1", FamilyRef: "family-a", OptionRef: "option-rest-hex", Position: CouncilVoteApproveV0, EvidenceRefs: []string{"source-a-1"}},
			{VoteRef: "vote-b-1", VoterRef: "agent-b-1", FamilyRef: "family-b", OptionRef: "option-rest-hex", Position: CouncilVoteApproveV0, EvidenceRefs: []string{"source-b-1"}},
			{VoteRef: "vote-c-1", VoterRef: "agent-c-1", FamilyRef: "family-c", OptionRef: "option-rest-hex", Position: CouncilVoteApproveV0, EvidenceRefs: []string{"source-c-1"}},
		},
	}
}

func assertCouncilRoleCountV0(
	t *testing.T,
	plan DecisionCouncilPlanV0,
	role string,
	want int,
) {
	t.Helper()
	got := 0
	for _, assignment := range plan.Assignments {
		if assignment.Role == role {
			got++
		}
	}
	if got != want {
		t.Fatalf("role %s count=%d, want %d", role, got, want)
	}
}

func assertCouncilErrorFieldV0(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err=nil, want %s", field)
	}
	publicErr, ok := err.(DecisionCouncilErrorV0)
	if !ok {
		t.Fatalf("err=%T, want DecisionCouncilErrorV0", err)
	}
	if publicErr.Field != field {
		t.Fatalf("field=%s, want %s", publicErr.Field, field)
	}
}
