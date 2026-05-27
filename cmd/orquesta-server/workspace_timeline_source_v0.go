package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

type serverWorkspaceTimelineSourceV0 struct {
	queue orquestamcp.MCPTransportRunQueuePriorityExecutorV0
	stats orquestamcp.MCPTransportDirectorStatsExecutorV0
	now   func() time.Time
}

func newServerWorkspaceTimelineSourceV0(
	bindings orquestamcp.MCPTransportBindingsV0,
) orquestaobservability.WorkspaceTimelineSourcePortV0 {
	return serverWorkspaceTimelineSourceV0{
		queue: bindings.RunQueuePriority,
		stats: bindings.DirectorStats,
		now:   time.Now,
	}
}

func (source serverWorkspaceTimelineSourceV0) QueryWorkspaceTimelineV0(
	ctx context.Context,
	query orquestaobservability.WorkspaceTimelineQueryV0,
) (orquestaobservability.WorkspaceTimelineV0, error) {
	if err := orquestaobservability.ValidateWorkspaceTimelineQueryV0(query); err != nil {
		return orquestaobservability.WorkspaceTimelineV0{}, err
	}
	occurredAt := source.timestampV0()
	items := []orquestaobservability.WorkspaceTimelineItemV0{}
	statuses := []orquestaobservability.WorkspaceTimelineSourceStatusV0{}
	for _, requested := range source.requestedSourcesV0(query) {
		sourceName := strings.TrimSpace(requested)
		available := false
		switch sourceName {
		case orquestaobservability.WorkspaceTimelineSourceRunQueueV0:
			var queueItems []orquestaobservability.WorkspaceTimelineItemV0
			queueItems, available = source.queueItemsV0(ctx, query, occurredAt)
			items = append(items, queueItems...)
		case orquestaobservability.WorkspaceTimelineSourceRuntimeProgressV0:
			var statsItems []orquestaobservability.WorkspaceTimelineItemV0
			statsItems, available = source.statsItemsV0(ctx, query, occurredAt)
			items = append(items, statsItems...)
		}
		statuses = append(statuses, source.statusV0(sourceName, available))
	}
	limit := source.limitV0(query)
	items = serverWorkspaceTimelineFilterItemsV0(items, query, occurredAt)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	timeline := orquestaobservability.WorkspaceTimelineV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineSchemaVersionV0,
		TimelineRef:   "timeline-ref-workspace-" + compactServerTimelineSuffixV0(query.CorrelationID),
		TimelineID:    "timeline-ref-workspace-" + compactServerTimelineSuffixV0(query.CorrelationID),
		GeneratedAt:   occurredAt,
		CorrelationID: strings.TrimSpace(query.CorrelationID),
		Scope:         orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		ProjectRef:    strings.TrimSpace(query.ProjectRef),
		AgentRef:      strings.TrimSpace(query.AgentRef),
		TaskRef:       strings.TrimSpace(query.TaskRef),
		TimeWindow:    query.TimeWindow,
		Page: orquestaobservability.WorkspaceTimelinePageV0{
			Limit:     limit,
			CursorRef: strings.TrimSpace(query.Page.CursorRef),
		},
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark-ref-workspace-timeline-" + compactServerTimelineSuffixV0(query.CorrelationID),
			MaxAgeSeconds: 30,
			Partial:       len(items) == 0,
		},
		Sources:  statuses,
		Items:    items,
		Counters: serverWorkspaceTimelineCountersV0(items),
		Privacy:  orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
	}
	if err := orquestaobservability.ValidateWorkspaceTimelineV0(timeline); err != nil {
		return orquestaobservability.WorkspaceTimelineV0{}, err
	}
	return timeline, nil
}

