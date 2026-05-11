package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeSchedulableReviewGateCandidatesV0(
	candidates []SchedulableReviewGateCandidateV0,
) []SchedulableReviewGateCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableReviewGateCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeSchedulableReviewGateCandidateV0(candidate))
	}
	return normalized
}

func normalizeSchedulableReviewGateCandidateV0(
	candidate SchedulableReviewGateCandidateV0,
) SchedulableReviewGateCandidateV0 {
	return SchedulableReviewGateCandidateV0{
		CandidateRef:       strings.TrimSpace(candidate.CandidateRef),
		RequestReview:      normalizeSchedulerRequestReviewCandidateV0(candidate.RequestReview),
		RecordReviewResult: normalizeSchedulerRecordReviewResultCandidateV0(candidate.RecordReviewResult),
		AcceptReview:       normalizeSchedulerAcceptReviewCandidateV0(candidate.AcceptReview),
		RequestRework:      normalizeSchedulerRequestReworkCandidateV0(candidate.RequestRework),
		EvidenceRefs:       compactSchedulerStringsV0(candidate.EvidenceRefs),
	}
}

func normalizeSchedulerRequestReviewCandidateV0(
	candidate *SchedulerRequestReviewCandidateV0,
) *SchedulerRequestReviewCandidateV0 {
	if candidate == nil {
		return nil
	}
	payload := candidate.Payload
	payload.ReviewRequestID = strings.TrimSpace(payload.ReviewRequestID)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.DeliveryRef = strings.TrimSpace(payload.DeliveryRef)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactSchedulerStringsV0(payload.EvidenceRefs)
	return &SchedulerRequestReviewCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     payload,
	}
}

func normalizeSchedulerRecordReviewResultCandidateV0(
	candidate *SchedulerRecordReviewResultCandidateV0,
) *SchedulerRecordReviewResultCandidateV0 {
	if candidate == nil {
		return nil
	}
	return &SchedulerRecordReviewResultCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     orquestacoreworkflow.NormalizeReviewResultV0(candidate.Payload),
	}
}

func normalizeSchedulerAcceptReviewCandidateV0(
	candidate *SchedulerAcceptReviewCandidateV0,
) *SchedulerAcceptReviewCandidateV0 {
	if candidate == nil {
		return nil
	}
	payload := candidate.Payload
	payload.AcceptedReviewRef = strings.TrimSpace(payload.AcceptedReviewRef)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.ReviewRequestID = strings.TrimSpace(payload.ReviewRequestID)
	payload.DeliveryRef = strings.TrimSpace(payload.DeliveryRef)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactSchedulerStringsV0(payload.EvidenceRefs)
	return &SchedulerAcceptReviewCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     payload,
	}
}

func normalizeSchedulerRequestReworkCandidateV0(
	candidate *SchedulerRequestReworkCandidateV0,
) *SchedulerRequestReworkCandidateV0 {
	if candidate == nil {
		return nil
	}
	payload := candidate.Payload
	payload.ReworkRequestRef = strings.TrimSpace(payload.ReworkRequestRef)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.ReviewResultRef = strings.TrimSpace(payload.ReviewResultRef)
	payload.ReviewRequestID = strings.TrimSpace(payload.ReviewRequestID)
	payload.DeliveryRef = strings.TrimSpace(payload.DeliveryRef)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactSchedulerStringsV0(payload.EvidenceRefs)
	return &SchedulerRequestReworkCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     payload,
	}
}
