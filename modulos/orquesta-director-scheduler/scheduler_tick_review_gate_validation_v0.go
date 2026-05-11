package orquestadirectorscheduler

import "strings"

func validateSchedulerReviewGateCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.ReviewGateCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("review_gate_candidates")
	}
	for _, candidate := range input.ReviewGateCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("review_gate_candidates.candidate_ref")
		}
		if candidate.RequestReview == nil && candidate.RecordReviewResult == nil &&
			candidate.AcceptReview == nil && candidate.RequestRework == nil {
			return schedulerTickErrorV0("review_gate_candidates")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) {
			return schedulerTickErrorV0("review_gate_candidates.evidence_refs")
		}
		if schedulerReviewGatePayloadRefsInvalidV0(candidate) {
			return schedulerTickErrorV0("review_gate_candidates.evidence_refs")
		}
		if err := validateSchedulerReviewGateRunRefsV0(input.RunRef, candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateSchedulerReviewGateRunRefsV0(
	runRef string,
	candidate SchedulableReviewGateCandidateV0,
) error {
	if candidate.RequestReview != nil && candidate.RequestReview.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("review_gate_candidates.request_review.command_meta.run_id")
	}
	if candidate.RecordReviewResult != nil && candidate.RecordReviewResult.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("review_gate_candidates.record_review_result.command_meta.run_id")
	}
	if candidate.AcceptReview != nil && candidate.AcceptReview.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("review_gate_candidates.accept_review.command_meta.run_id")
	}
	if candidate.RequestRework != nil && candidate.RequestRework.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("review_gate_candidates.request_rework.command_meta.run_id")
	}
	return nil
}

func schedulerReviewGatePayloadRefsInvalidV0(candidate SchedulableReviewGateCandidateV0) bool {
	return schedulerRequestReviewRefsInvalidV0(candidate.RequestReview) ||
		schedulerRecordReviewResultRefsInvalidV0(candidate.RecordReviewResult) ||
		schedulerAcceptReviewRefsInvalidV0(candidate.AcceptReview) ||
		schedulerRequestReworkRefsInvalidV0(candidate.RequestRework)
}

func schedulerRequestReviewRefsInvalidV0(candidate *SchedulerRequestReviewCandidateV0) bool {
	return candidate != nil && schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs)
}

func schedulerRecordReviewResultRefsInvalidV0(candidate *SchedulerRecordReviewResultCandidateV0) bool {
	return candidate != nil && schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs)
}

func schedulerAcceptReviewRefsInvalidV0(candidate *SchedulerAcceptReviewCandidateV0) bool {
	return candidate != nil && schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs)
}

func schedulerRequestReworkRefsInvalidV0(candidate *SchedulerRequestReworkCandidateV0) bool {
	return candidate != nil && schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs)
}

func schedulerReviewGateCandidatesTextFieldsV0(
	candidates []SchedulableReviewGateCandidateV0,
) []string {
	var values []string
	for _, candidate := range candidates {
		values = append(values, schedulerRequestReviewTextFieldsV0(candidate.RequestReview)...)
		values = append(values, schedulerRecordReviewResultTextFieldsV0(candidate.RecordReviewResult)...)
		values = append(values, schedulerAcceptReviewTextFieldsV0(candidate.AcceptReview)...)
		values = append(values, schedulerRequestReworkTextFieldsV0(candidate.RequestRework)...)
	}
	return values
}

func schedulerRequestReviewTextFieldsV0(candidate *SchedulerRequestReviewCandidateV0) []string {
	if candidate == nil {
		return nil
	}
	values := []string{
		candidate.Payload.Summary,
	}
	return values
}

func schedulerRecordReviewResultTextFieldsV0(candidate *SchedulerRecordReviewResultCandidateV0) []string {
	if candidate == nil {
		return nil
	}
	values := []string{
		string(candidate.Payload.Status),
		candidate.Payload.Summary,
	}
	return values
}

func schedulerAcceptReviewTextFieldsV0(candidate *SchedulerAcceptReviewCandidateV0) []string {
	if candidate == nil {
		return nil
	}
	values := []string{
		candidate.Payload.Summary,
	}
	return values
}

func schedulerRequestReworkTextFieldsV0(candidate *SchedulerRequestReworkCandidateV0) []string {
	if candidate == nil {
		return nil
	}
	values := []string{
		candidate.Payload.Summary,
	}
	return values
}
