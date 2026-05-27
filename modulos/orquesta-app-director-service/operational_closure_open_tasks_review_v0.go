package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const operationalClosureOpenTasksNoProgressReasonV0 = "operational-closure-open-tasks-no-progress"

func operationalDirectorPlanStateReopenReviewForClosureOpenTasksNoProgressV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return false, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	next, reopened, err := operationalDirectorPlanStateReopenReviewForOpenDeliveredTasksV0(
		ctx,
		request,
		ports,
		state,
		run,
		operationalClosureOpenTasksNoProgressReasonV0,
	)
	if err != nil || !reopened {
		return reopened, err
	}
	return true, ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func continueOperationalDirectorPlanStateAfterClosureOpenTasksNoProgressV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.RunStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		!operationalDirectorPlanStateHasReasonOrBlockerV0(state, activeStep, operationalClosureOpenTasksNoProgressReasonV0) {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	return operationalDirectorPlanStateReopenReviewForOpenDeliveredTasksV0(
		ctx,
		request,
		ports,
		state,
		run,
		operationalClosureOpenTasksNoProgressReasonV0,
	)
}

func continueOperationalDirectorPlanStateRefreshStaleClosureScopeV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.RunStore == nil || ports.DirectorTaskStore == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return state, false, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return state, false, nil
	}
	scope, err := operationalDirectorPlanStateOpenAcceptedLeafTasksForClosureV0(
		ctx,
		request,
		ports,
		run,
		operationalDirectorPlanStateClosureScopeFilterFromStepV0(state, activeStep),
	)
	if err != nil || len(scope.TaskRefs) == 0 {
		return state, false, err
	}
	if operationalDirectorPlanStateSameRefsV0(activeStep.TaskRefs, scope.TaskRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.AgentRefs, scope.AgentRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.DeliveryRefs, scope.DeliveryRefs) &&
		operationalDirectorPlanStateSameRefsV0(activeStep.ReviewResultRefs, scope.ReviewResultRefs) {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.TaskRefs = append([]string(nil), scope.TaskRefs...)
			nextStep.AgentRefs = append([]string(nil), scope.AgentRefs...)
			nextStep.DeliveryRefs = append([]string(nil), scope.DeliveryRefs...)
			nextStep.ReviewResultRefs = append([]string(nil), scope.ReviewResultRefs...)
			nextStep.BlockerRefs = nil
			nextStep.Reason = "closure-scope-refreshed-from-open-reviewed-tasks"
			nextStep.EvidenceRefs = compactServiceRefsV0(append(
				nextStep.EvidenceRefs,
				"evidence-ref-app-director-closure-scope-refreshed-v0",
			))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-closure-scope-refreshed-v0",
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

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

func operationalDirectorPlanStateReopenReviewForOpenDeliveredTasksV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	reason string,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		return state, false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 &&
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	taskRefs, agentRefs, deliveryRefs, err := operationalDirectorPlanStateOpenDeliveredTasksWithoutAcceptedReviewV0(ctx, request, ports, run)
	if err != nil || len(taskRefs) == 0 {
		return state, false, err
	}
	reviewStepID := operationalDirectorPlanStateStepIDByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0)
	if reviewStepID == "" {
		return state, false, nil
	}
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch step.StepID {
		case reviewStepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.TaskRefs = append([]string(nil), taskRefs...)
			nextStep.AgentRefs = append([]string(nil), agentRefs...)
			nextStep.DeliveryRefs = append([]string(nil), deliveryRefs...)
			nextStep.ReviewResultRefs = nil
			nextStep.AcceptedReviewRefs = nil
			nextStep.ReworkRequestRefs = nil
			nextStep.ReplanDecisionRefs = nil
			nextStep.BlockerRefs = nil
			nextStep.Reason = "closure-open-tasks-review-followups"
		case activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.BlockerRefs = nil
			nextStep.Reason = ""
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = reviewStepID
	state.ActiveWaveRef = ""
	state.ActiveCohortRef = ""
	state.ActiveParentTaskRef = ""
	state.PendingAgentRefs = append([]string(nil), agentRefs...)
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-closure-open-tasks-review-followups-v0",
		reason,
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanStateOpenDeliveredTasksWithoutAcceptedReviewV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]string, []string, []string, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return nil, nil, nil, nil
	}
	events, err := loadOperationalRunEventsV0(ctx, reader, request.RunRef)
	if err != nil {
		return nil, nil, nil, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	closed := serviceStringSetV0(run.ClosedTasks)
	taskRefs := []string{}
	agentRefs := []string{}
	deliveryRefs := []string{}
	for _, taskRef := range compactServiceRefsV0(run.Tasks) {
		if closed[taskRef] || !startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			continue
		}
		match, ok := operationalDirectorPlanAcceptedReviewMatchForTaskV0(
			orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{TaskRefs: []string{taskRef}},
			run,
			trace,
			taskRef,
		)
		if ok && match.AcceptedReviewRef != "" {
			continue
		}
		agentRef, deliveryRef := operationalDirectorPlanLatestDeliveredAgentAndDeliveryForTaskV0(run, trace, taskRef)
		if deliveryRef == "" {
			continue
		}
		taskRefs = append(taskRefs, taskRef)
		agentRefs = append(agentRefs, agentRef)
		deliveryRefs = append(deliveryRefs, deliveryRef)
	}
	return compactServiceRefsV0(taskRefs), compactServiceRefsV0(agentRefs), compactServiceRefsV0(deliveryRefs), nil
}

func operationalDirectorPlanLatestDeliveredAgentAndDeliveryForTaskV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (string, string) {
	for index := len(trace.DeliveryRefs) - 1; index >= 0; index-- {
		delivery := trace.Deliveries[trace.DeliveryRefs[index]]
		if strings.TrimSpace(delivery.TaskID) != taskRef ||
			!startAppDirectorStringInSetV0(run.Deliveries, delivery.DeliveryRef) {
			continue
		}
		if delivery.AgentRef != "" && !startAppDirectorStringInSetV0(run.DeliveredAgents, delivery.AgentRef) {
			continue
		}
		agentRef := strings.TrimSpace(delivery.AgentRef)
		if agentRef == "" {
			agentRef = orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
		}
		return agentRef, strings.TrimSpace(delivery.DeliveryRef)
	}
	return "", ""
}

func operationalDirectorPlanStateHasReasonOrBlockerV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	reason string,
) bool {
	reason = strings.TrimSpace(reason)
	return strings.TrimSpace(state.ClosureReason) == reason ||
		strings.TrimSpace(step.Reason) == reason ||
		startAppDirectorStringInSetV0(state.BlockerRefs, reason) ||
		startAppDirectorStringInSetV0(step.BlockerRefs, reason)
}

func operationalDirectorPlanStateRefsWithoutV0(values []string, ignored string) []string {
	ignored = strings.TrimSpace(ignored)
	out := make([]string, 0, len(values))
	for _, value := range compactServiceRefsV0(values) {
		if strings.TrimSpace(value) == ignored {
			continue
		}
		out = append(out, value)
	}
	return out
}

func operationalDirectorPlanStateSameRefsV0(left []string, right []string) bool {
	left = compactServiceRefsV0(left)
	right = compactServiceRefsV0(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
