package orquestagoal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	GoalRequiredTestAttestationStatusPassedV0 = "passed"
	GoalRequiredTestAttestationStatusFailedV0 = "failed"
)

// IndependentGoalRequiredTestAttestationClosureValidatorV0 makes independent
// receipts authoritative when the frozen spec requires them. The implementer
// supplied RequiredTestResults remain diagnostics and are intentionally not
// consulted by this validator.
type IndependentGoalRequiredTestAttestationClosureValidatorV0 struct {
	Base             GoalWorkClosureValidatorPortV0
	Reader           GoalRequiredTestAttestationReaderPortV0
	SnapshotReader   GoalRequiredTestFinalSnapshotReaderPortV0
	IdentityVerifier GoalRequiredTestIdentityVerifierPortV0
}

func EnforceIndependentGoalRequiredTestAttestationV0(
	base GoalWorkClosureValidatorPortV0,
	store GoalRequiredTestAttestationStorePortV0,
	verifier GoalRequiredTestIdentityVerifierPortV0,
) GoalWorkClosureValidatorPortV0 {
	switch current := base.(type) {
	case IndependentGoalRequiredTestAttestationClosureValidatorV0:
		base = current.Base
	case *IndependentGoalRequiredTestAttestationClosureValidatorV0:
		base = current.Base
	}
	return IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Base: base, Reader: store, SnapshotReader: store, IdentityVerifier: verifier,
	}
}

func (validator IndependentGoalRequiredTestAttestationClosureValidatorV0) ValidateGoalWorkClosureV0(
	ctx context.Context,
	spec GoalWorkSpecV0,
	result GoalWorkResultV0,
) (GoalClosureValidationV0, error) {
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		return validator.base().ValidateGoalWorkClosureV0(ctx, spec, result)
	}
	// A self-reported result cannot be used as proof. Disable only the legacy
	// result predicate while preserving all other base closure checks.
	baseSpec := NormalizeGoalWorkSpecV0(spec)
	baseSpec.ClosurePolicy.RequireRequiredTests = false
	closure, err := validator.base().ValidateGoalWorkClosureV0(ctx, baseSpec, result)
	if err != nil || !closure.Accepted {
		return closure, err
	}
	if validator.Reader == nil || validator.SnapshotReader == nil {
		return blockedGoalRequiredTestAttestationClosureV0(closure, ErrGoalRequiredTestAttestationMissingV0, "required_test_attestations"), nil
	}
	if validator.IdentityVerifier == nil {
		return blockedGoalRequiredTestAttestationClosureV0(closure, ErrGoalRequiredTestAttestorUntrustedV0, "required_test_identity_verifier"), nil
	}
	snapshot, err := validator.SnapshotReader.LoadGoalRequiredTestFinalSnapshotV0(ctx, spec.RunRef, spec.GoalRef)
	if err != nil {
		return blockedGoalRequiredTestAttestationClosureV0(closure, ErrGoalRequiredTestSnapshotMissingV0, "required_test_final_snapshot"), nil
	}
	if issues := ValidateGoalRequiredTestFinalSnapshotV0(snapshot); len(issues) > 0 ||
		snapshot.RunRef != spec.RunRef || snapshot.GoalRef != spec.GoalRef ||
		snapshot.WriteSetSHA256 != spec.WriteSetSHA256 {
		return blockedGoalRequiredTestAttestationClosureV0(closure, ErrGoalRequiredTestSnapshotMismatchV0, "required_test_final_snapshot"), nil
	}
	attestations, err := validator.Reader.ListGoalRequiredTestAttestationsV0(ctx, GoalRequiredTestAttestationQueryV0{
		RunRef:      spec.RunRef,
		GoalRef:     spec.GoalRef,
		RevisionRef: snapshot.RevisionRef,
	})
	if err != nil {
		return GoalClosureValidationV0{}, err
	}
	verifications, issue, err := validator.verifyAttestationsV0(ctx, spec, snapshot, attestations)
	closure.AttestationVerifications = verifications
	if err != nil {
		return GoalClosureValidationV0{}, err
	}
	if issue.Code != "" {
		return blockedGoalRequiredTestAttestationClosureV0(closure, issue.Code, issue.Field), nil
	}
	for _, attestation := range attestations {
		closure.EvidenceRefs = append(closure.EvidenceRefs, attestation.AttestationRef)
		closure.EvidenceRefs = append(closure.EvidenceRefs, attestation.EvidenceRefs...)
	}
	closure.EvidenceRefs = append(closure.EvidenceRefs, snapshot.SnapshotRef)
	closure.EvidenceRefs = append(closure.EvidenceRefs, snapshot.EvidenceRefs...)
	for _, verification := range verifications {
		closure.EvidenceRefs = append(closure.EvidenceRefs, verification.EvidenceRefs...)
	}
	closure.EvidenceRefs = compactGoalStringsV0(closure.EvidenceRefs)
	return closure, nil
}

