package orquestarunqueue

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
)

const (
	RunQueueWorksetReasonSerializedV0   = "run_workset_serialized"
	RunQueueWorksetReasonMissingClaimV0 = "run_workset_claim_missing"
	RunQueueWorksetReasonInvalidClaimV0 = "run_workset_claim_invalid"
)

func EvaluateRunQueueWorksetConcurrencyV0(
	candidates []RunSchedulingCandidateV0,
) RunQueueWorksetEvaluationV0 {
	return EvaluateRunQueueWorksetConcurrencyWithPolicyV0(candidates, RunQueueWorksetPolicyV0{})
}

func EvaluateRunQueueWorksetConcurrencyWithPolicyV0(
	candidates []RunSchedulingCandidateV0,
	policy RunQueueWorksetPolicyV0,
) RunQueueWorksetEvaluationV0 {
	evaluation := RunQueueWorksetEvaluationV0{}
	accepted := make([]RunSchedulingCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = cloneRunSchedulingCandidateV0(candidate)
		if !IsExecutableRunStatusV0(candidate.Status) {
			continue
		}
		if len(candidate.WorksetClaims) == 0 {
			if policy.RequireClaims {
				block := RunQueueWorksetBlockV0{
					RunRef:       candidate.RunRef,
					ConflictRefs: []string{"workset-claim-missing:" + candidate.RunRef},
					Reason:       RunQueueWorksetReasonMissingClaimV0,
				}
				evaluation.Blocks = append(evaluation.Blocks, block)
				evaluation.EvidenceRefs = compactRunQueueStringsV0(append(evaluation.EvidenceRefs, block.ConflictRefs...))
				continue
			}
			evaluation.AllowedRunRefs = append(evaluation.AllowedRunRefs, candidate.RunRef)
			accepted = append(accepted, candidate)
			continue
		}
		if block := invalidWorksetBlockV0(candidate); block.Reason != "" {
			evaluation.Blocks = append(evaluation.Blocks, block)
			evaluation.EvidenceRefs = compactRunQueueStringsV0(append(evaluation.EvidenceRefs, block.ConflictRefs...))
			continue
		}
		if block := conflictWorksetBlockV0(candidate, accepted); block.Reason != "" {
			evaluation.Blocks = append(evaluation.Blocks, block)
			evaluation.EvidenceRefs = compactRunQueueStringsV0(append(evaluation.EvidenceRefs, block.ConflictRefs...))
			continue
		}
		evaluation.AllowedRunRefs = append(evaluation.AllowedRunRefs, candidate.RunRef)
		accepted = append(accepted, candidate)
	}
	return evaluation
}

func filterRankedRunCandidatesByWorksetV0(
	ranked []RankedRunCandidateV0,
	policy RunQueueWorksetPolicyV0,
) []RankedRunCandidateV0 {
	candidates := make([]RunSchedulingCandidateV0, 0, len(ranked))
	byRun := make(map[string]RankedRunCandidateV0, len(ranked))
	for _, candidate := range ranked {
		candidates = append(candidates, candidate.RunSchedulingCandidateV0)
		byRun[candidate.RunRef] = candidate
	}
	evaluation := EvaluateRunQueueWorksetConcurrencyWithPolicyV0(candidates, policy)
	out := make([]RankedRunCandidateV0, 0, len(evaluation.AllowedRunRefs))
	for _, runRef := range evaluation.AllowedRunRefs {
		if candidate, ok := byRun[runRef]; ok {
			out = append(out, candidate)
		}
	}
	return out
}

func invalidWorksetBlockV0(candidate RunSchedulingCandidateV0) RunQueueWorksetBlockV0 {
	for _, claim := range candidate.WorksetClaims {
		if len(orquestacoreconcurrency.ValidateWorksetClaimV0(claim)) == 0 {
			continue
		}
		return RunQueueWorksetBlockV0{
			RunRef:       candidate.RunRef,
			ConflictRefs: []string{"workset-claim-invalid:" + candidate.RunRef},
			Reason:       RunQueueWorksetReasonInvalidClaimV0,
		}
	}
	return RunQueueWorksetBlockV0{}
}

func conflictWorksetBlockV0(
	candidate RunSchedulingCandidateV0,
	accepted []RunSchedulingCandidateV0,
) RunQueueWorksetBlockV0 {
	for _, blocker := range accepted {
		conflicts := orquestacoreconcurrency.DetectWorksetConflictsV0(
			append(cloneRunQueueWorksetClaimsV0(blocker.WorksetClaims), candidate.WorksetClaims...),
		)
		refs := conflictRefsV0(conflicts)
		if len(refs) == 0 {
			continue
		}
		return RunQueueWorksetBlockV0{
			RunRef:          candidate.RunRef,
			BlockedByRunRef: blocker.RunRef,
			ConflictRefs:    refs,
			Reason:          RunQueueWorksetReasonSerializedV0,
		}
	}
	return RunQueueWorksetBlockV0{}
}

func conflictRefsV0(conflicts []orquestacoreconcurrency.WorksetConflictV0) []string {
	refs := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		refs = append(refs, conflict.ConflictRef)
	}
	return compactRunQueueStringsV0(refs)
}

func cloneRunQueueWorksetClaimsV0(
	claims []orquestacoreconcurrency.WorksetClaimV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	out := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(claims))
	for _, claim := range claims {
		claim.ReadSet = cloneRunQueueScopeRefsV0(claim.ReadSet)
		claim.WriteSet = cloneRunQueueScopeRefsV0(claim.WriteSet)
		claim.DependsOn = append([]string(nil), claim.DependsOn...)
		claim.EvidenceRefs = append([]string(nil), claim.EvidenceRefs...)
		out = append(out, claim)
	}
	if out == nil {
		return []orquestacoreconcurrency.WorksetClaimV0{}
	}
	return out
}

func cloneRunQueueScopeRefsV0(
	refs []orquestacoreconcurrency.ScopeRefV0,
) []orquestacoreconcurrency.ScopeRefV0 {
	if len(refs) == 0 {
		return nil
	}
	out := make([]orquestacoreconcurrency.ScopeRefV0, len(refs))
	copy(out, refs)
	return out
}
