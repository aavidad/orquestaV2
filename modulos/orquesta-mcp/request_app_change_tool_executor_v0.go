package orquestamcp

import (
	"context"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

type MCPRequestAppChangeToolExecutorV0 struct {
	Ports orquestaappchange.AppChangePortsV0
}

func NewMCPRequestAppChangeToolExecutorV0(
	ports orquestaappchange.AppChangePortsV0,
) MCPRequestAppChangeToolExecutorV0 {
	return MCPRequestAppChangeToolExecutorV0{Ports: ports}
}

func (executor MCPRequestAppChangeToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRequestAppChangeToolInputV0,
) (MCPRequestAppChangeToolResultV0, error) {
	result, err := orquestaappchange.RequestAppChangeV0(
		ctx,
		ToAppChangeRequestFromMCPV0(input),
		executor.Ports,
	)
	if err != nil {
		return MCPRequestAppChangeToolResultV0{}, err
	}
	return NewMCPRequestAppChangeResultV0(result), nil
}
