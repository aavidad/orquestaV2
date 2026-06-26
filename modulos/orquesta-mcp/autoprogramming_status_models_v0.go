package orquestamcp

type MCPAutoprogrammingProjectV0 struct {
	ProjectRef     string   `json:"project_ref,omitempty"`
	AppRef         string   `json:"app_ref,omitempty"`
	RunRefs        []string `json:"run_refs,omitempty"`
	QueueCount     int      `json:"queue_count,omitempty"`
	TasksTotal     int      `json:"tasks_total,omitempty"`
	TasksClosed    int      `json:"tasks_closed,omitempty"`
	AgentsInFlight int      `json:"agents_in_flight,omitempty"`
	Blocked        bool     `json:"blocked,omitempty"`
}

type MCPAutoprogrammingQueueHealthV0 struct {
	Queued                    int `json:"queued,omitempty"`
	RunningLive               int `json:"running_live,omitempty"`
	RunningStale              int `json:"running_stale,omitempty"`
	RunningStaleNoProcess     int `json:"running_stale_no_process,omitempty"`
	RunningWithoutRecentStats int `json:"running_without_recent_stats,omitempty"`
	Blocked                   int `json:"blocked,omitempty"`
	Lost                      int `json:"lost,omitempty"`
	Completed                 int `json:"completed,omitempty"`
	Failed                    int `json:"failed,omitempty"`
	Unclassified              int `json:"unclassified,omitempty"`
	ObservedRuns              int `json:"observed_runs,omitempty"`
	QueueRuns                 int `json:"queue_runs,omitempty"`
	StatsRuns                 int `json:"stats_runs,omitempty"`
}

type MCPAutoprogrammingActionableRunV0 struct {
	Code              string   `json:"code"`
	Severity          string   `json:"severity,omitempty"`
	RunRef            string   `json:"run_ref,omitempty"`
	AppRef            string   `json:"app_ref,omitempty"`
	Status            string   `json:"status,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	RecommendedAction string   `json:"recommended_action,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingTaskV0 struct {
	RunRef              string   `json:"run_ref,omitempty"`
	TaskRef             string   `json:"task_ref"`
	Status              string   `json:"status,omitempty"`
	AgentRequestID      string   `json:"agent_request_id,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	LastReportRef       string   `json:"last_report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	Classification      string   `json:"classification,omitempty"`
	BudgetReason        string   `json:"budget_reason,omitempty"`
	AgeSeconds          int64    `json:"age_seconds,omitempty"`
	SecondsSinceAck     int64    `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds  int64    `json:"max_expected_seconds,omitempty"`
	NoActivityLimit     int64    `json:"no_activity_limit_seconds,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	DecisionRequired    bool     `json:"decision_required,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingAgentV0 struct {
	RunRef              string   `json:"run_ref,omitempty"`
	AgentRequestID      string   `json:"agent_request_id"`
	Status              string   `json:"status,omitempty"`
	InFlight            bool     `json:"in_flight,omitempty"`
	NeedsAttention      bool     `json:"needs_attention,omitempty"`
	Completed           bool     `json:"completed,omitempty"`
	Failed              bool     `json:"failed,omitempty"`
	Lost                bool     `json:"lost,omitempty"`
	StopRequested       bool     `json:"stop_requested,omitempty"`
	StopConfirmed       bool     `json:"stop_confirmed,omitempty"`
	StopReasonCode      string   `json:"stop_reason_code,omitempty"`
	StopReasonSource    string   `json:"stop_reason_source,omitempty"`
	StopReasonRef       string   `json:"stop_reason_ref,omitempty"`
	ControlRegistered   bool     `json:"control_registered,omitempty"`
	ControlState        string   `json:"control_state,omitempty"`
	CanStop             bool     `json:"can_stop,omitempty"`
	TaskRef             string   `json:"task_ref,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	ReportRef           string   `json:"report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	Classification      string   `json:"classification,omitempty"`
	BudgetReason        string   `json:"budget_reason,omitempty"`
	AgeSeconds          int64    `json:"age_seconds,omitempty"`
	SecondsSinceAck     int64    `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds  int64    `json:"max_expected_seconds,omitempty"`
	NoActivityLimit     int64    `json:"no_activity_limit_seconds,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	DecisionRequired    bool     `json:"decision_required,omitempty"`
	ProcessRef          string   `json:"process_ref,omitempty"`
	SessionRef          string   `json:"session_ref,omitempty"`
	LaunchRef           string   `json:"launch_ref,omitempty"`
	RuntimeKind         string   `json:"runtime_kind,omitempty"`
	ConnectorRef        string   `json:"connector_ref,omitempty"`
	ProfileRef          string   `json:"profile_ref,omitempty"`
	CapacityLevel       string   `json:"capacity_level,omitempty"`
	QuotaStatus         string   `json:"quota_status,omitempty"`
	TotalTokens         int64    `json:"total_tokens,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
