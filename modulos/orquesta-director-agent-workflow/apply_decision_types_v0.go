package orquestadirectoragentworkflow

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

type DirectorAgentRunStorePortV0 interface {
	LoadRunV0(ctx context.Context, runRef string) (orquestacoreworkflow.OrchestrationRunV0, error)
	SaveRunV0(ctx context.Context, run orquestacoreworkflow.OrchestrationRunV0) error
}

type DirectorAgentEventSinkPortV0 interface {
	AppendRunEventsV0(
		ctx context.Context,
		runRef string,
		events []orquestacoreworkflow.OrchestrationEventV0,
	) error
}

type DirectorAgentWorkflowTaskStorePortV0 interface {
	SaveWorkflowTaskV0(ctx context.Context, task orquestacoreworkflow.WorkflowTaskV0) error
}

type ApplyDirectorAgentDecisionRequestV0 struct {
	Decision      orquestadirectoragent.DirectorAgentDecisionV0
	OccurredAt    string
	CorrelationID string
	RequestedBy   string
}

type ApplyDirectorAgentDecisionPortsV0 struct {
	RunStore  DirectorAgentRunStorePortV0
	EventSink DirectorAgentEventSinkPortV0
	TaskStore DirectorAgentWorkflowTaskStorePortV0
}

type ApplyDirectorAgentDecisionResultV0 struct {
	Command     orquestacoreworkflow.OrchestrationCommandV0
	Run         orquestacoreworkflow.OrchestrationRunV0
	EventsCount int
	Idempotent  bool
	Issues      []DirectorAgentWorkflowIssueV0
}
