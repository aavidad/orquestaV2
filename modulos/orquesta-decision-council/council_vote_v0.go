package orquestadecisioncouncil

import "strings"

const (
	CouncilVoteApproveV0 = "approve"
	CouncilVoteRejectV0  = "reject"
	CouncilVoteBlockV0   = "block"
	CouncilVoteAbstainV0 = "abstain"
)

type DecisionCouncilVoteInputV0 struct {
	DecisionTopicRef        string          `json:"decision_topic_ref"`
	AuthorAgentRef          string          `json:"author_agent_ref,omitempty"`
	OptionRefs              []string        `json:"option_refs"`
	Votes                   []CouncilVoteV0 `json:"votes"`
	MinimumVotes            int             `json:"minimum_votes,omitempty"`
	MinimumNonAuthorVotes   int             `json:"minimum_non_author_votes,omitempty"`
	MinimumDistinctFamilies int             `json:"minimum_distinct_families,omitempty"`
	ApprovalThresholdPct    int             `json:"approval_threshold_pct,omitempty"`
}

type CouncilVoteV0 struct {
	VoteRef      string   `json:"vote_ref"`
	VoterRef     string   `json:"voter_ref"`
	FamilyRef    string   `json:"family_ref"`
	OptionRef    string   `json:"option_ref"`
	Position     string   `json:"position"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DecisionCouncilVoteResultV0 struct {
	DecisionTopicRef  string   `json:"decision_topic_ref"`
	Accepted          bool     `json:"accepted"`
	AcceptedOptionRef string   `json:"accepted_option_ref,omitempty"`
	ConsideredVotes   int      `json:"considered_votes"`
	NonAuthorVotes    int      `json:"non_author_votes"`
	DistinctFamilies  int      `json:"distinct_families"`
	ApprovalPct       int      `json:"approval_pct"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
	BlockVoteRefs     []string `json:"block_vote_refs,omitempty"`
	DissentVoteRefs   []string `json:"dissent_vote_refs,omitempty"`
	ReasonCode        string   `json:"reason_code,omitempty"`
}

func EvaluateDecisionCouncilVotesV0(
	input DecisionCouncilVoteInputV0,
) (DecisionCouncilVoteResultV0, error) {
	input = normalizeDecisionCouncilVoteInputV0(input)
	if err := validateDecisionCouncilVoteInputV0(input); err != nil {
		return DecisionCouncilVoteResultV0{}, err
	}
	result := buildDecisionCouncilVoteResultV0(input)
	result.Accepted = result.ReasonCode == ""
	return result, nil
}

func normalizeDecisionCouncilVoteInputV0(
	input DecisionCouncilVoteInputV0,
) DecisionCouncilVoteInputV0 {
	input.DecisionTopicRef = strings.TrimSpace(input.DecisionTopicRef)
	input.AuthorAgentRef = strings.TrimSpace(input.AuthorAgentRef)
	input.OptionRefs = compactCouncilStringsV0(input.OptionRefs)
	if input.MinimumVotes == 0 {
		input.MinimumVotes = 3
	}
	if input.MinimumNonAuthorVotes == 0 {
		input.MinimumNonAuthorVotes = 2
	}
	if input.MinimumDistinctFamilies == 0 {
		input.MinimumDistinctFamilies = 2
	}
	if input.ApprovalThresholdPct == 0 {
		input.ApprovalThresholdPct = 67
	}
	for index := range input.Votes {
		input.Votes[index] = normalizeCouncilVoteV0(input.Votes[index])
	}
	return input
}

func normalizeCouncilVoteV0(vote CouncilVoteV0) CouncilVoteV0 {
	vote.VoteRef = strings.TrimSpace(vote.VoteRef)
	vote.VoterRef = strings.TrimSpace(vote.VoterRef)
	vote.FamilyRef = strings.TrimSpace(vote.FamilyRef)
	vote.OptionRef = strings.TrimSpace(vote.OptionRef)
	vote.Position = strings.TrimSpace(vote.Position)
	vote.EvidenceRefs = compactCouncilStringsV0(vote.EvidenceRefs)
	return vote
}
