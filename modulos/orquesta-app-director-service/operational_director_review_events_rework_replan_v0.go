package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	"strings"
)

func operationalDirectorPlanNegativeReviewResultForRequestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	reviewRequestID := strings.TrimSpace(reviewRequest.ReviewRequestID)
	for _, reviewResultRef := range trace.ReviewResultRefs {
		reviewResult := trace.ReviewResults[reviewResultRef]
		status := orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(reviewResult.Status)))
		if strings.TrimSpace(reviewResult.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(reviewResult.DeliveryRef) == deliveryRef &&
			(status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
				status == orquestacoreworkflow.ReviewResultStatusRejectedV0) &&
			operationalDirectorPlanProjectionReflectedV0(reviewResult.ReviewResultRef, run.ReviewResults) {
			return reviewResult, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanReworkForReviewResultV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	reviewRequest orquestacoreworkflow.ReviewRequestedPayloadV0,
	deliveryRef string,
) (orquestacoreworkflow.ReworkRequestedPayloadV0, bool) {
	for _, reworkRequestRef := range trace.ReworkRequestRefs {
		rework := trace.ReworkRequests[reworkRequestRef]
		if strings.TrimSpace(rework.ReviewResultRef) == strings.TrimSpace(reviewResult.ReviewResultRef) &&
			strings.TrimSpace(rework.ReviewRequestID) == strings.TrimSpace(reviewRequest.ReviewRequestID) &&
			strings.TrimSpace(rework.DeliveryRef) == strings.TrimSpace(deliveryRef) &&
			operationalDirectorPlanProjectionReflectedV0(rework.ReworkRequestRef, run.ReworkRequests) {
			return rework, true
		}
	}
	return orquestacoreworkflow.ReworkRequestedPayloadV0{}, false
}

func operationalDirectorPlanReplanForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	rework orquestacoreworkflow.ReworkRequestedPayloadV0,
	taskRef string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, bool) {
	for _, replanDecisionRef := range trace.ReplanDecisionRefs {
		replan := trace.ReplanDecisions[replanDecisionRef]
		if strings.TrimSpace(replan.SourceRef) == strings.TrimSpace(rework.ReworkRequestRef) &&
			strings.TrimSpace(replan.TaskRef) == strings.TrimSpace(taskRef) &&
			strings.TrimSpace(replan.RunRef) == strings.TrimSpace(run.RunID) &&
			operationalDirectorPlanProjectionReflectedV0(replan.ReplanRef, run.ReplanDecisions) {
			return replan, true
		}
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, false
}
