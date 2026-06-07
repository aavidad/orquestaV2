package orquestadirectorsupervisor

const (
	DirectorSupervisorBriefingSchemaV0 = "orquesta.director_supervisor.briefing.v0"
)

const (
	DirectorSupervisorActionKindRunStepV0        = "run_director_step"
	DirectorSupervisorActionKindDispatchOutboxV0 = "dispatch_outbox"
	DirectorSupervisorActionKindWaitSignalV0     = "wait_external_signal"
	DirectorSupervisorActionKindAskDirectorV0    = "ask_director"
	DirectorSupervisorActionKindReviewBlockerV0  = "review_blocker"
	DirectorSupervisorActionKindCloseOrIdleV0    = "close_or_idle"
	DirectorSupervisorActionKindStopBudgetV0     = "stop_budget_exhausted"
	DirectorSupervisorActionKindInspectErrorV0   = "inspect_error"
)

type DirectorSupervisorBriefingInputV0 struct {
	Decision     DirectorSupervisorDecisionV0 `json:"decision"`
	ObjectiveRef string                       `json:"objective_ref,omitempty"`
	ContextRefs  []string                     `json:"context_refs,omitempty"`
}

type DirectorSupervisorBriefingV0 struct {
	SchemaVersion              string                                       `json:"schema_version"`
	RunRef                     string                                       `json:"run_ref"`
	ObjectiveRef               string                                       `json:"objective_ref,omitempty"`
	DecisionAction             DirectorSupervisorActionV0                   `json:"decision_action"`
	AutonomousRecommendation   DirectorSupervisorAutonomousRecommendationV0 `json:"autonomous_recommendation"`
	ReasonCode                 string                                       `json:"reason_code"`
	NextAction                 *DirectorSupervisorRecommendedActionV0       `json:"next_action,omitempty"`
	ActionQueue                []DirectorSupervisorRecommendedActionV0      `json:"action_queue,omitempty"`
	Timeline                   []DirectorSupervisorTimelineEventV0          `json:"timeline,omitempty"`
	PendingOutboxRefs          []string                                     `json:"pending_outbox_refs,omitempty"`
	WaitingReasons             []string                                     `json:"waiting_reasons,omitempty"`
	BlockedRefs                []string                                     `json:"blocked_refs,omitempty"`
	ContextRefs                []string                                     `json:"context_refs,omitempty"`
	EvidenceRefs               []string                                     `json:"evidence_refs,omitempty"`
	StopProjectionPublicReason string                                       `json:"stop_projection_public_reason,omitempty"`
}

type DirectorSupervisorRecommendedActionV0 struct {
	ActionRef        string                     `json:"action_ref"`
	Kind             string                     `json:"kind"`
	RunRef           string                     `json:"run_ref"`
	ReasonCode       string                     `json:"reason_code"`
	Priority         int                        `json:"priority"`
	SafeToApply      bool                       `json:"safe_to_apply"`
	RequiresDirector bool                       `json:"requires_director"`
	SourceAction     DirectorSupervisorActionV0 `json:"source_action"`
	TargetRefs       []string                   `json:"target_refs,omitempty"`
	EvidenceRefs     []string                   `json:"evidence_refs,omitempty"`
}

type DirectorSupervisorTimelineEventV0 struct {
	EventRef     string                     `json:"event_ref"`
	Kind         string                     `json:"kind"`
	RunRef       string                     `json:"run_ref"`
	StepNumber   int                        `json:"step_number"`
	ReasonCode   string                     `json:"reason_code"`
	SourceAction DirectorSupervisorActionV0 `json:"source_action"`
	TargetRefs   []string                   `json:"target_refs,omitempty"`
	EvidenceRefs []string                   `json:"evidence_refs,omitempty"`
}
