package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	codexStackAutoprogrammingAppSpecPrefixV0 = "app-spec-ref-autoprogramming-"
	codexStackAutoprogrammingTaskPrefixV0    = "task-autoprogramming-"
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

func RunHasOpenAutoprogrammingTasksV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	openRefs := codexStackOperationalClosureOpenTaskRefsV0(run)
	if len(openRefs) == 0 {
		return false, nil
	}
	if codexStackRunLooksAutoprogrammingV0(run, openRefs) {
		return true, nil
	}
	if taskStore == nil {
		return false, nil
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, openRefs)
	if err != nil {
		return false, err
	}
	for _, task := range tasks {
		if codexStackWorkflowTaskLooksAutoprogrammingV0(task) {
			return true, nil
		}
	}
	return false, nil
}

func RunHasOpenProgrammingOrAutonomyTasksV0(
	ctx context.Context,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	openRefs := codexStackOperationalClosureOpenTaskRefsV0(run)
	if len(openRefs) == 0 {
		return false, nil
	}
	if codexStackRunLooksAutoprogrammingV0(run, openRefs) {
		return true, nil
	}
	if taskStore == nil {
		return false, nil
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, openRefs)
	if err != nil {
		return false, err
	}
	for _, task := range tasks {
		if codexStackWorkflowTaskLooksProgrammingOrAutonomyV0(task) {
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
	if orquestacoreworkflow.NormalizeWorkProfileKindV0(task.WorkProfileKind) == orquestacoreworkflow.WorkProfileDomainWorkV0 {
		return codexStackWorkflowTaskHasOperationalDirectorFunctionContractV0(task) ||
			codexStackWorkflowTaskHasOperationalDirectorContextRefV0(task)
	}
	for _, value := range codexStackWorkflowTaskOperationalSignalsV0(task) {
		if codexStackOperationalDirectorSignalV0(value) {
			return true
		}
	}
	return false
}

func codexStackWorkflowTaskHasOperationalDirectorFunctionContractV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, ref := range task.FunctionContractRefs {
		if codexStackOperationalDirectorSignalV0(ref.ContractRef) ||
			codexStackOperationalDirectorSignalV0(ref.FunctionName) {
			return true
		}
	}
	return false
}

func codexStackWorkflowTaskHasOperationalDirectorContextRefV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, ref := range task.ContextRefs {
		if strings.TrimSpace(ref) == autoprogrammingBridgeOperationalTaskSourceRefV0 {
			return true
		}
	}
	return false
}

func codexStackRunLooksAutoprogrammingV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	openTaskRefs []string,
) bool {
	if strings.HasPrefix(strings.TrimSpace(run.AppSpecRef), codexStackAutoprogrammingAppSpecPrefixV0) {
		return true
	}
	for _, ref := range openTaskRefs {
		if strings.HasPrefix(strings.TrimSpace(ref), codexStackAutoprogrammingTaskPrefixV0) {
			return true
		}
	}
	for _, ref := range run.FunctionContracts {
		if codexStackAutoprogrammingSignalV0(ref) {
			return true
		}
	}
	return false
}

func codexStackWorkflowTaskLooksAutoprogrammingV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	if strings.HasPrefix(strings.TrimSpace(task.TaskID), codexStackAutoprogrammingTaskPrefixV0) {
		return true
	}
	for _, ref := range task.FunctionContractRefs {
		if codexStackAutoprogrammingSignalV0(ref.ContractRef) ||
			codexStackAutoprogrammingSignalV0(ref.FunctionName) {
			return true
		}
	}
	return false
}

func codexStackWorkflowTaskLooksProgrammingOrAutonomyV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	if codexStackWorkflowTaskLooksAutoprogrammingV0(task) {
		return true
	}
	if orquestacoreworkflow.NormalizeWorkProfileKindV0(task.WorkProfileKind) ==
		orquestacoreworkflow.WorkProfileImplementationV0 {
		return true
	}
	if strings.TrimSpace(string(task.WorkProfileKind)) != "" {
		if _, ok := orquestacoreworkflow.LookupWorkProfileDefinitionV0(task.WorkProfileKind); ok {
			return false
		}
	}
	if task.PhaseID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 &&
		(len(compactStringsV0(task.WriteSet)) > 0 ||
			len(compactStringsV0(task.RequiredTests)) > 0 ||
			len(task.FunctionContractRefs) > 0) {
		return true
	}
	return false
}

func codexStackWorkflowTaskOperationalSignalsV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) []string {
	signals := append([]string(nil), task.AcceptanceCriteria...)
	signals = append(signals, task.ContextRefs...)
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

func codexStackAutoprogrammingSignalV0(value string) bool {
	normalized := codexStackClosureSignalKeyV0(value)
	return strings.Contains(normalized, "autoprogramming")
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
