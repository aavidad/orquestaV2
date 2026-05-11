package orquestacoreconcurrency

import (
	"sort"
	"strings"
)

func compactDependencyIssuesV0(issues []WorksetDependencyIssueV0) []WorksetDependencyIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	byRef := make(map[string]WorksetDependencyIssueV0, len(issues))
	for _, issue := range issues {
		issue.ClaimRefs = sortedUniqueStringsV0(issue.ClaimRefs)
		issue.DependencyRefs = sortedUniqueStringsV0(issue.DependencyRefs)
		issue.EvidenceRefs = normalizeOpaqueRefsV0(issue.EvidenceRefs)
		issue.IssueRef = dependencyIssueRefV0(issue.IssueKind, issue.ClaimRefs, issue.DependencyRefs)
		issue.Summary = dependencyIssueSummaryV0(issue.IssueKind, issue.ClaimRefs, issue.DependencyRefs)
		byRef[issue.IssueRef] = issue
	}

	refs := make([]string, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	compact := make([]WorksetDependencyIssueV0, 0, len(refs))
	for _, ref := range refs {
		compact = append(compact, byRef[ref])
	}
	return compact
}

func newDependencyIssueV0(kind WorksetDependencyIssueKindV0, claimRefs []string, dependencyRefs []string, evidenceRefs []string) WorksetDependencyIssueV0 {
	claimRefs = sortedUniqueStringsV0(claimRefs)
	dependencyRefs = sortedUniqueStringsV0(dependencyRefs)
	return WorksetDependencyIssueV0{
		IssueRef:          dependencyIssueRefV0(kind, claimRefs, dependencyRefs),
		ClaimRefs:         claimRefs,
		DependencyRefs:    dependencyRefs,
		IssueKind:         kind,
		RecommendedAction: dependencyIssueActionV0(kind),
		Summary:           dependencyIssueSummaryV0(kind, claimRefs, dependencyRefs),
		EvidenceRefs:      normalizeOpaqueRefsV0(evidenceRefs),
	}
}

func dependencyIssueRefV0(kind WorksetDependencyIssueKindV0, claimRefs []string, dependencyRefs []string) string {
	return "dependency:" + string(kind) + ":" + strings.Join(claimRefs, "+") + ":" + strings.Join(dependencyRefs, "+")
}

func dependencyIssueActionV0(kind WorksetDependencyIssueKindV0) WorksetConflictRecommendedActionV0 {
	switch kind {
	case WorksetDependencyIssueKindMissingV0, WorksetDependencyIssueKindSelfV0:
		return WorksetConflictRecommendedActionRejectClaimV0
	case WorksetDependencyIssueKindCycleV0, WorksetDependencyIssueKindNotClosedV0:
		return WorksetConflictRecommendedActionAskDirectorV0
	default:
		return WorksetConflictRecommendedActionAskDirectorV0
	}
}

func dependencyIssueSummaryV0(kind WorksetDependencyIssueKindV0, claimRefs []string, dependencyRefs []string) string {
	return string(kind) + " en " + strings.Join(claimRefs, "+") + " por " + strings.Join(dependencyRefs, "+")
}

func evidenceRefsForClaimsV0(claimRefs []string, claimByRef map[string]WorksetClaimV0) []string {
	evidenceRefs := make([]string, 0)
	for _, claimRef := range claimRefs {
		claim, ok := claimByRef[claimRef]
		if !ok {
			continue
		}
		evidenceRefs = append(evidenceRefs, claim.EvidenceRefs...)
	}
	return normalizeOpaqueRefsV0(evidenceRefs)
}
