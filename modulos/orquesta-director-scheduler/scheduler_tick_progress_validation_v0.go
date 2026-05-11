package orquestadirectorscheduler

import "strings"

func validateSchedulerProgressSupervisionCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.ProgressSupervisionCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("progress_supervision_candidates")
	}
	for _, candidate := range input.ProgressSupervisionCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("progress_supervision_candidates.candidate_ref")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) ||
			schedulerRefsInvalidV0(candidate.SupervisionInput.Report.EvidenceRefs) {
			return schedulerTickErrorV0("progress_supervision_candidates.evidence_refs")
		}
		if err := validateSchedulerProgressSupervisionRunRefsV0(input.RunRef, candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateSchedulerProgressSupervisionRunRefsV0(
	runRef string,
	candidate SchedulableProgressSupervisionCandidateV0,
) error {
	supervision := candidate.SupervisionInput
	if strings.TrimSpace(supervision.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("progress_supervision_candidates.command_meta.run_id")
	}
	if strings.TrimSpace(supervision.Report.RunID) != runRef {
		return schedulerTickErrorV0("progress_supervision_candidates.report.run_id")
	}
	return nil
}

func schedulerProgressSupervisionCandidatesTextFieldsV0(
	candidates []SchedulableProgressSupervisionCandidateV0,
) []string {
	var values []string
	for _, candidate := range candidates {
		supervision := candidate.SupervisionInput
		values = append(values,
			string(supervision.Report.Status),
			supervision.Report.Summary,
		)
	}
	return values
}
