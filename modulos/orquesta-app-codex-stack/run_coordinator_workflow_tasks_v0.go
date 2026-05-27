package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (stack StackV0) repairQueuedAutoprogrammingWorkflowTasksV0(
	ctx context.Context,
	runRef string,
) error {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return err
	}
	openTaskRefs := stackDrainOpenTaskRefsV0(run)
	if len(openTaskRefs) == 0 {
		openTaskRefs = compactStringsV0(run.Tasks)
	}
	if !codexStackRunLooksAutoprogrammingV0(run, openTaskRefs) {
		return nil
	}
	tasks, err := stack.loadAvailableQueuedWorkflowTasksV0(ctx, run.RunID, openTaskRefs)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if codexStackWorkflowTaskLooksOperationalDirectorV0(task) ||
			!codexStackWorkflowTaskLooksAutoprogrammingV0(task) {
			continue
		}
		task.ContextRefs = compactStringsV0(append(
			[]string{autoprogrammingBridgeOperationalTaskSourceRefV0},
			task.ContextRefs...,
		))
		if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
			if codexStackWorkflowTaskImmutableConflictV0(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func codexStackWorkflowTaskImmutableConflictV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_task" &&
		strings.Contains(issue.Message, "microtarea existente con contrato distinto")
}

func (stack StackV0) queuedOperationalDirectorWaitAgentRefsV0(
	ctx context.Context,
	runRef string,
) ([]string, error) {
	if stack.Ports.RunStore == nil || stack.Ports.DirectorTaskStore == nil {
		return nil, nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return nil, err
	}
	openTaskRefs := stackDrainOpenTaskRefsV0(run)
	if len(openTaskRefs) == 0 {
		openTaskRefs = compactStringsV0(run.Tasks)
	}
	tasks, err := stack.loadAvailableQueuedWorkflowTasksV0(ctx, run.RunID, openTaskRefs)
	if err != nil {
		return nil, err
	}
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if !codexStackWorkflowTaskLooksOperationalDirectorV0(task) {
			continue
		}
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	if len(refs) > 0 {
		refs = append(refs, stackDrainPendingStartedAgentRefsV0(run)...)
	}
	return compactStringsV0(refs), nil
}

func (stack StackV0) loadAvailableQueuedWorkflowTasksV0(
	ctx context.Context,
	runRef string,
	taskRefs []string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	if stack.Ports.DirectorTaskStore == nil {
		return nil, nil
	}
	refs := compactStringsV0(taskRefs)
	if len(refs) == 0 {
		return nil, nil
	}
	tasks := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(refs))
	for _, taskRef := range refs {
		loaded, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, runRef, []string{taskRef})
		if err != nil {
			if codexStackWorkflowTaskMissingV0(err) {
				continue
			}
			return nil, err
		}
		tasks = append(tasks, loaded...)
	}
	return tasks, nil
}

func codexStackWorkflowTaskMissingV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_tasks" &&
		strings.Contains(issue.Message, "microtarea no encontrada")
}

func stackDrainOpenTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	closed := map[string]bool{}
	for _, taskRef := range compactStringsV0(run.ClosedTasks) {
		closed[taskRef] = true
	}
	for _, taskRef := range compactStringsV0(run.DeliveredTasks) {
		closed[taskRef] = true
	}
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if closed[taskRef] {
			continue
		}
		refs = append(refs, taskRef)
	}
	return refs
}

func stackDrainPendingStartedAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	terminal := map[string]bool{}
	for _, values := range [][]string{
		run.DeliveredAgents,
		run.FailedAgents,
		run.LostAgents,
		run.StoppedAgents,
		run.ConfirmedStoppedAgents,
	} {
		for _, agentRef := range compactStringsV0(values) {
			terminal[agentRef] = true
		}
	}
	refs := make([]string, 0, len(run.StartedAgents))
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if terminal[agentRef] {
			continue
		}
		refs = append(refs, agentRef)
	}
	return refs
}
