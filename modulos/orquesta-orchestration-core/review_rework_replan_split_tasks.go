package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkMicrotaskCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) ([]orquestadirector.ReplanMicrotaskCandidateV0, error) {
	if action != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 || len(plan.SplitTasks) == 0 {
		return nil, nil
	}
	if provider.TaskWriter == nil {
		return nil, errorV0(ErrNucleoOrquestacionInvalidoV0, "workflow_task_writer", "workflow_task_writer requerido")
	}
	tasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(plan.SplitTasks))
	for _, raw := range plan.SplitTasks {
		task, err := reviewReworkSplitTaskV0(request, raw)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := reviewReworkValidateRecursiveSplitTasksV0(ctx, request, provider.TaskWriter, tasks); err != nil {
		return nil, err
	}
	candidates := make([]orquestadirector.ReplanMicrotaskCandidateV0, 0, len(tasks))
	for _, task := range tasks {
		if err := provider.TaskWriter.SaveWorkflowTaskV0(ctx, task); err != nil {
			return nil, err
		}
		candidates = append(candidates, orquestadirector.ReplanMicrotaskCandidateV0{
			CommandMeta: provider.reviewReworkCommandMetaV0(request, "create-microtask", task.TaskID),
			Payload: orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
				Task: task,
			},
		})
	}
	return candidates, nil
}

func reviewReworkSplitTaskV0(
	request SchedulerCandidateRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	task = reviewReworkSplitTaskWithRunFunctionContractsV0(request, task)
	normalized, err := orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task", err.Error())
	}
	if strings.TrimSpace(normalized.RunID) != strings.TrimSpace(request.Run.RunID) {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.run_id", "run_id no coincide")
	}
	if !reviewReworkSplitTaskPhaseAllowedV0(request.Run, normalized.PhaseID) {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.phase_id", "phase_id no soportado")
	}
	return normalized, nil
}

func reviewReworkSplitTaskWithRunFunctionContractsV0(
	request SchedulerCandidateRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	if len(task.FunctionContractRefs) > 0 {
		return task
	}
	contracts := compactStringsV0(request.Run.FunctionContracts)
	if len(contracts) == 0 {
		return task
	}
	task.FunctionContractRefs = make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(contracts))
	for _, ref := range contracts {
		task.FunctionContractRefs = append(task.FunctionContractRefs, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef: ref,
		})
	}
	return task
}

func reviewReworkSplitTaskPhaseAllowedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	phase = orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(phase)))
	if err := orquestacoreworkflow.ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return false
	}
	current := orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(run.CurrentPhase)))
	if phase == current {
		return true
	}
	if current != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		return false
	}
	return reviewReworkSplitTaskReworkPhaseV0(phase)
}

func reviewReworkSplitTaskReworkPhaseV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	switch phase {
	case orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		orquestacoreworkflow.OrchestrationPhaseDocumentacionV0,
		orquestacoreworkflow.OrchestrationPhaseIntegracionV0:
		return true
	default:
		return false
	}
}

