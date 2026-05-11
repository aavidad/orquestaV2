package orquestacoreconcurrency

func detectDirectDependencyIssuesV0(claims []WorksetClaimV0, claimByRef map[string]WorksetClaimV0) []WorksetDependencyIssueV0 {
	issues := make([]WorksetDependencyIssueV0, 0)
	for _, claim := range claims {
		for _, dependencyRef := range claim.DependsOn {
			switch {
			case dependencyRef == claim.ClaimRef:
				issues = append(issues, newDependencyIssueV0(
					WorksetDependencyIssueKindSelfV0,
					[]string{claim.ClaimRef},
					[]string{dependencyRef},
					claim.EvidenceRefs,
				))
			case !dependencyClaimExistsV0(claimByRef, dependencyRef):
				issues = append(issues, newDependencyIssueV0(
					WorksetDependencyIssueKindMissingV0,
					[]string{claim.ClaimRef},
					[]string{dependencyRef},
					claim.EvidenceRefs,
				))
			}
		}
	}
	return issues
}

func dependencyClaimExistsV0(claimByRef map[string]WorksetClaimV0, claimRef string) bool {
	_, ok := claimByRef[claimRef]
	return ok
}
