package orquestaobservability

const (
	WorkspaceTimelineQuerySchemaVersionV0      = "workspace_timeline_query.v0"
	WorkspaceTimelineProjectionSchemaVersionV0 = "workspace_timeline_projection.v0"
	WorkspaceTimelineScopeWorkspaceV0          = "workspace"

	WorkspaceTimelineSourceEventsV0          = "events"
	WorkspaceTimelineSourceEventV0           = WorkspaceTimelineSourceEventsV0
	WorkspaceTimelineSourceAuditV0           = "audit"
	WorkspaceTimelineSourceRunQueueV0        = "run_queue"
	WorkspaceTimelineSourceRuntimeProgressV0 = "runtime_progress"
	WorkspaceTimelineSourceGitStatsV0        = "git_stats"
	WorkspaceTimelineSourceUsageCostV0       = "usage_cost"
	WorkspaceTimelineSourceDirectorStatsV0   = "director_stats"

	WorkspaceTimelineSourceAvailableV0    = "available"
	WorkspaceTimelineSourcePartialV0      = "partial"
	WorkspaceTimelineSourceNotAvailableV0 = "not_available"

	WorkspaceTimelineCategoryEventV0      = "event"
	WorkspaceTimelineCategoryAuditV0      = "audit"
	WorkspaceTimelineCategoryQueueV0      = "queue"
	WorkspaceTimelineCategoryRuntimeV0    = "runtime"
	WorkspaceTimelineCategoryGitV0        = "git"
	WorkspaceTimelineCategoryUsageV0      = "usage"
	WorkspaceTimelineCategoryTaskV0       = "task"
	WorkspaceTimelineCategoryReviewV0     = "review"
	WorkspaceTimelineCategoryTranscriptV0 = "transcript"

	WorkspaceTimelineTranscriptNoneV0       = "none"
	WorkspaceTimelineTranscriptMetadataV0   = "metadata_only"
	WorkspaceTimelineTranscriptRedactedV0   = "redacted"
	WorkspaceTimelineTranscriptSummaryV0    = "summary"
	WorkspaceTimelineTranscriptSignalOnlyV0 = "signal_only"

	WorkspaceTimelineSchemaVersionV0 = WorkspaceTimelineProjectionSchemaVersionV0

	ErrWorkspaceTimelineQueryInvalidaV0 = ErrOperationalStatusQueryInvalidaV0
	ErrWorkspaceTimelineNoDisponibleV0  = "workspace_timeline_no_disponible"
)

const (
	maxWorkspaceTimelineSourcesV0 = 7
	maxWorkspaceTimelineEventsV0  = 50
)

var (
	allowedWorkspaceTimelineSourcesV0 = setV0(
		WorkspaceTimelineSourceEventsV0,
		WorkspaceTimelineSourceAuditV0,
		WorkspaceTimelineSourceRunQueueV0,
		WorkspaceTimelineSourceRuntimeProgressV0,
		WorkspaceTimelineSourceGitStatsV0,
		WorkspaceTimelineSourceUsageCostV0,
		WorkspaceTimelineSourceDirectorStatsV0,
	)
	allowedWorkspaceTimelineSourceStatusesV0 = setV0(
		WorkspaceTimelineSourceAvailableV0,
		WorkspaceTimelineSourcePartialV0,
		WorkspaceTimelineSourceNotAvailableV0,
	)
	allowedWorkspaceTimelineCategoriesV0 = setV0(
		WorkspaceTimelineCategoryEventV0,
		WorkspaceTimelineCategoryAuditV0,
		WorkspaceTimelineCategoryQueueV0,
		WorkspaceTimelineCategoryRuntimeV0,
		WorkspaceTimelineCategoryGitV0,
		WorkspaceTimelineCategoryUsageV0,
		WorkspaceTimelineCategoryTaskV0,
		WorkspaceTimelineCategoryReviewV0,
		WorkspaceTimelineCategoryTranscriptV0,
	)
	allowedWorkspaceTranscriptClassesV0 = setV0(
		WorkspaceTimelineTranscriptNoneV0,
		WorkspaceTimelineTranscriptMetadataV0,
		WorkspaceTimelineTranscriptRedactedV0,
		WorkspaceTimelineTranscriptSummaryV0,
		WorkspaceTimelineTranscriptSignalOnlyV0,
	)
)

