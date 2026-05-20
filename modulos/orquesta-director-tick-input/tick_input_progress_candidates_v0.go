package orquestadirectortickinput

import (
	"strings"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func tickInputProgressCandidatesForRunV0(
	candidates []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0,
	runRef string,
) []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || len(candidates) == 0 {
		return candidates
	}
	filtered := make([]orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		reportRunRef := strings.TrimSpace(candidate.SupervisionInput.Report.RunID)
		commandRunRef := strings.TrimSpace(candidate.SupervisionInput.CommandMeta.RunID)
		if commandRunRef != runRef || reportRunRef != runRef {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}
