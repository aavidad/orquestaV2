package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type CodexStackMaterialProgressEvidenceV0 struct {
	Stack *StackV0
}

func (source CodexStackMaterialProgressEvidenceV0) ClassifyMaterialProgressV0(
	ctx context.Context,
	request orquestaautoprogramming.MaterialProgressEvidenceRequestV0,
) (orquestaautoprogramming.MaterialProgressEvidenceV0, error) {
	state := request.State
	result := orquestagoal.NormalizeGoalWorkResultV0(request.Result)
	base, err := source.materialProgressBaseEvidenceV0(state)
	if err != nil {
		return base, err
	}
	if refs := compactStringsV0(result.DomainReceiptRefs); len(refs) > 0 {
		base.MaterialClass = orquestaautoprogramming.MaterialProgressClassReceiptV0
		base.EvidenceRefs = compactStringsV0(append([]string{"evidence-ref-material-progress-domain-receipt"}, refs...))
		base.Verified = true
		return base, nil
	}
	if refs, ok := goalMaterialProgressIndependentTestEvidenceV0(state); ok {
		base.MaterialClass = orquestaautoprogramming.MaterialProgressClassTestV0
		base.EvidenceRefs = []string{goalMaterialProgressEvidenceRefV0("test", refs)}
		base.Verified = true
		return base, nil
	}
	if goalMaterialProgressTerminalResultV0(result) {
		base.MaterialClass = orquestaautoprogramming.MaterialProgressClassResultV0
		base.EvidenceRefs = []string{goalMaterialProgressResultEvidenceRefV0(result)}
		base.Verified = true
		return base, nil
	}
	return source.classifyMaterialProgressDiffV0(ctx, state, base)
}

func goalMaterialProgressIndependentTestEvidenceV0(
	state orquestagoal.GoalWorkStateV0,
) ([]string, bool) {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	closure := state.LastClosure
	if closure == nil || !closure.Accepted ||
		!spec.ClosurePolicy.RequireIndependentRequiredTestAttestation ||
		len(spec.RequiredTests) == 0 {
		return nil, false
	}
	refs := append([]string(nil), closure.EvidenceRefs...)
	for _, required := range spec.RequiredTests {
		testRef := strings.TrimSpace(required.TestRef)
		matched := false
		for _, verification := range closure.AttestationVerifications {
			if strings.TrimSpace(verification.TestRef) != testRef ||
				!goalMaterialProgressIdentityVerificationValidV0(spec, verification) {
				continue
			}
			refs = append(refs, verification.AttestationRef)
			refs = append(refs, verification.EvidenceRefs...)
			matched = true
			break
		}
		if testRef == "" || !matched {
			return nil, false
		}
	}
	refs = compactStringsV0(refs)
	return refs, len(refs) > 0
}

func goalMaterialProgressIdentityVerificationValidV0(
	spec orquestagoal.GoalWorkSpecV0,
	verification orquestagoal.GoalRequiredTestIdentityVerificationV0,
) bool {
	return verification.Verified && verification.Independent &&
		strings.TrimSpace(verification.AttestationRef) != "" &&
		strings.TrimSpace(verification.ImplementerPrincipalRef) != "" &&
		strings.TrimSpace(verification.AttestorPrincipalRef) != "" &&
		strings.TrimSpace(verification.ImplementerPrincipalRef) != strings.TrimSpace(verification.AttestorPrincipalRef) &&
		strings.TrimSpace(verification.AttestorCredentialRef) != "" &&
		strings.TrimSpace(verification.AttestorCredentialRef) != strings.TrimSpace(spec.ImplementerCredentialRef) &&
		strings.TrimSpace(verification.TrustPolicyRef) == strings.TrimSpace(spec.ClosurePolicy.RequiredAttestorTrustPolicyRef)
}

func (source CodexStackMaterialProgressEvidenceV0) materialProgressBaseEvidenceV0(
	state orquestagoal.GoalWorkStateV0,
) (orquestaautoprogramming.MaterialProgressEvidenceV0, error) {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	baselineRef := goalMaterialProgressContextRefV0(spec.ContextRefs, "worktree_baseline")
	writeSetSHA := strings.TrimSpace(spec.WriteSetSHA256)
	if writeSetSHA == "" {
		writeSetSHA = orquestagoal.GoalWriteSetSHA256V0(spec.WriteSet)
	}
	contextRef := goalMaterialProgressContextRevisionRefV0(spec, baselineRef, writeSetSHA)
	base := orquestaautoprogramming.MaterialProgressEvidenceV0{
		MaterialClass: orquestaautoprogramming.MaterialProgressClassNoneV0,
		BaselineRef:   baselineRef, WriteSetSHA256: writeSetSHA, ContextRevisionRef: contextRef,
	}
	if source.Stack == nil || baselineRef == "" || writeSetSHA == "" || contextRef == "" {
		return base, fmt.Errorf("material_progress_contract_unavailable")
	}
	return base, nil
}

