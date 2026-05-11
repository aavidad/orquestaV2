package orquestadecisioncouncil

const (
	ErrDecisionCouncilInvalidV0 = "decision_council_invalido"
)

const (
	CouncilCapacityLowV0    = "low"
	CouncilCapacityMediumV0 = "medium"
	CouncilCapacityHighV0   = "high"
	CouncilCapacityXHighV0  = "xhigh"
)

const (
	CouncilRoleProposalV0 = "architecture_proposal"
	CouncilRoleCritiqueV0 = "architecture_critique"
	CouncilRoleVoteV0     = "architecture_vote"
)

const (
	CouncilArtifactProposalV0 = "architecture_proposal.v0"
	CouncilArtifactCritiqueV0 = "architecture_critique.v0"
	CouncilArtifactVoteV0     = "architecture_vote.v0"
)

type DecisionCouncilPlanInputV0 struct {
	RunRef                  string                    `json:"run_ref"`
	DecisionTopicRef        string                    `json:"decision_topic_ref"`
	BrainstormRequestRef    string                    `json:"brainstorm_request_ref"`
	VoteRequestRef          string                    `json:"vote_request_ref"`
	MinimumCapacityLevel    string                    `json:"minimum_capacity_level,omitempty"`
	MinimumAgents           int                       `json:"minimum_agents,omitempty"`
	MinimumDistinctFamilies int                       `json:"minimum_distinct_families,omitempty"`
	RequiredFamilyRefs      []string                  `json:"required_family_refs,omitempty"`
	Candidates              []CouncilAgentCandidateV0 `json:"candidates"`
	EvidenceRefs            []string                  `json:"evidence_refs,omitempty"`
}

type CouncilAgentCandidateV0 struct {
	AgentRef      string `json:"agent_ref"`
	FamilyRef     string `json:"family_ref"`
	CapacityLevel string `json:"capacity_level"`
	Active        bool   `json:"active"`
}

type DecisionCouncilPlanV0 struct {
	RunRef           string                         `json:"run_ref"`
	DecisionTopicRef string                         `json:"decision_topic_ref"`
	Assignments      []CouncilAssignmentV0          `json:"assignments"`
	Gates            []CouncilSynchronizationGateV0 `json:"gates"`
	SelectedAgents   []CouncilAgentCandidateV0      `json:"selected_agents"`
	EvidenceRefs     []string                       `json:"evidence_refs,omitempty"`
}

type CouncilAssignmentV0 struct {
	AssignmentRef    string   `json:"assignment_ref"`
	Role             string   `json:"role"`
	AgentRef         string   `json:"agent_ref"`
	FamilyRef        string   `json:"family_ref"`
	DependsOnRefs    []string `json:"depends_on_refs,omitempty"`
	ExpectedArtifact string   `json:"expected_artifact"`
	ContextPolicy    string   `json:"context_policy"`
}

type CouncilSynchronizationGateV0 struct {
	GateRef                 string `json:"gate_ref"`
	WaitForRole             string `json:"wait_for_role"`
	MinimumArtifacts        int    `json:"minimum_artifacts"`
	MinimumDistinctFamilies int    `json:"minimum_distinct_families"`
	OnFailure               string `json:"on_failure"`
}

type DecisionCouncilErrorV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func (err DecisionCouncilErrorV0) Error() string {
	return err.Code
}
