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
	request.LegacyDirectorLoopOptInAvailable = request.LegacyDirectorLoopOptInAvailable ||
		stack.AllowLegacyAutoprogrammingRun ||
		request.AllowLegacyDirectorLoop
	if stack.AllowLegacyAutoprogrammingRun &&
		autoprogrammingBridgeLegacyDirectorLoopRequestedV0(request) {
		request.AllowLegacyDirectorLoop = true
	}
	if err := validateAutoprogrammingBridgePortsV0(stack.Ports); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	request = autoprogrammingBridgeRequestWithGoalFirstBackendMarkersV0(request, stack.Ports)
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request.Request)
	if work.Accepted {
		snapshotStore := stack.AutoprogrammingPromotion.GoalFirstSnapshotStore
		if stack.AutoprogrammingPromotion.Enabled {
			if snapshotStore == nil {
				work.Accepted = false
				work.Issues = append(work.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
					Code:    "worktree_baseline_store_missing",
					Field:   "autoprogramming_promotion.goal_first_snapshot_store",
					Message: "snapshot store requerido antes de preparar un goal promocionable",
				})
			}
		}
		if work.Accepted {
			prepared, issues := autoprogrammingPrepareWorktreeIsolationV0(
				ctx,
				stack.Codex.ProjectWorkDir,
				work.Work,
				snapshotStore,
			)
			if len(issues) > 0 {
				work.Accepted = false
				work.Issues = append(work.Issues, issues...)
			}
			work.Work = prepared
		}
	}
	return prepareAutoprogrammingRunWithWorkV0(ctx, request, stack.Ports, work)
}
