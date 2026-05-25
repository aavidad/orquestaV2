package orquestaobservability

import (
	"context"
	"strings"
)

type WorkspaceTimelineStaticReaderV0 struct {
	timelines map[workspaceTimelineStaticKeyV0]WorkspaceTimelineProjectionV0
}

type workspaceTimelineStaticKeyV0 struct {
	correlationID string
	agentRef      string
	projectRef    string
	taskRef       string
	runRef        string
}

func NewWorkspaceTimelineStaticReaderV0(
	timelines []WorkspaceTimelineProjectionV0,
) (WorkspaceTimelineStaticReaderV0, error) {
	reader := WorkspaceTimelineStaticReaderV0{timelines: map[workspaceTimelineStaticKeyV0]WorkspaceTimelineProjectionV0{}}
	for _, timeline := range timelines {
		if err := ValidateWorkspaceTimelineProjectionV0(timeline); err != nil {
			return WorkspaceTimelineStaticReaderV0{}, err
		}
		reader.timelines[workspaceTimelineStaticKeyFromProjectionV0(timeline)] = timeline
	}
	return reader, nil
}

func (reader WorkspaceTimelineStaticReaderV0) QueryWorkspaceTimelineV0(
	_ context.Context,
	query WorkspaceTimelineQueryV0,
) (WorkspaceTimelineProjectionV0, error) {
	if err := ValidateWorkspaceTimelineQueryV0(query); err != nil {
		return WorkspaceTimelineProjectionV0{}, err
	}
	timeline, ok := reader.timelines[workspaceTimelineStaticKeyFromQueryV0(query)]
	if !ok {
		return WorkspaceTimelineProjectionV0{}, operationalStatusValidationErrorV0(ErrWorkspaceTimelineNoDisponibleV0, "timeline")
	}
	return filterWorkspaceTimelineProjectionForQueryV0(timeline, query), nil
}

func workspaceTimelineStaticKeyFromProjectionV0(timeline WorkspaceTimelineProjectionV0) workspaceTimelineStaticKeyV0 {
	return workspaceTimelineStaticKeyV0{
		correlationID: strings.TrimSpace(timeline.CorrelationID),
		agentRef:      firstNonEmptyOperationalStatusV0(timeline.AgentRef, timeline.Filters.AgentRef),
		projectRef:    firstNonEmptyOperationalStatusV0(timeline.ProjectRef, timeline.Filters.ProjectRef),
		taskRef:       firstNonEmptyOperationalStatusV0(timeline.TaskRef, timeline.Filters.TaskRef),
		runRef:        strings.TrimSpace(timeline.RunRef),
	}
}

func workspaceTimelineStaticKeyFromQueryV0(query WorkspaceTimelineQueryV0) workspaceTimelineStaticKeyV0 {
	return workspaceTimelineStaticKeyV0{
		correlationID: strings.TrimSpace(query.CorrelationID),
		agentRef:      strings.TrimSpace(query.AgentRef),
		projectRef:    strings.TrimSpace(query.ProjectRef),
		taskRef:       strings.TrimSpace(query.TaskRef),
		runRef:        strings.TrimSpace(query.RunRef),
	}
}
