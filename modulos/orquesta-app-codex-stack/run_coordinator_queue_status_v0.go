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
	result = stack.stackDrainQueueStatusResultWithLatestRunV0(ctx, result)
	status := stackDrainQueueStatusV0(result)
	if status == orquestarunqueue.RunStatusClosedV0 {
		complete, _, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, result.Final.Run)
		if err != nil || !complete {
			return "", err
		}
		return status, nil
	}
	if status != orquestarunqueue.RunStatusDeliveredV0 {
		return status, nil
	}
	run := result.Final.Run
	hold, err := RunHasOpenOperationalDirectorTasksV0(ctx, stack.Stores.TaskStore, run)
	if err != nil || hold {
		return "", err
	}
	hold, err = RunHasOpenProgrammingOrAutonomyTasksV0(ctx, stack.Stores.TaskStore, run)
	if err != nil || hold {
		return "", err
	}
	return status, nil
}

func (stack StackV0) stackDrainQueueStatusResultWithLatestRunV0(
	ctx context.Context,
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) orquestacionnucleoapp.ManagedProgressiveLoopResultV0 {
	runRef := result.Final.Run.RunID
	if runRef == "" || stack.Stores.RunStore == nil {
		return result
	}
	latest, err := stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil || latest.RunID == "" {
		return result
	}
	result.Final.Run = latest
	return result
}

func stackDrainQueueStatusV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) string {
	run := result.Final.Run
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0
	}
	if stackDrainRunAwaitsLateDirectorDecisionsV0(run) {
		return ""
	}
	if stackDrainRunHasAllTasksDeliveredOrClosedV0(run) &&
		!stackDrainRunHasUndeliveredStartedAgentsV0(run) &&
		!drainRunHasPendingExternalAgentsV0(run, nil) {
		return orquestarunqueue.RunStatusDeliveredV0
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusActiveV0 &&
		len(stackDrainOpenTaskRefsV0(run)) > 0 {
		return ""
	}
	if result.Status == orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 &&
		result.Final.PendingOutboxCount == 0 &&
		result.Final.FirstPendingCount == 0 &&
		!stackDrainRunHasUndeliveredStartedAgentsV0(run) &&
		!drainRunHasPendingExternalAgentsV0(run, nil) {
		return orquestarunqueue.RunStatusStoppedV0
	}
	return ""
}

func stackDrainRunAwaitsLateDirectorDecisionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return run.Status == orquestacoreworkflow.OrchestrationRunStatusActiveV0 &&
		run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 &&
		len(compactCodexStackStringsV0(run.Tasks)) == 0 &&
		len(compactCodexStackStringsV0(run.StartedAgents)) > 0 &&
		(len(compactCodexStackStringsV0(run.PhaseArtifacts)) > 0 ||
			len(compactCodexStackStringsV0(run.DeliveredAgents)) > 0 ||
			len(compactCodexStackStringsV0(run.Deliveries)) > 0)
}

func stackDrainRunHasUndeliveredStartedAgentsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	started := compactCodexStackStringsV0(run.StartedAgents)
	if len(started) == 0 {
		return false
	}
	terminalRefs := append([]string{}, run.DeliveredAgents...)
	terminalRefs = append(terminalRefs, run.FailedAgents...)
	terminalRefs = append(terminalRefs, run.LostAgents...)
	terminalRefs = append(terminalRefs, run.ConfirmedStoppedAgents...)
	terminalRefs = compactCodexStackStringsV0(terminalRefs)
	if len(terminalRefs) == 0 {
		return len(compactCodexStackStringsV0(run.Deliveries)) < len(started)
	}
	for _, agentRef := range started {
		if !codexStackStringInSetV0(terminalRefs, agentRef) {
			return true
		}
	}
	return false
}
