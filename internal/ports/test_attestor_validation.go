package ports

import (
	"errors"

	"orquesta/internal/goal"
)

type TestAttestorContractError struct{ Code string }

func (err *TestAttestorContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func TestAttestorContractErrorCode(err error) string {
	var contractErr *TestAttestorContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateTestAttestationRequest(request TestAttestationRequest) error {
	if err := validateTestSubject(request.Subject); err != nil {
		return err
	}
	if request.SubjectDigest != TestSubjectDigest(request.Subject) {
		return &TestAttestorContractError{Code: "test_attestor.subject_digest_mismatch"}
	}
	if request.RequestedAt.IsZero() || !validWorkspaceLogicalRef(request.IdempotencyKey) || len(request.RequiredTests) == 0 {
		return &TestAttestorContractError{Code: "test_attestor.request_invalid"}
	}
	seen := make(map[goal.RequiredTestRef]struct{}, len(request.RequiredTests))
	for _, testSpec := range request.RequiredTests {
		canonical, err := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
			Ref: testSpec.Ref(), ToolRef: testSpec.ToolRef(), Arguments: testSpec.Arguments(),
			WorkingDirectory: testSpec.WorkingDirectory(),
		})
		if err != nil || canonical.Digest() != testSpec.Digest() {
			return &TestAttestorContractError{Code: "test_attestor.required_tests_invalid"}
		}
		if _, duplicate := seen[testSpec.Ref()]; duplicate {
			return &TestAttestorContractError{Code: "test_attestor.required_tests_invalid"}
		}
		seen[testSpec.Ref()] = struct{}{}
	}
	if request.Subject.RequiredTestsDigest != goal.RequiredTestsDigest(request.RequiredTests) {
		return &TestAttestorContractError{Code: "test_attestor.required_tests_mismatch"}
	}
	return nil
}

func ValidateTestAttestationResult(request TestAttestationRequest, result TestAttestationResult) error {
	if err := validateTestResultIdentity(request, result); err != nil {
		return err
	}
	wantManifest, manifestErr := BuildTestSubjectManifest(request.Subject)
	wantReport, reportErr := BuildTestAttestationReport(TestAttestationReportInput{
		SubjectDigest: result.SubjectDigest, Verdict: result.Verdict, Tests: result.Tests,
		AttestorRef: result.AttestorRef, PolicyRef: result.PolicyRef, PolicyDigest: result.PolicyDigest,
	})
	if manifestErr != nil || reportErr != nil || !equalTestAttestationArtifact(result.Manifest, wantManifest) ||
		!equalTestAttestationArtifact(result.Report, wantReport) {
		return &TestAttestorContractError{Code: "test_attestor.artifact_invalid"}
	}
	wantReceipt, receiptErr := TestAttestationReceiptRef(wantReport)
	if receiptErr != nil || result.ReceiptRef != wantReceipt {
		return &TestAttestorContractError{Code: "test_attestor.identity_mismatch"}
	}
	if err := ValidateRequiredTestOutcomes(request.RequiredTests, result.Verdict, result.Tests); err != nil {
		return err
	}
	if result.StartedAt.IsZero() || result.FinishedAt.IsZero() ||
		result.StartedAt.Before(request.RequestedAt) || result.FinishedAt.Before(result.StartedAt) {
		return &TestAttestorContractError{Code: "test_attestor.timestamps_invalid"}
	}
	return nil
}

func validateTestResultIdentity(request TestAttestationRequest, result TestAttestationResult) error {
	if err := ValidateTestAttestationRequest(request); err != nil {
		return err
	}
	if result.Subject != request.Subject || result.SubjectDigest != TestSubjectDigest(request.Subject) {
		return &TestAttestorContractError{Code: "test_attestor.subject_mismatch"}
	}
	if !validTestVerdict(result.Verdict) {
		return &TestAttestorContractError{Code: "test_attestor.verdict_invalid"}
	}
	if !validWorkspaceLogicalRef(result.AttestorRef) || !validWorkspaceLogicalRef(result.ReceiptRef) ||
		result.PolicyRef != request.Subject.PolicyRef || result.PolicyDigest != request.Subject.PolicyDigest {
		return &TestAttestorContractError{Code: "test_attestor.identity_mismatch"}
	}
	return nil
}

// ValidateRequiredTestOutcomes is the canonical required-test evidence check
// shared by the attestor boundary, application gate, and state adapter.
func ValidateRequiredTestOutcomes(
	required []goal.RequiredTestSpec,
	verdict TestAttestationVerdict,
	outcomes []RequiredTestOutcome,
) error {
	if len(required) == 0 || len(outcomes) != len(required) {
		return &TestAttestorContractError{Code: "test_attestor.outcomes_invalid"}
	}
	hasFailure := false
	for index, outcome := range outcomes {
		if outcome.RequiredTestRef != required[index].Ref() || outcome.ExitCode < 0 ||
			outcome.ExitCode > 255 || !validWorkspaceDigest(outcome.OutputDigest) {
			return &TestAttestorContractError{Code: "test_attestor.outcomes_invalid"}
		}
		hasFailure = hasFailure || outcome.ExitCode != 0
	}
	if !validTestVerdict(verdict) || (verdict == TestAttestationPassed) == hasFailure {
		return &TestAttestorContractError{Code: "test_attestor.verdict_mismatch"}
	}
	return nil
}

func validateTestSubject(subject TestSubject) error {
	if subject.GoalRef.String() == "" || subject.WorkItemRef.String() == "" || subject.ExecutionRef.String() == "" ||
		subject.ExecutionAttempt == 0 || subject.PlanGeneration == 0 || subject.WorkItemGeneration == 0 ||
		subject.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(subject.AppSpecHash) ||
		subject.WorkspaceRef.String() == "" || subject.ChangeSetRef.String() == "" || subject.RepositoryRef.String() == "" {
		return &TestAttestorContractError{Code: "test_attestor.subject_identity_invalid"}
	}
	if !validWorkspaceDigest(subject.WorkspaceBindingDigest) || !validWorkspaceDigest(subject.ChangeSetDigest) ||
		!validGitObjectFormat(subject.ObjectFormat) || !validGitOID(subject.BaseOID, subject.ObjectFormat) ||
		!validGitOID(subject.ParentOID, subject.ObjectFormat) || !validGitOID(subject.HeadOID, subject.ObjectFormat) ||
		!validGitOID(subject.TreeOID, subject.ObjectFormat) || !validWorkspaceDigest(subject.DiffDigest) ||
		!validWorkspaceDigest(subject.WriteSetDigest) || !validWorkspaceDigest(subject.RequiredTestsDigest) ||
		!validWorkspaceLogicalRef(subject.PolicyRef) || !validWorkspaceDigest(subject.PolicyDigest) {
		return &TestAttestorContractError{Code: "test_attestor.subject_binding_invalid"}
	}
	return nil
}

func validTestVerdict(verdict TestAttestationVerdict) bool {
	return verdict == TestAttestationPassed || verdict == TestAttestationFailed
}
