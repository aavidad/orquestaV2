package orquestadirectoragent

type DirectorAgentAskQuestionCommandV0 struct {
	QuestionID   string   `json:"question_id"`
	SourceGroup  string   `json:"source_group"`
	TargetGroup  string   `json:"target_group,omitempty"`
	Summary      string   `json:"summary"`
	Options      []string `json:"options,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Blocking     bool     `json:"blocking"`
}

type DirectorAgentRequestCapacityCommandV0 struct {
	CapacityRequestID          string   `json:"capacity_request_id"`
	PhaseID                    string   `json:"phase_id"`
	TaskRef                    string   `json:"task_ref,omitempty"`
	ReasonCode                 string   `json:"reason_code"`
	Summary                    string   `json:"summary"`
	MinimumRecommendedCapacity string   `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentRequestAgentCommandV0 struct {
	AgentRequestID     string   `json:"agent_request_id"`
	PhaseID            string   `json:"phase_id"`
	TaskRef            string   `json:"task_ref,omitempty"`
	CapacityRequestRef string   `json:"capacity_request_ref"`
	Role               string   `json:"role"`
	Summary            string   `json:"summary"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentRequestReworkCommandV0 struct {
	ReworkRequestRef string   `json:"rework_request_ref"`
	PhaseID          string   `json:"phase_id"`
	ReviewResultRef  string   `json:"review_result_ref"`
	ReviewRequestID  string   `json:"review_request_id"`
	DeliveryRef      string   `json:"delivery_ref"`
	Summary          string   `json:"summary"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentReplanDecisionCommandV0 struct {
	ReplanRef      string   `json:"replan_ref"`
	RunRef         string   `json:"run_ref"`
	TaskRef        string   `json:"task_ref"`
	SourceRef      string   `json:"source_ref"`
	AcceptedAction string   `json:"accepted_action"`
	FollowupRefs   []string `json:"followup_refs"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}
