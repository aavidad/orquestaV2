package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
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
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeAutoprogrammingBridgeRequestV0(request)
	if stack.AllowLegacyAutoprogrammingRun {
		request.AllowLegacyDirectorLoop = true
	}
	if err := validateAutoprogrammingBridgePortsV0(stack.Ports); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	request = autoprogrammingBridgeRequestWithGoalFirstBackendMarkersV0(request, stack.Ports)
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request.Request)
	if work.Accepted {
		prepared, issues := autoprogrammingPrepareWorktreeIsolationV0(ctx, stack.Codex.ProjectWorkDir, work.Work)
		if len(issues) > 0 {
			work.Accepted = false
			work.Issues = append(work.Issues, issues...)
		}
		work.Work = prepared
	}
	return prepareAutoprogrammingRunWithWorkV0(ctx, request, stack.Ports, work)
}
