package orquestacoreconcurrency

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

type ParallelGroupPlanV0 struct {
	PlanRef          string   `json:"plan_ref"`
	RunRef           string   `json:"run_ref,omitempty"`
	ReadyClaimRefs   []string `json:"ready_claim_refs,omitempty"`
	BlockedClaimRefs []string `json:"blocked_claim_refs,omitempty"`
	ConflictRefs     []string `json:"conflict_refs,omitempty"`
	Summary          string   `json:"summary"`
}

func EvaluateParallelGroupsV0(claims []WorksetClaimV0) ParallelGroupPlanV0 {
	normalized := normalizeClaimsForDependencyV0(claims)
	dependencies := EvaluateWorksetDependenciesV0(normalized)
	conflicts := DetectWorksetConflictsV0(normalized)

	conflictRefs := parallelGroupConflictRefsV0(conflicts)
	invalidClaimRefs := parallelGroupInvalidClaimRefsV0(claims)
	conflictClaimRefs := parallelGroupConflictClaimRefsV0(conflicts)
	blockedCandidates := append([]string{}, dependencies.BlockedClaimRefs...)
	blockedCandidates = append(blockedCandidates, invalidClaimRefs...)
	blockedCandidates = append(blockedCandidates, conflictClaimRefs...)
	blocked := sortedUniqueStringsV0(blockedCandidates)
	ready := parallelGroupReadyClaimRefsV0(dependencies.ReadyClaimRefs, blocked)
	runRef := parallelGroupRunRefV0(normalized)

	return ParallelGroupPlanV0{
		PlanRef:          parallelGroupPlanRefV0(runRef, normalized),
		RunRef:           runRef,
		ReadyClaimRefs:   ready,
		BlockedClaimRefs: blocked,
		ConflictRefs:     conflictRefs,
		Summary:          parallelGroupSummaryV0(ready, blocked, conflictRefs, dependencies.Issues),
	}
}

func parallelGroupInvalidClaimRefsV0(claims []WorksetClaimV0) []string {
	refs := make([]string, 0)
	for _, claim := range claims {
		normalized, issues := NormalizeWorksetClaimV0(claim)
		if len(issues) == 0 {
			continue
		}
		refs = append(refs, normalized.ClaimRef)
	}
	return sortedUniqueStringsV0(refs)
}

func parallelGroupReadyClaimRefsV0(dependencyReady []string, blocked []string) []string {
	blockedSet := map[string]bool{}
	for _, claimRef := range blocked {
		blockedSet[claimRef] = true
	}
	ready := make([]string, 0, len(dependencyReady))
	for _, claimRef := range dependencyReady {
		if !blockedSet[claimRef] {
			ready = append(ready, claimRef)
		}
	}
	return sortedUniqueStringsV0(ready)
}

func parallelGroupConflictRefsV0(conflicts []WorksetConflictV0) []string {
	refs := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		refs = append(refs, conflict.ConflictRef)
	}
	return sortedUniqueStringsV0(refs)
}

func parallelGroupConflictClaimRefsV0(conflicts []WorksetConflictV0) []string {
	refs := make([]string, 0)
	for _, conflict := range conflicts {
		refs = append(refs, conflict.ClaimRefs...)
	}
	return sortedUniqueStringsV0(refs)
}

func parallelGroupRunRefV0(claims []WorksetClaimV0) string {
	refs := make([]string, 0, len(claims))
	for _, claim := range claims {
		refs = append(refs, claim.RunRef)
	}
	return strings.Join(sortedUniqueStringsV0(refs), "+")
}

func parallelGroupPlanRefV0(runRef string, claims []WorksetClaimV0) string {
	claimRefs := make([]string, 0, len(claims))
	for _, claim := range claims {
		claimRefs = append(claimRefs, claim.ClaimRef)
	}
	claimKey := strings.Join(sortedUniqueStringsV0(claimRefs), "+")
	if claimKey == "" {
		claimKey = "claims:none"
	}
	if runRef == "" {
		runRef = "run:none"
	}
	return "parallel_group_plan:" + runRef + ":" + shortConcurrencyDigestV0(claimKey)
}

func shortConcurrencyDigestV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:24]
}

func parallelGroupSummaryV0(ready []string, blocked []string, conflicts []string, issues []WorksetDependencyIssueV0) string {
	return "parallel_group_plan ready=" + strconv.Itoa(len(ready)) +
		" blocked=" + strconv.Itoa(len(blocked)) +
		" conflicts=" + strconv.Itoa(len(conflicts)) +
		" dependency_issues=" + strconv.Itoa(len(issues))
}
