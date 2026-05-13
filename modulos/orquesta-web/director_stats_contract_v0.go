package orquestaweb

const (
	WebDirectorStatsInboundEndpointV0      = "/api/v0/director/stats"
	WebDirectorStatsPageEndpointV0         = "/director-stats"
	WebDirectorStatsCorrelationHeaderV0    = "X-Correlation-ID"
	WebDirectorStatsPageSchemaV0           = "web_director_stats_page.v0"
	WebDirectorStatsPanelSchemaV0          = "web_director_stats_panel.v0"
	WebDirectorStatsRefreshIntervalMsV0    = 5000
	WebDirectorStatsInboundEstadoOKV0      = "ok"
	WebDirectorStatsInboundEstadoErrorV0   = "error"
	WebDirectorStatsEstadoOKV0             = "ok"
	WebDirectorStatsEstadoAtencionV0       = "attention"
	WebDirectorStatsEstadoErrorV0          = "error"
	WebDirectorStatsErrRunRefRequeridoV0   = "run_ref_requerido"
	WebDirectorStatsErrTransporteV0        = "error_transporte"
	WebDirectorStatsErrRespuestaInvalidaV0 = "respuesta_invalida"
)

const DirectorStatsEndpointV0 = WebDirectorStatsInboundEndpointV0

type WebDirectorStatsQueryV0 struct {
	RequestID            string `json:"request_id,omitempty"`
	CorrelationID        string `json:"correlation_id,omitempty"`
	Locale               string `json:"locale,omitempty"`
	RunRef               string `json:"run_ref"`
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
	Stats         *WebDirectorRunStatsContractV0  `json:"stats,omitempty"`
	Errores       []WebDirectorStatsPublicIssueV0 `json:"errores_publicos,omitempty"`
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
	UsageSummary               *WebDirectorRunUsageSummaryV0      `json:"usage_summary,omitempty"`
	Agents                     []WebDirectorAgentStatsContractV0  `json:"agents,omitempty"`
	CheckpointAgentsPending    int                                `json:"checkpoint_agents_pending,omitempty"`
	PendingCheckpointAgentRefs []string                           `json:"pending_checkpoint_agent_refs,omitempty"`
	CheckpointEvidenceRefs     []string                           `json:"checkpoint_evidence_refs,omitempty"`
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
	AgentRequestID string                      `json:"agent_request_id"`
	Status         string                      `json:"status"`
	ControlState   string                      `json:"control_state,omitempty"`
	InFlight       bool                        `json:"in_flight"`
	NeedsAttention bool                        `json:"needs_attention"`
	CanStop        bool                        `json:"can_stop"`
	LastProgress   *WebDirectorAgentProgressV0 `json:"last_progress,omitempty"`
	Usage          *WebDirectorAgentUsageV0    `json:"usage,omitempty"`
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
	ModelAlias       string `json:"model_alias,omitempty"`
	CapacityLevel    string `json:"capacity_level,omitempty"`
	ReasoningEffort  string `json:"reasoning_effort,omitempty"`
	QuotaStatus      string `json:"quota_status,omitempty"`
	QuotaRemaining   int64  `json:"quota_remaining,omitempty"`
	QuotaLimit       int64  `json:"quota_limit,omitempty"`
	QuotaResetAt     string `json:"quota_reset_at,omitempty"`
	PromptTokens     int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens int64  `json:"completion_tokens,omitempty"`
	TotalTokens      int64  `json:"total_tokens,omitempty"`
	CostMicros       int64  `json:"cost_micros,omitempty"`
}

type WebDirectorRunUsageSummaryV0 struct {
	AgentsObserved   int    `json:"agents_observed"`
	QuotaStatus      string `json:"quota_status,omitempty"`
	QuotaRemaining   int64  `json:"quota_remaining,omitempty"`
	QuotaLimit       int64  `json:"quota_limit,omitempty"`
	PromptTokens     int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens int64  `json:"completion_tokens,omitempty"`
	TotalTokens      int64  `json:"total_tokens,omitempty"`
	CostMicros       int64  `json:"cost_micros,omitempty"`
}

type WebDirectorStatsPublicIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}
