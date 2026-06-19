package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanAcceptedReviewMatchForTaskV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	taskRef string,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
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
		match, ok := operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(run, trace, delivery)
		if !ok {
			continue
		}
		if len(activeStep.ReviewResultRefs) > 0 &&
			!startAppDirectorStringInSetV0(activeStep.ReviewResultRefs, match.ReviewResultRef) {
			continue
		}
		match.TaskRef = taskRef
		match.AgentRef = strings.TrimSpace(delivery.AgentRef)
		return match, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanAcceptedReviewMatchForDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	delivery orquestacoreworkflow.DeliveryRegisteredPayloadV0,
) (operationalDirectorPlanAcceptedReviewMatchV0, bool) {
	deliveryRef := strings.TrimSpace(delivery.DeliveryRef)
	for _, reviewRequestID := range trace.ReviewRequestRefs {
		reviewRequest := trace.ReviewRequests[reviewRequestID]
		if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.Reviews, reviewRequest.ReviewRequestID) {
			continue
		}
		reviewResult, acceptedReview, ok := operationalDirectorPlanAcceptedReviewPairForRequestV0(run, trace, reviewRequest, deliveryRef)
		if !ok {
			continue
		}
		return operationalDirectorPlanAcceptedReviewMatchV0{
			DeliveryRef:       deliveryRef,
			ReviewRequestID:   strings.TrimSpace(reviewRequest.ReviewRequestID),
			ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
			AcceptedReviewRef: strings.TrimSpace(acceptedReview.AcceptedReviewRef),
			EvidenceRefs: compactServiceRefsV0(append(append(append(
				append([]string(nil), delivery.EvidenceRefs...),
				reviewRequest.EvidenceRefs...),
				reviewResult.EvidenceRefs...),
				acceptedReview.EvidenceRefs...)),
		}, true
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{}, false
}

func operationalDirectorPlanAcceptedReviewPairForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, orquestacoreworkflow.ReviewAcceptedPayloadV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	var fallbackResult orquestacoreworkflow.ReviewResultV0
	var fallbackAccepted orquestacoreworkflow.ReviewAcceptedPayloadV0
	haveFallback := false
	for _, acceptedReviewRef := range trace.AcceptedReviewRefs {
		acceptedReview := trace.AcceptedReviews[acceptedReviewRef]
		if strings.TrimSpace(acceptedReview.ReviewRequestID) != reviewRequestID ||
			strings.TrimSpace(acceptedReview.DeliveryRef) != deliveryRef ||
			!startAppDirectorStringInSetV0(run.AcceptedReviews, acceptedReview.AcceptedReviewRef) {
			continue
		}
		reviewResult, direct, ok := operationalDirectorPlanAcceptedReviewResultForAcceptedV0(run, trace, acceptedReview, deliveryRef)
		if !ok {
			continue
		}
		if direct {
			return reviewResult, acceptedReview, true
		}
		if !haveFallback {
			fallbackResult = reviewResult
			fallbackAccepted = acceptedReview
			haveFallback = true
		}
	}
	return fallbackResult, fallbackAccepted, haveFallback
}

func operationalDirectorPlanAcceptedReviewResultForAcceptedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	acceptedReview orquestacoreworkflow.ReviewAcceptedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool, bool) {
	acceptedEvidence := operationalDirectorPlanReviewEvidenceSetV0(acceptedReview.EvidenceRefs)
	var fallback orquestacoreworkflow.ReviewResultV0
	haveFallback := false
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		if strings.TrimSpace(reviewResult.ReviewRequestID) != strings.TrimSpace(acceptedReview.ReviewRequestID) ||
			strings.TrimSpace(reviewResult.DeliveryRef) != deliveryRef ||
			orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status))) != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
			!operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			continue
		}
		if operationalDirectorPlanReviewResultMatchesAcceptedV0(reviewResult, acceptedEvidence) {
			return reviewResult, true, true
		}
		if !haveFallback {
			fallback = reviewResult
			haveFallback = true
		}
	}
	return fallback, false, haveFallback
}

func operationalDirectorPlanReviewResultMatchesAcceptedV0(
	reviewResult orquestacoreworkflow.ReviewResultV0,
	acceptedEvidence map[string]bool,
) bool {
	if len(acceptedEvidence) == 0 {
		return false
	}
	if acceptedEvidence[strings.TrimSpace(reviewResult.ReviewResultRef)] {
		return true
	}
	for _, evidenceRef := range reviewResult.EvidenceRefs {
		if acceptedEvidence[strings.TrimSpace(evidenceRef)] {
			return true
		}
	}
	return false
}

func operationalDirectorPlanReviewEvidenceSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range compactServiceRefsV0(values) {
		out[strings.TrimSpace(value)] = true
	}
	return out
}
