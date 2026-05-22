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
	normalized, err := orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task", err.Error())
	}
	if strings.TrimSpace(normalized.RunID) != strings.TrimSpace(request.Run.RunID) {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.run_id", "run_id no coincide")
	}
	if normalized.PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return orquestacoreworkflow.WorkflowTaskV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.phase_id", "phase_id no soportado")
	}
	return normalized, nil
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
	for _, parentRef := range parentRefs {
		if _, ok := parentsByRef[parentRef]; !ok {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.parent_task_ref", "parent_task_ref no encontrada")
		}
	}
	for _, task := range tasks {
		parentRef := strings.TrimSpace(task.ParentTaskRef)
		if parentRef == "" {
			continue
		}
		parent := parentsByRef[parentRef]
		if err := reviewReworkValidateRecursiveSplitChildTaskV0(parent, task); err != nil {
			return err
		}
		fanoutByParentRef[parentRef] = append(fanoutByParentRef[parentRef], task.TaskID)
	}
	for _, parentRef := range parentRefs {
		parent := parentsByRef[parentRef]
		if len(compactStringsV0(fanoutByParentRef[parentRef])) > parent.MaxChildAgents {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.max_child_agents", "fanout supera max_child_agents")
		}
	}
	return nil
}

func reviewReworkValidateRecursiveSplitChildTaskV0(
	parent orquestacoreworkflow.WorkflowTaskV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
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
