package orquestacionnucleoapp

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const defaultWorkflowTaskCandidateFrontierLimitV0 = 0

func workflowTaskCandidateFrontierV0(
	activeClaims []orquestacoreconcurrency.WorksetClaimV0,
	pending []orquestadirectorscheduler.SchedulableWorkCandidateV0,
	limit int,
) []orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	selected := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, len(pending))
	claims := workflowTaskActiveClaimsV0(nil, activeClaims)
	for _, candidate := range pending {
		if limit > 0 && len(selected) >= limit {
			break
		}
		if workflowTaskClaimsConflictV0(candidate.Claims, claims) {
			continue
		}
		selected = append(selected, candidate)
		claims = workflowTaskActiveClaimsV0(claims, candidate.Claims)
	}
	return selected
}

func workflowTaskClaimsForSchedulerV0(
	activeClaims []orquestacoreconcurrency.WorksetClaimV0,
	frontier []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	claims := workflowTaskActiveClaimsV0(nil, activeClaims)
	for _, candidate := range frontier {
		claims = workflowTaskActiveClaimsV0(claims, candidate.Claims)
	}
	return claims
}

func workflowTaskActiveClaimsV0(
	base []orquestacoreconcurrency.WorksetClaimV0,
	next []orquestacoreconcurrency.WorksetClaimV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	if len(next) == 0 {
		return base
	}
	out := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(base)+len(next))
	out = append(out, base...)
	out = append(out, next...)
	return out
}

func workflowTaskClaimsConflictV0(
	left []orquestacoreconcurrency.WorksetClaimV0,
	right []orquestacoreconcurrency.WorksetClaimV0,
) bool {
	for _, leftClaim := range left {
		for _, rightClaim := range right {
			if len(orquestacoreconcurrency.DetectWorksetConflictsV0([]orquestacoreconcurrency.WorksetClaimV0{
				leftClaim,
				rightClaim,
			})) > 0 {
				return true
			}
		}
	}
	return false
}
