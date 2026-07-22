package ports

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type TestSubject struct {
	GoalRef                goal.GoalRef
	WorkItemRef            goal.WorkItemRef
	ExecutionRef           goal.ExecutionRef
	ExecutionAttempt       uint64
	PlanGeneration         goal.PlanGeneration
	WorkItemGeneration     goal.Revision
	AppSpecGeneration      goal.AppSpecGeneration
	AppSpecHash            string
	WorkspaceRef           ExecutionWorkspaceRef
	WorkspaceBindingDigest string
	ChangeSetRef           ChangeSetRef
	ChangeSetDigest        string
	RepositoryRef          identity.RepositoryRef
	ObjectFormat           GitObjectFormat
	BaseOID                string
	ParentOID              string
	HeadOID                string
	TreeOID                string
	DiffDigest             string
	WriteSetDigest         string
	RequiredTestsDigest    string
	PolicyRef              string
	PolicyDigest           string
}

type TestAttestationRequest struct {
	Subject        TestSubject
	SubjectDigest  string
	RequiredTests  []goal.RequiredTestSpec
	IdempotencyKey string
	RequestedAt    time.Time
}

type TestAttestationRun struct {
	Request  TestAttestationRequest
	Snapshot SnapshotVerificationRequest
}

type TestAttestationVerdict string

const (
	TestAttestationPassed TestAttestationVerdict = "passed"
	TestAttestationFailed TestAttestationVerdict = "failed"
)

type RequiredTestOutcome struct {
	RequiredTestRef goal.RequiredTestRef
	ExitCode        int
	OutputDigest    string
}

type TestAttestationResult struct {
	Subject       TestSubject
	SubjectDigest string
	Verdict       TestAttestationVerdict
	Manifest      PutArtifactRequest
	Report        PutArtifactRequest
	Tests         []RequiredTestOutcome
	AttestorRef   string
	ReceiptRef    string
	PolicyRef     string
	PolicyDigest  string
	StartedAt     time.Time
	FinishedAt    time.Time
}

func TestAttestationReceiptRef(report PutArtifactRequest) (string, error) {
	if !validTestAttestationArtifact(report, TestAttestationReportMediaType) {
		return "", &TestAttestorContractError{Code: "test_attestor.report_invalid"}
	}
	digest := sha256.Sum256(report.Content)
	return "test-attestation-receipt:" + hex.EncodeToString(digest[:]), nil
}

func TestSubjectDigest(subject TestSubject) string {
	digest := sha256.New()
	writeTestAttestorField(digest, "orquesta.test-subject.v1")
	fields := []string{
		subject.GoalRef.String(), subject.WorkItemRef.String(), subject.ExecutionRef.String(),
		strconv.FormatUint(subject.ExecutionAttempt, 10), strconv.FormatUint(uint64(subject.PlanGeneration), 10),
		strconv.FormatUint(uint64(subject.WorkItemGeneration), 10),
		strconv.FormatUint(uint64(subject.AppSpecGeneration), 10), subject.AppSpecHash,
		subject.WorkspaceRef.String(), subject.WorkspaceBindingDigest,
		subject.ChangeSetRef.String(), subject.ChangeSetDigest, subject.RepositoryRef.String(),
		string(subject.ObjectFormat), subject.BaseOID, subject.ParentOID, subject.HeadOID, subject.TreeOID,
		subject.DiffDigest, subject.WriteSetDigest, subject.RequiredTestsDigest,
		subject.PolicyRef, subject.PolicyDigest,
	}
	for _, field := range fields {
		writeTestAttestorField(digest, field)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func writeTestAttestorField(digest hash.Hash, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(value))
}
