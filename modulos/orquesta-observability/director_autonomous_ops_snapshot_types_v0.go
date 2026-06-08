package orquestaobservability

const (
	DirectorAutonomousOpsSnapshotSchemaVersionV0 = "director_autonomous_ops_snapshot.v0"

	DirectorAutonomousOpsActionReviewReplanV0    = "review_replan"
	DirectorAutonomousOpsActionWaitDeliveriesV0  = "wait_deliveries"
	DirectorAutonomousOpsActionSuperviseQueueV0  = "supervise_queue"
	DirectorAutonomousOpsActionCloseOrValidateV0 = "close_or_validate"
	DirectorAutonomousOpsActionClosedV0          = "closed"
	DirectorAutonomousOpsActionContinueRunV0     = "continue_run"
	DirectorAutonomousOpsActionIdleV0            = "idle"
)

type DirectorAutonomousOpsSnapshotV0 struct {
	SchemaVersion string                          `json:"schema_version"`
	SnapshotRef   string                          `json:"snapshot_ref,omitempty"`
	ObservedAt    string                          `json:"observed_at,omitempty"`
	Queue         DirectorAutonomousOpsQueueV0    `json:"queue"`
	Runs          []DirectorAutonomousOpsRunV0    `json:"runs,omitempty"`
	Agents        []DirectorAutonomousOpsAgentV0  `json:"agents,omitempty"`
	Decision      DirectorAutonomousOpsDecisionV0 `json:"decision"`
	Privacy       DiagnosticoPrivacyV0            `json:"privacy"`
}

type DirectorAutonomousOpsQueueV0 struct {
	QueueRef      string   `json:"queue_ref,omitempty"`
	Live          bool     `json:"live"`
	Count         int      `json:"count"`
	RankedRunRefs []string `json:"ranked_run_refs,omitempty"`
}

type DirectorAutonomousOpsRunV0 struct {
	RunRef              string   `json:"run_ref"`
	AppRef              string   `json:"app_ref,omitempty"`
	Status              string   `json:"status,omitempty"`
	CurrentPhase        string   `json:"current_phase,omitempty"`
	ClosureStatus       string   `json:"closure_status,omitempty"`
	Blocked             bool     `json:"blocked,omitempty"`
	PercentComplete     int      `json:"percent_complete"`
	TasksTotal          int      `json:"tasks_total"`
	TasksClosed         int      `json:"tasks_closed"`
	TasksOpen           int      `json:"tasks_open"`
	AgentsInFlight      int      `json:"agents_in_flight"`
	AgentsNeedAttention int      `json:"agents_need_attention"`
	AgentsFailed        int      `json:"agents_failed"`
	ReworkRequests      int      `json:"rework_requests"`
	ReplanDecisions     int      `json:"replan_decisions"`
	QuotaStatus         string   `json:"quota_status,omitempty"`
	QuotaRemaining      int64    `json:"quota_remaining,omitempty"`
	QuotaLimit          int64    `json:"quota_limit,omitempty"`
	TotalTokens         int64    `json:"total_tokens,omitempty"`
	WaitAgentRefs       []string `json:"wait_agent_refs,omitempty"`
	WaveRefs            []string `json:"wave_refs,omitempty"`
	CohortRefs          []string `json:"cohort_refs,omitempty"`
}

type DirectorAutonomousOpsAgentV0 struct {
	RunRef         string `json:"run_ref,omitempty"`
	AgentRef       string `json:"agent_ref"`
	TaskRef        string `json:"task_ref,omitempty"`
	Status         string `json:"status,omitempty"`
	InFlight       bool   `json:"in_flight,omitempty"`
	NeedsAttention bool   `json:"needs_attention,omitempty"`
	CanStop        bool   `json:"can_stop,omitempty"`
	ProgressStatus string `json:"progress_status,omitempty"`
	RuntimeKind    string `json:"runtime_kind,omitempty"`
	CapacityLevel  string `json:"capacity_level,omitempty"`
	QuotaStatus    string `json:"quota_status,omitempty"`
	TotalTokens    int64  `json:"total_tokens,omitempty"`
}

type DirectorAutonomousOpsDecisionV0 struct {
	Action       string   `json:"action"`
	Scope        string   `json:"scope,omitempty"`
	RunRef       string   `json:"run_ref,omitempty"`
	Attention    bool     `json:"attention,omitempty"`
	ReasonCode   string   `json:"reason_code,omitempty"`
	SummaryKey   string   `json:"summary_key,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