func (validator IndependentGoalRequiredTestAttestationClosureValidatorV0) verifyAttestationsV0(
	ctx context.Context,
	spec GoalWorkSpecV0,
	snapshot GoalRequiredTestFinalSnapshotV0,
	attestations []GoalRequiredTestAttestationV0,
) ([]GoalRequiredTestIdentityVerificationV0, GoalWorkIssueV0, error) {
	bindings, issue := independentGoalRequiredTestAttestationBindingsV0(spec, snapshot, attestations)
	if issue.Code != "" {
		return nil, issue, nil
	}
	verifications := make([]GoalRequiredTestIdentityVerificationV0, 0, len(bindings))
	for _, attestation := range bindings {
		verification, err := validator.IdentityVerifier.VerifyGoalRequiredTestIdentityV0(ctx, GoalRequiredTestIdentityVerificationRequestV0{
			RunRef: spec.RunRef, GoalRef: spec.GoalRef, RevisionRef: attestation.RevisionRef,
			AttestationRef: attestation.AttestationRef, TestRef: attestation.TestRef,
			ImplementerAgentRef: spec.ImplementerAgentRef, ImplementerCredentialRef: spec.ImplementerCredentialRef,
			AttestorAgentRef: attestation.AttestorAgentRef, AttestorCredentialRef: attestation.AttestorCredentialRef,
			RequiredTrustPolicyRef: spec.ClosurePolicy.RequiredAttestorTrustPolicyRef,
		})
		if err != nil {
			return nil, GoalWorkIssueV0{}, err
		}
		verification = normalizeGoalRequiredTestIdentityVerificationV0(verification)
		verification.TestRef = attestation.TestRef
		verifications = append(verifications, verification)
		if !validGoalRequiredTestIdentityVerificationV0(spec, attestation, verification) {
			return verifications, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestorUntrustedV0, Field: "required_test_attestor_identity"}, nil
		}
	}
	return verifications, GoalWorkIssueV0{}, nil
}

func (validator IndependentGoalRequiredTestAttestationClosureValidatorV0) base() GoalWorkClosureValidatorPortV0 {
	if validator.Base != nil {
		return validator.Base
	}
	return DefaultGoalWorkClosureValidatorV0{}
}

func RunAndPersistGoalRequiredTestAttestationsV0(
	ctx context.Context,
	request GoalRequiredTestAttestationRequestV0,
	claim GoalRequiredTestAttestationClaimV0,
	attestor GoalRequiredTestAttestorPortV0,
	store GoalRequiredTestAttestationStorePortV0,
) ([]GoalRequiredTestAttestationV0, error) {
	request = NormalizeGoalRequiredTestAttestationRequestV0(request)
	if attestatorIssue := ValidateGoalRequiredTestAttestationRequestV0(request); len(attestatorIssue) > 0 {
		return nil, GoalWorkLifecycleIssueErrorV0{Field: "required_test_attestation_request", Issues: attestatorIssue}
	}
	if attestor == nil {
		return nil, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_required_test_attestor"}
	}
	if store == nil {
		return nil, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_required_test_attestation_store"}
	}
	attestations, err := attestor.AttestGoalRequiredTestsV0(ctx, request)
	if err != nil {
		return nil, err
	}
	if issue := goalRequiredTestAttestationResponseIssueV0(request, attestations); issue.Code != "" {
		return nil, GoalWorkLifecycleIssueErrorV0{Field: "required_test_attestations", Issues: []GoalWorkIssueV0{issue}}
	}
	if len(attestations) != 1 || len(request.RequiredTests) != 1 {
		return nil, GoalWorkLifecycleIssueErrorV0{Field: "required_test_attestations", Issues: []GoalWorkIssueV0{{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "canonical_test_receipt"}}}
	}
	if err := store.CompleteGoalRequiredTestAttestationClaimV0(ctx, claim, attestations[0]); err != nil {
		return nil, err
	}
	return attestations, nil
}

func GoalRequiredTestAttestationRequestFromSpecV0(
	spec GoalWorkSpecV0,
	snapshot GoalRequiredTestFinalSnapshotV0,
) GoalRequiredTestAttestationRequestV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	return NormalizeGoalRequiredTestAttestationRequestV0(GoalRequiredTestAttestationRequestV0{
		RunRef:                   spec.RunRef,
		GoalRef:                  spec.GoalRef,
		ImplementerAgentRef:      spec.ImplementerAgentRef,
		ImplementerCredentialRef: spec.ImplementerCredentialRef,
		AttestorTrustPolicyRef:   spec.ClosurePolicy.RequiredAttestorTrustPolicyRef,
		FinalSnapshot:            snapshot,
		RequiredTests:            append([]GoalRequiredTestV0(nil), spec.RequiredTests...),
	})
}

