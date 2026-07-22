// Package review contains pure rules for independent review rounds.
package review

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"
)

const AssessmentMediaType = "application/vnd.orquesta.review-assessment+json"

type Role string

const (
	RoleAuthor      Role = "author"
	RolePrimary     Role = "primary"
	RoleAdversarial Role = "adversarial"
)

type Verdict string

const (
	VerdictApprove          Verdict = "approve"
	VerdictChangesRequested Verdict = "changes_requested"
)

type GateStatus string

const (
	GateMissing          GateStatus = "missing"
	GateChangesRequested GateStatus = "changes_requested"
	GateApproved         GateStatus = "approved"
)

// Subject binds one round to an exact author launch, ChangeSet and test PASS.
type Subject struct {
	GoalRef, WorkItemRef, AuthorExecutionRef string
	AuthorExecutionAttempt                   uint64
	PlanGeneration, WorkItemGeneration       uint64
	AppSpecGeneration                        uint64
	SpecHash, AuthorLaunchReceiptRef         string
	AuthorExternalRef                        string
	WorkspaceBindingDigest, ChangeSetRef     string
	ChangeSetDigest, TreeOID, DiffDigest     string
	WriteSetDigest, RequiredTestsDigest      string
	TestAttestationRef, TestSubjectDigest    string
	TestPolicyDigest                         string
}

func NewSubject(subject Subject) (Subject, error) {
	if !validOpaque(subject.GoalRef, 512) || !validOpaque(subject.WorkItemRef, 512) ||
		!validOpaque(subject.AuthorExecutionRef, 512) ||
		subject.AuthorExecutionAttempt == 0 || subject.PlanGeneration == 0 || subject.WorkItemGeneration == 0 ||
		subject.AppSpecGeneration == 0 {
		return Subject{}, errors.New("review.subject_invalid")
	}
	for _, value := range []string{subject.AuthorLaunchReceiptRef, subject.AuthorExternalRef, subject.ChangeSetRef, subject.TestAttestationRef} {
		if !validOpaque(value, 1024) {
			return Subject{}, errors.New("review.subject_invalid")
		}
	}
	for _, value := range []string{subject.SpecHash, subject.WorkspaceBindingDigest, subject.ChangeSetDigest,
		subject.DiffDigest, subject.WriteSetDigest, subject.RequiredTestsDigest, subject.TestSubjectDigest,
		subject.TestPolicyDigest} {
		if !validHash(value) {
			return Subject{}, errors.New("review.subject_invalid")
		}
	}
	if !validOID(subject.TreeOID) {
		return Subject{}, errors.New("review.subject_invalid")
	}
	return subject, nil
}

