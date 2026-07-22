package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestTestSubjectDigestIsDeterministicAndOperationIndependent(t *testing.T) {
	request := validTestAttestationRequest(t)
	want := TestSubjectDigest(request.Subject)
	if got := TestSubjectDigest(request.Subject); got != want {
		t.Fatalf("digest changed: got=%s want=%s", got, want)
	}
	request.IdempotencyKey = "attest:another-operation"
	request.RequestedAt = request.RequestedAt.Add(time.Hour)
	if got := TestSubjectDigest(request.Subject); got != want {
		t.Fatalf("operation fields changed subject digest: got=%s want=%s", got, want)
	}
}

func TestTestSubjectDigestChangesOnEveryBoundFact(t *testing.T) {
	base := validTestAttestationRequest(t)
	want := TestSubjectDigest(base.Subject)
	otherGoal, _ := goal.NewGoalRef("goal:test-attestor:changed")
	otherWorkItem, _ := goal.NewWorkItemRef("work-item:test-attestor:changed")
	otherExecution, _ := goal.NewExecutionRef("execution:test-attestor:changed")
	otherWorkspace, _ := NewExecutionWorkspaceRef("workspace:test-attestor:changed")
	otherChange, _ := NewChangeSetRef("change:test-attestor:changed")
	otherRepository, _ := identity.NewRepositoryRef("repository:test-attestor:changed")
	mutations := []func(*TestSubject){
		func(subject *TestSubject) { subject.GoalRef = otherGoal },
		func(subject *TestSubject) { subject.WorkItemRef = otherWorkItem },
		func(subject *TestSubject) { subject.ExecutionRef = otherExecution },
		func(subject *TestSubject) { subject.ExecutionAttempt++ },
		func(subject *TestSubject) { subject.PlanGeneration++ },
		func(subject *TestSubject) { subject.WorkItemGeneration++ },
		func(subject *TestSubject) { subject.AppSpecGeneration++ },
		func(subject *TestSubject) { subject.AppSpecHash = testAttestorDigest("spec:changed") },
		func(subject *TestSubject) { subject.WorkspaceRef = otherWorkspace },
		func(subject *TestSubject) { subject.WorkspaceBindingDigest = testAttestorDigest("workspace:changed") },
		func(subject *TestSubject) { subject.ChangeSetRef = otherChange },
		func(subject *TestSubject) { subject.ChangeSetDigest = testAttestorDigest("change:changed") },
		func(subject *TestSubject) { subject.RepositoryRef = otherRepository },
		func(subject *TestSubject) { subject.ObjectFormat = GitObjectFormatSHA256 },
		func(subject *TestSubject) { subject.BaseOID = strings.Repeat("b", 40) },
		func(subject *TestSubject) { subject.ParentOID = strings.Repeat("c", 40) },
		func(subject *TestSubject) { subject.HeadOID = strings.Repeat("d", 40) },
		func(subject *TestSubject) { subject.TreeOID = strings.Repeat("e", 40) },
		func(subject *TestSubject) { subject.DiffDigest = testAttestorDigest("diff:changed") },
		func(subject *TestSubject) { subject.WriteSetDigest = testAttestorDigest("writes:changed") },
		func(subject *TestSubject) { subject.RequiredTestsDigest = testAttestorDigest("tests:changed") },
		func(subject *TestSubject) { subject.PolicyRef = "policy:sandbox:changed" },
		func(subject *TestSubject) { subject.PolicyDigest = testAttestorDigest("policy:changed") },
	}
	seen := map[string]struct{}{want: {}}
	for index, mutate := range mutations {
		subject := base.Subject
		mutate(&subject)
		got := TestSubjectDigest(subject)
		if got == want {
			t.Errorf("mutation %d did not change digest", index)
		}
		if _, duplicate := seen[got]; duplicate {
			t.Errorf("mutation %d collided", index)
		}
		seen[got] = struct{}{}
	}
}

func TestTestSubjectManifestBindsExactWorkItemGeneration(t *testing.T) {
	request := validTestAttestationRequest(t)
	manifest, err := BuildTestSubjectManifest(request.Subject)
	if err != nil {
		t.Fatal(err)
	}
	want := `"work_item_generation":4`
	if !strings.Contains(string(manifest.Content), want) {
		t.Fatalf("manifest missing %s: %s", want, manifest.Content)
	}
	request.Subject.WorkItemGeneration = 0
	if code := TestAttestorContractErrorCode(ValidateTestAttestationRequest(request)); code != "test_attestor.subject_identity_invalid" {
		t.Fatalf("zero work item generation code=%q", code)
	}
}

func TestTestAttestationResultAcceptsOnlyCanonicalManifestAndReport(t *testing.T) {
	request := validTestAttestationRequest(t)
	result := validTestAttestationResult(t, request)
	if err := ValidateTestAttestationResult(request, result); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	tampered := result
	tampered.Report.Content = append(append([]byte(nil), result.Report.Content...), ' ')
	if code := TestAttestorContractErrorCode(ValidateTestAttestationResult(request, tampered)); code != "test_attestor.artifact_invalid" {
		t.Fatalf("tampered report code=%q", code)
	}

	ackLike := TestAttestationResult{AttestorRef: "agent:ack"}
	if err := ValidateTestAttestationResult(request, ackLike); err == nil {
		t.Fatal("ACK-like value accepted as test attestation")
	}
}

