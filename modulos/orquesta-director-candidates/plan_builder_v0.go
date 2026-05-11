package orquestadirectorcandidates

import (
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func BuildSchedulableWorkCandidatesFromPlanV0(
	input CompactBacklogPlanCandidatesInputV0,
) ([]orquestadirectorscheduler.SchedulableWorkCandidateV0, error) {
	normalized := normalizePlanInputV0(input)
	if err := validatePlanInputV0(normalized); err != nil {
		return nil, err
	}
	candidates := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, len(normalized.WorkItems))
	for _, item := range normalized.WorkItems {
		candidate, err := BuildSchedulableWorkCandidateV0(candidateInputFromPlanItemV0(normalized, item))
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func candidateInputFromPlanItemV0(
	input CompactBacklogPlanCandidatesInputV0,
	item CompactBacklogPlanWorkItemV0,
) SchedulableWorkCandidateInputV0 {
	return SchedulableWorkCandidateInputV0{
		CandidateRef:     item.CandidateRef,
		RunRef:           input.RunRef,
		PhaseID:          input.PhaseID,
		TaskRef:          item.TaskRef,
		SubjectClaimRefs: item.SubjectClaimRefs,
		ScopeClaims:      item.ScopeClaims,
		Commands:         item.Commands,
		Capacity:         item.Capacity,
		Agent:            item.Agent,
		GateEvidenceRefs: item.GateEvidenceRefs,
		EvidenceRefs:     item.EvidenceRefs,
	}
}