func NormalizeGoalRequiredTestAttestationRequestV0(request GoalRequiredTestAttestationRequestV0) GoalRequiredTestAttestationRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ImplementerAgentRef = strings.TrimSpace(request.ImplementerAgentRef)
	request.ImplementerCredentialRef = strings.TrimSpace(request.ImplementerCredentialRef)
	request.AttestorTrustPolicyRef = strings.TrimSpace(request.AttestorTrustPolicyRef)
	request.FinalSnapshot = NormalizeGoalRequiredTestFinalSnapshotV0(request.FinalSnapshot)
	request.RequiredTests = append([]GoalRequiredTestV0(nil), request.RequiredTests...)
	for i := range request.RequiredTests {
		request.RequiredTests[i] = normalizeGoalRequiredTestAttestationTestV0(request.RequiredTests[i])
	}
	return request
}

func ValidateGoalRequiredTestAttestationRequestV0(request GoalRequiredTestAttestationRequestV0) []GoalWorkIssueV0 {
	request = NormalizeGoalRequiredTestAttestationRequestV0(request)
	issues := []GoalWorkIssueV0{}
	for field, value := range map[string]string{
		"run_ref": request.RunRef, "goal_ref": request.GoalRef, "implementer_agent_ref": request.ImplementerAgentRef,
		"implementer_credential_ref": request.ImplementerCredentialRef, "attestor_trust_policy_ref": request.AttestorTrustPolicyRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
	}
	if snapshotIssues := ValidateGoalRequiredTestFinalSnapshotV0(request.FinalSnapshot); len(snapshotIssues) > 0 ||
		request.FinalSnapshot.RunRef != request.RunRef || request.FinalSnapshot.GoalRef != request.GoalRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "final_snapshot"})
	}
	if len(request.RequiredTests) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "required_tests"})
	}
	for _, test := range request.RequiredTests {
		validateFrozenGoalRequiredTestV0(&issues, test, "required_tests")
	}
	return issues
}

func NormalizeGoalRequiredTestAttestationV0(attestation GoalRequiredTestAttestationV0) GoalRequiredTestAttestationV0 {
	attestation.SchemaVersion = GoalRequiredTestAttestationSchemaV0
	attestation.AttestationRef = strings.TrimSpace(attestation.AttestationRef)
	attestation.RunRef = strings.TrimSpace(attestation.RunRef)
	attestation.GoalRef = strings.TrimSpace(attestation.GoalRef)
	attestation.FinalSnapshotRef = strings.TrimSpace(attestation.FinalSnapshotRef)
	attestation.CheckoutRef = strings.TrimSpace(attestation.CheckoutRef)
	attestation.RevisionRef = strings.TrimSpace(attestation.RevisionRef)
	attestation.WriteSetSHA256 = strings.ToLower(strings.TrimSpace(attestation.WriteSetSHA256))
	attestation.TestRef = strings.TrimSpace(attestation.TestRef)
	attestation.CommandRef = strings.TrimSpace(attestation.CommandRef)
	attestation.CommandSHA256 = strings.ToLower(strings.TrimSpace(attestation.CommandSHA256))
	attestation.DefinitionSHA256 = strings.ToLower(strings.TrimSpace(attestation.DefinitionSHA256))
	attestation.Status = strings.TrimSpace(attestation.Status)
	attestation.ImplementerAgentRef = strings.TrimSpace(attestation.ImplementerAgentRef)
	attestation.AttestorAgentRef = strings.TrimSpace(attestation.AttestorAgentRef)
	attestation.AttestorCredentialRef = strings.TrimSpace(attestation.AttestorCredentialRef)
	attestation.StartedAt = strings.TrimSpace(attestation.StartedAt)
	attestation.FinishedAt = strings.TrimSpace(attestation.FinishedAt)
	attestation.IsolatedEnvironmentRef = strings.TrimSpace(attestation.IsolatedEnvironmentRef)
	attestation.HashesBefore = normalizeGoalAttestedHashesV0(attestation.HashesBefore)
	attestation.HashesAfter = normalizeGoalAttestedHashesV0(attestation.HashesAfter)
	attestation.EvidenceRefs = compactGoalStringsV0(attestation.EvidenceRefs)
	return attestation
}

