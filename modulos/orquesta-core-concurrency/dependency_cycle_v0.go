package orquestacoreconcurrency

import "strings"

func detectDependencyCyclesV0(claims []WorksetClaimV0, claimByRef map[string]WorksetClaimV0) []WorksetDependencyIssueV0 {
	visiting := map[string]int{}
	visited := map[string]bool{}
	seenCycles := map[string]bool{}
	issues := make([]WorksetDependencyIssueV0, 0)

	var visit func(claim WorksetClaimV0, stack []string)
	visit = func(claim WorksetClaimV0, stack []string) {
		if claim.ClaimRef == "" || visited[claim.ClaimRef] {
			return
		}
		visiting[claim.ClaimRef] = len(stack)
		stack = append(stack, claim.ClaimRef)

		for _, dependencyRef := range claim.DependsOn {
			dependency, ok := claimByRef[dependencyRef]
			if !ok || dependencyRef == claim.ClaimRef {
				continue
			}
			if index, ok := visiting[dependencyRef]; ok {
				claimRefs := sortedUniqueStringsV0(stack[index:])
				key := strings.Join(claimRefs, "+")
				if !seenCycles[key] {
					seenCycles[key] = true
					issues = append(issues, newDependencyIssueV0(
						WorksetDependencyIssueKindCycleV0,
						claimRefs,
						claimRefs,
						evidenceRefsForClaimsV0(claimRefs, claimByRef),
					))
				}
				continue
			}
			visit(dependency, stack)
		}

		delete(visiting, claim.ClaimRef)
		visited[claim.ClaimRef] = true
	}

	for _, claim := range claims {
		visit(claim, nil)
	}
	return issues
}
