package ports

import (
	"bytes"
	"encoding/json"
)

const (
	TestSubjectManifestMediaType    = "application/vnd.orquesta.test-subject-manifest+json;v=1"
	TestAttestationReportMediaType  = "application/vnd.orquesta.test-attestation-report+json;v=1"
	maxTestAttestationArtifactBytes = 1024 * 1024
)

type TestAttestationReportInput struct {
	SubjectDigest string
	Verdict       TestAttestationVerdict
	Tests         []RequiredTestOutcome
	AttestorRef   string
	PolicyRef     string
	PolicyDigest  string
}

func BuildTestSubjectManifest(subject TestSubject) (PutArtifactRequest, error) {
	if err := validateTestSubject(subject); err != nil {
		return PutArtifactRequest{}, err
	}
	payload := testSubjectManifestProjection(subject)
	content, err := json.Marshal(payload)
	if err != nil {
		return PutArtifactRequest{}, &TestAttestorContractError{Code: "test_attestor.manifest_invalid"}
	}
	return PutArtifactRequest{MediaType: TestSubjectManifestMediaType, Content: content}, nil
}

type testSubjectManifest struct {
	Schema                 string `json:"schema"`
	SubjectDigest          string `json:"subject_digest"`
	GoalRef                string `json:"goal_ref"`
	WorkItemRef            string `json:"work_item_ref"`
	ExecutionRef           string `json:"execution_ref"`
	ExecutionAttempt       uint64 `json:"execution_attempt"`
	PlanGeneration         uint64 `json:"plan_generation"`
	WorkItemGeneration     uint64 `json:"work_item_generation"`
	AppSpecGeneration      uint64 `json:"app_spec_generation"`
	AppSpecHash            string `json:"app_spec_hash"`
	WorkspaceRef           string `json:"workspace_ref"`
	WorkspaceBindingDigest string `json:"workspace_binding_digest"`
	ChangeSetRef           string `json:"change_set_ref"`
	ChangeSetDigest        string `json:"change_set_digest"`
	RepositoryRef          string `json:"repository_ref"`
	ObjectFormat           string `json:"object_format"`
	BaseOID                string `json:"base_oid"`
	ParentOID              string `json:"parent_oid"`
	HeadOID                string `json:"head_oid"`
	TreeOID                string `json:"tree_oid"`
	DiffDigest             string `json:"diff_digest"`
	WriteSetDigest         string `json:"write_set_digest"`
	RequiredTestsDigest    string `json:"required_tests_digest"`
	PolicyRef              string `json:"policy_ref"`
	PolicyDigest           string `json:"policy_digest"`
}

func testSubjectManifestProjection(subject TestSubject) testSubjectManifest {
	return testSubjectManifest{
		Schema: "orquesta.test-subject-manifest.v1", SubjectDigest: TestSubjectDigest(subject),
		GoalRef: subject.GoalRef.String(), WorkItemRef: subject.WorkItemRef.String(),
		ExecutionRef: subject.ExecutionRef.String(), ExecutionAttempt: subject.ExecutionAttempt,
		PlanGeneration: uint64(subject.PlanGeneration), WorkItemGeneration: uint64(subject.WorkItemGeneration),
		AppSpecGeneration: uint64(subject.AppSpecGeneration), AppSpecHash: subject.AppSpecHash,
		WorkspaceRef: subject.WorkspaceRef.String(), WorkspaceBindingDigest: subject.WorkspaceBindingDigest,
		ChangeSetRef: subject.ChangeSetRef.String(), ChangeSetDigest: subject.ChangeSetDigest,
		RepositoryRef: subject.RepositoryRef.String(), ObjectFormat: string(subject.ObjectFormat),
		BaseOID: subject.BaseOID, ParentOID: subject.ParentOID, HeadOID: subject.HeadOID,
		TreeOID: subject.TreeOID, DiffDigest: subject.DiffDigest, WriteSetDigest: subject.WriteSetDigest,
		RequiredTestsDigest: subject.RequiredTestsDigest, PolicyRef: subject.PolicyRef, PolicyDigest: subject.PolicyDigest,
	}
}

func BuildTestAttestationReport(input TestAttestationReportInput) (PutArtifactRequest, error) {
	tests, err := testReportOutcomes(input)
	if err != nil {
		return PutArtifactRequest{}, err
	}
	payload := testAttestationReport{
		Schema: "orquesta.test-attestation-report.v1", SubjectDigest: input.SubjectDigest,
		Verdict: string(input.Verdict), Tests: tests, AttestorRef: input.AttestorRef,
		PolicyRef: input.PolicyRef, PolicyDigest: input.PolicyDigest,
	}
	content, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return PutArtifactRequest{}, &TestAttestorContractError{Code: "test_attestor.report_invalid"}
	}
	return PutArtifactRequest{MediaType: TestAttestationReportMediaType, Content: content}, nil
}

type testAttestationReport struct {
	Schema        string              `json:"schema"`
	SubjectDigest string              `json:"subject_digest"`
	Verdict       string              `json:"verdict"`
	Tests         []testReportOutcome `json:"tests"`
	AttestorRef   string              `json:"attestor_ref"`
	PolicyRef     string              `json:"policy_ref"`
	PolicyDigest  string              `json:"policy_digest"`
}

type testReportOutcome struct {
	RequiredTestRef string `json:"required_test_ref"`
	ExitCode        int    `json:"exit_code"`
	OutputDigest    string `json:"output_digest"`
}

func testReportOutcomes(input TestAttestationReportInput) ([]testReportOutcome, error) {
	if !validWorkspaceDigest(input.SubjectDigest) || !validTestVerdict(input.Verdict) || len(input.Tests) == 0 ||
		!validWorkspaceLogicalRef(input.AttestorRef) || !validWorkspaceLogicalRef(input.PolicyRef) ||
		!validWorkspaceDigest(input.PolicyDigest) {
		return nil, &TestAttestorContractError{Code: "test_attestor.report_invalid"}
	}
	tests := make([]testReportOutcome, len(input.Tests))
	hasFailure := false
	for index, outcome := range input.Tests {
		if outcome.RequiredTestRef.String() == "" || outcome.ExitCode < 0 || outcome.ExitCode > 255 ||
			!validWorkspaceDigest(outcome.OutputDigest) {
			return nil, &TestAttestorContractError{Code: "test_attestor.report_invalid"}
		}
		tests[index] = testReportOutcome{outcome.RequiredTestRef.String(), outcome.ExitCode, outcome.OutputDigest}
		hasFailure = hasFailure || outcome.ExitCode != 0
	}
	if (input.Verdict == TestAttestationPassed) == hasFailure {
		return nil, &TestAttestorContractError{Code: "test_attestor.report_invalid"}
	}
	return tests, nil
}

func validTestAttestationArtifact(request PutArtifactRequest, mediaType string) bool {
	return request.MediaType == mediaType && len(request.Content) > 0 &&
		len(request.Content) <= maxTestAttestationArtifactBytes && ValidateArtifactMediaType(request.MediaType) == nil
}

func equalTestAttestationArtifact(got, want PutArtifactRequest) bool {
	return validTestAttestationArtifact(got, want.MediaType) && bytes.Equal(got.Content, want.Content)
}
