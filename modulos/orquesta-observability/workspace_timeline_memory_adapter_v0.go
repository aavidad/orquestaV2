package orquestaobservability

import "context"

type WorkspaceTimelineSourcePortV0 interface {
	QueryWorkspaceTimelineV0(context.Context, WorkspaceTimelineQueryV0) (WorkspaceTimelineV0, error)
}

type WorkspaceTimelineMemoryAdapterV0 = WorkspaceTimelineStaticReaderV0

func NewWorkspaceTimelineMemoryAdapterV0(
	timelines []WorkspaceTimelineV0,
) (WorkspaceTimelineMemoryAdapterV0, error) {
	return NewWorkspaceTimelineStaticReaderV0(timelines)
}
