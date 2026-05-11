package orquestadirectorcandidates

import orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"

const (
	TeamComplexityLowV0    = "low"
	TeamComplexityMediumV0 = "medium"
	TeamComplexityHighV0   = "high"
	TeamComplexityXHighV0  = "xhigh"
)

type TeamPlanFromComplexityInputV0 struct {
	RunRef               string   `json:"run_ref"`
	DecisionTopicRef     string   `json:"decision_topic_ref"`
	BrainstormRequestRef string   `json:"brainstorm_request_ref"`
	VoteRequestRef       string   `json:"vote_request_ref"`
	Complexity           string   `json:"complexity"`
	EvidenceRefs         []string `json:"evidence_refs,omitempty"`
}

type TeamCapacityPlanV0 struct {
	Complexity              string                                            `json:"complexity"`
	MinimumCapacityLevel    string                                            `json:"minimum_capacity_level"`
	MinimumAgents           int                                               `json:"minimum_agents"`
	MinimumDistinctFamilies int                                               `json:"minimum_distinct_families"`
	Candidates              []orquestadecisioncouncil.CouncilAgentCandidateV0 `json:"candidates"`
}
