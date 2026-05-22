package orquestaappcodexstack

import (
	"context"
	"fmt"
)

type CodexStackAutoprogrammingExecutorV0 struct {
	Stack *StackV0
}

func NewCodexStackAutoprogrammingExecutorV0(
	stack *StackV0,
) CodexStackAutoprogrammingExecutorV0 {
	return CodexStackAutoprogrammingExecutorV0{Stack: stack}
}

func (executor CodexStackAutoprogrammingExecutorV0) Execute(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
) (AutoprogrammingBridgeResultV0, error) {
	if executor.Stack == nil {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("stack requerido")
	}
	return PrepareAutoprogrammingRunFromStackV0(ctx, *executor.Stack, request)
}

func PrepareAutoprogrammingRunFromStackV0(
	ctx context.Context,
	stack StackV0,
	request AutoprogrammingBridgeRequestV0,
) (AutoprogrammingBridgeResultV0, error) {
	return PrepareAutoprogrammingRunV0(ctx, request, stack.Ports)
}
