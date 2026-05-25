package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func nextReviewGateCandidateV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) (orquestadirectorscheduler.SchedulableReviewGateCandidateV0, bool) {
	switch {
	case reviewGateNeedsRequestReviewV0(run, candidate.RequestReview):
		return reviewGateCandidateOnlyRequestReviewV0(candidate), true
	case reviewGateNeedsRecordResultV0(run, candidate.RecordReviewResult):
		return reviewGateCandidateOnlyRecordReviewResultV0(candidate), true
	case reviewGateNeedsAcceptReviewV0(run, candidate.AcceptReview):
		return reviewGateCandidateOnlyAcceptReviewV0(candidate), true
	case reviewGateNeedsRequestReworkV0(run, candidate.RequestRework):
		return reviewGateCandidateOnlyRequestReworkV0(candidate), true
	default:
		return orquestadirectorscheduler.SchedulableReviewGateCandidateV0{}, false
	}
}

func reviewGateCandidateOnlyRequestReviewV0(
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	candidate.RecordReviewResult = nil
	candidate.AcceptReview = nil
	candidate.RequestRework = nil
	return candidate
}

func reviewGateCandidateOnlyRecordReviewResultV0(
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	candidate.RequestReview = nil
	candidate.AcceptReview = nil
	candidate.RequestRework = nil
	return candidate
}

func reviewGateCandidateOnlyAcceptReviewV0(
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	candidate.RequestReview = nil
	candidate.RecordReviewResult = nil
	candidate.RequestRework = nil
	return candidate
}

func reviewGateCandidateOnlyRequestReworkV0(
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	candidate.RequestReview = nil
	candidate.RecordReviewResult = nil
	candidate.AcceptReview = nil
	return candidate
}

func reviewGateNeedsRequestReviewV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	candidate *orquestadirectorscheduler.SchedulerRequestReviewCandidateV0,
) bool {
	return candidate != nil && !stringInSetV0(candidate.Payload.ReviewRequestID, run.Reviews)
}

func reviewGateNeedsRecordResultV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	candidate *orquestadirectorscheduler.SchedulerRecordReviewResultCandidateV0,
) bool {
	return candidate != nil &&
		stringInSetV0(candidate.Payload.ReviewRequestID, run.Reviews) &&
		!reviewGateResultKnownV0(run, candidate.Payload.ReviewResultRef)
}

func reviewGateNeedsAcceptReviewV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	candidate *orquestadirectorscheduler.SchedulerAcceptReviewCandidateV0,
) bool {
	return candidate != nil &&
		reviewGateAcceptedResultKnownV0(run, candidate.Payload.ReviewRequestID, candidate.Payload.DeliveryRef) &&
		!stringInSetV0(candidate.Payload.AcceptedReviewRef, run.AcceptedReviews)
}

func reviewGateNeedsRequestReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	candidate *orquestadirectorscheduler.SchedulerRequestReworkCandidateV0,
) bool {
	return candidate != nil &&
		reviewGateReworkResultKnownV0(
			run,
			candidate.Payload.ReviewResultRef,
			candidate.Payload.ReviewRequestID,
			candidate.Payload.DeliveryRef,
		) &&
		!reviewGateReworkKnownV0(run, candidate.Payload.ReworkRequestRef)
}

func reviewGateResultKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	resultRef string,
) bool {
	resultRef = strings.TrimSpace(resultRef)
	for _, projection := range run.ReviewResults {
		if projection == resultRef || strings.HasPrefix(projection, resultRef+"#review_result:") {
			return true
		}
	}
	return false
}

func reviewGateAcceptedResultKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRef string,
	deliveryRef string,
) bool {
	status, ok := reviewGateResultStatusForTargetV0(run, reviewRef, deliveryRef)
	return ok && status == string(orquestacoreworkflow.ReviewResultStatusAcceptedV0)
}

func reviewGateReworkResultKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	resultRef string,
	reviewRef string,
	deliveryRef string,
) bool {
	resultRef = strings.TrimSpace(resultRef)
	for _, projection := range run.ReviewResults {
		if projection != resultRef && !strings.HasPrefix(projection, resultRef+"#review_result:") {
			continue
		}
		status, request, delivery, ok := reviewGateResultProjectionV0(projection)
		if ok && request == strings.TrimSpace(reviewRef) &&
			delivery == strings.TrimSpace(deliveryRef) &&
			reviewGateStatusSupportsReworkV0(orquestacoreworkflow.ReviewResultStatusV0(status)) {
			return true
		}
	}
	return false
}

func reviewGateResultStatusForTargetV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reviewRef string,
	deliveryRef string,
) (string, bool) {
	var first string
	for _, projection := range run.ReviewResults {
		status, request, delivery, ok := reviewGateResultProjectionV0(projection)
		if ok && request == strings.TrimSpace(reviewRef) &&
			delivery == strings.TrimSpace(deliveryRef) {
			if status == string(orquestacoreworkflow.ReviewResultStatusAcceptedV0) {
				return status, true
			}
			if first == "" {
				first = status
			}
		}
	}
	if first != "" {
		return first, true
	}
	return "", false
}

func reviewGateResultProjectionV0(projection string) (string, string, string, bool) {
	_, tail, ok := strings.Cut(strings.TrimSpace(projection), "#review_result:")
	if !ok {
		return "", "", "", false
	}
	status, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return "", "", "", false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return "", "", "", false
	}
	return strings.TrimSpace(status), strings.TrimSpace(reviewRef), strings.TrimSpace(deliveryRef), true
}

func reviewGateReworkKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reworkRef string,
) bool {
	reworkRef = strings.TrimSpace(reworkRef)
	for _, projection := range run.ReworkRequests {
		if projection == reworkRef || strings.HasPrefix(projection, reworkRef+"#review_result:") {
			return true
		}
	}
	return false
}