func TestAttestationReceiptRefBindsCanonicalReportNotOnlySubjectDigest(t *testing.T) {
	request := validTestAttestationRequest(t)
	passed := validTestAttestationResult(t, request)
	failedOutcomes := []RequiredTestOutcome{{
		RequiredTestRef: request.RequiredTests[0].Ref(), ExitCode: 7,
		OutputDigest: testAttestorDigest("different failed output"),
	}}
	failedReport, err := BuildTestAttestationReport(TestAttestationReportInput{
		SubjectDigest: request.SubjectDigest, Verdict: TestAttestationFailed, Tests: failedOutcomes,
		AttestorRef: passed.AttestorRef, PolicyRef: passed.PolicyRef, PolicyDigest: passed.PolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	failedReceipt, err := TestAttestationReceiptRef(failedReport)
	if err != nil {
		t.Fatal(err)
	}
	failed := passed
	failed.Verdict, failed.Tests, failed.Report, failed.ReceiptRef =
		TestAttestationFailed, failedOutcomes, failedReport, failedReceipt
	if passed.SubjectDigest != failed.SubjectDigest || passed.ReceiptRef == failed.ReceiptRef ||
		ValidateTestAttestationResult(request, passed) != nil || ValidateTestAttestationResult(request, failed) != nil {
		t.Fatalf("receipt is not report-bound: subject=%s pass=%s failed=%s",
			passed.SubjectDigest, passed.ReceiptRef, failed.ReceiptRef)
	}
	failed.ReceiptRef = passed.ReceiptRef
	if TestAttestorContractErrorCode(ValidateTestAttestationResult(request, failed)) != "test_attestor.identity_mismatch" {
		t.Fatal("receipt replayed across different canonical report")
	}
}

func TestTestAttestationRequestRejectsRequiredTestDrift(t *testing.T) {
	request := validTestAttestationRequest(t)
	ref, _ := goal.NewRequiredTestRef("required-test:unit")
	toolRef, _ := goal.NewToolRef("tool:go-test")
	drifted, _ := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: ref, ToolRef: toolRef, Arguments: []string{"./internal/..."}, WorkingDirectory: ".",
	})
	request.RequiredTests = []goal.RequiredTestSpec{drifted}
	if code := TestAttestorContractErrorCode(ValidateTestAttestationRequest(request)); code != "test_attestor.required_tests_mismatch" {
		t.Fatalf("test drift code=%q", code)
	}
}

func validTestAttestationRequest(t *testing.T) TestAttestationRequest {
	t.Helper()
	goalRef, _ := goal.NewGoalRef("goal:test-attestor")
	workItemRef, _ := goal.NewWorkItemRef("work-item:test-attestor")
	executionRef, _ := goal.NewExecutionRef("execution:test-attestor")
	workspaceRef, _ := NewExecutionWorkspaceRef("workspace:test-attestor")
	changeRef, _ := NewChangeSetRef("change:test-attestor")
	repositoryRef, _ := identity.NewRepositoryRef("repository:test-attestor")
	testRef, _ := goal.NewRequiredTestRef("required-test:unit")
	toolRef, _ := goal.NewToolRef("tool:go-test")
	testSpec, err := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: testRef, ToolRef: toolRef, Arguments: []string{"./..."}, WorkingDirectory: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := TestAttestationRequest{
		Subject: TestSubject{
			GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
			ExecutionAttempt: 1, PlanGeneration: 2, WorkItemGeneration: 4, AppSpecGeneration: 3,
			AppSpecHash: testAttestorDigest("app-spec"), WorkspaceRef: workspaceRef,
			WorkspaceBindingDigest: testAttestorDigest("workspace"), ChangeSetRef: changeRef,
			ChangeSetDigest: testAttestorDigest("change"), RepositoryRef: repositoryRef,
			ObjectFormat: GitObjectFormatSHA1, BaseOID: strings.Repeat("1", 40),
			ParentOID: strings.Repeat("2", 40), HeadOID: strings.Repeat("3", 40),
			TreeOID: strings.Repeat("4", 40), DiffDigest: testAttestorDigest("diff"),
			WriteSetDigest: testAttestorDigest("write-set"), RequiredTestsDigest: goal.RequiredTestsDigest([]goal.RequiredTestSpec{testSpec}),
			PolicyRef: "policy:sandbox:v1", PolicyDigest: testAttestorDigest("policy"),
		},
		RequiredTests: []goal.RequiredTestSpec{testSpec}, IdempotencyKey: "attest:operation:1",
		RequestedAt: time.Date(2026, 7, 22, 1, 0, 0, 0, time.UTC),
	}
	request.SubjectDigest = TestSubjectDigest(request.Subject)
	return request
}

func validTestAttestationResult(t *testing.T, request TestAttestationRequest) TestAttestationResult {
	t.Helper()
	outcomes := []RequiredTestOutcome{{
		RequiredTestRef: request.RequiredTests[0].Ref(), ExitCode: 0,
		OutputDigest: testAttestorDigest("output"),
	}}
	manifest, err := BuildTestSubjectManifest(request.Subject)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildTestAttestationReport(TestAttestationReportInput{
		SubjectDigest: TestSubjectDigest(request.Subject), Verdict: TestAttestationPassed,
		Tests: outcomes, AttestorRef: "attestor:fake", PolicyRef: request.Subject.PolicyRef,
		PolicyDigest: request.Subject.PolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	receiptRef, err := TestAttestationReceiptRef(report)
	if err != nil {
		t.Fatal(err)
	}
	return TestAttestationResult{
		Subject: request.Subject, SubjectDigest: TestSubjectDigest(request.Subject),
		Verdict: TestAttestationPassed, Manifest: manifest, Report: report, Tests: outcomes,
		AttestorRef: "attestor:fake", ReceiptRef: receiptRef, PolicyRef: request.Subject.PolicyRef,
		PolicyDigest: request.Subject.PolicyDigest, StartedAt: request.RequestedAt,
		FinishedAt: request.RequestedAt.Add(time.Second),
	}
}

func testAttestorDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
