package orquestadirectorscheduler

import "strings"

func validateSchedulerDeliveryCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.DeliveryCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("delivery_candidates")
	}
	for _, candidate := range input.DeliveryCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("delivery_candidates.candidate_ref")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) ||
			schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs) {
			return schedulerTickErrorV0("delivery_candidates.evidence_refs")
		}
		if strings.TrimSpace(candidate.CommandMeta.RunID) != input.RunRef {
			return schedulerTickErrorV0("delivery_candidates.command_meta.run_id")
		}
	}
	return nil
}

func schedulerDeliveryCandidatesTextFieldsV0(
	candidates []SchedulableDeliveryCandidateV0,
) []string {
	var values []string
	for _, candidate := range candidates {
		values = append(values,
			candidate.Payload.Summary,
		)
	}
	return values
}
