package orquestadirectorscheduler

import "strings"

func validateSchedulerLeaseCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.LeaseActionCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("lease_action_candidates")
	}
	for _, candidate := range input.LeaseActionCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("lease_action_candidates.candidate_ref")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) ||
			schedulerRefsInvalidV0(candidate.PostLeaseActionInput.EvidenceRefs) {
			return schedulerTickErrorV0("lease_action_candidates.evidence_refs")
		}
		if err := validateSchedulerLeaseCandidateRunRefsV0(input.RunRef, candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateSchedulerLeaseCandidateRunRefsV0(
	runRef string,
	candidate SchedulableLeaseActionCandidateV0,
) error {
	actionInput := candidate.PostLeaseActionInput
	if strings.TrimSpace(actionInput.RunRef) != runRef {
		return schedulerTickErrorV0("lease_action_candidates.post_lease_action_input.run_ref")
	}
	if strings.TrimSpace(actionInput.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("lease_action_candidates.post_lease_action_input.command_meta.run_id")
	}
	return nil
}

func schedulerLeaseCandidatesTextFieldsV0(candidates []SchedulableLeaseActionCandidateV0) []string {
	var values []string
	for _, candidate := range candidates {
		input := candidate.PostLeaseActionInput
		values = append(values,
			input.ReasonCode,
			string(input.RecommendedAction),
		)
	}
	return values
}
