package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type CodexStackAutoprogrammingObserveGoalExecutorV0 struct {
	stack *StackV0
}

func NewCodexStackAutoprogrammingObserveGoalExecutorV0(
	stack *StackV0,
) CodexStackAutoprogrammingObserveGoalExecutorV0 {
	return CodexStackAutoprogrammingObserveGoalExecutorV0{stack: stack}
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
) (orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0, error) {
	if executor.stack == nil {
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, fmt.Errorf("stack requerido")
	}
	result, err := executor.stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestamcp.ToAutoprogrammingObserveGoalRequestV0(input),
	)
	if err != nil {
		if publicResult, ok := orquestamcp.NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(input, err); ok {
			return publicResult, nil
		}
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, err
	}
	return orquestamcp.NewMCPAutoprogrammingObserveGoalResultV0(input, result), nil
}
