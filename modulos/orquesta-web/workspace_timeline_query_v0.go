package orquestaweb

import orquestaobservability "orquesta/modulos/orquesta-observability"

type WebWorkspaceTimelineQueryInputV0 struct {
	RequestID     string
	CorrelationID string
	Locale        string
	AgentRef      string
	ProjectRef    string
	TaskRef       string
	From          string
	To            string
	Preset        string
	Limit         int
	CursorRef     string
	Sources       []string
}

func NewWebWorkspaceTimelineQueryV0(input WebWorkspaceTimelineQueryInputV0) orquestaobservability.WorkspaceTimelineQueryV0 {
	locale := trimOperationalStatusV0(input.Locale)
	if locale == "" {
		locale = "es-ES"
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	sources := append([]string(nil), input.Sources...)
	if len(sources) == 0 {
		sources = []string{
			orquestaobservability.WorkspaceTimelineSourceEventsV0,
			orquestaobservability.WorkspaceTimelineSourceAuditV0,
			orquestaobservability.WorkspaceTimelineSourceRunQueueV0,
			orquestaobservability.WorkspaceTimelineSourceRuntimeProgressV0,
			orquestaobservability.WorkspaceTimelineSourceGitStatsV0,
			orquestaobservability.WorkspaceTimelineSourceUsageCostV0,
		}
	}
	return orquestaobservability.WorkspaceTimelineQueryV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     trimOperationalStatusV0(input.RequestID),
		CorrelationID: trimOperationalStatusV0(input.CorrelationID),
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:     locale,
		Scope:      orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		AgentRef:   trimOperationalStatusV0(input.AgentRef),
		ProjectRef: trimOperationalStatusV0(input.ProjectRef),
		TaskRef:    trimOperationalStatusV0(input.TaskRef),
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{
			From:   trimOperationalStatusV0(input.From),
			To:     trimOperationalStatusV0(input.To),
			Preset: trimOperationalStatusV0(input.Preset),
		},
		Page: orquestaobservability.WorkspaceTimelinePageRequestV0{
			Limit:     limit,
			CursorRef: trimOperationalStatusV0(input.CursorRef),
		},
		IncludeSources: sources,
		Sources:        sources,
		Limit:          limit,
	}
}
