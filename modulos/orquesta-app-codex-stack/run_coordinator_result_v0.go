package orquestaappcodexstack

import (
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func normalizeGlobalDrainLimitsV0(
	limits orquestaruncoordinator.RunDrainLimitsV0,
) orquestaruncoordinator.RunDrainLimitsV0 {
	if limits.MaxBursts <= 0 {
		limits.MaxBursts = 4
	}
	if limits.MaxStepsPerBurst <= 0 {
		limits.MaxStepsPerBurst = 6
	}
	if limits.MaxDispatchesPerWait <= 0 {
		limits.MaxDispatchesPerWait = 10
	}
	if limits.MaxCommands <= 0 {
		limits.MaxCommands = 20
	}
	if limits.MaxOutboxPerCycle <= 0 {
		limits.MaxOutboxPerCycle = 10
	}
	if limits.MaxDecisionCycles <= 0 {
		limits.MaxDecisionCycles = 1
	}
	if limits.MaxExternalWaits <= 0 {
		limits.MaxExternalWaits = defaultDrainRunMaxExternalWaitsV0
	}
	return limits
}

func stackDrainOutcomeV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) string {
	if result.Status != "" {
		return string(result.Status)
	}
	return string(result.Final.Status)
}

func stackDrainEvidenceRefsV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) []string {
	refs := append([]string{"evidence-ref-run-coordinator-drain"}, result.Final.FirstPendingRefs...)
	for _, wait := range result.ExternalWaits {
		refs = append(refs, wait.EvidenceRefs...)
	}
	return compactCodexStackStringsV0(refs)
}

func stackDrainRunHasAllTasksDeliveredOrClosedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	tasks := compactCodexStackStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return false
	}
	for _, taskRef := range tasks {
		if !codexStackStringInSetV0(run.DeliveredTasks, taskRef) &&
			!codexStackStringInSetV0(run.ClosedTasks, taskRef) {
			return false
		}
	}
	return true
}

func formatStackCoordinatorTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
