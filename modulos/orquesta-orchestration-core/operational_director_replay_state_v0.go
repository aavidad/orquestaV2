package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type OperationalDirectorReplayStateCheckRequestV0 struct {
	Run            orquestacoreworkflow.OrchestrationRunV0
	PlanRef        string
	PlanStateStore OperationalDirectorPlanStateStorePortV0
	TaskStore      WorkflowTaskStorePortV0
	WaitStateStore WorkflowTaskWaitStateStorePortV0
}

type OperationalDirectorReplayStateCheckResultV0 struct {
	Restored   bool
	PlanState  OperationalDirectorPlanStateV0
	Tasks      []orquestacoreworkflow.WorkflowTaskV0
	WaitStates []WorkflowTaskWaitStateV0
	Issues     []ErrorV0
}

func CheckOperationalDirectorReplayStateV0(
	ctx context.Context,
	request OperationalDirectorReplayStateCheckRequestV0,
) (OperationalDirectorReplayStateCheckResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OperationalDirectorReplayStateCheckResultV0{}, err
	}
	request.PlanRef = strings.TrimSpace(request.PlanRef)
	result := OperationalDirectorReplayStateCheckResultV0{}
	if issues := validateReplayStateCheckRequestV0(request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	state, err := request.PlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.Run.RunID, request.PlanRef)
	if err != nil {
		result.Issues = append(result.Issues, replayStateIssueV0("plan_state", err))
		return result, nil
	}
	result.PlanState = state
	result.Issues = append(result.Issues, checkReplayPlanStateScopeV0(request.Run, state)...)
	tasks, taskIssues := loadReplayStateTasksV0(ctx, request, state)
	result.Tasks = tasks
	result.Issues = append(result.Issues, taskIssues...)
	waitStates, waitIssues := loadReplayWaitStatesV0(ctx, request, state)
	result.WaitStates = waitStates
	result.Issues = append(result.Issues, waitIssues...)
	result.Issues = append(result.Issues, checkReplayStateMetadataV0(state, tasks, waitStates)...)
	result.Restored = len(result.Issues) == 0
	return result, nil
}

func validateReplayStateCheckRequestV0(
	request OperationalDirectorReplayStateCheckRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if strings.TrimSpace(request.Run.RunID) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido"))
	}
	if request.PlanRef == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan_ref", "plan_ref requerido"))
	}
	if request.PlanStateStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan_state_store", "plan_state_store requerido"))
	}
	if request.TaskStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "workflow_task_store", "workflow_task_store requerido"))
	}
	if request.WaitStateStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "wait_state_store", "wait_state_store requerido"))
	}
	return issues
}

func checkReplayPlanStateScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	state OperationalDirectorPlanStateV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if state.RunRef != run.RunID {
		issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "plan_state.run_ref", "run_ref inconsistente"))
	}
	for _, taskRef := range replayStateTaskRefsV0(state) {
		if !stringInSetV0(taskRef, run.Tasks) {
			issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "plan_state.task_refs", "task_ref no autorizado por run"))
		}
		if stringInSetV0(taskRef, run.ClosedTasks) {
			issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "plan_state.task_refs", "task_ref cerrado no puede estar en scope vivo"))
		}
	}
	return issues
}

func loadReplayStateTasksV0(
	ctx context.Context,
	request OperationalDirectorReplayStateCheckRequestV0,
	state OperationalDirectorPlanStateV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, []ErrorV0) {
	taskRefs := replayStateTaskRefsV0(state)
	if len(taskRefs) == 0 {
		return nil, nil
	}
	tasks, err := request.TaskStore.LoadWorkflowTasksV0(ctx, request.Run.RunID, taskRefs)
	if err != nil {
		return nil, []ErrorV0{replayStateIssueV0("workflow_task_store", err)}
	}
	return tasks, nil
}

