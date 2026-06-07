package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (source serverWorkspaceTimelineSourceV0) eventItemsV0(
	ctx context.Context,
	query orquestaobservability.WorkspaceTimelineQueryV0,
) ([]orquestaobservability.WorkspaceTimelineItemV0, bool) {
	runRef := strings.TrimSpace(query.RunRef)
	if source.eventReader == nil || runRef == "" {
		return nil, false
	}
	events, ok := source.readTimelineEventsV0(ctx, runRef, query)
	if !ok {
		return nil, false
	}
	items := make([]orquestaobservability.WorkspaceTimelineItemV0, 0, len(events))
	for index, event := range events {
		items = append(items, serverWorkspaceTimelineEventItemV0(query, event, index+1))
	}
	return items, true
}

func (source serverWorkspaceTimelineSourceV0) readTimelineEventsV0(
	ctx context.Context,
	runRef string,
	query orquestaobservability.WorkspaceTimelineQueryV0,
) ([]orquestacoreworkflow.OrchestrationEventV0, bool) {
	limit := source.limitV0(query)
	if limit <= 0 {
		limit = 50
	}
	if paged, ok := source.eventReader.(orquestacionnucleoapp.RunEventPagedReaderPortV0); ok {
		page, err := paged.LoadRunEventsPageV0(ctx, orquestacionnucleoapp.RunEventPageRequestV0{
			RunRef: runRef,
			Limit:  limit,
			Cursor: strings.TrimSpace(query.Page.CursorRef),
		})
		if err != nil {
			return nil, false
		}
		return page.Events, true
	}
	events, err := source.eventReader.LoadRunEventsV0(ctx, runRef)
	if err != nil {
		return nil, false
	}
	if len(events) > limit {
		events = events[:limit]
	}
	return events, true
}

func serverWorkspaceTimelineEventItemV0(
	query orquestaobservability.WorkspaceTimelineQueryV0,
	event orquestacoreworkflow.OrchestrationEventV0,
	index int,
) orquestaobservability.WorkspaceTimelineItemV0 {
	refs, evidenceRefs := serverWorkspaceTimelineEventRefsV0(query, event)
	return orquestaobservability.WorkspaceTimelineItemV0{
		EventRef:   firstServerNonEmptyV0(strings.TrimSpace(event.EventID), fmt.Sprintf("event-ref-workspace-timeline-events-%03d", index)),
		OccurredAt: strings.TrimSpace(event.OccurredAt),
		Source:     orquestaobservability.WorkspaceTimelineSourceEventsV0,
		Category:   serverWorkspaceTimelineEventCategoryV0(event.EventType),
		Status:     "observed",
		SummaryKey: serverWorkspaceTimelineEventSummaryKeyV0(event.EventType),
		Summary:    "Evento durable registrado.",
		Refs:       refs,
		RunRef:     refs.RunRef,
		AgentRef:   refs.AgentRef,
		Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
			Class:      orquestaobservability.WorkspaceTimelineTranscriptMetadataV0,
			Confidence: 1,
			Compact:    true,
		},
		EvidenceRefs: evidenceRefs,
	}
}

func serverWorkspaceTimelineEventRefsV0(
	query orquestaobservability.WorkspaceTimelineQueryV0,
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestaobservability.WorkspaceTimelineRefsV0, []string) {
	payload := map[string]any{}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}
	refs := orquestaobservability.WorkspaceTimelineRefsV0{
		RunRef:   firstServerNonEmptyV0(event.RunID, query.RunRef),
		TaskRef:  serverWorkspaceTimelinePayloadStringV0(payload, "task_ref"),
		AgentRef: serverWorkspaceTimelinePayloadStringV0(payload, "agent_request_id"),
		ProjectRef: firstServerNonEmptyV0(
			serverWorkspaceTimelinePayloadStringV0(payload, "project_ref"),
			query.ProjectRef,
		),
		WorktreeRef: serverWorkspaceTimelinePayloadStringV0(payload, "worktree_ref"),
		AuditRef: firstServerNonEmptyV0(
			serverWorkspaceTimelinePayloadStringV0(payload, "delivery_ref"),
			serverWorkspaceTimelinePayloadStringV0(payload, "review_request_id"),
			serverWorkspaceTimelinePayloadStringV0(payload, "review_result_ref"),
			serverWorkspaceTimelinePayloadStringV0(payload, "blocker_id"),
			serverWorkspaceTimelinePayloadStringV0(payload, "closure_ref"),
		),
	}
	evidenceRefs := serverWorkspaceTimelinePayloadStringsV0(payload, "evidence_refs")
	return refs, evidenceRefs
}

