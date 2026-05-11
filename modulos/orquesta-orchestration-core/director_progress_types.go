package orquestacionnucleoapp

const (
	DirectorProgressSourceNotConfiguredV0 = "not_configured"
	DirectorProgressSourceLoadedV0        = "loaded"
	DirectorProgressSourceErrorV0         = "error"
)

const (
	DirectorTaskProgressPendingV0      = "pending"
	DirectorTaskProgressInProgressV0   = "in_progress"
	DirectorTaskProgressStalledV0      = "stalled"
	DirectorTaskProgressLoopDetectedV0 = "loop_detected"
	DirectorTaskProgressStoppedV0      = "stopped"
	DirectorTaskProgressClosedV0       = "closed"
)

const (
	DirectorProgressClassificationWorkingV0              = "working"
	DirectorProgressClassificationStalledV0              = "stalled"
	DirectorProgressClassificationOverBudgetButActiveV0  = "over_budget_but_active"
	DirectorProgressClassificationOverBudgetNoActivityV0 = "over_budget_no_activity"
	DirectorProgressClassificationAckCleanupV0           = "ack_registered_cleanup"
)

type DirectorProgressStatsV0 struct {
	SourceStatus               string                    `json:"source_status"`
	PercentComplete            int                       `json:"percent_complete"`
	TasksTotal                 int                       `json:"tasks_total"`
	TasksClosed                int                       `json:"tasks_closed"`
	TasksObserved              int                       `json:"tasks_observed"`
	ObservedAgents             int                       `json:"observed_agents"`
	ProgressingAgents          int                       `json:"progressing_agents"`
	StalledAgents              int                       `json:"stalled_agents"`
	OverBudgetButActiveAgents  int                       `json:"over_budget_but_active_agents"`
	OverBudgetNoActivityAgents int                       `json:"over_budget_no_activity_agents"`
	LoopDetectedAgents         int                       `json:"loop_detected_agents"`
	StoppedAgents              int                       `json:"stopped_agents"`
	NoSignalAgentRefs          []string                  `json:"no_signal_agent_refs,omitempty"`
	Tasks                      []DirectorTaskProgressV0  `json:"tasks,omitempty"`
	Issues                     []DirectorProgressIssueV0 `json:"issues,omitempty"`
}

type DirectorTaskProgressV0 struct {
	TaskRef             string `json:"task_ref"`
	Status              string `json:"status"`
	AgentRequestID      string `json:"agent_request_id,omitempty"`
	DeliveryRef         string `json:"delivery_ref,omitempty"`
	LastReportRef       string `json:"last_report_ref,omitempty"`
	ProgressStatus      string `json:"progress_status,omitempty"`
	NoProgressTicks     int    `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int    `json:"repeated_action_count,omitempty"`
	DirectorProgressTemporalV0
	Summary      string   `json:"summary,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DirectorProgressTemporalV0 struct {
	Classification         string `json:"classification,omitempty"`
	BudgetReason           string `json:"budget_reason,omitempty"`
	AgeSeconds             int64  `json:"age_seconds,omitempty"`
	SecondsSinceActivity   int64  `json:"seconds_since_activity,omitempty"`
	SecondsSinceAck        int64  `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds     int64  `json:"max_expected_seconds,omitempty"`
	NoActivityLimitSeconds int64  `json:"no_activity_limit_seconds,omitempty"`
	StartedAt              string `json:"started_at,omitempty"`
	LastActivityAt         string `json:"last_activity_at,omitempty"`
	LastAckAt              string `json:"last_ack_at,omitempty"`
	DecisionRequired       bool   `json:"decision_required,omitempty"`
}

type DirectorProgressIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorAgentProgressV0 struct {
	TaskRef             string `json:"task_ref,omitempty"`
	DeliveryRef         string `json:"delivery_ref,omitempty"`
	ReportRef           string `json:"report_ref,omitempty"`
	Status              string `json:"status,omitempty"`
	NoProgressTicks     int    `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int    `json:"repeated_action_count,omitempty"`
	DirectorProgressTemporalV0
	Summary      string   `json:"summary,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DirectorProgressSourceRequestV0 struct {
	StepNumber    int
	MaxSteps      int
	OccurredAt    string
	CorrelationID string
	EvidenceRefs  []string
}
