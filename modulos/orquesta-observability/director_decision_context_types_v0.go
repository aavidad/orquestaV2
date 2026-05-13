package orquestaobservability

const (
	DirectorDecisionContextSchemaVersionV0 = "director_decision_context.v0"

	DirectorDecisionActivityPhaseCurrentV0   = "phase_current"
	DirectorDecisionActivityTaskProgressV0   = "task_progress"
	DirectorDecisionActivityAgentLifecycleV0 = "agent_lifecycle"
	DirectorDecisionActivityClosureBlockedV0 = "closure_blocked"
	DirectorDecisionActivityReworkRequestV0  = "rework_request"
	DirectorDecisionActivityReplanDecisionV0 = "replan_decision"
)

type DirectorDecisionContextV0 struct {
	SchemaVersion string                         `json:"schema_version"`
	RunRef        string                         `json:"run_ref"`
	ObservedAt    string                         `json:"observed_at,omitempty"`
	CurrentPhase  string                         `json:"current_phase,omitempty"`
	Progress      DirectorDecisionProgressV0     `json:"progress"`
	Lifecycle     DirectorDecisionLifecycleV0    `json:"lifecycle"`
	Closure       DirectorDecisionClosureV0      `json:"closure"`
	Quietness     DirectorDecisionQuietnessV0    `json:"quietness"`
	ReworkReplan  DirectorDecisionReworkReplanV0 `json:"rework_replan"`
	Phases        []DirectorDecisionPhaseV0      `json:"phases,omitempty"`
	Tasks         []DirectorDecisionTaskV0       `json:"tasks,omitempty"`
	Agents        []DirectorDecisionAgentV0      `json:"agents,omitempty"`
	Blockers      []DirectorDecisionBlockerV0    `json:"blockers,omitempty"`
	Activity      []DirectorDecisionActivityV0   `json:"activity_recent,omitempty"`
	Warnings      []DirectorDecisionWarningV0    `json:"warnings,omitempty"`
	Privacy       DiagnosticoPrivacyV0           `json:"privacy"`
}

type DirectorDecisionProgressV0 struct {
	PercentComplete    int      `json:"percent_complete"`
	TasksTotal         int      `json:"tasks_total"`
	TasksClosed        int      `json:"tasks_closed"`
	TasksOpen          int      `json:"tasks_open"`
	TasksObserved      int      `json:"tasks_observed"`
	ObservedAgents     int      `json:"observed_agents"`
	ProgressingAgents  int      `json:"progressing_agents"`
	StalledAgents      int      `json:"stalled_agents"`
	LoopDetectedAgents int      `json:"loop_detected_agents"`
	StoppedAgents      int      `json:"stopped_agents"`
	NoSignalAgentRefs  []string `json:"no_signal_agent_refs,omitempty"`
}

type DirectorDecisionLifecycleV0 struct {
	AgentsRequested         int `json:"agents_requested"`
	AgentsStarted           int `json:"agents_started"`
	AgentsRunning           int `json:"agents_running"`
	AgentsFailed            int `json:"agents_failed"`
	AgentsStopRequested     int `json:"agents_stop_requested"`
	AgentsStopped           int `json:"agents_stopped"`
	AgentsInFlight          int `json:"agents_in_flight"`
	AgentsControlRegistered int `json:"agents_control_registered"`
	AgentsControlMissing    int `json:"agents_control_missing"`
	AgentsNeedAttention     int `json:"agents_need_attention"`
}

type DirectorDecisionClosureV0 struct {
	Status      string   `json:"status"`
	Blocked     bool     `json:"blocked"`
	Ready       bool     `json:"ready"`
	Closed      bool     `json:"closed"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	BlockerRefs []string `json:"blocker_refs,omitempty"`
}

type DirectorDecisionQuietnessV0 struct {
	NoSignalAgentRefs  []string `json:"no_signal_agent_refs,omitempty"`
	TasksWithoutSignal int      `json:"tasks_without_signal"`
	StalledAgents      int      `json:"stalled_agents"`
	LoopDetectedAgents int      `json:"loop_detected_agents"`
	StoppedAgents      int      `json:"stopped_agents"`
	MaxNoProgressTicks int      `json:"max_no_progress_ticks,omitempty"`
}

type DirectorDecisionReworkReplanV0 struct {
	ReworkRequests     int      `json:"rework_requests"`
	ReworkRequestRefs  []string `json:"rework_request_refs,omitempty"`
	ReplanDecisions    int      `json:"replan_decisions"`
	ReplanDecisionRefs []string `json:"replan_decision_refs,omitempty"`
}

type DirectorDecisionPhaseV0 struct {
	PhaseID             string `json:"phase_id"`
	Status              string `json:"status,omitempty"`
	Current             bool   `json:"current,omitempty"`
	OpenedAt            string `json:"opened_at,omitempty"`
	ClosedAt            string `json:"closed_at,omitempty"`
	DurationSeconds     int64  `json:"duration_seconds,omitempty"`
	RecommendedCapacity string `json:"recommended_capacity,omitempty"`
	EntryCriteria       int    `json:"entry_criteria"`
	ExitCriteria        int    `json:"exit_criteria"`
	EvidenceRequired    int    `json:"evidence_required"`
}

type DirectorDecisionTaskV0 struct {
	TaskRef             string   `json:"task_ref"`
	Status              string   `json:"status"`
	AgentRequestID      string   `json:"agent_request_id,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	LastReportRef       string   `json:"last_report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type DirectorDecisionAgentV0 struct {
	AgentRequestID      string   `json:"agent_request_id"`
	Status              string   `json:"status"`
	ControlState        string   `json:"control_state,omitempty"`
	Running             bool     `json:"running"`
	Failed              bool     `json:"failed"`
	StopRequested       bool     `json:"stop_requested"`
	Stopped             bool     `json:"stopped"`
	StopReasonCode      string   `json:"stop_reason_code,omitempty"`
	StopReasonSource    string   `json:"stop_reason_source,omitempty"`
	InFlight            bool     `json:"in_flight"`
	NeedsAttention      bool     `json:"needs_attention"`
	CanStop             bool     `json:"can_stop"`
	ProcessRef          string   `json:"process_ref,omitempty"`
	SessionRef          string   `json:"session_ref,omitempty"`
	LaunchRef           string   `json:"launch_ref,omitempty"`
	ReadinessRef        string   `json:"readiness_ref,omitempty"`
	ProgressTaskRef     string   `json:"progress_task_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type DirectorDecisionBlockerV0 struct {
	BlockerRef string `json:"blocker_ref"`
	Cause      string `json:"cause,omitempty"`
	Source     string `json:"source"`
	SummaryKey string `json:"summary_key"`
}

type DirectorDecisionActivityV0 struct {
	ActivityRef    string `json:"activity_ref"`
	Kind           string `json:"kind"`
	OccurredAt     string `json:"occurred_at,omitempty"`
	SourceRef      string `json:"source_ref,omitempty"`
	AgentRequestID string `json:"agent_request_id,omitempty"`
	TaskRef        string `json:"task_ref,omitempty"`
	SummaryKey     string `json:"summary_key"`
}

type DirectorDecisionWarningV0 struct {
	Code       string `json:"code"`
	SummaryKey string `json:"summary_key"`
}