func (subject Subject) Digest() string {
	values := []string{"orquesta.review-subject.v1", subject.GoalRef, subject.WorkItemRef, subject.AuthorExecutionRef,
		strconv.FormatUint(subject.AuthorExecutionAttempt, 10), strconv.FormatUint(subject.PlanGeneration, 10),
		strconv.FormatUint(subject.WorkItemGeneration, 10), strconv.FormatUint(subject.AppSpecGeneration, 10), subject.SpecHash,
		subject.AuthorLaunchReceiptRef, subject.AuthorExternalRef, subject.WorkspaceBindingDigest, subject.ChangeSetRef, subject.ChangeSetDigest,
		subject.TreeOID, subject.DiffDigest, subject.WriteSetDigest, subject.RequiredTestsDigest, subject.TestAttestationRef,
		subject.TestSubjectDigest, subject.TestPolicyDigest}
	digest := sha256.New()
	for _, value := range values {
		writeField(digest, value)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

// Assessment is immutable evidence from one reviewer execution.
type Assessment struct {
	SubjectDigest                           string
	Role                                    Role
	Verdict                                 Verdict
	ReviewerExecutionRef                    string
	ReviewerExecutionAttempt                uint64
	LaunchReceiptRef, ReviewerExternalRef   string
	AssessmentArtifactRef, AssessmentDigest string
	RecordedAt                              time.Time
}

func NewAssessment(assessment Assessment) (Assessment, error) {
	if !validDigest(assessment.SubjectDigest) || !validReviewerRole(assessment.Role) || !validVerdict(assessment.Verdict) ||
		!validOpaque(assessment.ReviewerExecutionRef, 512) || assessment.ReviewerExecutionAttempt == 0 ||
		!validOpaque(assessment.LaunchReceiptRef, 1024) || !validOpaque(assessment.ReviewerExternalRef, 1024) ||
		!validOpaque(assessment.AssessmentArtifactRef, 1024) ||
		!validHash(assessment.AssessmentDigest) || assessment.RecordedAt.IsZero() {
		return Assessment{}, errors.New("review.assessment_invalid")
	}
	return assessment, nil
}

// Artifact is the strict provider-neutral payload accepted from a reviewer.
type Artifact struct {
	SchemaVersion int       `json:"schema_version"`
	SubjectDigest string    `json:"subject_digest"`
	Role          Role      `json:"role"`
	Verdict       Verdict   `json:"verdict"`
	Summary       string    `json:"summary"`
	Findings      []Finding `json:"findings"`
}

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Finding struct {
	Code        string   `json:"code"`
	Severity    Severity `json:"severity"`
	EvidenceRef string   `json:"evidence_ref"`
}

func DecodeArtifact(content []byte) (Artifact, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var artifact Artifact
	if err := decoder.Decode(&artifact); err != nil {
		return Artifact{}, errors.New("review.assessment_invalid")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || artifact.SchemaVersion != 1 ||
		!validDigest(artifact.SubjectDigest) || !validReviewerRole(artifact.Role) || !validVerdict(artifact.Verdict) ||
		!validOpaque(artifact.Summary, 2000) || len(artifact.Findings) > 64 ||
		(artifact.Verdict == VerdictChangesRequested && len(artifact.Findings) == 0) || !validFindings(artifact.Findings) {
		return Artifact{}, errors.New("review.assessment_invalid")
	}
	return artifact, nil
}

type Gate struct {
	Status GateStatus
	Digest string
}

// EvaluateGate never guesses: malformed, duplicate, stale or non-independent
// facts are rejected; a changes request is reported only after both roles land.
func EvaluateGate(subject Subject, assessments []Assessment) (Gate, error) {
	if _, err := NewSubject(subject); err != nil {
		return Gate{}, err
	}
	digest := subject.Digest()
	seenRoles, seenExecutions, seenLaunches, seenExternal := map[Role]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	seenExternal[subject.AuthorExternalRef] = true
	byRole := make(map[Role]Assessment, 2)
	for _, assessment := range assessments {
		if _, err := NewAssessment(assessment); err != nil || assessment.SubjectDigest != digest ||
			assessment.ReviewerExecutionRef == subject.AuthorExecutionRef || assessment.LaunchReceiptRef == subject.AuthorLaunchReceiptRef ||
			seenRoles[assessment.Role] || seenExecutions[assessment.ReviewerExecutionRef] || seenLaunches[assessment.LaunchReceiptRef] ||
			seenExternal[assessment.ReviewerExternalRef] {
			return Gate{}, errors.New("review.subject_mismatch")
		}
		seenRoles[assessment.Role], seenExecutions[assessment.ReviewerExecutionRef], seenLaunches[assessment.LaunchReceiptRef],
			seenExternal[assessment.ReviewerExternalRef] = true, true, true, true
		byRole[assessment.Role] = assessment
	}
	if !seenRoles[RolePrimary] || !seenRoles[RoleAdversarial] {
		return Gate{Status: GateMissing}, nil
	}
	status := GateApproved
	if byRole[RolePrimary].Verdict == VerdictChangesRequested || byRole[RoleAdversarial].Verdict == VerdictChangesRequested {
		status = GateChangesRequested
	}
	return Gate{Status: status, Digest: gateDigest(subject, byRole)}, nil
}

func gateDigest(subject Subject, byRole map[Role]Assessment) string {
	digest := sha256.New()
	for _, value := range []string{"orquesta.review-gate.v1", subject.Digest(), subject.AuthorLaunchReceiptRef,
		string(RolePrimary), string(byRole[RolePrimary].Verdict), byRole[RolePrimary].ReviewerExecutionRef,
		byRole[RolePrimary].LaunchReceiptRef, byRole[RolePrimary].ReviewerExternalRef,
		byRole[RolePrimary].AssessmentArtifactRef, byRole[RolePrimary].AssessmentDigest,
		string(RoleAdversarial), string(byRole[RoleAdversarial].Verdict), byRole[RoleAdversarial].ReviewerExecutionRef,
		byRole[RoleAdversarial].LaunchReceiptRef, byRole[RoleAdversarial].ReviewerExternalRef,
		byRole[RoleAdversarial].AssessmentArtifactRef,
		byRole[RoleAdversarial].AssessmentDigest} {
		writeField(digest, value)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func validReviewerRole(role Role) bool { return role == RolePrimary || role == RoleAdversarial }
func validVerdict(verdict Verdict) bool {
	return verdict == VerdictApprove || verdict == VerdictChangesRequested
}
func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
func validHash(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}
func validOID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}
func validOpaque(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\r\n\x00")
}
func validFindings(findings []Finding) bool {
	for _, finding := range findings {
		if !validOpaque(finding.Code, 128) || !validOpaque(finding.EvidenceRef, 512) ||
			(finding.Severity != SeverityLow && finding.Severity != SeverityMedium && finding.Severity != SeverityHigh) {
			return false
		}
	}
	return true
}
func writeField(digest hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}