func (source serverWorkspaceTimelineSourceV0) queueItemsV0(
	ctx context.Context,
	query orquestaobservability.WorkspaceTimelineQueryV0,
	occurredAt string,
) ([]orquestaobservability.WorkspaceTimelineItemV0, bool) {
	if source.queue == nil {
		return nil, false
	}
	result, err := source.queue.Execute(ctx, orquestamcp.MCPRunQueuePriorityToolInputV0{
		RequestID:     query.RequestID,
		CorrelationID: query.CorrelationID,
		Action:        orquestamcp.MCPRunQueuePriorityActionRankV0,
		Limit:         source.limitV0(query),
		OccurredAt:    occurredAt,
	})
	if err != nil || result.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		return nil, false
	}
	items := make([]orquestaobservability.WorkspaceTimelineItemV0, 0, len(result.Ranked))
	for index, ranked := range result.Ranked {
		items = append(items, orquestaobservability.WorkspaceTimelineItemV0{
			EventRef:   fmt.Sprintf("event-ref-workspace-timeline-queue-%03d", index+1),
			OccurredAt: firstServerNonEmptyV0(ranked.UpdatedAt, occurredAt),
			Source:     orquestaobservability.WorkspaceTimelineSourceRunQueueV0,
			Category:   orquestaobservability.WorkspaceTimelineCategoryQueueV0,
			Status:     ranked.Status,
			SummaryKey: "workspace.timeline.run_queue",
			Summary:    "Run visible en cola.",
			Refs: orquestaobservability.WorkspaceTimelineRefsV0{
				RunRef:     ranked.RunRef,
				ProjectRef: ranked.AppRef,
			},
			Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
				Class:      orquestaobservability.WorkspaceTimelineTranscriptNoneV0,
				Confidence: 1,
				Compact:    true,
			},
		})
	}
	return items, true
}

func (source serverWorkspaceTimelineSourceV0) statsItemsV0(
	ctx context.Context,
	query orquestaobservability.WorkspaceTimelineQueryV0,
	occurredAt string,
) ([]orquestaobservability.WorkspaceTimelineItemV0, bool) {
	runRef := firstServerNonEmptyV0(query.RunRef, query.TaskRef)
	if source.stats == nil || strings.TrimSpace(runRef) == "" {
		return nil, false
	}
	result, err := source.stats.Execute(ctx, orquestamcp.MCPDirectorStatsToolInputV0{
		RequestID:            query.RequestID,
		CorrelationID:        query.CorrelationID,
		RunRef:               runRef,
		OccurredAt:           occurredAt,
		IncludeAgentProgress: true,
		IncludeAgentUsage:    true,
	})
	if err != nil || result.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 || result.Stats == nil {
		return nil, false
	}
	return []orquestaobservability.WorkspaceTimelineItemV0{{
		EventRef:   "event-ref-workspace-timeline-runtime-progress-001",
		OccurredAt: occurredAt,
		Source:     orquestaobservability.WorkspaceTimelineSourceRuntimeProgressV0,
		Category:   orquestaobservability.WorkspaceTimelineCategoryRuntimeV0,
		Status:     result.Stats.Status,
		SummaryKey: "workspace.timeline.runtime_progress",
		Summary:    "Run con estadisticas compactas.",
		Refs: orquestaobservability.WorkspaceTimelineRefsV0{
			RunRef:     result.Stats.RunRef,
			ProjectRef: result.Stats.ProjectRef,
			AgentRef:   query.AgentRef,
		},
		Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
			Class:      orquestaobservability.WorkspaceTimelineTranscriptNoneV0,
			Confidence: 1,
			Compact:    true,
		},
	}}, true
}

func (source serverWorkspaceTimelineSourceV0) statusV0(
	sourceName string,
	available bool,
) orquestaobservability.WorkspaceTimelineSourceStatusV0 {
	if available {
		return orquestaobservability.WorkspaceTimelineSourceStatusV0{
			Source: sourceName,
			Status: orquestaobservability.WorkspaceTimelineSourceAvailableV0,
		}
	}
	return orquestaobservability.WorkspaceTimelineSourceStatusV0{
		Source:       sourceName,
		Status:       orquestaobservability.WorkspaceTimelineSourceNotAvailableV0,
		ReasonCode:   "source_port_not_bound",
		EvidenceRefs: []string{"evidence-ref-workspace-timeline-source-not-bound"},
	}
}

func (source serverWorkspaceTimelineSourceV0) requestedSourcesV0(
	query orquestaobservability.WorkspaceTimelineQueryV0,
) []string {
	if len(query.Sources) > 0 {
		return query.Sources
	}
	return query.IncludeSources
}

func (source serverWorkspaceTimelineSourceV0) limitV0(query orquestaobservability.WorkspaceTimelineQueryV0) int {
	if query.Page.Limit > 0 {
		return query.Page.Limit
	}
	return query.Limit
}

func (source serverWorkspaceTimelineSourceV0) timestampV0() string {
	now := time.Now
	if source.now != nil {
		now = source.now
	}
	return now().UTC().Format(time.RFC3339)
}

func compactServerTimelineSuffixV0(value string) string {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), "-_.:")
	if value == "" {
		return "global"
	}
	if len(value) > 48 {
		return value[:48]
	}
	return value
}

func firstServerNonEmptyV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
