package orquestacoreconcurrency

import (
	"sort"
	"strings"
)

type WorksetConflictKindV0 string

const (
	WorksetConflictKindWriteWriteV0 WorksetConflictKindV0 = "write_write"
	WorksetConflictKindReadWriteV0  WorksetConflictKindV0 = "read_write"
)

type WorksetConflictRecommendedActionV0 string

const (
	WorksetConflictRecommendedActionSerializeV0   WorksetConflictRecommendedActionV0 = "serialize"
	WorksetConflictRecommendedActionAskDirectorV0 WorksetConflictRecommendedActionV0 = "ask_director"
	WorksetConflictRecommendedActionSplitTaskV0   WorksetConflictRecommendedActionV0 = "split_task"
	WorksetConflictRecommendedActionRejectClaimV0 WorksetConflictRecommendedActionV0 = "reject_claim"
)

type WorksetConflictV0 struct {
	ConflictRef       string                             `json:"conflict_ref"`
	ClaimRefs         []string                           `json:"claim_refs"`
	ConflictKind      WorksetConflictKindV0              `json:"conflict_kind"`
	ScopeRefs         []ScopeRefV0                       `json:"scope_refs,omitempty"`
	RecommendedAction WorksetConflictRecommendedActionV0 `json:"recommended_action"`
	Summary           string                             `json:"summary"`
	EvidenceRefs      []string                           `json:"evidence_refs,omitempty"`
}

func DetectWorksetConflictsV0(claims []WorksetClaimV0) []WorksetConflictV0 {
	normalized := normalizeClaimsForConflictV0(claims)
	conflicts := make([]WorksetConflictV0, 0)

	for left := 0; left < len(normalized); left++ {
		for right := left + 1; right < len(normalized); right++ {
			writeWriteRefs := overlappingScopeRefsV0(normalized[left].WriteSet, normalized[right].WriteSet)
			if len(writeWriteRefs) > 0 {
				conflicts = append(conflicts, newWorksetConflictV0(
					WorksetConflictKindWriteWriteV0,
					normalized[left],
					normalized[right],
					writeWriteRefs,
				))
				continue
			}

			readWriteRefs := overlappingScopeRefsV0(normalized[left].ReadSet, normalized[right].WriteSet)
			readWriteRefs = append(readWriteRefs, overlappingScopeRefsV0(normalized[right].ReadSet, normalized[left].WriteSet)...)
			readWriteRefs = compactScopeRefsForConflictV0(readWriteRefs)
			if len(readWriteRefs) > 0 {
				conflicts = append(conflicts, newWorksetConflictV0(
					WorksetConflictKindReadWriteV0,
					normalized[left],
					normalized[right],
					readWriteRefs,
				))
			}
		}
	}

	sort.SliceStable(conflicts, func(left, right int) bool {
		return conflicts[left].ConflictRef < conflicts[right].ConflictRef
	})
	return conflicts
}

func normalizeClaimsForConflictV0(claims []WorksetClaimV0) []WorksetClaimV0 {
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

func overlappingScopeRefsV0(left []ScopeRefV0, right []ScopeRefV0) []ScopeRefV0 {
	refs := make([]ScopeRefV0, 0)
	for _, leftRef := range left {
		for _, rightRef := range right {
			if ScopeRefsOverlapV0(leftRef, rightRef) {
				refs = append(refs, leftRef, rightRef)
			}
		}
	}
	return compactScopeRefsForConflictV0(refs)
}

func compactScopeRefsForConflictV0(refs []ScopeRefV0) []ScopeRefV0 {
	if len(refs) == 0 {
		return nil
	}
	raw := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.Ref) != "" {
			raw = append(raw, ref.Ref)
		}
	}
	normalized, _ := normalizeScopeRefsV0("scope_refs", raw)
	return normalized
}

func newWorksetConflictV0(kind WorksetConflictKindV0, left WorksetClaimV0, right WorksetClaimV0, refs []ScopeRefV0) WorksetConflictV0 {
	claimRefs := []string{left.ClaimRef, right.ClaimRef}
	sort.Strings(claimRefs)
	scopeKey := joinScopeRefsForConflictV0(refs)
	conflictRef := "conflict:" + string(kind) + ":" + strings.Join(claimRefs, "+") + ":" + scopeKey

	return WorksetConflictV0{
		ConflictRef:       conflictRef,
		ClaimRefs:         claimRefs,
		ConflictKind:      kind,
		ScopeRefs:         refs,
		RecommendedAction: WorksetConflictRecommendedActionSerializeV0,
		Summary:           string(kind) + " entre " + strings.Join(claimRefs, " y ") + " en " + scopeKey,
		EvidenceRefs:      mergeEvidenceRefsForConflictV0(left.EvidenceRefs, right.EvidenceRefs),
	}
}

func joinScopeRefsForConflictV0(refs []ScopeRefV0) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts = append(parts, ref.Ref)
	}
	sort.Strings(parts)
	return strings.Join(parts, "+")
}

func mergeEvidenceRefsForConflictV0(left []string, right []string) []string {
	return normalizeOpaqueRefsV0(append(append([]string{}, left...), right...))
}
