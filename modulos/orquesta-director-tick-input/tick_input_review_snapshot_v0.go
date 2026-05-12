package orquestadirectortickinput

import (
	"strings"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	tickInputReviewResultSeparatorV0   = "#review_result:"
	tickInputReviewRequestSeparatorV0  = "#review_request:"
	tickInputReviewDeliverySeparatorV0 = "#delivery:"
)

func compactTickInputSnapshotForReviewGateV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	refs := collectTickInputReviewGateRefsV0(input.ReviewGateCandidates)
	snapshot := input.Snapshot
	snapshot.Tasks = nil
	snapshot.CapacityRequests = nil
	snapshot.CapacityDecisions = nil
	snapshot.ConcurrencyGates = nil
	snapshot.Agents = nil
	snapshot.StartedAgents = nil
	snapshot.FailedAgents = nil
	snapshot.StoppedAgents = nil
	snapshot.PhaseArtifacts = nil
	snapshot.Deliveries = filterTickInputRefsByExactV0(snapshot.Deliveries, refs.Deliveries)
	snapshot.Reviews = filterTickInputRefsByExactV0(snapshot.Reviews, refs.Reviews)
	snapshot.ReviewResults = filterTickInputReviewResultsV0(snapshot.ReviewResults, refs)
	snapshot.AcceptedReviews = filterTickInputRefsByExactV0(snapshot.AcceptedReviews, refs.Accepted)
	snapshot.ReworkRequests = filterTickInputReworkRequestsV0(snapshot.ReworkRequests, refs)
	snapshot.AgentAssessments = nil
	snapshot.DirectorQuestions = nil
	snapshot.DirectorAnsweredQuestions = nil
	snapshot.ExpiredLeaseRefs = nil
	snapshot.ReplanRefs = nil
	snapshot.BlockingQualityGateRefs = nil
	input.Snapshot = snapshot
	return input
}

type tickInputReviewGateRefsV0 struct {
	Deliveries map[string]bool
	Reviews    map[string]bool
	Results    map[string]bool
	Accepted   map[string]bool
	Reworks    map[string]bool
}

func collectTickInputReviewGateRefsV0(
	candidates []orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) tickInputReviewGateRefsV0 {
	refs := tickInputReviewGateRefsV0{
		Deliveries: map[string]bool{},
		Reviews:    map[string]bool{},
		Results:    map[string]bool{},
		Accepted:   map[string]bool{},
		Reworks:    map[string]bool{},
	}
	for _, candidate := range candidates {
		tickInputCollectRequestReviewRefsV0(refs, candidate.RequestReview)
		tickInputCollectRecordReviewRefsV0(refs, candidate.RecordReviewResult)
		tickInputCollectAcceptReviewRefsV0(refs, candidate.AcceptReview)
		tickInputCollectRequestReworkRefsV0(refs, candidate.RequestRework)
	}
	return refs
}

func tickInputCollectRequestReviewRefsV0(
	refs tickInputReviewGateRefsV0,
	candidate *orquestadirectorscheduler.SchedulerRequestReviewCandidateV0,
) {
	if candidate == nil {
		return
	}
	refs.Deliveries[strings.TrimSpace(candidate.Payload.DeliveryRef)] = true
	refs.Reviews[strings.TrimSpace(candidate.Payload.ReviewRequestID)] = true
}

func tickInputCollectRecordReviewRefsV0(
	refs tickInputReviewGateRefsV0,
	candidate *orquestadirectorscheduler.SchedulerRecordReviewResultCandidateV0,
) {
	if candidate == nil {
		return
	}
	refs.Deliveries[strings.TrimSpace(candidate.Payload.DeliveryRef)] = true
	refs.Reviews[strings.TrimSpace(candidate.Payload.ReviewRequestID)] = true
	refs.Results[strings.TrimSpace(candidate.Payload.ReviewResultRef)] = true
}

func tickInputCollectAcceptReviewRefsV0(
	refs tickInputReviewGateRefsV0,
	candidate *orquestadirectorscheduler.SchedulerAcceptReviewCandidateV0,
) {
	if candidate == nil {
		return
	}
	refs.Deliveries[strings.TrimSpace(candidate.Payload.DeliveryRef)] = true
	refs.Reviews[strings.TrimSpace(candidate.Payload.ReviewRequestID)] = true
	refs.Accepted[strings.TrimSpace(candidate.Payload.AcceptedReviewRef)] = true
}

func tickInputCollectRequestReworkRefsV0(
	refs tickInputReviewGateRefsV0,
	candidate *orquestadirectorscheduler.SchedulerRequestReworkCandidateV0,
) {
	if candidate == nil {
		return
	}
	refs.Deliveries[strings.TrimSpace(candidate.Payload.DeliveryRef)] = true
	refs.Reviews[strings.TrimSpace(candidate.Payload.ReviewRequestID)] = true
	refs.Results[strings.TrimSpace(candidate.Payload.ReviewResultRef)] = true
	refs.Reworks[strings.TrimSpace(candidate.Payload.ReworkRequestRef)] = true
}

func filterTickInputReviewResultsV0(
	values []string,
	refs tickInputReviewGateRefsV0,
) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		resultRef, reviewRef, deliveryRef, ok := tickInputReviewResultProjectionV0(value)
		if !ok {
			continue
		}
		if refs.Results[resultRef] ||
			(refs.Reviews[reviewRef] && refs.Deliveries[deliveryRef]) {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return compactTickInputRefsV0(out)
}

func filterTickInputReworkRequestsV0(
	values []string,
	refs tickInputReviewGateRefsV0,
) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		reworkRef, resultRef, reviewRef, deliveryRef, ok := tickInputReworkProjectionV0(value)
		if !ok {
			continue
		}
		if refs.Reworks[reworkRef] ||
			(refs.Results[resultRef] && refs.Reviews[reviewRef] && refs.Deliveries[deliveryRef]) {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return compactTickInputRefsV0(out)
}

func tickInputReviewResultProjectionV0(value string) (string, string, string, bool) {
	resultRef, tail, ok := strings.Cut(strings.TrimSpace(value), tickInputReviewResultSeparatorV0)
	if !ok {
		return "", "", "", false
	}
	_, tail, ok = strings.Cut(tail, tickInputReviewRequestSeparatorV0)
	if !ok {
		return "", "", "", false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, tickInputReviewDeliverySeparatorV0)
	if !ok {
		return "", "", "", false
	}
	resultRef = strings.TrimSpace(resultRef)
	reviewRef = strings.TrimSpace(reviewRef)
	deliveryRef = strings.TrimSpace(deliveryRef)
	return resultRef, reviewRef, deliveryRef, resultRef != "" && reviewRef != "" && deliveryRef != ""
}

func tickInputReworkProjectionV0(value string) (string, string, string, string, bool) {
	reworkRef, tail, ok := strings.Cut(strings.TrimSpace(value), tickInputReviewResultSeparatorV0)
	if !ok {
		return "", "", "", "", false
	}
	resultRef, tail, ok := strings.Cut(tail, tickInputReviewRequestSeparatorV0)
	if !ok {
		return "", "", "", "", false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, tickInputReviewDeliverySeparatorV0)
	if !ok {
		return "", "", "", "", false
	}
	reworkRef = strings.TrimSpace(reworkRef)
	resultRef = strings.TrimSpace(resultRef)
	reviewRef = strings.TrimSpace(reviewRef)
	deliveryRef = strings.TrimSpace(deliveryRef)
	return reworkRef, resultRef, reviewRef, deliveryRef,
		reworkRef != "" && resultRef != "" && reviewRef != "" && deliveryRef != ""
}
