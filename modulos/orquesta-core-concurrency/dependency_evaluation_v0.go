package orquestacoreconcurrency

import "sort"

func EvaluateWorksetDependenciesV0(claims []WorksetClaimV0) WorksetDependencyEvaluationV0 {
	normalized := normalizeClaimsForDependencyV0(claims)
	claimByRef := indexClaimsByRefV0(normalized)
	issues := make([]WorksetDependencyIssueV0, 0)

	issues = append(issues, detectDirectDependencyIssuesV0(normalized, claimByRef)...)
	issues = append(issues, detectDependencyCyclesV0(normalized, claimByRef)...)
	issues = append(issues, detectNotClosedDependenciesV0(normalized, claimByRef)...)
	issues = compactDependencyIssuesV0(issues)

	blocked := dependencyBlockedClaimRefsV0(normalized, issues)
	ready := dependencyReadyClaimRefsV0(normalized, blocked)

	return WorksetDependencyEvaluationV0{
		ReadyClaimRefs:   ready,
		BlockedClaimRefs: blocked,
		Issues:           issues,
	}
}

func normalizeClaimsForDependencyV0(claims []WorksetClaimV0) []WorksetClaimV0 {
	normalized := make([]WorksetClaimV0, 0, len(claims))
	for _, claim := range claims {
		value, _ := NormalizeWorksetClaimV0(claim)
		normalized = append(normalized, value)
	}
	sort.SliceStable(normalized, func(left, right int) bool {
		if normalized[left].ClaimRef != normalized[right].ClaimRef {
			return normalized[left].ClaimRef < normalized[right].ClaimRef
		}
		return normalized[left].TaskRef < normalized[right].TaskRef
	})
	return normalized
}

func indexClaimsByRefV0(claims []WorksetClaimV0) map[string]WorksetClaimV0 {
	indexed := make(map[string]WorksetClaimV0, len(claims))
	for _, claim := range claims {
		if claim.ClaimRef == "" {
			continue
		}
		if _, exists := indexed[claim.ClaimRef]; exists {
			continue
		}
		indexed[claim.ClaimRef] = claim
	}
	return indexed
}

func dependencyBlockedClaimRefsV0(claims []WorksetClaimV0, issues []WorksetDependencyIssueV0) []string {
	blocked := map[string]bool{}
	for _, claim := range claims {
		if len(claim.DependsOn) > 0 {
			blocked[claim.ClaimRef] = true
		}
	}
	for _, issue := range issues {
		for _, claimRef := range issue.ClaimRefs {
			blocked[claimRef] = true
		}
	}
	return sortedMapKeysV0(blocked)
}

func dependencyReadyClaimRefsV0(claims []WorksetClaimV0, blocked []string) []string {
	blockedSet := map[string]bool{}
	for _, claimRef := range blocked {
		blockedSet[claimRef] = true
	}
	ready := map[string]bool{}
	for _, claim := range claims {
		if claim.ClaimRef != "" && !blockedSet[claim.ClaimRef] {
			ready[claim.ClaimRef] = true
		}
	}
	return sortedMapKeysV0(ready)
}
