package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	schedulerReviewResultStatusSepV0   = "#review_result:"
	schedulerReviewResultRequestSepV0  = "#review_request:"
	schedulerReviewResultDeliverySepV0 = "#delivery:"
)

func (collector *schedulerTickCollectorV0) collectReviewGateCandidateV0(
	candidate SchedulableReviewGateCandidateV0,
) error {
	if collector.input.Snapshot.CurrentPhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if done, err := collector.collectReviewRequestV0(candidate.RequestReview); done || err != nil {
		return err
	}
	if done, err := collector.collectReviewResultV0(candidate.RecordReviewResult); done || err != nil {
		return err
	}
	if done, err := collector.collectReviewAcceptV0(candidate.AcceptReview); done || err != nil {
		return err
	}
	if done, err := collector.collectReviewReworkV0(candidate.RequestRework); done || err != nil {
		return err
	}
	return nil
}

func (collector *schedulerTickCollectorV0) collectReviewRequestV0(
	candidate *SchedulerRequestReviewCandidateV0,
) (bool, error) {
	if candidate == nil {
		return false, nil
	}
	payload := candidate.Payload
	if collector.reviews[payload.ReviewRequestID] || collector.plannedReviews[payload.ReviewRequestID] {
		return false, nil
	}
	if !collector.deliveries[payload.DeliveryRef] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	command, err := orquestacoreworkflow.NewRequestReviewCommandV0(candidate.CommandMeta, payload)
	if err != nil {
		return true, err
	}
	collector.addReadyCommandV0(command)
	collector.plannedReviews[payload.ReviewRequestID] = true
	return true, nil
}

func (collector *schedulerTickCollectorV0) collectReviewResultV0(
	candidate *SchedulerRecordReviewResultCandidateV0,
) (bool, error) {
	if candidate == nil {
		return false, nil
	}
	payload := candidate.Payload
	if collector.reviewResultKnownV0(payload.ReviewResultRef) ||
		collector.plannedReviewResults[payload.ReviewResultRef] {
		return false, nil
	}
	if !collector.reviews[payload.ReviewRequestID] && !collector.plannedReviews[payload.ReviewRequestID] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	if !collector.deliveries[payload.DeliveryRef] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	command, err := orquestacoreworkflow.NewRecordReviewResultCommandV0(candidate.CommandMeta, payload)
	if err != nil {
		return true, err
	}
	collector.addReadyCommandV0(command)
	collector.plannedReviewResults[payload.ReviewResultRef] = true
	return true, nil
}

func (collector *schedulerTickCollectorV0) collectReviewAcceptV0(
	candidate *SchedulerAcceptReviewCandidateV0,
) (bool, error) {
	if candidate == nil {
		return false, nil
	}
	payload := candidate.Payload
	if collector.acceptedReviews[payload.AcceptedReviewRef] ||
		collector.plannedAcceptedReviews[payload.AcceptedReviewRef] {
		return true, nil
	}
	if !collector.reviews[payload.ReviewRequestID] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	if !collector.acceptedReviewResultKnownV0(payload.ReviewRequestID, payload.DeliveryRef) {
		collector.addWaitingV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	command, err := orquestacoreworkflow.NewAcceptReviewCommandV0(candidate.CommandMeta, payload)
	if err != nil {
		return true, err
	}
	collector.addReadyCommandV0(command)
	collector.plannedAcceptedReviews[payload.AcceptedReviewRef] = true
	return true, nil
}

func (collector *schedulerTickCollectorV0) collectReviewReworkV0(
	candidate *SchedulerRequestReworkCandidateV0,
) (bool, error) {
	if candidate == nil {
		return false, nil
	}
	payload := candidate.Payload
	if collector.reworkRequestKnownV0(payload.ReworkRequestRef) ||
		collector.plannedReworkRequests[payload.ReworkRequestRef] {
		return false, nil
	}
	if !collector.reworkReviewResultKnownV0(
		payload.ReviewResultRef,
		payload.ReviewRequestID,
		payload.DeliveryRef,
	) {
		collector.addWaitingV0(SchedulerWaitingCandidateMissingV0)
		return true, nil
	}
	command, err := orquestacoreworkflow.NewRequestReworkCommandV0(candidate.CommandMeta, payload)
	if err != nil {
		return true, err
	}
	collector.addReadyCommandV0(command)
	collector.plannedReworkRequests[payload.ReworkRequestRef] = true
	return true, nil
}

func (collector *schedulerTickCollectorV0) reviewResultKnownV0(reviewResultRef string) bool {
	reviewResultRef = strings.TrimSpace(reviewResultRef)
	for _, projection := range collector.reviewResults {
		if schedulerProjectionHasResultRefV0(projection, reviewResultRef) {
			return true
		}
	}
	return false
}

func (collector *schedulerTickCollectorV0) acceptedReviewResultKnownV0(
	reviewRequestID string,
	deliveryRef string,
) bool {
	reviewRequestID = strings.TrimSpace(reviewRequestID)
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, projection := range collector.reviewResults {
		status, request, delivery, ok := schedulerProjectionReviewFieldsV0(projection)
		if ok &&
			status == string(orquestacoreworkflow.ReviewResultStatusAcceptedV0) &&
			request == reviewRequestID &&
			delivery == deliveryRef {
			return true
		}
	}
	return false
}

func (collector *schedulerTickCollectorV0) reworkReviewResultKnownV0(
	reviewResultRef string,
	reviewRequestID string,
	deliveryRef string,
) bool {
	reviewResultRef = strings.TrimSpace(reviewResultRef)
	reviewRequestID = strings.TrimSpace(reviewRequestID)
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, projection := range collector.reviewResults {
		if !schedulerProjectionHasResultRefV0(projection, reviewResultRef) {
			continue
		}
		status, request, delivery, ok := schedulerProjectionReviewFieldsV0(projection)
		if ok && schedulerReviewStatusSupportsReworkV0(status) &&
			request == reviewRequestID && delivery == deliveryRef {
			return true
		}
	}
	return false
}

func (collector *schedulerTickCollectorV0) reworkRequestKnownV0(
	reworkRequestRef string,
) bool {
	reworkRequestRef = strings.TrimSpace(reworkRequestRef)
	for _, projection := range collector.reworkRequests {
		if projection == reworkRequestRef ||
			strings.HasPrefix(projection, reworkRequestRef+schedulerReviewResultStatusSepV0) {
			return true
		}
	}
	return false
}

func schedulerProjectionHasResultRefV0(projection string, reviewResultRef string) bool {
	projection = strings.TrimSpace(projection)
	return projection == reviewResultRef ||
		strings.HasPrefix(projection, reviewResultRef+schedulerReviewResultStatusSepV0)
}

func schedulerProjectionReviewFieldsV0(projection string) (string, string, string, bool) {
	_, tail, ok := strings.Cut(strings.TrimSpace(projection), schedulerReviewResultStatusSepV0)
	if !ok {
		return "", "", "", false
	}
	status, tail, ok := strings.Cut(tail, schedulerReviewResultRequestSepV0)
	if !ok {
		return "", "", "", false
	}
	request, delivery, ok := strings.Cut(tail, schedulerReviewResultDeliverySepV0)
	if !ok {
		return "", "", "", false
	}
	return strings.TrimSpace(status), strings.TrimSpace(request), strings.TrimSpace(delivery), true
}

func schedulerReviewStatusSupportsReworkV0(status string) bool {
	status = strings.TrimSpace(status)
	return status == string(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0) ||
		status == string(orquestacoreworkflow.ReviewResultStatusRejectedV0)
}
