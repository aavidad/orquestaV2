package orquestadirectortickinput

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const tickInputPayloadPressureKeepRefsV0 = 64

func compactTickInputPayloadV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	for schedulerInputPayloadTooLargeV0(input) && len(input.WorkCandidates) > 1 {
		input.WorkCandidates = input.WorkCandidates[:len(input.WorkCandidates)-1]
		input.WorkClaims = compactTickInputWorkClaimsV0(input)
		input = orquestadirectorscheduler.NormalizeDirectorSchedulerTickInputV0(input)
	}
	if schedulerInputPayloadTooLargeV0(input) {
		input = compactTickInputSnapshotForPayloadPressureV0(input)
		input = orquestadirectorscheduler.NormalizeDirectorSchedulerTickInputV0(input)
	}
	return input
}

func schedulerInputPayloadTooLargeV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) bool {
	err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(input)
	return tickInputSchedulerErrorFieldV0(err) == "payload"
}

func compactTickInputWorkClaimsV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	candidateClaims := tickInputCandidateClaimRefsV0(input.WorkCandidates)
	activeAgents := tickInputActiveAgentRefsV0(input.Snapshot)
	out := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(input.WorkClaims))
	for _, claim := range input.WorkClaims {
		if candidateClaims[claim.ClaimRef] || activeAgents[claim.AgentRequestID] {
			out = append(out, claim)
		}
	}
	return out
}

func tickInputCandidateClaimRefsV0(
	candidates []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) map[string]bool {
	refs := map[string]bool{}
	for _, candidate := range candidates {
		for _, ref := range candidate.SubjectClaimRefs {
			if ref != "" {
				refs[ref] = true
			}
		}
		for _, claim := range candidate.Claims {
			if claim.ClaimRef != "" {
				refs[claim.ClaimRef] = true
			}
		}
	}
	return refs
}

func tickInputActiveAgentRefsV0(
	snapshot orquestadirectorscheduler.RunSchedulingSnapshotV0,
) map[string]bool {
	refs := map[string]bool{}
	for _, ref := range append(snapshot.Agents, snapshot.StartedAgents...) {
		if ref != "" {
			refs[ref] = true
		}
	}
	return refs
}

func compactTickInputSnapshotForPayloadPressureV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	keep := collectTickInputPayloadPressureRefsV0(input)
	snapshot := input.Snapshot
	snapshot.Tasks = compactTickInputRefsForPayloadPressureV0(snapshot.Tasks, keep.tasks)
	snapshot.CapacityRequests = compactTickInputRefsForPayloadPressureV0(snapshot.CapacityRequests, keep.capacityRequests)
	snapshot.CapacityDecisions = compactTickInputRefsForPayloadPressureV0(snapshot.CapacityDecisions, nil)
	snapshot.ConcurrencyGates = compactTickInputRefsForPayloadPressureV0(snapshot.ConcurrencyGates, nil)
	snapshot.Agents = compactTickInputRefsForPayloadPressureV0(snapshot.Agents, keep.agents)
	snapshot.StartedAgents = compactTickInputRefsForPayloadPressureV0(snapshot.StartedAgents, keep.agents)
	snapshot.FailedAgents = compactTickInputRefsForPayloadPressureV0(snapshot.FailedAgents, keep.agents)
	snapshot.LostAgents = compactTickInputRefsForPayloadPressureV0(snapshot.LostAgents, keep.agents)
	snapshot.StoppedAgents = compactTickInputRefsForPayloadPressureV0(snapshot.StoppedAgents, keep.agents)
	snapshot.ConfirmedStoppedAgents = compactTickInputRefsForPayloadPressureV0(snapshot.ConfirmedStoppedAgents, keep.agents)
	snapshot.PhaseArtifacts = compactTickInputRefsForPayloadPressureV0(snapshot.PhaseArtifacts, keep.artifacts)
	snapshot.Deliveries = compactTickInputRefsForPayloadPressureV0(snapshot.Deliveries, keep.deliveries)
	snapshot.Reviews = compactTickInputRefsForPayloadPressureV0(snapshot.Reviews, keep.reviews)
	snapshot.ReviewResults = compactTickInputRefsForPayloadPressureV0(snapshot.ReviewResults, keep.reviewResults)
	snapshot.AcceptedReviews = compactTickInputRefsForPayloadPressureV0(snapshot.AcceptedReviews, keep.acceptedReviews)
	snapshot.ReworkRequests = compactTickInputRefsForPayloadPressureV0(snapshot.ReworkRequests, keep.reworkRequests)
	snapshot.AgentAssessments = compactTickInputRefsForPayloadPressureV0(snapshot.AgentAssessments, keep.assessments)
	snapshot.DirectorQuestions = compactTickInputRefsForPayloadPressureV0(snapshot.DirectorQuestions, keep.questions)
	snapshot.DirectorAnsweredQuestions = compactTickInputRefsForPayloadPressureV0(
		snapshot.DirectorAnsweredQuestions,
		keep.questions,
	)
	snapshot.ExpiredLeaseRefs = compactTickInputRefsForPayloadPressureV0(snapshot.ExpiredLeaseRefs, nil)
	snapshot.ReplanRefs = compactTickInputRefsForPayloadPressureV0(snapshot.ReplanRefs, keep.replans)
	snapshot.BlockingQualityGateRefs = compactTickInputRefsForPayloadPressureV0(snapshot.BlockingQualityGateRefs, nil)
	snapshot.PendingOutboxRefs = compactTickInputRefsForPayloadPressureV0(snapshot.PendingOutboxRefs, nil)
	input.Snapshot = snapshot
	return input
}

