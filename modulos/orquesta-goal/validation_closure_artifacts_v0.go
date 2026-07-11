package orquestagoal

import (
	"path/filepath"
	"strings"
)

func goalPartialArtifactClosureIssuesV0(spec GoalWorkSpecV0, result GoalWorkResultV0) []GoalWorkIssueV0 {
	issues := []GoalWorkIssueV0{}
	partial := false
	for _, artifact := range result.MaterializedArtifacts {
		if artifact.Status != GoalMaterializedArtifactStatusValidV0 {
			partial = true
			issues = append(issues, GoalWorkIssueV0{
				Code:  ErrGoalMaterializedArtifactInvalidV0,
				Field: "materialized_artifacts.status",
			})
		}
	}
	if spec.ClosurePolicy.RequireChecklist && len(result.Checklist.ExpectedRefs) == 0 {
		partial = true
		issues = append(issues, GoalWorkIssueV0{
			Code:  ErrGoalChecklistIncompleteV0,
			Field: "checklist.expected_refs",
		})
	}
	if len(result.Checklist.MissingRefs) > 0 {
		partial = true
		issues = append(issues, GoalWorkIssueV0{
			Code:  ErrGoalChecklistIncompleteV0,
			Field: "checklist.missing_refs",
		})
	}
	if partial && spec.ClosurePolicy.RequireReworkPlanForPartialArtifacts && len(result.ReworkPlanRefs) == 0 {
		issues = append(issues, GoalWorkIssueV0{
			Code:  ErrGoalReworkPlanRequiredV0,
			Field: "rework_plan_refs",
		})
	}
	return issues
}

func goalClosureEvidenceRefsV0(result GoalWorkResultV0) []string {
	refs := append([]string(nil), result.EvidenceRefs...)
	for _, artifact := range result.MaterializedArtifacts {
		refs = append(refs, artifact.EvidenceRefs...)
	}
	refs = append(refs, result.Checklist.EvidenceRefs...)
	return compactGoalStringsV0(refs)
}

func goalMaterializedArtifactPathsForScopeV0(result GoalWorkResultV0) []string {
	var paths []string
	for _, artifact := range result.MaterializedArtifacts {
		if artifact.Path != "" {
			paths = append(paths, artifact.Path)
		}
	}
	return paths
}

func goalArtifactPathsOutsideWriteSetV0(writeSet []GoalWriteScopeV0, artifactPaths []string) []string {
	if len(artifactPaths) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(writeSet))
	for _, scope := range writeSet {
		path := filepath.ToSlash(strings.TrimSpace(scope.Path))
		if path != "" {
			scopes = append(scopes, path)
		}
	}
	var out []string
	for _, artifactPath := range artifactPaths {
		path := filepath.ToSlash(strings.TrimSpace(artifactPath))
		if path == "" {
			continue
		}
		if !goalArtifactPathInsideAnyScopeV0(path, scopes) {
			out = append(out, path)
		}
	}
	return out
}

func goalArtifactPathInsideAnyScopeV0(path string, scopes []string) bool {
	for _, scope := range scopes {
		if path == scope || strings.HasPrefix(path, scope+"/") {
			return true
		}
	}
	return false
}

func missingGoalEvidenceRefsV0(required, actual []string) []string {
	if len(required) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(actual))
	for _, ref := range actual {
		seen[ref] = true
	}
	var missing []string
	for _, ref := range required {
		if ref != "" && !seen[ref] {
			missing = append(missing, ref)
		}
	}
	return missing
}

func allRequiredGoalTestsPassedV0(required []GoalRequiredTestV0, results []GoalRequiredTestResultV0) bool {
	if len(required) == 0 {
		return true
	}
	requiresEvidence := make(map[string]bool, len(required))
	for _, test := range required {
		requiresEvidence[test.TestRef] = len(compactGoalStringsV0(test.EvidenceRefs)) > 0
	}
	passed := make(map[string]bool, len(results))
	for _, result := range results {
		if result.Status == GoalStatusAcceptedV0 || result.Status == "passed" {
			if !requiresEvidence[result.TestRef] || len(compactGoalStringsV0(result.EvidenceRefs)) > 0 {
				passed[result.TestRef] = true
			}
		}
	}
	for _, test := range required {
		if !passed[test.TestRef] {
			return false
		}
	}
	return true
}

func allRequiredGoalArtifactsPresentV0(required []GoalArtifactContractV0, actual []string) bool {
	present := make(map[string]bool, len(actual))
	for _, ref := range actual {
		present[ref] = true
	}
	for _, artifact := range required {
		if artifact.Required && !present[artifact.ArtifactRef] {
			return false
		}
	}
	return true
}
