package orquestaweb

const (
	WebDirectorStatsInboundEndpointV0      = "/api/v0/director/stats"
	WebDirectorStatsPageEndpointV0         = "/director-stats"
	WebDirectorStatsGoalObserveEndpointV0  = "/api/v0/apps/director/goal/observe"
	WebDirectorStatsCorrelationHeaderV0    = "X-Correlation-ID"
	WebDirectorStatsPageSchemaV0           = "web_director_stats_page.v0"
	WebDirectorStatsPanelSchemaV0          = "web_director_stats_panel.v0"
	WebDirectorStatsRefreshIntervalMsV0    = 5000
	WebDirectorStatsInboundEstadoOKV0      = "ok"
	WebDirectorStatsInboundEstadoErrorV0   = "error"
	WebDirectorStatsEstadoOKV0             = "ok"
	WebDirectorStatsEstadoAtencionV0       = "attention"
	WebDirectorStatsEstadoErrorV0          = "error"
	WebDirectorStatsErrRunRefRequeridoV0   = webPublicErrRunRefRequiredV0
	WebDirectorStatsErrTransporteV0        = webPublicErrTransportV0
	WebDirectorStatsErrRespuestaInvalidaV0 = webPublicErrResponseInvalidV0
)

const DirectorStatsEndpointV0 = WebDirectorStatsInboundEndpointV0

type WebDirectorStatsQueryV0 struct {
	RequestID            string `json:"request_id,omitempty"`
	CorrelationID        string `json:"correlation_id,omitempty"`
	Locale               string `json:"locale,omitempty"`
	RunRef               string `json:"run_ref"`
	AppRef               string `json:"app_ref,omitempty"`
	ExternalJobRef       string `json:"external_job_ref,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
	IncludeProcessRefs   bool   `json:"include_process_refs,omitempty"`
	IncludeAgentProgress bool   `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    bool   `json:"include_agent_usage,omitempty"`
}