type WorkspaceTimelineQueryV0 struct {
	SchemaVersion  string                               `json:"schema_version"`
	RequestID      string                               `json:"request_id"`
	CorrelationID  string                               `json:"correlation_id"`
	Consumer       OperationalStatusConsumerV0          `json:"consumer"`
	Locale         string                               `json:"locale"`
	Scope          string                               `json:"scope,omitempty"`
	WorkspaceRef   string                               `json:"workspace_ref,omitempty"`
	ProjectRef     string                               `json:"project_ref,omitempty"`
	AgentRef       string                               `json:"agent_ref,omitempty"`
	TaskRef        string                               `json:"task_ref,omitempty"`
	RunRef         string                               `json:"run_ref,omitempty"`
	TimeWindow     OperationalStatusTimeWindowV0        `json:"time_window,omitempty"`
	IncludeSources []string                             `json:"include_sources,omitempty"`
	Sources        []string                             `json:"sources,omitempty"`
	Limit          int                                  `json:"limit,omitempty"`
	Page           WorkspaceTimelinePageRequestV0       `json:"page,omitempty"`
	Freshness      *OperationalStatusFreshnessRequestV0 `json:"freshness,omitempty"`
}

type WorkspaceTimelineProjectionV0 struct {
	SchemaVersion string                            `json:"schema_version"`
	TimelineRef   string                            `json:"timeline_ref"`
	TimelineID    string                            `json:"timeline_id,omitempty"`
	GeneratedAt   string                            `json:"generated_at"`
	CorrelationID string                            `json:"correlation_id"`
	Scope         string                            `json:"scope,omitempty"`
	WorkspaceRef  string                            `json:"workspace_ref,omitempty"`
	ProjectRef    string                            `json:"project_ref,omitempty"`
	AgentRef      string                            `json:"agent_ref,omitempty"`
	TaskRef       string                            `json:"task_ref,omitempty"`
	RunRef        string                            `json:"run_ref,omitempty"`
	Filters       WorkspaceTimelineFiltersV0        `json:"filters,omitempty"`
	TimeWindow    OperationalStatusTimeWindowV0     `json:"time_window,omitempty"`
	Page          WorkspaceTimelinePageV0           `json:"page,omitempty"`
	Freshness     DiagnosticoFreshnessV0            `json:"freshness"`
	Sources       []WorkspaceTimelineSourceStatusV0 `json:"sources,omitempty"`
	Items         []WorkspaceTimelineItemV0         `json:"items"`
	Counters      map[string]float64                `json:"counters,omitempty"`
	Warnings      []DiagnosticoWarningV0            `json:"warnings,omitempty"`
	Privacy       DiagnosticoPrivacyV0              `json:"privacy"`
}

type WorkspaceTimelineV0 = WorkspaceTimelineProjectionV0

type WorkspaceTimelineFiltersV0 struct {
	AgentRef   string `json:"agent_ref,omitempty"`
	ProjectRef string `json:"project_ref,omitempty"`
	TaskRef    string `json:"task_ref,omitempty"`
}

type WorkspaceTimelinePageRequestV0 struct {
	Limit     int    `json:"limit"`
	CursorRef string `json:"cursor_ref,omitempty"`
}

type WorkspaceTimelinePageV0 struct {
	Limit     int    `json:"limit"`
	CursorRef string `json:"cursor_ref,omitempty"`
	HasMore   bool   `json:"has_more"`
}

type WorkspaceTimelineSourceStatusV0 struct {
	Source       string   `json:"source"`
	Status       string   `json:"status"`
	ReasonCode   string   `json:"reason_code,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type WorkspaceTimelineItemV0 struct {
	ItemRef                  string                              `json:"item_ref,omitempty"`
	EventRef                 string                              `json:"event_ref"`
	OccurredAt               string                              `json:"occurred_at"`
	Source                   string                              `json:"source"`
	Category                 string                              `json:"category"`
	Status                   string                              `json:"status,omitempty"`
	SummaryKey               string                              `json:"summary_key"`
	Summary                  string                              `json:"summary,omitempty"`
	Refs                     WorkspaceTimelineRefsV0             `json:"refs"`
	RunRef                   string                              `json:"run_ref,omitempty"`
	AgentRef                 string                              `json:"agent_ref,omitempty"`
	Transcript               WorkspaceTranscriptClassificationV0 `json:"transcript"`
	TranscriptClassification string                              `json:"transcript_classification,omitempty"`
	EvidenceRefs             []string                            `json:"evidence_refs,omitempty"`
}

type WorkspaceTimelineRefsV0 struct {
	RunRef      string `json:"run_ref,omitempty"`
	TaskRef     string `json:"task_ref,omitempty"`
	AgentRef    string `json:"agent_ref,omitempty"`
	ProjectRef  string `json:"project_ref,omitempty"`
	WorktreeRef string `json:"worktree_ref,omitempty"`
	AuditRef    string `json:"audit_ref,omitempty"`
}

type WorkspaceTranscriptClassificationV0 struct {
	Class      string  `json:"class"`
	Confidence float64 `json:"confidence"`
	Compact    bool    `json:"compact"`
}
