package orquestacoreconcurrency

func detectNotClosedDependenciesV0(claims []WorksetClaimV0, claimByRef map[string]WorksetClaimV0) []WorksetDependencyIssueV0 {
	issues := make([]WorksetDependencyIssueV0, 0)
	for _, root := range claims {
		missingByClaim := map[string][]string{}
		collectTransitiveMissingDependenciesV0(root, root, claimByRef, map[string]bool{}, missingByClaim)
		for claimRef, dependencyRefs := range missingByClaim {
			if claimRef == root.ClaimRef {
				continue
			}
			issues = append(issues, newDependencyIssueV0(
				WorksetDependencyIssueKindNotClosedV0,
				sortedUniqueStringsV0([]string{root.ClaimRef, claimRef}),
				dependencyRefs,
				evidenceRefsForClaimsV0([]string{root.ClaimRef, claimRef}, claimByRef),
			))
		}
	}
	return issues
}

func collectTransitiveMissingDependenciesV0(root WorksetClaimV0, current WorksetClaimV0, claimByRef map[string]WorksetClaimV0, visited map[string]bool, missingByClaim map[string][]string) {
	if current.ClaimRef == "" || visited[current.ClaimRef] {
		return
	}
	visited[current.ClaimRef] = true

	for _, dependencyRef := range current.DependsOn {
		if dependencyRef == current.ClaimRef {
			continue
		}
		dependency, ok := claimByRef[dependencyRef]
		if !ok {
			missingByClaim[current.ClaimRef] = append(missingByClaim[current.ClaimRef], dependencyRef)
			continue
		}
		if dependency.ClaimRef == root.ClaimRef {
			continue
		}
		collectTransitiveMissingDependenciesV0(root, dependency, claimByRef, visited, missingByClaim)
	}
}
