package orquestamcp

import (
	"context"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
)

type MCPArrancarDirectorAppToolExecutorV0 struct {
	Ports orquestaappdirectorservice.StartAppDirectorPortsV0
}

func NewMCPArrancarDirectorAppToolExecutorV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) MCPArrancarDirectorAppToolExecutorV0 {
	return MCPArrancarDirectorAppToolExecutorV0{Ports: ports}
}

func (executor MCPArrancarDirectorAppToolExecutorV0) Execute(
	ctx context.Context,
	input MCPArrancarDirectorAppToolInputV0,
) (MCPArrancarDirectorAppToolResultV0, error) {
	result, err := orquestaappdirectorservice.StartAppDirectorV0(
		ctx,
		ToStartAppDirectorRequestV0(input),
		executor.Ports,
	)
	if err != nil {
		return MCPArrancarDirectorAppToolResultV0{}, err
	}
	return NewMCPArrancarDirectorAppResultV0(result), nil
}
