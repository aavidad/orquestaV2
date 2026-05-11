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
	candidates := make([]orquestadirector.ReplanMicrotaskCandidateV0, 0, len(plan.SplitTasks))
	for _, raw := range plan.SplitTasks {
		task, err := reviewReworkSplitTaskV0(request, raw)
		if err != nil {
			return nil, err
		}
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

func reviewReworkSplitTaskRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return compactStringsV0(refs)
}