type tickInputPayloadPressureRefsV0 struct {
	tasks            map[string]bool
	capacityRequests map[string]bool
	agents           map[string]bool
	artifacts        map[string]bool
	deliveries       map[string]bool
	reviews          map[string]bool
	reviewResults    map[string]bool
	acceptedReviews  map[string]bool
	reworkRequests   map[string]bool
	assessments      map[string]bool
	questions        map[string]bool
	replans          map[string]bool
}

func collectTickInputPayloadPressureRefsV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) tickInputPayloadPressureRefsV0 {
	refs := tickInputPayloadPressureRefsV0{
		tasks:            map[string]bool{},
		capacityRequests: map[string]bool{},
		agents:           map[string]bool{},
		artifacts:        map[string]bool{},
		deliveries:       map[string]bool{},
		reviews:          map[string]bool{},
		reviewResults:    map[string]bool{},
		acceptedReviews:  map[string]bool{},
		reworkRequests:   map[string]bool{},
		assessments:      map[string]bool{},
		questions:        map[string]bool{},
		replans:          map[string]bool{},
	}
	for _, candidate := range input.WorkCandidates {
		collectTickInputWorkCandidateRefsV0(refs, candidate)
	}
	for _, agentRef := range tickInputInFlightAgentRefsForPayloadPressureV0(input.Snapshot) {
		refs.agents[agentRef] = true
	}
	for _, candidate := range input.PhaseArtifactCandidates {
		refs.artifacts[candidate.Payload.ArtifactRef] = true
		refs.agents[candidate.Payload.AgentRef] = true
	}
	for _, candidate := range input.DeliveryCandidates {
		refs.deliveries[candidate.Payload.DeliveryRef] = true
		refs.tasks[candidate.Payload.TaskID] = true
		refs.agents[candidate.Payload.AgentRef] = true
	}
	for _, candidate := range input.ReviewGateCandidates {
		collectTickInputReviewGateRefsForPayloadV0(refs, candidate)
	}
	for _, candidate := range input.ProgressSupervisionCandidates {
		supervision := candidate.SupervisionInput
		refs.agents[supervision.Report.AgentRequestID] = true
		refs.tasks[supervision.TaskRef] = true
		refs.deliveries[supervision.DeliveryRef] = true
		refs.assessments[supervision.AssessmentRef] = true
		refs.questions[supervision.QuestionID] = true
	}
	for _, candidate := range input.ReplanFollowupCandidates {
		collectTickInputReplanRefsForPayloadV0(refs, candidate)
	}
	return refs
}