func ValidateGoalRequiredTestAttestationV0(attestation GoalRequiredTestAttestationV0) []GoalWorkIssueV0 {
	attestation = NormalizeGoalRequiredTestAttestationV0(attestation)
	issues := []GoalWorkIssueV0{}
	for field, value := range map[string]string{
		"attestation_ref": attestation.AttestationRef, "run_ref": attestation.RunRef, "goal_ref": attestation.GoalRef,
		"final_snapshot_ref": attestation.FinalSnapshotRef, "checkout_ref": attestation.CheckoutRef, "revision_ref": attestation.RevisionRef,
		"test_ref": attestation.TestRef, "implementer_agent_ref": attestation.ImplementerAgentRef,
		"attestor_agent_ref": attestation.AttestorAgentRef, "attestor_credential_ref": attestation.AttestorCredentialRef,
		"isolated_environment_ref": attestation.IsolatedEnvironmentRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
	}
	validateGoalRefsV0(&issues, "command_ref", attestation.CommandRef)
	if !validGoalSHA256V0(attestation.CommandSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "command_sha256"})
	}
	if attestation.AttestationRef != GoalRequiredTestAttestationCanonicalRefV0(attestation) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "attestation_ref"})
	}
	if !validGoalSHA256V0(attestation.DefinitionSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "definition_sha256"})
	}
	if attestation.Status != GoalRequiredTestAttestationStatusPassedV0 && attestation.Status != GoalRequiredTestAttestationStatusFailedV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "status"})
	}
	if !validGoalSHA256V0(attestation.WriteSetSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "write_set_sha256"})
	}
	if len(attestation.HashesBefore) == 0 || len(attestation.HashesAfter) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "hashes_before_after"})
	}
	for _, hash := range append(append([]GoalAttestedHashV0(nil), attestation.HashesBefore...), attestation.HashesAfter...) {
		validateRequiredGoalRefV0(&issues, "hashes.ref", hash.Ref)
		if !validGoalSHA256V0(hash.SHA256) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "hashes.sha256"})
		}
	}
	started, startedErr := time.Parse(time.RFC3339Nano, attestation.StartedAt)
	finished, finishedErr := time.Parse(time.RFC3339Nano, attestation.FinishedAt)
	if startedErr != nil || finishedErr != nil || finished.Before(started) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "timestamps"})
	}
	if attestation.Status == GoalRequiredTestAttestationStatusPassedV0 && attestation.ExitCode != 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationFailedV0, Field: "exit_code"})
	}
	for _, ref := range attestation.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", ref)
	}
	if len(attestation.EvidenceRefs) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "evidence_refs"})
	}
	return issues
}

func independentGoalRequiredTestAttestationBindingsV0(
	spec GoalWorkSpecV0,
	snapshot GoalRequiredTestFinalSnapshotV0,
	attestations []GoalRequiredTestAttestationV0,
) ([]GoalRequiredTestAttestationV0, GoalWorkIssueV0) {
	spec = NormalizeGoalWorkSpecV0(spec)
	snapshot = NormalizeGoalRequiredTestFinalSnapshotV0(snapshot)
	bindings := make([]GoalRequiredTestAttestationV0, 0, len(spec.RequiredTests))
	for _, required := range spec.RequiredTests {
		matched := []GoalRequiredTestAttestationV0{}
		for _, raw := range attestations {
			attestation := NormalizeGoalRequiredTestAttestationV0(raw)
			if attestation.TestRef != required.TestRef {
				continue
			}
			matched = append(matched, attestation)
			if issues := ValidateGoalRequiredTestAttestationV0(attestation); len(issues) > 0 ||
				attestation.RunRef != spec.RunRef || attestation.GoalRef != spec.GoalRef ||
				attestation.FinalSnapshotRef != snapshot.SnapshotRef || attestation.RevisionRef != snapshot.RevisionRef ||
				attestation.CheckoutRef != snapshot.CheckoutRef || attestation.WriteSetSHA256 != snapshot.WriteSetSHA256 ||
				!reflect.DeepEqual(attestation.HashesBefore, snapshot.Hashes) ||
				attestation.ImplementerAgentRef != spec.ImplementerAgentRef || attestation.CommandRef != required.CommandRef ||
				attestation.CommandSHA256 != required.CommandSHA256 || attestation.DefinitionSHA256 != required.DefinitionSHA256 {
				return nil, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "required_test_attestations"}
			}
			if !reflect.DeepEqual(attestation.HashesBefore, attestation.HashesAfter) {
				return nil, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "hashes_before_after"}
			}
			if attestation.Status != GoalRequiredTestAttestationStatusPassedV0 || attestation.ExitCode != 0 {
				return nil, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationFailedV0, Field: "required_test_attestations"}
			}
		}
		if len(matched) == 0 {
			return nil, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "required_test_attestations"}
		}
		if len(matched) != 1 {
			return nil, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "contradictory_required_test_attestations"}
		}
		bindings = append(bindings, matched[0])
	}
	return bindings, GoalWorkIssueV0{}
}

