package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type codexStackDirectorDecisionExecutorV0 struct {
	Inner       orquestamcp.MCPTransportDirectorAgentDecisionExecutorV0
	Coordinator *goalFirstObservationCoordinatorV0
}

func (executor codexStackDirectorDecisionExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPDirectorAgentDecisionToolInputV0,
) (orquestamcp.MCPDirectorAgentDecisionToolResultV0, error) {
	if executor.Inner == nil || executor.Coordinator == nil {
		return orquestamcp.MCPDirectorAgentDecisionToolResultV0{}, fmt.Errorf("director_decision_executor_unavailable")
	}
	release, err := executor.Coordinator.acquireV0(ctx, input.Decision.RunID)
	if err != nil {
		return orquestamcp.MCPDirectorAgentDecisionToolResultV0{}, err
	}
	defer release()
	return executor.Inner.Execute(ctx, input)
}
