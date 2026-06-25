package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type CodexStackObserveAppDirectorGoalExecutorV0 struct {
	stack *StackV0
}

func NewCodexStackObserveAppDirectorGoalExecutorV0(
	stack *StackV0,
) CodexStackObserveAppDirectorGoalExecutorV0 {
	return CodexStackObserveAppDirectorGoalExecutorV0{stack: stack}
}

func (executor CodexStackObserveAppDirectorGoalExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	if executor.stack == nil {
		return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{}, fmt.Errorf("stack requerido")
	}
	result, err := executor.stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestamcp.ToObserveAppDirectorGoalRequestV0(input),
	)
	if err != nil {
		return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	return orquestamcp.NewMCPObserveAppDirectorGoalResultV0(input, result), nil
}
