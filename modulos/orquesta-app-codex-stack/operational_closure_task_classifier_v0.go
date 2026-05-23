package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func RunHasOpenOperationalDirectorTasksV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if taskStore == nil {
		return false, nil
	}
	openRefs := codexStackOperationalClosureOpenTaskRefsV0(run)
	if len(openRefs) == 0 {
		return false, nil
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, openRefs)
	if err != nil {
		return false, err
	}
	for _, task := range tasks {
		if codexStackWorkflowTaskLooksOperationalDirectorV0(task) {
			return true, nil
		}
	}
	return false, nil
}

func codexStackOperationalClosureOpenTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	closed := codexStackOperationalClosureSetV0(run.ClosedTasks)
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range codexStackOperationalClosureCompactRefsV0(run.Tasks) {
		if !closed[taskRef] {
			refs = append(refs, taskRef)
		}
	}
	return refs
}

func codexStackWorkflowTaskLooksOperationalDirectorV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, value := range codexStackWorkflowTaskOperationalSignalsV0(task) {
		if codexStackOperationalDirectorSignalV0(value) {
			return true
		}
	}
	return false
}

func codexStackWorkflowTaskOperationalSignalsV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) []string {
	signals := append([]string(nil), task.AcceptanceCriteria...)
	signals = append(signals, task.CohortRef, task.WaveRef, task.ParentTaskRef)
	for _, ref := range task.FunctionContractRefs {
		signals = append(signals, ref.ContractRef, ref.FunctionName)
	}
	return signals
}

func codexStackOperationalDirectorSignalV0(value string) bool {
	normalized := codexStackClosureSignalKeyV0(value)
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"operationaldirector",
		"directoroperativo",
		"operativedirector",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func codexStackClosureSignalKeyV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, value)
}
