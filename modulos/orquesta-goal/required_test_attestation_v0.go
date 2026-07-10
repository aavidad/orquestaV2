package orquestagoal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	Base   GoalWorkClosureValidatorPortV0
	Reader GoalRequiredTestAttestationReaderPortV0
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
	if validator.Reader == nil {
		return blockedGoalRequiredTestAttestationClosureV0(closure, ErrGoalRequiredTestAttestationMissingV0, "required_test_attestations"), nil
	}
	attestations, err := validator.Reader.ListGoalRequiredTestAttestationsV0(ctx, GoalRequiredTestAttestationQueryV0{
		RunRef:      spec.RunRef,
		GoalRef:     spec.GoalRef,
		RevisionRef: spec.RevisionRef,
	})
	if err != nil {
		return GoalClosureValidationV0{}, err
	}
	if issue := independentGoalRequiredTestAttestationIssueV0(spec, attestations); issue.Code != "" {
		return blockedGoalRequiredTestAttestationClosureV0(closure, issue.Code, issue.Field), nil
	}
	for _, attestation := range attestations {
		closure.EvidenceRefs = append(closure.EvidenceRefs, attestation.AttestationRef)
		closure.EvidenceRefs = append(closure.EvidenceRefs, attestation.EvidenceRefs...)
	}
	closure.EvidenceRefs = compactGoalStringsV0(closure.EvidenceRefs)
	return closure, nil
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
	for _, attestation := range attestations {
		if err := store.SaveGoalRequiredTestAttestationV0(ctx, attestation); err != nil {
			return nil, err
		}
	}
	return attestations, nil
}

func GoalRequiredTestAttestationRequestFromSpecV0(spec GoalWorkSpecV0) GoalRequiredTestAttestationRequestV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	return NormalizeGoalRequiredTestAttestationRequestV0(GoalRequiredTestAttestationRequestV0{
		RunRef:              spec.RunRef,
		GoalRef:             spec.GoalRef,
		RevisionRef:         spec.RevisionRef,
		ImplementerAgentRef: spec.ImplementerAgentRef,
		RequiredTests:       append([]GoalRequiredTestV0(nil), spec.RequiredTests...),
	})
}

func NormalizeGoalRequiredTestAttestationRequestV0(request GoalRequiredTestAttestationRequestV0) GoalRequiredTestAttestationRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.RevisionRef = strings.TrimSpace(request.RevisionRef)
	request.ImplementerAgentRef = strings.TrimSpace(request.ImplementerAgentRef)
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
		"run_ref": request.RunRef, "goal_ref": request.GoalRef, "revision_ref": request.RevisionRef, "implementer_agent_ref": request.ImplementerAgentRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
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
	attestation.RevisionRef = strings.TrimSpace(attestation.RevisionRef)
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
		"revision_ref": attestation.RevisionRef, "test_ref": attestation.TestRef, "implementer_agent_ref": attestation.ImplementerAgentRef,
		"attestor_agent_ref": attestation.AttestorAgentRef, "attestor_credential_ref": attestation.AttestorCredentialRef,
		"isolated_environment_ref": attestation.IsolatedEnvironmentRef,
	} {
		validateRequiredGoalRefV0(&issues, field, value)
	}
	validateGoalRefsV0(&issues, "command_ref", attestation.CommandRef)
	if !validGoalSHA256V0(attestation.CommandSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "command_sha256"})
	}
	if !validGoalSHA256V0(attestation.DefinitionSHA256) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "definition_sha256"})
	}
	if attestation.Status != GoalRequiredTestAttestationStatusPassedV0 && attestation.Status != GoalRequiredTestAttestationStatusFailedV0 {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "status"})
	}
	if attestation.AttestorAgentRef == attestation.ImplementerAgentRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "attestor_agent_ref"})
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

func independentGoalRequiredTestAttestationIssueV0(spec GoalWorkSpecV0, attestations []GoalRequiredTestAttestationV0) GoalWorkIssueV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	for _, required := range spec.RequiredTests {
		matched := false
		for _, raw := range attestations {
			attestation := NormalizeGoalRequiredTestAttestationV0(raw)
			if attestation.TestRef != required.TestRef {
				continue
			}
			matched = true
			if issues := ValidateGoalRequiredTestAttestationV0(attestation); len(issues) > 0 ||
				attestation.RunRef != spec.RunRef || attestation.GoalRef != spec.GoalRef || attestation.RevisionRef != spec.RevisionRef ||
				attestation.ImplementerAgentRef != spec.ImplementerAgentRef || attestation.CommandRef != required.CommandRef ||
				attestation.CommandSHA256 != required.CommandSHA256 || attestation.DefinitionSHA256 != required.DefinitionSHA256 {
				return GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMismatchV0, Field: "required_test_attestations"}
			}
			if attestation.Status != GoalRequiredTestAttestationStatusPassedV0 || attestation.ExitCode != 0 {
				return GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationFailedV0, Field: "required_test_attestations"}
			}
			break
		}
		if !matched {
			return GoalWorkIssueV0{Code: ErrGoalRequiredTestAttestationMissingV0, Field: "required_test_attestations"}
		}
	}
	return GoalWorkIssueV0{}
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
	return out
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