func blockedGoalRequiredTestAttestationClosureV0(closure GoalClosureValidationV0, code, field string) GoalClosureValidationV0 {
	closure.Status = GoalStatusBlockedV0
	closure.Accepted = false
	closure.NeedsRework = true
	closure.Issues = append(closure.Issues, GoalWorkIssueV0{Code: code, Field: field})
	return closure
}

func normalizeGoalRequiredTestAttestationTestV0(test GoalRequiredTestV0) GoalRequiredTestV0 {
	test.TestRef = strings.TrimSpace(test.TestRef)
	test.CommandRef = strings.TrimSpace(test.CommandRef)
	test.Command = strings.TrimSpace(test.Command)
	test.CommandSHA256 = strings.ToLower(strings.TrimSpace(test.CommandSHA256))
	test.DefinitionSHA256 = strings.ToLower(strings.TrimSpace(test.DefinitionSHA256))
	return test
}

func validateFrozenGoalRequiredTestV0(issues *[]GoalWorkIssueV0, test GoalRequiredTestV0, field string) {
	test = normalizeGoalRequiredTestAttestationTestV0(test)
	validateRequiredGoalRefV0(issues, field+".test_ref", test.TestRef)
	validateRequiredGoalRefV0(issues, field+".command_ref", test.CommandRef)
	if test.Command == "" || test.CommandSHA256 != goalRequiredTestCommandSHA256V0(test.Command) {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: field + ".command_sha256"})
	}
	if !validGoalSHA256V0(test.CommandSHA256) {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: field + ".command_sha256"})
	}
	if !validGoalSHA256V0(test.DefinitionSHA256) {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: field + ".definition_sha256"})
	} else if test.DefinitionSHA256 != FreezeGoalRequiredTestV0(test).DefinitionSHA256 {
		*issues = append(*issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: field + ".definition_sha256"})
	}
}

func goalRequiredTestCommandSHA256V0(command string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(command)))
	return hex.EncodeToString(sum[:])
}

func normalizeGoalAttestedHashesV0(hashes []GoalAttestedHashV0) []GoalAttestedHashV0 {
	out := append([]GoalAttestedHashV0(nil), hashes...)
	for i := range out {
		out[i].Ref = strings.TrimSpace(out[i].Ref)
		out[i].SHA256 = strings.ToLower(strings.TrimSpace(out[i].SHA256))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ref == out[j].Ref {
			return out[i].SHA256 < out[j].SHA256
		}
		return out[i].Ref < out[j].Ref
	})
	return out
}

func FreezeGoalRequiredTestV0(test GoalRequiredTestV0) GoalRequiredTestV0 {
	test = normalizeGoalRequiredTestAttestationTestV0(test)
	test.CommandSHA256 = goalRequiredTestCommandSHA256V0(test.Command)
	payload := struct {
		TestRef                string   `json:"test_ref"`
		CommandRef             string   `json:"command_ref"`
		CommandSHA256          string   `json:"command_sha256"`
		AcceptanceCriteria     []string `json:"acceptance_criteria,omitempty"`
		AcceptanceCriteriaRefs []string `json:"acceptance_criteria_refs,omitempty"`
	}{test.TestRef, test.CommandRef, test.CommandSHA256, test.AcceptanceCriteria, test.AcceptanceCriteriaRefs}
	encoded, _ := json.Marshal(payload)
	test.DefinitionSHA256 = goalRequiredTestCommandSHA256V0(string(encoded))
	return test
}

func GoalWriteSetSHA256V0(writeSet []GoalWriteScopeV0) string {
	paths := make([]string, 0, len(writeSet))
	for _, scope := range writeSet {
		paths = append(paths, strings.TrimSpace(scope.Path))
	}
	sort.Strings(paths)
	encoded, _ := json.Marshal(paths)
	return goalRequiredTestCommandSHA256V0(string(encoded))
}

func ValidateGoalRequiredTestAttestationBindingV0(spec GoalWorkSpecV0) []GoalWorkIssueV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	if !spec.ClosurePolicy.RequireIndependentRequiredTestAttestation {
		return nil
	}
	issues := []GoalWorkIssueV0{}
	validateRequiredGoalRefV0(&issues, "implementer_agent_ref", spec.ImplementerAgentRef)
	validateRequiredGoalRefV0(&issues, "implementer_credential_ref", spec.ImplementerCredentialRef)
	validateRequiredGoalRefV0(&issues, "closure_policy.required_attestor_trust_policy_ref", spec.ClosurePolicy.RequiredAttestorTrustPolicyRef)
	return issues
}

