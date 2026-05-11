package orquestadirectorcandidates

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
)

func buildClaimsV0(
	input SchedulableWorkCandidateInputV0,
) ([]orquestacoreconcurrency.WorksetClaimV0, error) {
	claims := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(input.Claims)+len(input.ScopeClaims))
	for _, claim := range input.Claims {
		normalized, issues := orquestacoreconcurrency.NormalizeWorksetClaimV0(claim)
		if len(issues) > 0 {
			return nil, candidateErrorV0("claims")
		}
		claims = append(claims, normalized)
	}
	for _, claim := range input.ScopeClaims {
		built, err := buildScopeClaimV0(input, claim)
		if err != nil {
			return nil, err
		}
		claims = append(claims, built)
	}
	return claims, nil
}

func buildScopeClaimV0(
	input SchedulableWorkCandidateInputV0,
	claim WorkCandidateScopeClaimV0,
) (orquestacoreconcurrency.WorksetClaimV0, error) {
	readSet, issues := orquestacoreconcurrency.NormalizeScopeRefsV0(claim.ReadScopes)
	if len(issues) > 0 {
		return orquestacoreconcurrency.WorksetClaimV0{}, candidateErrorV0("scope_claims.read_scopes")
	}
	writeSet, issues := orquestacoreconcurrency.NormalizeScopeRefsV0(claim.WriteScopes)
	if len(issues) > 0 {
		return orquestacoreconcurrency.WorksetClaimV0{}, candidateErrorV0("scope_claims.write_scopes")
	}
	normalized, issues := orquestacoreconcurrency.NormalizeWorksetClaimV0(orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claim.ClaimRef,
		RunRef:         input.RunRef,
		TaskRef:        input.TaskRef,
		GroupRef:       claim.GroupRef,
		AgentRequestID: claim.AgentRequestID,
		ReadSet:        readSet,
		WriteSet:       writeSet,
		DependsOn:      claim.DependsOn,
		EvidenceRefs:   claim.EvidenceRefs,
	})
	if len(issues) > 0 {
		return orquestacoreconcurrency.WorksetClaimV0{}, candidateErrorV0("scope_claims")
	}
	return normalized, nil
}