func tickInputInFlightAgentRefsForPayloadPressureV0(
	snapshot orquestadirectorscheduler.RunSchedulingSnapshotV0,
) []string {
	failed := tickInputStringSetForPayloadPressureV0(snapshot.FailedAgents)
	stopped := tickInputStringSetForPayloadPressureV0(snapshot.StoppedAgents)
	refs := make([]string, 0, len(snapshot.StartedAgents))
	for _, agentRef := range compactTickInputRefsV0(snapshot.StartedAgents) {
		if failed[agentRef] || stopped[agentRef] {
			continue
		}
		refs = append(refs, agentRef)
	}
	return compactTickInputRefsV0(refs)
}

func tickInputStringSetForPayloadPressureV0(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range compactTickInputRefsV0(values) {
		set[value] = true
	}
	return set
}

func collectTickInputWorkCandidateRefsV0(
	refs tickInputPayloadPressureRefsV0,
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) {
	for _, claim := range candidate.Claims {
		refs.tasks[claim.TaskRef] = true
		refs.agents[claim.AgentRequestID] = true
	}
	if candidate.CapacityCandidate != nil {
		refs.capacityRequests[candidate.CapacityCandidate.Payload.CapacityRequestID] = true
		refs.tasks[candidate.CapacityCandidate.Payload.TaskRef] = true
	}
	if candidate.AgentCandidate != nil {
		refs.capacityRequests[candidate.AgentCandidate.Payload.CapacityRequestRef] = true
		refs.tasks[candidate.AgentCandidate.Payload.TaskRef] = true
		refs.agents[candidate.AgentCandidate.Payload.AgentRequestID] = true
	}
}

func collectTickInputReviewGateRefsForPayloadV0(
	refs tickInputPayloadPressureRefsV0,
	candidate orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) {
	if candidate.RequestReview != nil {
		refs.deliveries[candidate.RequestReview.Payload.DeliveryRef] = true
		refs.reviews[candidate.RequestReview.Payload.ReviewRequestID] = true
	}
	if candidate.RecordReviewResult != nil {
		refs.deliveries[candidate.RecordReviewResult.Payload.DeliveryRef] = true
		refs.reviews[candidate.RecordReviewResult.Payload.ReviewRequestID] = true
		refs.reviewResults[candidate.RecordReviewResult.Payload.ReviewResultRef] = true
	}
	if candidate.AcceptReview != nil {
		refs.deliveries[candidate.AcceptReview.Payload.DeliveryRef] = true
		refs.reviews[candidate.AcceptReview.Payload.ReviewRequestID] = true
		refs.acceptedReviews[candidate.AcceptReview.Payload.AcceptedReviewRef] = true
	}
	if candidate.RequestRework != nil {
		refs.deliveries[candidate.RequestRework.Payload.DeliveryRef] = true
		refs.reviews[candidate.RequestRework.Payload.ReviewRequestID] = true
		refs.reviewResults[candidate.RequestRework.Payload.ReviewResultRef] = true
		refs.reworkRequests[candidate.RequestRework.Payload.ReworkRequestRef] = true
	}
}

func collectTickInputReplanRefsForPayloadV0(
	refs tickInputPayloadPressureRefsV0,
	candidate orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0,
) {
	input := candidate.ReplanFollowupsInput
	refs.replans[input.DecisionPayload.ReplanRef] = true
	refs.tasks[input.DecisionPayload.TaskRef] = true
	if input.CapacityCandidate != nil {
		refs.capacityRequests[input.CapacityCandidate.Payload.CapacityRequestID] = true
	}
	if input.AgentCandidate != nil {
		refs.capacityRequests[input.AgentCandidate.Payload.CapacityRequestRef] = true
		refs.agents[input.AgentCandidate.Payload.AgentRequestID] = true
	}
	if input.AskDirectorCandidate != nil {
		refs.questions[input.AskDirectorCandidate.Payload.QuestionID] = true
	}
}

func compactTickInputRefsForPayloadPressureV0(values []string, keep map[string]bool) []string {
	values = compactTickInputRefsV0(values)
	if len(values) <= tickInputPayloadPressureKeepRefsV0 {
		return values
	}
	out := make([]string, 0, tickInputPayloadPressureKeepRefsV0+len(keep))
	seen := map[string]bool{}
	for _, value := range values {
		if keep != nil && keep[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	start := len(values) - tickInputPayloadPressureKeepRefsV0
	if start < 0 {
		start = 0
	}
	for _, value := range values[start:] {
		if !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	return compactTickInputRefsV0(out)
}
