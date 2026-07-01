package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type CodexStackAutoprogrammingObserveGoalExecutorV0 struct {
	stack *StackV0
}

var _ orquestamcp.MCPTransportObserveAppDirectorGoalTimeoutSnapshotExecutorV0 = CodexStackAutoprogrammingObserveGoalExecutorV0{}

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
			return executor.withPartialSnapshotAfterObserveErrorV0(ctx, input, publicResult), nil
		}
		return orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0{}, err
	}
	return orquestamcp.NewMCPAutoprogrammingObserveGoalResultV0(input, result), nil
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	ctx context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	return NewCodexStackObserveAppDirectorGoalExecutorV0(executor.stack).
		ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, input)
}

func (executor CodexStackAutoprogrammingObserveGoalExecutorV0) withPartialSnapshotAfterObserveErrorV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0,
	publicResult orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0,
) orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0 {
	snapshot, err := executor.ObserveAppDirectorGoalTimeoutSnapshotV0(ctx, orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		RunRef:        input.RunRef,
		OccurredAt:    input.OccurredAt,
		RequestedBy:   input.RequestedBy,
	})
	if err != nil {
		return publicResult
	}
	return orquestamcp.NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(publicResult, snapshot)
}
