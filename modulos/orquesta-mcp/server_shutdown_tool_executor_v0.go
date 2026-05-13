package orquestamcp

import (
	"context"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

type MCPServerShutdownToolExecutorV0 struct {
	Deps orquestaservershutdown.ServerShutdownDepsV0
}

func NewMCPServerShutdownToolExecutorV0(
	deps orquestaservershutdown.ServerShutdownDepsV0,
) MCPServerShutdownToolExecutorV0 {
	return MCPServerShutdownToolExecutorV0{Deps: deps}
}

func (executor MCPServerShutdownToolExecutorV0) Execute(
	ctx context.Context,
	input MCPServerShutdownToolInputV0,
) (MCPServerShutdownToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := orquestaservershutdown.ShutdownServerV0(
		ctx,
		executor.Deps,
		serverShutdownCommandFromMCPV0(input),
	)
	if err != nil {
		return MCPServerShutdownToolResultV0{}, err
	}
	return newMCPServerShutdownResultV0(input, result), nil
}