func NormalizeGoalRequiredTestFinalSnapshotV0(snapshot GoalRequiredTestFinalSnapshotV0) GoalRequiredTestFinalSnapshotV0 {
	snapshot.SchemaVersion = GoalRequiredTestFinalSnapshotSchemaV0
	snapshot.SnapshotRef = strings.TrimSpace(snapshot.SnapshotRef)
	snapshot.RunRef = strings.TrimSpace(snapshot.RunRef)
	snapshot.GoalRef = strings.TrimSpace(snapshot.GoalRef)
	snapshot.CheckoutRef = strings.TrimSpace(snapshot.CheckoutRef)
	snapshot.RevisionRef = strings.TrimSpace(snapshot.RevisionRef)
	snapshot.WriteSetSHA256 = strings.ToLower(strings.TrimSpace(snapshot.WriteSetSHA256))
	snapshot.Hashes = normalizeGoalAttestedHashesV0(snapshot.Hashes)
	snapshot.ObservedAt = strings.TrimSpace(snapshot.ObservedAt)
	snapshot.EvidenceRefs = compactGoalStringsV0(snapshot.EvidenceRefs)
	return snapshot
}

func FreezeGoalRequiredTestFinalSnapshotV0(snapshot GoalRequiredTestFinalSnapshotV0) GoalRequiredTestFinalSnapshotV0 {
	snapshot = NormalizeGoalRequiredTestFinalSnapshotV0(snapshot)
	payload := struct {
		RunRef         string               `json:"run_ref"`
		GoalRef        string               `json:"goal_ref"`
		CheckoutRef    string               `json:"checkout_ref"`
		RevisionRef    string               `json:"revision_ref"`
		WriteSetSHA256 string               `json:"write_set_sha256"`
		Hashes         []GoalAttestedHashV0 `json:"hashes"`
	}{snapshot.RunRef, snapshot.GoalRef, snapshot.CheckoutRef, snapshot.RevisionRef, snapshot.WriteSetSHA256, snapshot.Hashes}
	encoded, _ := json.Marshal(payload)
	snapshot.SnapshotRef = "goal-required-test-final-snapshot-ref-" + goalRequiredTestCommandSHA256V0(string(encoded))
	return snapshot
}

func GoalRequiredTestFinalSnapshotIdentityEqualV0(
	left GoalRequiredTestFinalSnapshotV0,
	right GoalRequiredTestFinalSnapshotV0,
) bool {
	left = NormalizeGoalRequiredTestFinalSnapshotV0(left)
	right = NormalizeGoalRequiredTestFinalSnapshotV0(right)
	return left.SnapshotRef == right.SnapshotRef && left.RunRef == right.RunRef && left.GoalRef == right.GoalRef &&
		left.CheckoutRef == right.CheckoutRef && left.RevisionRef == right.RevisionRef &&
		left.WriteSetSHA256 == right.WriteSetSHA256 && reflect.DeepEqual(left.Hashes, right.Hashes)
}

func ValidateGoalRequiredTestFinalSnapshotRequestV0(request GoalRequiredTestFinalSnapshotRequestV0) []GoalWorkIssueV0 {
	issues := []GoalWorkIssueV0{}
	validateRequiredGoalRefV0(&issues, "run_ref", strings.TrimSpace(request.RunRef))
	validateRequiredGoalRefV0(&issues, "goal_ref", strings.TrimSpace(request.GoalRef))
	if len(request.WriteSet) == 0 || strings.ToLower(strings.TrimSpace(request.WriteSetSHA256)) != GoalWriteSetSHA256V0(request.WriteSet) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "write_set"})
	}
	return issues
}

