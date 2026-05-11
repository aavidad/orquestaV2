package orquestadecisioncouncil

func validateDecisionCouncilVoteInputV0(input DecisionCouncilVoteInputV0) error {
	switch {
	case input.DecisionTopicRef == "":
		return councilErrorV0("decision_topic_ref")
	case len(input.OptionRefs) == 0:
		return councilErrorV0("option_refs")
	case len(input.Votes) == 0:
		return councilErrorV0("votes")
	case input.MinimumVotes < 1:
		return councilErrorV0("minimum_votes")
	case input.MinimumNonAuthorVotes < 0:
		return councilErrorV0("minimum_non_author_votes")
	case input.MinimumDistinctFamilies < 1:
		return councilErrorV0("minimum_distinct_families")
	case input.ApprovalThresholdPct < 1 || input.ApprovalThresholdPct > 100:
		return councilErrorV0("approval_threshold_pct")
	}
	optionSet := stringSetV0(input.OptionRefs)
	for _, vote := range input.Votes {
		if err := validateCouncilVoteV0(vote, optionSet); err != nil {
			return err
		}
	}
	return nil
}

func validateCouncilVoteV0(vote CouncilVoteV0, optionSet map[string]bool) error {
	switch {
	case vote.VoteRef == "":
		return councilErrorV0("votes.vote_ref")
	case vote.VoterRef == "":
		return councilErrorV0("votes.voter_ref")
	case vote.FamilyRef == "":
		return councilErrorV0("votes.family_ref")
	case !validCouncilVotePositionV0(vote.Position):
		return councilErrorV0("votes.position")
	case vote.Position != CouncilVoteAbstainV0 && vote.OptionRef == "":
		return councilErrorV0("votes.option_ref")
	case vote.OptionRef != "" && !optionSet[vote.OptionRef]:
		return councilErrorV0("votes.option_ref")
	case vote.Position != CouncilVoteAbstainV0 && len(vote.EvidenceRefs) == 0:
		return councilErrorV0("votes.evidence_refs")
	}
	return nil
}

func validCouncilVotePositionV0(position string) bool {
	switch position {
	case CouncilVoteApproveV0, CouncilVoteRejectV0, CouncilVoteBlockV0, CouncilVoteAbstainV0:
		return true
	default:
		return false
	}
}
