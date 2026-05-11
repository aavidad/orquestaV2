package orquestadecisioncouncil

func buildDecisionCouncilVoteResultV0(
	input DecisionCouncilVoteInputV0,
) DecisionCouncilVoteResultV0 {
	result := DecisionCouncilVoteResultV0{DecisionTopicRef: input.DecisionTopicRef}
	approvalsByOption := map[string]int{}
	for _, vote := range input.Votes {
		if vote.Position == CouncilVoteAbstainV0 {
			continue
		}
		result.EvidenceRefs = append(result.EvidenceRefs, vote.EvidenceRefs...)
		result.ConsideredVotes++
		if vote.VoterRef != input.AuthorAgentRef {
			result.NonAuthorVotes++
		}
		if vote.Position == CouncilVoteBlockV0 {
			result.BlockVoteRefs = append(result.BlockVoteRefs, vote.VoteRef)
			continue
		}
		if vote.Position == CouncilVoteApproveV0 {
			approvalsByOption[vote.OptionRef]++
			continue
		}
		result.DissentVoteRefs = append(result.DissentVoteRefs, vote.VoteRef)
	}
	result.EvidenceRefs = compactCouncilStringsV0(result.EvidenceRefs)
	result.DistinctFamilies = len(councilVoteFamiliesV0(input.Votes))
	result.AcceptedOptionRef, result.ApprovalPct = bestCouncilOptionV0(
		input.OptionRefs,
		approvalsByOption,
		result.ConsideredVotes,
	)
	result.ReasonCode = councilVoteRejectionReasonV0(input, result)
	return result
}

func bestCouncilOptionV0(
	optionRefs []string,
	approvalsByOption map[string]int,
	consideredVotes int,
) (string, int) {
	bestOption := ""
	bestApprovals := 0
	for _, optionRef := range optionRefs {
		if approvalsByOption[optionRef] > bestApprovals {
			bestOption = optionRef
			bestApprovals = approvalsByOption[optionRef]
		}
	}
	if consideredVotes == 0 {
		return bestOption, 0
	}
	return bestOption, bestApprovals * 100 / consideredVotes
}

func councilVoteRejectionReasonV0(
	input DecisionCouncilVoteInputV0,
	result DecisionCouncilVoteResultV0,
) string {
	switch {
	case len(result.BlockVoteRefs) > 0:
		return "blocking_vote"
	case result.ConsideredVotes < input.MinimumVotes:
		return "minimum_votes"
	case result.NonAuthorVotes < input.MinimumNonAuthorVotes:
		return "minimum_non_author_votes"
	case result.DistinctFamilies < input.MinimumDistinctFamilies:
		return "minimum_distinct_families"
	case result.AcceptedOptionRef == "":
		return "accepted_option_ref"
	case result.ApprovalPct < input.ApprovalThresholdPct:
		return "approval_threshold"
	default:
		return ""
	}
}