func (source CodexStackMaterialProgressEvidenceV0) classifyMaterialProgressDiffV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	base orquestaautoprogramming.MaterialProgressEvidenceV0,
) (orquestaautoprogramming.MaterialProgressEvidenceV0, error) {
	store := source.Stack.AutoprogrammingPromotion.GoalFirstSnapshotStore
	projectDir, projectDirErr := source.Stack.autoprogrammingGoalProjectWorkDirV0(ctx, state)
	if store == nil || projectDirErr != nil {
		return base, fmt.Errorf("material_progress_diff_verifier_unavailable")
	}
	baseline, err := store.LoadWorktreeSnapshotV0(ctx, base.BaselineRef)
	if err != nil || len(baseline.OmittedPaths) > 0 {
		return base, fmt.Errorf("material_progress_baseline_unavailable")
	}
	writeSet := make([]string, 0, len(state.Spec.WriteSet))
	for _, scope := range state.Spec.WriteSet {
		writeSet = append(writeSet, scope.Path)
	}
	verified, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(ctx, orquestaruntimeworktree.WorktreeVerifyRequestV0{
		Baseline: baseline, ProjectWorkDir: projectDir, WriteSet: writeSet,
		DestructiveAuthorizations: autoprogrammingWorktreeDestructiveAuthorizationsV0(state.Spec.DestructiveAuthorizations),
		IgnorePrefixes:            codexStackWorktreeIgnorePrefixesV0(), AllowPartialSnapshot: false,
	})
	if len(issues) > 0 || !verified.OK {
		base.Verified = true
		base.EvidenceRefs = []string{"evidence-ref-material-progress-diff-not-verified"}
		for _, issue := range issues {
			base.EvidenceRefs = append(base.EvidenceRefs, "evidence-ref-material-progress-worktree-issue-"+string(issue.Code))
		}
		base.EvidenceRefs = compactStringsV0(base.EvidenceRefs)
		return base, nil
	}
	if len(verified.ChangedPaths) == 0 {
		base.Verified = true
		base.EvidenceRefs = []string{"evidence-ref-material-progress-no-diff"}
		return base, nil
	}
	base.Verified = true
	base.MaterialClass = orquestaautoprogramming.MaterialProgressClassDiffV0
	base.EvidenceRefs = []string{goalMaterialProgressDiffEvidenceRefV0(base.BaselineRef, verified.ChangedPaths)}
	return base, nil
}

func goalMaterialProgressTerminalResultV0(result orquestagoal.GoalWorkResultV0) bool {
	return result.Status == orquestagoal.GoalStatusCompleteV0 || result.Status == orquestagoal.GoalStatusAcceptedV0
}

func goalMaterialProgressContextRefV0(refs []orquestagoal.GoalContextRefV0, kind string) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.Kind) == kind {
			return strings.TrimSpace(ref.Ref)
		}
	}
	return ""
}

func goalMaterialProgressContextRevisionRefV0(spec orquestagoal.GoalWorkSpecV0, baselineRef, writeSetSHA string) string {
	payload, err := json.Marshal(struct {
		BaselineRef    string                          `json:"baseline_ref"`
		WriteSetSHA256 string                          `json:"write_set_sha256"`
		ContextRefs    []orquestagoal.GoalContextRefV0 `json:"context_refs"`
	}{baselineRef, writeSetSHA, spec.ContextRefs})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("context-revision-ref-%x", sum[:])
}

func goalMaterialProgressDiffEvidenceRefV0(baselineRef string, paths []string) string {
	paths = compactStringsV0(paths)
	sort.Strings(paths)
	sum := sha256.Sum256([]byte(strings.Join(append([]string{baselineRef}, paths...), "\n")))
	return fmt.Sprintf("evidence-ref-material-progress-diff-%x", sum[:])
}

func goalMaterialProgressEvidenceRefV0(kind string, refs []string) string {
	refs = compactStringsV0(refs)
	sort.Strings(refs)
	sum := sha256.Sum256([]byte(strings.Join(append([]string{kind}, refs...), "\n")))
	return fmt.Sprintf("evidence-ref-material-progress-%s-%x", kind, sum[:])
}

func goalMaterialProgressResultEvidenceRefV0(result orquestagoal.GoalWorkResultV0) string {
	payload, _ := json.Marshal(result)
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("evidence-ref-material-progress-result-%x", sum[:])
}
