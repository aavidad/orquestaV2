package orquestaobservability

import (
	"context"
	"strings"
	"time"
)

type WorkspaceTimelineUnavailableReaderV0 struct {
	now func() time.Time
}

func NewWorkspaceTimelineUnavailableReaderV0(now func() time.Time) WorkspaceTimelineUnavailableReaderV0 {
	return WorkspaceTimelineUnavailableReaderV0{now: now}
}

func (reader WorkspaceTimelineUnavailableReaderV0) QueryWorkspaceTimelineV0(
	_ context.Context,
	query WorkspaceTimelineQueryV0,
) (WorkspaceTimelineProjectionV0, error) {
	if err := ValidateWorkspaceTimelineQueryV0(query); err != nil {
		return WorkspaceTimelineProjectionV0{}, err
	}
	projection := WorkspaceTimelineProjectionV0{
		SchemaVersion: WorkspaceTimelineProjectionSchemaVersionV0,
		TimelineRef:   "timeline-ref-workspace-" + compactWorkspaceTimelineSuffixV0(query.CorrelationID),
		TimelineID:    "timeline-ref-workspace-" + compactWorkspaceTimelineSuffixV0(query.CorrelationID),
		GeneratedAt:   workspaceTimelineNowV0(reader.now).Format(time.RFC3339),
		CorrelationID: strings.TrimSpace(query.CorrelationID),
		Scope:         WorkspaceTimelineScopeWorkspaceV0,
		WorkspaceRef:  strings.TrimSpace(query.WorkspaceRef),
		ProjectRef:    strings.TrimSpace(query.ProjectRef),
		AgentRef:      strings.TrimSpace(query.AgentRef),
		TaskRef:       strings.TrimSpace(query.TaskRef),
		RunRef:        strings.TrimSpace(query.RunRef),
		Filters: WorkspaceTimelineFiltersV0{
			AgentRef:   strings.TrimSpace(query.AgentRef),
			ProjectRef: strings.TrimSpace(query.ProjectRef),
			TaskRef:    strings.TrimSpace(query.TaskRef),
		},
		TimeWindow: query.TimeWindow,
		Page: WorkspaceTimelinePageV0{
			Limit:     workspaceTimelineQueryLimitV0(query),
			CursorRef: strings.TrimSpace(query.Page.CursorRef),
			HasMore:   false,
		},
		Freshness: DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark-ref-workspace-timeline-" + compactWorkspaceTimelineSuffixV0(query.CorrelationID),
			MaxAgeSeconds: 0,
			Partial:       true,
		},
		Sources: workspaceTimelineUnavailableSourcesV0(workspaceTimelineQuerySourcesV0(query)),
		Counters: map[string]float64{
			"events": 0,
		},
		Warnings: []DiagnosticoWarningV0{{
			Code:    ErrWorkspaceTimelineNoDisponibleV0,
			Summary: "Fuentes declaradas sin adaptador residente disponible.",
		}},
		Privacy: DiagnosticoPrivacyV0{},
	}
	if err := ValidateWorkspaceTimelineProjectionV0(projection); err != nil {
		return WorkspaceTimelineProjectionV0{}, err
	}
	return projection, nil
}

func workspaceTimelineNowV0(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}

func workspaceTimelineUnavailableSourcesV0(sources []string) []WorkspaceTimelineSourceStatusV0 {
	statuses := make([]WorkspaceTimelineSourceStatusV0, 0, len(sources))
	for _, source := range sources {
		statuses = append(statuses, WorkspaceTimelineSourceStatusV0{
			Source:       strings.TrimSpace(source),
			Status:       WorkspaceTimelineSourceNotAvailableV0,
			ReasonCode:   "source_port_not_bound",
			EvidenceRefs: []string{"evidence-ref-workspace-timeline-source-not-bound"},
		})
	}
	return statuses
}

func compactWorkspaceTimelineSuffixV0(value string) string {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), "-_.:")
	if value == "" {
		return "global"
	}
	if len(value) > 48 {
		return value[:48]
	}
	return value
}
