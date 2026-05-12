package orquestadirectortickinput

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func compactTickInputPayloadV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	for schedulerInputPayloadTooLargeV0(input) && len(input.WorkCandidates) > 1 {
		input.WorkCandidates = input.WorkCandidates[:len(input.WorkCandidates)-1]
		input.WorkClaims = compactTickInputWorkClaimsV0(input)
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