func serverWorkspaceTimelinePayloadStringV0(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return serverWorkspaceTimelineSafeOpaqueRefV0(value)
}

func serverWorkspaceTimelinePayloadStringsV0(payload map[string]any, key string) []string {
	values, _ := payload[key].([]any)
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, raw := range values {
		value, _ := raw.(string)
		value = serverWorkspaceTimelineSafeOpaqueRefV0(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func serverWorkspaceTimelineSafeOpaqueRefV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 200 {
		return ""
	}
	for _, r := range value {
		if unicode.IsSpace(r) || r == '/' || r == '\\' || r == '=' || r == '"' || r == '\'' {
			return ""
		}
	}
	return value
}

func serverWorkspaceTimelineEventCategoryV0(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventTaskClosedV0:
		return orquestaobservability.WorkspaceTimelineCategoryTaskV0
	case orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewAcceptedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReworkRequestedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0,
		orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
		return orquestaobservability.WorkspaceTimelineCategoryReviewV0
	case orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStartedV0,
		orquestacoreworkflow.OrchestrationEventAgentFailedV0,
		orquestacoreworkflow.OrchestrationEventAgentLostV0,
		orquestacoreworkflow.OrchestrationEventAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
		orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0:
		return orquestaobservability.WorkspaceTimelineCategoryRuntimeV0
	default:
		return orquestaobservability.WorkspaceTimelineCategoryEventV0
	}
}

func serverWorkspaceTimelineEventSummaryKeyV0(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case orquestacoreworkflow.OrchestrationEventRunStartedV0,
		orquestacoreworkflow.OrchestrationEventPhaseOpenedV0,
		orquestacoreworkflow.OrchestrationEventPhaseClosedV0,
		orquestacoreworkflow.OrchestrationEventRunBlockedV0,
		orquestacoreworkflow.OrchestrationEventRunBlockerResolvedV0,
		orquestacoreworkflow.OrchestrationEventDirectorQuestionRaisedV0,
		orquestacoreworkflow.OrchestrationEventDirectorQuestionAnsweredV0,
		orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0,
		orquestacoreworkflow.OrchestrationEventVoteRequestedV0,
		orquestacoreworkflow.OrchestrationEventArchitectureDecisionAcceptedV0,
		orquestacoreworkflow.OrchestrationEventFunctionContractPublishedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStartedV0,
		orquestacoreworkflow.OrchestrationEventAgentFailedV0,
		orquestacoreworkflow.OrchestrationEventAgentLostV0,
		orquestacoreworkflow.OrchestrationEventAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
		orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0,
		orquestacoreworkflow.OrchestrationEventConcurrencyGateRecordedV0,
		orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0,
		orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0,
		orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0,
		orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewAcceptedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReworkRequestedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventTaskClosedV0,
		orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0,
		orquestacoreworkflow.OrchestrationEventRunClosedV0:
	default:
		return "workspace.timeline.events.generic"
	}
	normalized := strings.ToLower(strings.TrimSpace(eventType))
	var builder strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('_')
	}
	suffix := strings.Trim(builder.String(), "_")
	if suffix == "" {
		return "workspace.timeline.events.generic"
	}
	return "workspace.timeline.events." + suffix
}
