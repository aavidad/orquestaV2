package orquestadirectorscheduler

import "strings"

func validateSchedulerPhaseArtifactCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.PhaseArtifactCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("phase_artifact_candidates")
	}
	for _, candidate := range input.PhaseArtifactCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("phase_artifact_candidates.candidate_ref")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) ||
			schedulerRefsInvalidV0(candidate.Payload.EvidenceRefs) {
			return schedulerTickErrorV0("phase_artifact_candidates.evidence_refs")
		}
		if strings.TrimSpace(candidate.CommandMeta.RunID) != input.RunRef {
			return schedulerTickErrorV0("phase_artifact_candidates.command_meta.run_id")
		}
	}
	return nil
}

func schedulerPhaseArtifactCandidatesTextFieldsV0(
	candidates []SchedulablePhaseArtifactCandidateV0,
) []string {
	var values []string
	for _, candidate := range candidates {
		values = append(values,
			candidate.Payload.Summary,
		)
	}
	return values
}
