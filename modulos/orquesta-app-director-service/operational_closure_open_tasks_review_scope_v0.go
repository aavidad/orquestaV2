package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type operationalDirectorPlanStateClosureScopeV0 struct {
	TaskRefs         []string
	AgentRefs        []string
	DeliveryRefs     []string
	ReviewResultRefs []string
}

type operationalDirectorPlanStateClosureScopeFilterV0 struct {
	WaveRef       string
	CohortRef     string
	ParentTaskRef string
}

func operationalDirectorPlanStateClosureScopeFilterFromStepV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) operationalDirectorPlanStateClosureScopeFilterV0 {
	filter := operationalDirectorPlanStateClosureScopeFilterV0{
		WaveRef:       strings.TrimSpace(step.WaveRef),
		CohortRef:     strings.TrimSpace(step.CohortRef),
		ParentTaskRef: strings.TrimSpace(step.ParentTaskRef),
	}
	if filter.WaveRef == "" {
		filter.WaveRef = strings.TrimSpace(state.ActiveWaveRef)
	}
	if filter.CohortRef == "" {
		filter.CohortRef = strings.TrimSpace(state.ActiveCohortRef)
	}
	if filter.ParentTaskRef == "" {
		filter.ParentTaskRef = strings.TrimSpace(state.ActiveParentTaskRef)
	}
	return filter
}

func operationalDirectorPlanStateOpenAcceptedLeafTasksForClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	filter operationalDirectorPlanStateClosureScopeFilterV0,
) (operationalDirectorPlanStateClosureScopeV0, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return operationalDirectorPlanStateClosureScopeV0{}, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return operationalDirectorPlanStateClosureScopeV0{}, err
	}
	tasks, err := ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, request.RunRef, run.Tasks)
	if err != nil {
		return operationalDirectorPlanStateClosureScopeV0{}, err
	}
	tasksByRef := make(map[string]orquestacoreworkflow.WorkflowTaskV0, len(tasks))
	for _, task := range tasks {
		tasksByRef[strings.TrimSpace(task.TaskID)] = task
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	closed := serviceStringSetV0(run.ClosedTasks)
	scope := operationalDirectorPlanStateClosureScopeV0{}
	for _, taskRef := range compactServiceRefsV0(run.Tasks) {
		if task, ok := tasksByRef[taskRef]; ok && !operationalDirectorPlanStateTaskMatchesClosureScopeFilterV0(task, filter) {
			continue
		}
		if closed[taskRef] ||
			!startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) ||
			operationalDirectorPlanStateTaskHasOpenChildrenV0(run, trace, taskRef, tasksByRef, closed) {
			continue
		}
		match, ok := operationalDirectorPlanAcceptedReviewMatchForTaskV0(
			orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{TaskRefs: []string{taskRef}},
			run,
			trace,
			taskRef,
		)
		if !ok || match.AcceptedReviewRef == "" {
			continue
		}
		scope.TaskRefs = append(scope.TaskRefs, taskRef)
		scope.AgentRefs = append(scope.AgentRefs, match.AgentRef)
		scope.DeliveryRefs = append(scope.DeliveryRefs, match.DeliveryRef)
		scope.ReviewResultRefs = append(scope.ReviewResultRefs, match.ReviewResultRef)
	}
	scope.TaskRefs = compactServiceRefsV0(scope.TaskRefs)
	scope.AgentRefs = compactServiceRefsV0(scope.AgentRefs)
	scope.DeliveryRefs = compactServiceRefsV0(scope.DeliveryRefs)
	scope.ReviewResultRefs = compactServiceRefsV0(scope.ReviewResultRefs)
	return scope, nil
}

func operationalDirectorPlanStateTaskMatchesClosureScopeFilterV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	filter operationalDirectorPlanStateClosureScopeFilterV0,
) bool {
	if filter.WaveRef != "" {
		taskWaveRef := strings.TrimSpace(task.WaveRef)
		if taskWaveRef != "" && taskWaveRef != filter.WaveRef {
			return false
		}
	}
	if filter.CohortRef != "" {
		taskCohortRef := strings.TrimSpace(task.CohortRef)
		if taskCohortRef != "" && taskCohortRef != filter.CohortRef {
			return false
		}
	}
	if filter.ParentTaskRef != "" {
		taskRef := strings.TrimSpace(task.TaskID)
		taskParentRef := strings.TrimSpace(task.ParentTaskRef)
		if taskRef != filter.ParentTaskRef && taskParentRef != "" && taskParentRef != filter.ParentTaskRef {
			return false
		}
	}
	return true
}

func operationalDirectorPlanStateTaskHasOpenChildrenV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
	tasksByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
	closed map[string]bool,
) bool {
	for _, childRef := range operationalDirectorPlanStateChildTaskRefsV0(run, trace, taskRef, tasksByRef) {
		if startAppDirectorStringInSetV0(run.Tasks, childRef) && !closed[childRef] {
			return true
		}
	}
	return false
}

func operationalDirectorPlanStateChildTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
	tasksByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
) []string {
	taskRef = strings.TrimSpace(taskRef)
	refs := []string(nil)
	if task, ok := tasksByRef[taskRef]; ok {
		refs = append(refs, task.ChildTaskRefs...)
	}
	for _, replanRef := range trace.ReplanDecisionRefs {
		replan := trace.ReplanDecisions[replanRef]
		if strings.TrimSpace(replan.TaskRef) != taskRef ||
			replan.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
			!operationalDirectorPlanProjectionReflectedV0(replan.ReplanRef, run.ReplanDecisions) {
			continue
		}
		refs = append(refs, replan.FollowupRefs...)
	}
	return compactServiceRefsV0(refs)
}