func ValidateGoalRequiredTestFinalSnapshotV0(snapshot GoalRequiredTestFinalSnapshotV0) []GoalWorkIssueV0 {
	snapshot = NormalizeGoalRequiredTestFinalSnapshotV0(snapshot)
	issues := []GoalWorkIssueV0{}
	for field, value := range map[string]string{
		"snapshot_ref": snapshot.SnapshotRef, "run_ref": snapshot.RunRef, "goal_ref": snapshot.GoalRef,
		"checkout_ref": snapshot.CheckoutRef, "revision_ref": snapshot.RevisionRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
	}
	if !validGoalSHA256V0(snapshot.WriteSetSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "write_set_sha256"})
	}
	if len(snapshot.Hashes) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMissingV0, Field: "hashes"})
	}
	seen := map[string]bool{}
	for _, hash := range snapshot.Hashes {
		validateRequiredGoalRefV0(&issues, "hashes.ref", hash.Ref)
		if seen[hash.Ref] || !validGoalSHA256V0(hash.SHA256) {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "hashes"})
		}
		seen[hash.Ref] = true
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshot.ObservedAt); err != nil {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "observed_at"})
	}
	if len(snapshot.EvidenceRefs) == 0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMissingV0, Field: "evidence_refs"})
	}
	for _, ref := range snapshot.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", ref)
	}
	expected := FreezeGoalRequiredTestFinalSnapshotV0(snapshot)
	if snapshot.SnapshotRef != expected.SnapshotRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestSnapshotMismatchV0, Field: "snapshot_ref"})
	}
	return issues
}

func GoalRequiredTestAttestationClaimRefV0(request GoalRequiredTestAttestationClaimRequestV0) string {
	return "goal-required-test-attestation-claim-ref-" + goalRequiredTestCommandSHA256V0(strings.Join([]string{
		strings.TrimSpace(request.RunRef), strings.TrimSpace(request.GoalRef), strings.TrimSpace(request.RevisionRef),
		strings.TrimSpace(request.TestRef), strings.ToLower(strings.TrimSpace(request.DefinitionSHA256)),
	}, "\x00"))
}

func GoalRequiredTestAttestationCanonicalRefV0(attestation GoalRequiredTestAttestationV0) string {
	return "goal-required-test-attestation-ref-" + goalRequiredTestCommandSHA256V0(strings.Join([]string{
		strings.TrimSpace(attestation.RunRef), strings.TrimSpace(attestation.GoalRef), strings.TrimSpace(attestation.RevisionRef),
		strings.TrimSpace(attestation.TestRef), strings.ToLower(strings.TrimSpace(attestation.DefinitionSHA256)),
	}, "\x00"))
}

func NormalizeGoalRequiredTestAttestationClaimV0(claim GoalRequiredTestAttestationClaimV0) GoalRequiredTestAttestationClaimV0 {
	claim.SchemaVersion = GoalRequiredTestAttestationClaimSchemaV0
	claim.ClaimRef = strings.TrimSpace(claim.ClaimRef)
	claim.RunRef = strings.TrimSpace(claim.RunRef)
	claim.GoalRef = strings.TrimSpace(claim.GoalRef)
	claim.RevisionRef = strings.TrimSpace(claim.RevisionRef)
	claim.TestRef = strings.TrimSpace(claim.TestRef)
	claim.DefinitionSHA256 = strings.ToLower(strings.TrimSpace(claim.DefinitionSHA256))
	claim.Status = strings.TrimSpace(claim.Status)
	claim.AttestationRef = strings.TrimSpace(claim.AttestationRef)
	claim.ClaimedAt = strings.TrimSpace(claim.ClaimedAt)
	claim.CompletedAt = strings.TrimSpace(claim.CompletedAt)
	claim.FailedAt = strings.TrimSpace(claim.FailedAt)
	claim.FailureCode = strings.TrimSpace(claim.FailureCode)
	return claim
}

func ValidateGoalRequiredTestAttestationClaimV0(claim GoalRequiredTestAttestationClaimV0) []GoalWorkIssueV0 {
	claim = NormalizeGoalRequiredTestAttestationClaimV0(claim)
	issues := []GoalWorkIssueV0{}
	for field, value := range map[string]string{
		"claim_ref": claim.ClaimRef, "run_ref": claim.RunRef, "goal_ref": claim.GoalRef,
		"revision_ref": claim.RevisionRef, "test_ref": claim.TestRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
	}
	request := GoalRequiredTestAttestationClaimRequestV0{
		RunRef: claim.RunRef, GoalRef: claim.GoalRef, RevisionRef: claim.RevisionRef,
		TestRef: claim.TestRef, DefinitionSHA256: claim.DefinitionSHA256,
	}
	if !validGoalSHA256V0(claim.DefinitionSHA256) || claim.ClaimRef != GoalRequiredTestAttestationClaimRefV0(request) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "claim_identity"})
	}
	if claim.Status != GoalRequiredTestAttestationClaimStatusPendingV0 && claim.Status != GoalRequiredTestAttestationClaimStatusCompletedV0 && claim.Status != GoalRequiredTestAttestationClaimStatusFailedV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "claim_status"})
	}
	if _, err := time.Parse(time.RFC3339Nano, claim.ClaimedAt); err != nil {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "claimed_at"})
	}
	if claim.Status == GoalRequiredTestAttestationClaimStatusCompletedV0 {
		validateRequiredGoalRefV0(&issues, "attestation_ref", claim.AttestationRef)
		if _, err := time.Parse(time.RFC3339Nano, claim.CompletedAt); err != nil {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "completed_at"})
		}
	}
	if claim.Status == GoalRequiredTestAttestationClaimStatusFailedV0 {
		validateRequiredGoalRefV0(&issues, "failure_code", claim.FailureCode)
		if _, err := time.Parse(time.RFC3339Nano, claim.FailedAt); err != nil {
			issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "failed_at"})
		}
	}
	return issues
}