func reviewReworkValidateRecursiveSplitTasksV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
	writer WorkflowTaskWriterPortV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) error {
	parentRefs := reviewReworkRecursiveSplitParentRefsV0(tasks)
	if len(parentRefs) == 0 {
		return nil
	}
	store, ok := writer.(WorkflowTaskStorePortV0)
	if !ok {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_store",
			"workflow_task_store requerido para split_task recursivo",
		)
	}
	parentTasks, err := store.LoadWorkflowTasksV0(ctx, request.Run.RunID, parentRefs)
	if err != nil {
		return err
	}
	parentsByRef := map[string]orquestacoreworkflow.WorkflowTaskV0{}
	fanoutByParentRef := map[string][]string{}
	for _, parent := range parentTasks {
		parentRef := strings.TrimSpace(parent.TaskID)
		if !reviewReworkRunContainsRefV0(request.Run.Tasks, parentRef) {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.parent_task_ref", "parent_task_ref no reflejada")
		}
		if parent.MaxChildAgents <= 0 {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.max_child_agents", "parent no permite hijos")
		}
		parentsByRef[parentRef] = parent
		fanoutByParentRef[parentRef] = compactStringsV0(parent.ChildTaskRefs)
	}
	treeStore, ok := writer.(WorkflowTaskByParentStorePortV0)
	if !ok {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_by_parent_store",
			"workflow_task_by_parent_store requerido para split_task recursivo",
		)
	}
	for _, parentRef := range parentRefs {
		if _, ok := parentsByRef[parentRef]; !ok {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.parent_task_ref", "parent_task_ref no encontrada")
		}
		persistedChildren, err := treeStore.LoadWorkflowTasksByParentV0(ctx, request.Run.RunID, parentRef)
		if err != nil {
			return err
		}
		fanoutByParentRef[parentRef] = append(fanoutByParentRef[parentRef], reviewReworkSplitTaskRefsV0(persistedChildren)...)
	}
	for i := range tasks {
		parentRef := strings.TrimSpace(tasks[i].ParentTaskRef)
		if parentRef == "" {
			continue
		}
		parent := parentsByRef[parentRef]
		reviewReworkInheritRecursiveLimitsV0(&tasks[i], parent)
		if err := reviewReworkValidateRecursiveSplitChildTaskV0(parent, tasks[i]); err != nil {
			return err
		}
		fanoutByParentRef[parentRef] = append(fanoutByParentRef[parentRef], tasks[i].TaskID)
	}
	for _, parentRef := range parentRefs {
		parent := parentsByRef[parentRef]
		fanout := len(compactStringsV0(fanoutByParentRef[parentRef]))
		limit, field := reviewReworkEffectiveFanoutLimitV0(parent)
		if fanout > limit {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, field, "fanout supera limite recursivo")
		}
	}
	if err := reviewReworkValidateRecursiveTreeBudgetsV0(ctx, request, store, treeStore, parentsByRef, tasks); err != nil {
		return err
	}
	return nil
}

func reviewReworkValidateRecursiveSplitChildTaskV0(
	parent orquestacoreworkflow.WorkflowTaskV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
	if parent.MaxDelegationDepth > 0 && task.DelegationDepth > parent.MaxDelegationDepth {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.max_delegation_depth", "delegation_depth supera max_delegation_depth")
	}
	if task.DelegationDepth != parent.DelegationDepth+1 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.delegation_depth", "delegation_depth no sigue al parent")
	}
	if strings.TrimSpace(task.WaveRef) != strings.TrimSpace(parent.WaveRef) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.wave_ref", "wave_ref no coincide con parent")
	}
	if strings.TrimSpace(task.CohortRef) != strings.TrimSpace(parent.CohortRef) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.cohort_ref", "cohort_ref no coincide con parent")
	}
	if len(parent.ChildTaskRefs) > 0 && !stringInSetV0(task.TaskID, parent.ChildTaskRefs) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.child_task_refs", "hijo no declarado por parent")
	}
	return nil
}

func reviewReworkInheritRecursiveLimitsV0(
	task *orquestacoreworkflow.WorkflowTaskV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
) {
	if parent.MaxDelegationDepth > 0 &&
		(task.MaxDelegationDepth == 0 || task.MaxDelegationDepth > parent.MaxDelegationDepth) {
		task.MaxDelegationDepth = parent.MaxDelegationDepth
	}
	if parent.MaxSubagentsPerAgent > 0 &&
		(task.MaxSubagentsPerAgent == 0 || task.MaxSubagentsPerAgent > parent.MaxSubagentsPerAgent) {
		task.MaxSubagentsPerAgent = parent.MaxSubagentsPerAgent
	}
	if parent.MaxSubagentsPerAgent > 0 && task.MaxChildAgents > parent.MaxSubagentsPerAgent {
		task.MaxChildAgents = parent.MaxSubagentsPerAgent
	}
	if parent.MaxRecursiveAgents > 0 &&
		(task.MaxRecursiveAgents == 0 || task.MaxRecursiveAgents > parent.MaxRecursiveAgents) {
		task.MaxRecursiveAgents = parent.MaxRecursiveAgents
	}
}

func reviewReworkEffectiveFanoutLimitV0(
	parent orquestacoreworkflow.WorkflowTaskV0,
) (int, string) {
	limit := parent.MaxChildAgents
	field := "split_task.max_child_agents"
	if parent.MaxSubagentsPerAgent > 0 && parent.MaxSubagentsPerAgent < limit {
		return parent.MaxSubagentsPerAgent, "split_task.max_subagents_per_agent"
	}
	return limit, field
}

func reviewReworkRecursiveSplitParentRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.ParentTaskRef)
	}
	return compactStringsV0(refs)
}

func reviewReworkSplitTaskRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return compactStringsV0(refs)
}
