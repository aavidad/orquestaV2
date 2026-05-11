package orquestadirectorcandidates

import "strings"

func normalizeRefsV0(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		value := strings.TrimSpace(ref)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeScopeClaimsV0(claims []WorkCandidateScopeClaimV0) []WorkCandidateScopeClaimV0 {
	if len(claims) == 0 {
		return nil
	}
	normalized := make([]WorkCandidateScopeClaimV0, 0, len(claims))
	for _, claim := range claims {
		normalized = append(normalized, normalizeScopeClaimV0(claim))
	}
	return normalized
}

func normalizeScopeClaimV0(claim WorkCandidateScopeClaimV0) WorkCandidateScopeClaimV0 {
	return WorkCandidateScopeClaimV0{
		ClaimRef:       strings.TrimSpace(claim.ClaimRef),
		AgentRequestID: strings.TrimSpace(claim.AgentRequestID),
		GroupRef:       strings.TrimSpace(claim.GroupRef),
		ReadScopes:     normalizeRefsV0(claim.ReadScopes),
		WriteScopes:    normalizeRefsV0(claim.WriteScopes),
		DependsOn:      normalizeRefsV0(claim.DependsOn),
		EvidenceRefs:   normalizeRefsV0(claim.EvidenceRefs),
	}
}
