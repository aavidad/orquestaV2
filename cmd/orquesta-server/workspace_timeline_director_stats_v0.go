package main

import (
	"context"
	"fmt"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func (source serverWorkspaceTimelineSourceV0) directorStatsItemsV0(
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
	if err != nil ||
		result.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 ||
		result.DecisionContext == nil {
		return nil, false
	}
	decisionContext := *result.DecisionContext
	if strings.TrimSpace(decisionContext.RunRef) == "" {
		decisionContext.RunRef = firstServerNonEmptyV0(result.RunRef, runRef)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(decisionContext); err != nil {
		return nil, false
	}
	observedAt := firstServerNonEmptyV0(decisionContext.ObservedAt, occurredAt)
	projectRef := strings.TrimSpace(query.ProjectRef)
	if result.Stats != nil {
		projectRef = firstServerNonEmptyV0(result.Stats.ProjectRef, projectRef)
	}
	items := []orquestaobservability.WorkspaceTimelineItemV0{
		serverWorkspaceTimelineDirectorSnapshotItemV0(decisionContext, result, projectRef, observedAt),
	}
	for index, activity := range decisionContext.Activity {
		items = append(items, serverWorkspaceTimelineDirectorActivityItemV0(
			decisionContext,
			activity,
			projectRef,
			observedAt,
			index+1,
		))
	}
	return items, true
}

func serverWorkspaceTimelineDirectorSnapshotItemV0(
	decisionContext orquestaobservability.DirectorDecisionContextV0,
	result orquestamcp.MCPDirectorStatsToolResultV0,
	projectRef string,
	observedAt string,
) orquestaobservability.WorkspaceTimelineItemV0 {
	status := firstServerNonEmptyV0(decisionContext.Closure.Status, "observed")
	if result.Stats != nil {
		status = firstServerNonEmptyV0(decisionContext.Closure.Status, result.Stats.Status, "observed")
	}
	return orquestaobservability.WorkspaceTimelineItemV0{
		EventRef:   "event-ref-workspace-timeline-director-stats-" + compactServerTimelineSuffixV0(decisionContext.RunRef),
		OccurredAt: observedAt,
		Source:     orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0,
		Category:   orquestaobservability.WorkspaceTimelineCategoryRuntimeV0,
		Status:     status,
		SummaryKey: "workspace.timeline.director_stats.snapshot",
		Summary: fmt.Sprintf(
			"Director: fase %s, %d%%, %d/%d tareas cerradas.",
			firstServerNonEmptyV0(decisionContext.CurrentPhase, "sin_fase"),
			decisionContext.Progress.PercentComplete,
			decisionContext.Progress.TasksClosed,
			decisionContext.Progress.TasksTotal,
		),
		Refs: orquestaobservability.WorkspaceTimelineRefsV0{
			RunRef:     decisionContext.RunRef,
			ProjectRef: projectRef,
		},
		EvidenceRefs: compactServerTimelineRefsV0(decisionContext.Closure.BlockerRefs),
		Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
			Class:      orquestaobservability.WorkspaceTimelineTranscriptMetadataV0,
			Confidence: 1,
			Compact:    true,
		},
	}
}

func serverWorkspaceTimelineDirectorActivityItemV0(
	decisionContext orquestaobservability.DirectorDecisionContextV0,
	activity orquestaobservability.DirectorDecisionActivityV0,
	projectRef string,
	observedAt string,
	index int,
) orquestaobservability.WorkspaceTimelineItemV0 {
	refs := orquestaobservability.WorkspaceTimelineRefsV0{
		RunRef:     decisionContext.RunRef,
		ProjectRef: projectRef,
		TaskRef:    strings.TrimSpace(activity.TaskRef),
		AgentRef:   strings.TrimSpace(activity.AgentRequestID),
	}
	sourceRef := strings.TrimSpace(activity.SourceRef)
	if refs.TaskRef == "" && refs.AgentRef == "" && sourceRef != "" && sourceRef != decisionContext.RunRef {
		refs.AuditRef = sourceRef
	}
	return orquestaobservability.WorkspaceTimelineItemV0{
		EventRef:   fmt.Sprintf("event-ref-workspace-timeline-director-stats-%03d-%s", index, compactServerTimelineSuffixV0(firstServerNonEmptyV0(sourceRef, activity.Kind))),
		OccurredAt: firstServerNonEmptyV0(activity.OccurredAt, observedAt),
		Source:     orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0,
		Category:   serverWorkspaceTimelineDirectorActivityCategoryV0(activity.Kind),
		Status:     "observed",
		SummaryKey: firstServerNonEmptyV0(activity.SummaryKey, "workspace.timeline.director_stats.activity"),
		Summary:    "Director: actividad " + firstServerNonEmptyV0(activity.Kind, "observed") + ".",
		Refs:       refs,
		Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
			Class:      orquestaobservability.WorkspaceTimelineTranscriptMetadataV0,
			Confidence: 1,
			Compact:    true,
		},
	}
}

func serverWorkspaceTimelineDirectorActivityCategoryV0(kind string) string {
	switch strings.TrimSpace(kind) {
	case orquestaobservability.DirectorDecisionActivityTaskProgressV0:
		return orquestaobservability.WorkspaceTimelineCategoryTaskV0
	case orquestaobservability.DirectorDecisionActivityClosureBlockedV0,
		orquestaobservability.DirectorDecisionActivityReworkRequestV0,
		orquestaobservability.DirectorDecisionActivityReplanDecisionV0:
		return orquestaobservability.WorkspaceTimelineCategoryReviewV0
	default:
		return orquestaobservability.WorkspaceTimelineCategoryRuntimeV0
	}
}

func compactServerTimelineRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
