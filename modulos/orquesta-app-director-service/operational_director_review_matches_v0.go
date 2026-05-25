package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanAcceptedReviewMatchesV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]operationalDirectorPlanAcceptedReviewMatchV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return nil, false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return nil, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	taskRefs := compactServiceRefsV0(activeStep.TaskRefs)
	if len(taskRefs) == 0 {
		return nil, false, nil
	}
	matches := make([]operationalDirectorPlanAcceptedReviewMatchV0, 0, len(taskRefs))
	for _, taskRef := range taskRefs {
		match, ok := operationalDirectorPlanAcceptedReviewMatchForTaskV0(activeStep, run, trace, taskRef)
		if !ok {
			return nil, false, nil
		}
		matches = append(matches, match)
	}
	return matches, true, nil
}

func operationalDirectorPlanReviewReworkReplanMatchForActiveStepV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	for _, taskRef := range compactServiceRefsV0(activeStep.TaskRefs) {
		match, ok := operationalDirectorPlanReviewReworkReplanMatchForTaskV0(activeStep, run, trace, taskRef)
		if ok {
			return match, true, nil
		}
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false, nil
}

func operationalDirectorPlanReviewReworkReplanMatchForTaskV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, deliveryRef := range trace.DeliveryRefs {
		delivery := trace.Deliveries[deliveryRef]
		if strings.TrimSpace(delivery.TaskID) != taskRef {
			continue
		}
		if len(activeStep.AgentRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.AgentRefs, delivery.AgentRef) {
			continue
		}
		if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, delivery.DeliveryRef) {
			continue
		}
		if !startAppDirectorStringInSetV0(run.Deliveries, delivery.DeliveryRef) ||
			!startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			continue
		}
		if strings.TrimSpace(delivery.AgentRef) != "" &&
			!startAppDirectorStringInSetV0(run.DeliveredAgents, delivery.AgentRef) {
			continue
		}
		match, ok := operationalDirectorPlanReviewReworkReplanMatchForDeliveryV0(run, trace, delivery, taskRef)
		if !ok {
			continue
		}
		if len(activeStep.ReviewResultRefs) > 0 &&
			!startAppDirectorStringInSetV0(activeStep.ReviewResultRefs, match.ReviewResultRef) {
			continue
		}
		match.AgentRef = strings.TrimSpace(delivery.AgentRef)
		return match, true
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false
}

func operationalDirectorPlanReviewReworkReplanMatchForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
	taskRef string,
) (operationalDirectorPlanReviewReworkReplanMatchV0, bool) {
	deliveryRef := strings.TrimSpace(delivery.DeliveryRef)
	for _, reviewRequestID := range trace.ReviewRequestRefs {
		reviewRequest := trace.ReviewRequests[reviewRequestID]
		if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.Reviews, reviewRequest.ReviewRequestID) {
			continue
		}
		reviewResult, ok := operationalDirectorPlanNegativeReviewResultForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		rework, ok := operationalDirectorPlanReworkForReviewResultV0(run, trace, reviewResult, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		replan, ok := operationalDirectorPlanReplanForReworkV0(run, trace, rework, taskRef)
		if !ok {
			continue
		}
		return operationalDirectorPlanReviewReworkReplanMatchV0{
			TaskRef:           strings.TrimSpace(taskRef),
			DeliveryRef:       deliveryRef,
			ReviewRequestID:   strings.TrimSpace(reviewRequest.ReviewRequestID),
			ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
			ReworkRequestRef:  strings.TrimSpace(rework.ReworkRequestRef),
			ReplanDecisionRef: strings.TrimSpace(replan.ReplanRef),
			AcceptedAction:    replan.AcceptedAction,
			FollowupRefs:      append([]string(nil), replan.FollowupRefs...),
			EvidenceRefs: compactServiceRefsV0(append(append(append(
				append([]string(nil), delivery.EvidenceRefs...),
				reviewRequest.EvidenceRefs...),
				reviewResult.EvidenceRefs...),
				append(rework.EvidenceRefs, replan.EvidenceRefs...)...)),
		}, true
	}
	return operationalDirectorPlanReviewReworkReplanMatchV0{}, false
}
