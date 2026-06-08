package orquestadecisioncouncil

import "testing"

func TestEvaluateDecisionCouncilVotesV0AceptaConsensoTresFamilias(t *testing.T) {
	result, err := EvaluateDecisionCouncilVotesV0(validCouncilVoteInputV0())
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("accepted=false reason=%s result=%+v", result.ReasonCode, result)
	}
	if result.AcceptedOptionRef != "option-rest-hex" {
		t.Fatalf("accepted_option_ref=%s", result.AcceptedOptionRef)
	}
	if result.DistinctFamilies != 3 {
		t.Fatalf("distinct_families=%d, want 3", result.DistinctFamilies)
	}
	if len(result.EvidenceRefs) != 3 {
		t.Fatalf("evidence_refs=%v, want 3 source refs", result.EvidenceRefs)
	}
}

func TestEvaluateDecisionCouncilVotesV0BloqueoImpideDecision(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.Votes[1].Position = CouncilVoteBlockV0

	result, err := EvaluateDecisionCouncilVotesV0(input)
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	if result.Accepted {
		t.Fatalf("accepted=true with block")
	}
	if result.ReasonCode != "blocking_vote" {
		t.Fatalf("reason=%s, want blocking_vote", result.ReasonCode)
	}
}

func TestEvaluateDecisionCouncilVotesV0RechazaQuorumFamiliasInsuficiente(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.Votes[1].FamilyRef = "family-a"
	input.Votes[2].FamilyRef = "family-a"

	result, err := EvaluateDecisionCouncilVotesV0(input)
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	if result.Accepted {
		t.Fatalf("accepted=true without family quorum")
	}
	if result.ReasonCode != "minimum_distinct_families" {
		t.Fatalf("reason=%s, want minimum_distinct_families", result.ReasonCode)
	}
}

func TestEvaluateDecisionCouncilVotesV0FamiliasNoCuentaAbstenciones(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.MinimumVotes = 2
	input.MinimumNonAuthorVotes = 1
	input.MinimumDistinctFamilies = 3
	input.Votes[2].Position = CouncilVoteAbstainV0
	input.Votes[2].OptionRef = ""
	input.Votes[2].EvidenceRefs = nil

	result, err := EvaluateDecisionCouncilVotesV0(input)
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	if result.Accepted {
		t.Fatalf("accepted=true with abstained family counted")
	}
	if result.DistinctFamilies != 2 || result.ReasonCode != "minimum_distinct_families" {
		t.Fatalf("result=%+v, want 2 considered families and minimum_distinct_families", result)
	}
}

func TestEvaluateDecisionCouncilVotesV0ConservaDisenso(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.ApprovalThresholdPct = 50
	input.Votes = append(input.Votes, CouncilVoteV0{
		VoteRef:      "vote-local-1",
		VoterRef:     "agent-local-1",
		FamilyRef:    "family-local",
		OptionRef:    "option-rest-hex",
		Position:     CouncilVoteRejectV0,
		EvidenceRefs: []string{"source-local-1"},
	})

	result, err := EvaluateDecisionCouncilVotesV0(input)
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("accepted=false reason=%s", result.ReasonCode)
	}
	if len(result.DissentVoteRefs) != 1 || result.DissentVoteRefs[0] != "vote-local-1" {
		t.Fatalf("dissent=%v", result.DissentVoteRefs)
	}
}

func TestEvaluateDecisionCouncilVotesV0RechazaVotoSinEvidencia(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.Votes[0].EvidenceRefs = nil

	_, err := EvaluateDecisionCouncilVotesV0(input)
	assertCouncilErrorFieldV0(t, err, "votes.evidence_refs")
}

func TestEvaluateDecisionCouncilVotesV0ConservaSourceRefsSinTranscript(t *testing.T) {
	input := validCouncilVoteInputV0()
	input.Votes[0].EvidenceRefs = []string{"source-shared", "source-a"}
	input.Votes[1].EvidenceRefs = []string{"source-shared", "source-b"}
	input.Votes[2].EvidenceRefs = []string{"source-c"}

	result, err := EvaluateDecisionCouncilVotesV0(input)
	if err != nil {
		t.Fatalf("EvaluateDecisionCouncilVotesV0: %v", err)
	}
	want := []string{"source-shared", "source-a", "source-b", "source-c"}
	if len(result.EvidenceRefs) != len(want) {
		t.Fatalf("evidence_refs=%v, want %v", result.EvidenceRefs, want)
	}
	for index, ref := range want {
		if result.EvidenceRefs[index] != ref {
			t.Fatalf("evidence_refs=%v, want %v", result.EvidenceRefs, want)
		}
	}
}
