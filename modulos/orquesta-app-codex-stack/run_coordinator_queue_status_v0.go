package orquestaappcodexstack

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack StackV0) stackDrainQueueStatusForCoordinatorV0(
	ctx context.Context,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) (string, error) {
	status := stackDrainQueueStatusV0(result)
	if status != orquestarunqueue.RunStatusDeliveredV0 {
		return status, nil
	}
	run := result.Final.Run
	hold, err := RunHasOpenOperationalDirectorTasksV0(ctx, stack.Stores.TaskStore, run)
	if err != nil || hold {
		return "", err
	}
	hold, err = RunHasOpenAutoprogrammingTasksV0(ctx, stack.Stores.TaskStore, run)
	if err != nil || hold {
		return "", err
	}
	return status, nil
}

func stackDrainQueueStatusV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) string {
	run := result.Final.Run
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0
	}
	if stackDrainRunHasAllTasksDeliveredOrClosedV0(run) &&
		!drainRunHasPendingExternalAgentsV0(run, nil) {
		return orquestarunqueue.RunStatusDeliveredV0
	}
	return ""
}
