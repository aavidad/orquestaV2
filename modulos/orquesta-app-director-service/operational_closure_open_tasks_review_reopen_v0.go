package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

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