func MissingGoalRequiredTestsForAttestationV0(
	spec GoalWorkSpecV0,
	attestations []GoalRequiredTestAttestationV0,
) []GoalRequiredTestV0 {
	present := map[string]bool{}
	for _, attestation := range attestations {
		present[strings.TrimSpace(attestation.TestRef)] = true
	}
	missing := []GoalRequiredTestV0{}
	for _, test := range NormalizeGoalWorkSpecV0(spec).RequiredTests {
		if !present[test.TestRef] {
			missing = append(missing, test)
		}
	}
	return missing
}

func goalRequiredTestAttestationResponseIssueV0(
	request GoalRequiredTestAttestationRequestV0,
	attestations []GoalRequiredTestAttestationV0,
) GoalWorkIssueV0 {
	request = NormalizeGoalRequiredTestAttestationRequestV0(request)
	wanted := map[string]GoalRequiredTestV0{}
	for _, test := range request.RequiredTests {
		wanted[test.TestRef] = test
	}
	seen := map[string]bool{}
	for _, raw := range attestations {
		attestation := NormalizeGoalRequiredTestAttestationV0(raw)
		test, ok := wanted[attestation.TestRef]
		if !ok || seen[attestation.TestRef] || len(ValidateGoalRequiredTestAttestationV0(attestation)) > 0 ||
			attestation.RunRef != request.RunRef || attestation.GoalRef != request.GoalRef ||
			attestation.FinalSnapshotRef != request.FinalSnapshot.SnapshotRef ||
			attestation.CheckoutRef != request.FinalSnapshot.CheckoutRef || attestation.RevisionRef != request.FinalSnapshot.RevisionRef ||
			attestation.ImplementerAgentRef != request.ImplementerAgentRef ||
			attestation.WriteSetSHA256 != request.FinalSnapshot.WriteSetSHA256 ||
			attestation.CommandRef != test.CommandRef || attestation.CommandSHA256 != test.CommandSHA256 ||
			attestation.DefinitionSHA256 != test.DefinitionSHA256 {
			return GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "required_test_attestations"}
		}
		seen[attestation.TestRef] = true
	}
	if len(seen) != len(wanted) {
		return GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "required_test_attestations"}
	}
	return GoalWorkIssueV0{}
}

func normalizeGoalRequiredTestIdentityVerificationV0(
	verification GoalRequiredTestIdentityVerificationV0,
) GoalRequiredTestIdentityVerificationV0 {
	verification.AttestationRef = strings.TrimSpace(verification.AttestationRef)
	verification.TestRef = strings.TrimSpace(verification.TestRef)
	verification.ImplementerPrincipalRef = strings.TrimSpace(verification.ImplementerPrincipalRef)
	verification.AttestorPrincipalRef = strings.TrimSpace(verification.AttestorPrincipalRef)
	verification.AttestorCredentialRef = strings.TrimSpace(verification.AttestorCredentialRef)
	verification.TrustPolicyRef = strings.TrimSpace(verification.TrustPolicyRef)
	verification.EvidenceRefs = compactGoalStringsV0(verification.EvidenceRefs)
	return verification
}

func validGoalRequiredTestIdentityVerificationV0(
	spec GoalWorkSpecV0,
	attestation GoalRequiredTestAttestationV0,
	verification GoalRequiredTestIdentityVerificationV0,
) bool {
	if !verification.Verified || !verification.Independent ||
		verification.AttestationRef != attestation.AttestationRef ||
		verification.TestRef != attestation.TestRef ||
		verification.AttestorCredentialRef != attestation.AttestorCredentialRef ||
		verification.TrustPolicyRef != spec.ClosurePolicy.RequiredAttestorTrustPolicyRef ||
		verification.ImplementerPrincipalRef == "" || verification.AttestorPrincipalRef == "" ||
		len(verification.EvidenceRefs) == 0 {
		return false
	}
	for _, ref := range verification.EvidenceRefs {
		if !validGoalRefTokenV0(ref) {
			return false
		}
	}
	return true
}

func validGoalSHA256V0(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}