type WebDirectorStatsInboundResultV0 struct {
	Estado        string                          `json:"estado"`
	RequestID     string                          `json:"request_id,omitempty"`
	CorrelationID string                          `json:"correlation_id,omitempty"`
	RunRef        string                          `json:"run_ref,omitempty"`
	ExternalJob   *WebDirectorExternalJobV0       `json:"external_job,omitempty"`
	Goal          *WebDirectorGoalStatsContractV0 `json:"goal,omitempty"`
	Stats         *WebDirectorRunStatsContractV0  `json:"stats,omitempty"`
	Errores       []WebDirectorStatsPublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebDirectorExternalJobV0 struct {
	Available    bool                                 `json:"available,omitempty"`
	AppRef       string                               `json:"app_ref,omitempty"`
	JobRef       string                               `json:"job_ref,omitempty"`
	WorkKind     string                               `json:"work_kind,omitempty"`
	ChangeRef    string                               `json:"change_ref,omitempty"`
	RunRef       string                               `json:"run_ref,omitempty"`
	TaskRef      string                               `json:"task_ref,omitempty"`
	AgentRef     string                               `json:"agent_ref,omitempty"`
	Status       string                               `json:"status,omitempty"`
	StatusReason string                               `json:"status_reason,omitempty"`
	DeliveryRefs []string                             `json:"delivery_refs,omitempty"`
	IssueRefs    []string                             `json:"issue_refs,omitempty"`
	EvidenceRefs []string                             `json:"evidence_refs,omitempty"`
	Diagnostics  []WebDirectorExternalJobDiagnosticV0 `json:"diagnostics,omitempty"`
}

type WebDirectorExternalJobDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type WebDirectorGoalStatsContractV0 struct {
	DirectorExecutionMode string   `json:"director_execution_mode,omitempty"`
	RunRef                string   `json:"run_ref,omitempty"`
	GoalRef               string   `json:"goal_ref"`
	ExternalGoalRef       string   `json:"external_goal_ref,omitempty"`
	Status                string   `json:"status,omitempty"`
	ClosureStatus         string   `json:"closure_status,omitempty"`
	ClosureAccepted       bool     `json:"closure_accepted,omitempty"`
	ClosureNeedsRework    bool     `json:"closure_needs_rework,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
}

type WebDirectorRunStatsContractV0 struct {
	SchemaVersion              string                             `json:"schema_version"`
	RunRef                     string                             `json:"run_ref"`
	ProjectRef                 string                             `json:"project_ref,omitempty"`
	AppSpecRef                 string                             `json:"app_spec_ref,omitempty"`
	Status                     string                             `json:"status,omitempty"`
	CurrentPhase               string                             `json:"current_phase,omitempty"`
	Counts                     map[string]int                     `json:"counts"`
	Progress                   WebDirectorProgressStatsContractV0 `json:"progress"`
	Closure                    WebDirectorClosureStatsContractV0  `json:"closure"`
	StopControl                WebDirectorStopControlContractV0   `json:"stop_control"`
	UsageSummary               *WebDirectorRunUsageSummaryV0      `json:"usage_summary,omitempty"`
	Agents                     []WebDirectorAgentStatsContractV0  `json:"agents,omitempty"`
	CheckpointAgentsPending    int                                `json:"checkpoint_agents_pending,omitempty"`
	PendingCheckpointAgentRefs []string                           `json:"pending_checkpoint_agent_refs,omitempty"`
	CheckpointEvidenceRefs     []string                           `json:"checkpoint_evidence_refs,omitempty"`
}

type WebDirectorClosureStatsContractV0 struct {
	Status      string   `json:"status"`
	Blocked     bool     `json:"blocked"`
	Ready       bool     `json:"ready"`
	Closed      bool     `json:"closed"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	BlockerRefs []string `json:"blocker_refs,omitempty"`
}

type WebDirectorStopControlContractV0 struct {
	Status                 string   `json:"status"`
	Requested              bool     `json:"requested"`
	Propagated             bool     `json:"propagated"`
	Pending                bool     `json:"pending"`
	Confirmed              bool     `json:"confirmed"`
	RunControlStatus       string   `json:"run_control_status,omitempty"`
	CheckpointRecorded     bool     `json:"checkpoint_recorded,omitempty"`
	Forced                 bool     `json:"forced,omitempty"`
	StartedAgents          int      `json:"started_agents"`
	StopRequestedAgents    int      `json:"stop_requested_agents"`
	StopConfirmedAgents    int      `json:"stop_confirmed_agents"`
	StopPendingAgents      int      `json:"stop_pending_agents"`
	PendingAgentRefs       []string `json:"pending_agent_refs,omitempty"`
	StopRequestedAgentRefs []string `json:"stop_requested_agent_refs,omitempty"`
	StopConfirmedAgentRefs []string `json:"stop_confirmed_agent_refs,omitempty"`
	EvidenceRefs           []string `json:"evidence_refs,omitempty"`
}

type WebDirectorProgressStatsContractV0 struct {
	SourceStatus       string                          `json:"source_status"`
	PercentComplete    int                             `json:"percent_complete"`
	TasksTotal         int                             `json:"tasks_total"`
	TasksClosed        int                             `json:"tasks_closed"`
	TasksObserved      int                             `json:"tasks_observed"`
	ObservedAgents     int                             `json:"observed_agents"`
	ProgressingAgents  int                             `json:"progressing_agents"`
	StalledAgents      int                             `json:"stalled_agents"`
	LoopDetectedAgents int                             `json:"loop_detected_agents"`
	StoppedAgents      int                             `json:"stopped_agents"`
	NoSignalAgentRefs  []string                        `json:"no_signal_agent_refs,omitempty"`
	Tasks              []WebDirectorTaskProgressV0     `json:"tasks,omitempty"`
	Issues             []WebDirectorStatsPublicIssueV0 `json:"issues,omitempty"`
}

type WebDirectorAgentStatsContractV0 struct {
	AgentRequestID   string                      `json:"agent_request_id"`
	Status           string                      `json:"status"`
	ControlState     string                      `json:"control_state,omitempty"`
	InFlight         bool                        `json:"in_flight"`
	NeedsAttention   bool                        `json:"needs_attention"`
	CanStop          bool                        `json:"can_stop"`
	StopReasonCode   string                      `json:"stop_reason_code,omitempty"`
	StopReasonSource string                      `json:"stop_reason_source,omitempty"`
	StopReasonRef    string                      `json:"stop_reason_ref,omitempty"`
	LastProgress     *WebDirectorAgentProgressV0 `json:"last_progress,omitempty"`
	Usage            *WebDirectorAgentUsageV0    `json:"usage,omitempty"`
}

type WebDirectorTaskProgressV0 struct {
	TaskRef             string   `json:"task_ref"`
	Status              string   `json:"status"`
	AgentRequestID      string   `json:"agent_request_id,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	LastReportRef       string   `json:"last_report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type WebDirectorAgentProgressV0 struct {
	TaskRef             string   `json:"task_ref,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	ReportRef           string   `json:"report_ref,omitempty"`
	Status              string   `json:"status,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type WebDirectorAgentUsageV0 struct {
	RuntimeKind      string `json:"runtime_kind,omitempty"`
	ConnectorRef     string `json:"connector_ref,omitempty"`
	ProfileRef       string `json:"profile_ref,omitempty"`
	CapacityLevel    string `json:"capacity_level,omitempty"`
	QuotaStatus      string `json:"quota_status,omitempty"`
	QuotaRemaining   int64  `json:"quota_remaining,omitempty"`
	QuotaLimit       int64  `json:"quota_limit,omitempty"`
	QuotaResetAt     string `json:"quota_reset_at,omitempty"`
	PromptTokens     int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens int64  `json:"completion_tokens,omitempty"`
	TotalTokens      int64  `json:"total_tokens,omitempty"`
}

type WebDirectorRunUsageSummaryV0 struct {
	AgentsObserved   int    `json:"agents_observed"`
	QuotaStatus      string `json:"quota_status,omitempty"`
	QuotaRemaining   int64  `json:"quota_remaining,omitempty"`
	QuotaLimit       int64  `json:"quota_limit,omitempty"`
	PromptTokens     int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens int64  `json:"completion_tokens,omitempty"`
	TotalTokens      int64  `json:"total_tokens,omitempty"`
}

type WebDirectorStatsPublicIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}