func loadReplayWaitStatesV0(
	ctx context.Context,
	request OperationalDirectorReplayStateCheckRequestV0,
	state OperationalDirectorPlanStateV0,
) ([]WorkflowTaskWaitStateV0, []ErrorV0) {
	waitRefs := replayStateWaitRefsV0(state)
	if len(waitRefs) == 0 {
		return nil, nil
	}
	out := make([]WorkflowTaskWaitStateV0, 0, len(waitRefs))
	issues := make([]ErrorV0, 0)
	for _, waitRef := range waitRefs {
		waitState, err := request.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.Run.RunID, waitRef)
		if err != nil {
			issues = append(issues, replayStateIssueV0("workflow_task_wait_state_store", err))
			continue
		}
		out = append(out, waitState)
	}
	return out, issues
}

func checkReplayStateMetadataV0(
	state OperationalDirectorPlanStateV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	waitStates []WorkflowTaskWaitStateV0,
) []ErrorV0 {
	taskByRef := map[string]orquestacoreworkflow.WorkflowTaskV0{}
	for _, task := range tasks {
		taskByRef[task.TaskID] = task
	}
	waitByRef := map[string]WorkflowTaskWaitStateV0{}
	for _, waitState := range waitStates {
		waitByRef[waitState.WaitRef] = waitState
	}
	issues := make([]ErrorV0, 0)
	for _, step := range state.Steps {
		issues = append(issues, checkReplayStepTaskMetadataV0(step, taskByRef)...)
		issues = append(issues, checkReplayStepWaitMetadataV0(step, waitByRef)...)
	}
	return issues
}

func checkReplayStepTaskMetadataV0(
	step OperationalDirectorPlanStepStateV0,
	taskByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for _, taskRef := range step.TaskRefs {
		task, ok := taskByRef[taskRef]
		if !ok {
			issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "workflow_task_store", "metadata de task no restaurada"))
			continue
		}
		issues = append(issues, checkReplayMetadataRefV0("task.wave_ref", step.WaveRef, task.WaveRef)...)
		issues = append(issues, checkReplayMetadataRefV0("task.cohort_ref", step.CohortRef, task.CohortRef)...)
		issues = append(issues, checkReplayMetadataRefV0("task.parent_task_ref", step.ParentTaskRef, task.ParentTaskRef)...)
	}
	return issues
}

func checkReplayStepWaitMetadataV0(
	step OperationalDirectorPlanStepStateV0,
	waitByRef map[string]WorkflowTaskWaitStateV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for _, waitRef := range step.WaitRefs {
		waitState, ok := waitByRef[waitRef]
		if !ok {
			issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "workflow_task_wait_state_store", "estado de espera no restaurado"))
			continue
		}
		issues = append(issues, checkReplayMetadataRefV0("wait.wave_ref", step.WaveRef, waitState.WaveRef)...)
		issues = append(issues, checkReplayMetadataRefV0("wait.cohort_ref", step.CohortRef, waitState.CohortRef)...)
		issues = append(issues, checkReplayMetadataRefV0("wait.parent_task_ref", step.ParentTaskRef, waitState.ParentTaskRef)...)
		for _, taskRef := range waitState.TaskRefs {
			if !stringInSetV0(taskRef, step.TaskRefs) {
				issues = append(issues, errorV0(ErrNucleoOrquestacionStoreV0, "wait.task_refs", "wait state apunta fuera del scope del step"))
			}
		}
	}
	return issues
}

func checkReplayMetadataRefV0(field string, expected string, actual string) []ErrorV0 {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == expected {
		return nil
	}
	return []ErrorV0{errorV0(ErrNucleoOrquestacionStoreV0, field, "metadata viva no coincide con plan state")}
}

func replayStateTaskRefsV0(state OperationalDirectorPlanStateV0) []string {
	refs := make([]string, 0)
	for _, step := range state.Steps {
		refs = append(refs, step.TaskRefs...)
	}
	return compactStringsV0(refs)
}

func replayStateWaitRefsV0(state OperationalDirectorPlanStateV0) []string {
	refs := make([]string, 0)
	for _, step := range state.Steps {
		refs = append(refs, step.WaitRefs...)
	}
	return compactStringsV0(refs)
}

func replayStateIssueV0(field string, err error) ErrorV0 {
	return errorV0(ErrNucleoOrquestacionStoreV0, field, fmt.Sprintf("estado operacional no restaurado: %v", err))
}
