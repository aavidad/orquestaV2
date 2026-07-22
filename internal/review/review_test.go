package review

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func fixtureSubject(t *testing.T) Subject {
	t.Helper()
	value := strings.Repeat("a", 64)
	subject, err := NewSubject(Subject{
		GoalRef: "goal:one", WorkItemRef: "work-item:one", AuthorExecutionRef: "execution:author",
		AuthorExecutionAttempt: 1, PlanGeneration: 1, WorkItemGeneration: 1, AppSpecGeneration: 1,
		SpecHash: value, AuthorLaunchReceiptRef: "receipt:launch:author", AuthorExternalRef: "process:author",
		WorkspaceBindingDigest: value,
		ChangeSetRef:           "change-set:one", ChangeSetDigest: value, TreeOID: strings.Repeat("1", 40),
		DiffDigest: value, WriteSetDigest: value, RequiredTestsDigest: value,
		TestAttestationRef: "attestation:tests", TestSubjectDigest: value, TestPolicyDigest: value,
	})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func fixtureAssessment(subject Subject, role Role, verdict Verdict) Assessment {
	digestByte := "b"
	if role == RoleAdversarial {
		digestByte = "c"
	}
	return Assessment{SubjectDigest: subject.Digest(), Role: role, Verdict: verdict,
		ReviewerExecutionRef: "execution:" + string(role), ReviewerExecutionAttempt: 1,
		LaunchReceiptRef: "receipt:launch:" + string(role), ReviewerExternalRef: "process:" + string(role),
		AssessmentArtifactRef: "artifact:" + string(role),
		AssessmentDigest:      strings.Repeat(digestByte, 64), RecordedAt: time.Unix(1, 0).UTC()}
}

func TestReviewSubjectDigestBindsExactGenerationTreeDiffAndTests(t *testing.T) {
	base := fixtureSubject(t)
	mutations := []func(*Subject){
		func(v *Subject) { v.PlanGeneration++ }, func(v *Subject) { v.WorkItemGeneration++ },
		func(v *Subject) { v.TreeOID = strings.Repeat("2", 40) }, func(v *Subject) { v.DiffDigest = strings.Repeat("b", 64) },
		func(v *Subject) { v.RequiredTestsDigest = strings.Repeat("c", 64) },
		func(v *Subject) { v.TestPolicyDigest = strings.Repeat("d", 64) },
	}
	for index, mutate := range mutations {
		changed := base
		mutate(&changed)
		if changed.Digest() == base.Digest() {
			t.Fatalf("mutation %d not bound", index)
		}
	}
}

func TestAuthorPrimaryAndAdversarialUseThreeDistinctLaunches(t *testing.T) {
	subject := fixtureSubject(t)
	gate, err := EvaluateGate(subject, []Assessment{
		fixtureAssessment(subject, RolePrimary, VerdictApprove),
		fixtureAssessment(subject, RoleAdversarial, VerdictApprove),
	})
	if err != nil || gate.Status != GateApproved || gate.Digest == "" {
		t.Fatalf("gate=%+v err=%v", gate, err)
	}
}

func TestReviewRolesCannotReuseAuthorOrEachOtherLaunch(t *testing.T) {
	subject := fixtureSubject(t)
	primary := fixtureAssessment(subject, RolePrimary, VerdictApprove)
	adversarial := fixtureAssessment(subject, RoleAdversarial, VerdictApprove)
	for name, mutate := range map[string]func(){
		"author":   func() { primary.LaunchReceiptRef = subject.AuthorLaunchReceiptRef },
		"reviewer": func() { adversarial.LaunchReceiptRef = primary.LaunchReceiptRef },
	} {
		t.Run(name, func(t *testing.T) {
			primary = fixtureAssessment(subject, RolePrimary, VerdictApprove)
			adversarial = fixtureAssessment(subject, RoleAdversarial, VerdictApprove)
			mutate()
			if _, err := EvaluateGate(subject, []Assessment{primary, adversarial}); err == nil {
				t.Fatal("reused launch accepted")
			}
		})
	}
}

func TestReviewRolesCannotReuseAuthorOrEachOtherProcess(t *testing.T) {
	subject := fixtureSubject(t)
	primary := fixtureAssessment(subject, RolePrimary, VerdictApprove)
	adversarial := fixtureAssessment(subject, RoleAdversarial, VerdictApprove)
	primary.ReviewerExternalRef = subject.AuthorExternalRef
	if _, err := EvaluateGate(subject, []Assessment{primary, adversarial}); err == nil {
		t.Fatal("author process reused by reviewer")
	}
	primary = fixtureAssessment(subject, RolePrimary, VerdictApprove)
	adversarial.ReviewerExternalRef = primary.ReviewerExternalRef
	if _, err := EvaluateGate(subject, []Assessment{primary, adversarial}); err == nil {
		t.Fatal("reviewer process reused")
	}
}

func TestReviewAssessmentsRequireExactSubjectAndStructuredVerdict(t *testing.T) {
	subject := fixtureSubject(t)
	content := fmt.Sprintf(`{"schema_version":1,"subject_digest":%q,"role":"primary","verdict":"approve","summary":"exact review complete","findings":[]}`, subject.Digest())
	artifact, err := DecodeArtifact([]byte(content))
	if err != nil || artifact.Role != RolePrimary {
		t.Fatalf("artifact=%+v err=%v", artifact, err)
	}
	for _, invalid := range []string{
		`{"schema_version":1,"subject_digest":"bad","role":"primary","verdict":"approve","summary":"bad","findings":[]}`,
		content[:len(content)-1] + `,"extra":true}`,
		`approve`,
	} {
		if _, err := DecodeArtifact([]byte(invalid)); err == nil {
			t.Fatalf("invalid artifact accepted: %s", invalid)
		}
	}
}

func TestPrimaryAndAdversarialMustBothApproveBeforeIntegrationAdmission(t *testing.T) {
	subject := fixtureSubject(t)
	gate, err := EvaluateGate(subject, []Assessment{fixtureAssessment(subject, RolePrimary, VerdictApprove)})
	if err != nil || gate.Status != GateMissing || gate.Digest != "" {
		t.Fatalf("gate=%+v err=%v", gate, err)
	}
	gate, err = EvaluateGate(subject, []Assessment{
		fixtureAssessment(subject, RolePrimary, VerdictChangesRequested),
		fixtureAssessment(subject, RoleAdversarial, VerdictApprove),
	})
	if err != nil || gate.Status != GateChangesRequested || gate.Digest == "" {
		t.Fatalf("gate=%+v err=%v", gate, err)
	}
}

func TestReworkCreatesFreshSubjectAndInvalidatesEarlierReviews(t *testing.T) {
	first := fixtureSubject(t)
	next := first
	next.ChangeSetRef = "change-set:two"
	if next.Digest() == first.Digest() {
		t.Fatal("subject reused")
	}
	_, err := EvaluateGate(next, []Assessment{
		fixtureAssessment(first, RolePrimary, VerdictApprove), fixtureAssessment(first, RoleAdversarial, VerdictApprove),
	})
	if err == nil {
		t.Fatal("stale reviews accepted")
	}
}
