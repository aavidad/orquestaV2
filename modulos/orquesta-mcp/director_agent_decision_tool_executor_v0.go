package orquestamcp

import (
	"context"

	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

type MCPDirectorAgentDecisionToolExecutorV0 struct {
	Ports orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0
}

func NewMCPDirectorAgentDecisionToolExecutorV0(
	ports orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0,
) MCPDirectorAgentDecisionToolExecutorV0 {
	return MCPDirectorAgentDecisionToolExecutorV0{Ports: ports}
}

func (executor MCPDirectorAgentDecisionToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDirectorAgentDecisionToolInputV0,
) (MCPDirectorAgentDecisionToolResultV0, error) {
	result, err := orquestadirectoragentworkflow.ApplyDirectorAgentDecisionV0(
		ctx,
		ToApplyDirectorAgentDecisionRequestV0(input),
		executor.Ports,
	)
	if err != nil {
		return NewMCPDirectorAgentDecisionErrorV0(input, err), nil
	}
	return NewMCPDirectorAgentDecisionResultV0(input, result), nil
}
