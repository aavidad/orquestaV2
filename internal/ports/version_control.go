package ports

import (
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type ChangeSetRef struct{ value string }

func NewChangeSetRef(value string) (ChangeSetRef, error) {
	if !validWorkspaceOpaqueRef(value) {
		return ChangeSetRef{}, &VersionControlContractError{Code: "version_control.change_set_ref_invalid"}
	}
	return ChangeSetRef{value: value}, nil
}

func (ref ChangeSetRef) String() string { return ref.value }

type CommitRequest struct {
	ChangeSetRef      ChangeSetRef
	WorkspaceRef      ExecutionWorkspaceRef
	PrincipalRef      identity.PrincipalRef
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	RepositoryRef     identity.RepositoryRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	ExecutionAttempt  uint64
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	AppSpecHash       string
	BaseOID           string
	ObjectFormat      GitObjectFormat
	WriteSet          []string
	WriteSetDigest    string
	ParentChangeRef   ChangeSetRef
	IntentRef         string
	AttemptRef        string
	ActionFence       uint64
	IdempotencyKey    string
	CommittedAt       time.Time
}

type CommitResult struct {
	ChangeSetRef    ChangeSetRef
	WorkspaceRef    ExecutionWorkspaceRef
	RepositoryRef   identity.RepositoryRef
	ExecutionRef    goal.ExecutionRef
	BaseOID         string
	ParentOID       string
	HeadOID         string
	TreeOID         string
	ObjectFormat    GitObjectFormat
	DiffDigest      string
	ChangedPaths    []string
	WriteSetDigest  string
	ParentChangeRef ChangeSetRef
	AdapterRef      string
	ReceiptRef      string
	CommittedAt     time.Time
}

// SnapshotVerificationRequest asks the version-control adapter to prove the
// exact immutable Git-object subject. TestSubject carries logical identities
// only; a mutable worktree is never verification authority.
type SnapshotVerificationRequest struct {
	Subject       TestSubject
	SubjectDigest string
	ChangedPaths  []string
	WriteSet      []string
}

type IntegrationPreviewRequest struct {
	ChangeSetRef   ChangeSetRef
	RepositoryRef  identity.RepositoryRef
	SourceOID      string
	TargetRef      string
	TargetOID      string
	ObjectFormat   GitObjectFormat
	IdempotencyKey string
	RequestedAt    time.Time
}

type MergeStatus string

const (
	MergeStatusClean      MergeStatus = "clean"
	MergeStatusConflicted MergeStatus = "conflicted"
	MergeStatusStale      MergeStatus = "stale"
)

// IntegrationStatus is an effect outcome, distinct from a merge preview.
type IntegrationStatus string

const (
	IntegrationStatusIntegrated IntegrationStatus = "integrated"
	IntegrationStatusConflicted IntegrationStatus = "conflicted"
	IntegrationStatusStale      IntegrationStatus = "stale"
)

type IntegrationPreview struct {
	ChangeSetRef     ChangeSetRef
	RepositoryRef    identity.RepositoryRef
	SourceOID        string
	TargetRef        string
	TargetOID        string
	ObjectFormat     GitObjectFormat
	Status           MergeStatus
	CandidateTreeOID string
	ConflictDigest   string
	AdapterRef       string
	ObservedAt       time.Time
}

type IntegrationRequest struct {
	ChangeSetRef      ChangeSetRef
	RepositoryRef     identity.RepositoryRef
	PrincipalRef      identity.PrincipalRef
	ProjectRef        goal.ProjectRef
	SourceOID         string
	TargetRef         string
	ExpectedTargetOID string
	ObjectFormat      GitObjectFormat
	IntentRef         string
	AttemptRef        string
	ActionFence       uint64
	IdempotencyKey    string
	RequestedAt       time.Time
}

type IntegrationResult struct {
	ChangeSetRef    ChangeSetRef
	RepositoryRef   identity.RepositoryRef
	SourceOID       string
	TargetRef       string
	TargetBeforeOID string
	TargetAfterOID  string
	TreeOID         string
	ObjectFormat    GitObjectFormat
	Status          IntegrationStatus
	MarkerRef       string
	ConflictDigest  string
	AdapterRef      string
	ReceiptRef      string
	RecordedAt      time.Time
}

type VersionControlContractError struct{ Code string }

func (err *VersionControlContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ValidateCommitRequest(request CommitRequest) error {
	switch {
	case request.ChangeSetRef.String() == "" || request.WorkspaceRef.String() == "":
		return &VersionControlContractError{Code: "version_control.commit_ref_required"}
	case request.PrincipalRef.String() == "" || request.ActorRef.String() == "" || request.ProjectRef.String() == "" || request.RepositoryRef.String() == "":
		return &VersionControlContractError{Code: "version_control.commit_scope_required"}
	case request.GoalRef.String() == "" || request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" || request.ExecutionAttempt == 0 || request.PlanGeneration == 0 || request.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.AppSpecHash):
		return &VersionControlContractError{Code: "version_control.commit_causality_invalid"}
	case !validGitObjectFormat(request.ObjectFormat) || !validGitOID(request.BaseOID, request.ObjectFormat):
		return &VersionControlContractError{Code: "version_control.commit_base_invalid"}
	case !validWorkspaceWriteSet(request.WriteSet) || request.WriteSetDigest != WorkspaceWriteSetDigest(request.WriteSet):
		return &VersionControlContractError{Code: "version_control.commit_write_set_invalid"}
	case request.ParentChangeRef.String() != "" && request.ParentChangeRef == request.ChangeSetRef:
		return &VersionControlContractError{Code: "version_control.commit_parent_invalid"}
	case !validWorkspaceLogicalRef(request.IntentRef) || !validWorkspaceLogicalRef(request.AttemptRef) || !validWorkspaceLogicalRef(request.IdempotencyKey) || request.ActionFence == 0 || request.CommittedAt.IsZero():
		return &VersionControlContractError{Code: "version_control.commit_effect_invalid"}
	default:
		return nil
	}
}

func ValidateCommitResult(request CommitRequest, result CommitResult) error {
	if err := ValidateCommitRequest(request); err != nil {
		return err
	}
	if result.ChangeSetRef != request.ChangeSetRef || result.WorkspaceRef != request.WorkspaceRef || result.RepositoryRef != request.RepositoryRef || result.ExecutionRef != request.ExecutionRef || result.BaseOID != request.BaseOID || result.ObjectFormat != request.ObjectFormat || result.WriteSetDigest != request.WriteSetDigest || result.ParentChangeRef != request.ParentChangeRef || !validGitOID(result.ParentOID, result.ObjectFormat) || !validGitOID(result.HeadOID, result.ObjectFormat) || !validGitOID(result.TreeOID, result.ObjectFormat) || !validWorkspaceDigest(result.DiffDigest) || !validWorkspaceWriteSet(result.ChangedPaths) || !withinWriteSet(result.ChangedPaths, request.WriteSet) || !validWorkspaceLogicalRef(result.AdapterRef) || !validWorkspaceLogicalRef(result.ReceiptRef) || result.CommittedAt.IsZero() {
		return &VersionControlContractError{Code: "version_control.commit_result_invalid"}
	}
	return nil
}

func ValidateSnapshotVerificationRequest(request SnapshotVerificationRequest) error {
	if err := validateTestSubject(request.Subject); err != nil {
		return &VersionControlContractError{Code: "version_control.snapshot_subject_invalid"}
	}
	if request.SubjectDigest != TestSubjectDigest(request.Subject) {
		return &VersionControlContractError{Code: "version_control.snapshot_subject_mismatch"}
	}
	if !validWorkspaceWriteSet(request.ChangedPaths) || !validWorkspaceWriteSet(request.WriteSet) ||
		!withinWriteSet(request.ChangedPaths, request.WriteSet) ||
		WorkspaceWriteSetDigest(request.WriteSet) != request.Subject.WriteSetDigest {
		return &VersionControlContractError{Code: "version_control.snapshot_scope_invalid"}
	}
	return nil
}

func ValidateIntegrationPreviewRequest(request IntegrationPreviewRequest) error {
	if request.ChangeSetRef.String() == "" || request.RepositoryRef.String() == "" || !validWorkspaceLogicalRef(request.TargetRef) || !validGitObjectFormat(request.ObjectFormat) || !validGitOID(request.SourceOID, request.ObjectFormat) || !validGitOID(request.TargetOID, request.ObjectFormat) || !validWorkspaceLogicalRef(request.IdempotencyKey) || request.RequestedAt.IsZero() {
		return &VersionControlContractError{Code: "version_control.preview_request_invalid"}
	}
	return nil
}

func ValidateIntegrationPreview(request IntegrationPreviewRequest, preview IntegrationPreview) error {
	if err := ValidateIntegrationPreviewRequest(request); err != nil {
		return err
	}
	if preview.ChangeSetRef != request.ChangeSetRef || preview.RepositoryRef != request.RepositoryRef || preview.SourceOID != request.SourceOID || preview.TargetRef != request.TargetRef || preview.TargetOID != request.TargetOID || preview.ObjectFormat != request.ObjectFormat || !validMergeStatus(preview.Status) || !validWorkspaceLogicalRef(preview.AdapterRef) || preview.ObservedAt.IsZero() {
		return &VersionControlContractError{Code: "version_control.preview_invalid"}
	}
	if preview.Status == MergeStatusClean {
		if !validGitOID(preview.CandidateTreeOID, preview.ObjectFormat) || preview.ConflictDigest != "" {
			return &VersionControlContractError{Code: "version_control.preview_clean_invalid"}
		}
	} else if preview.CandidateTreeOID != "" || !validWorkspaceDigest(preview.ConflictDigest) {
		return &VersionControlContractError{Code: "version_control.preview_nonclean_invalid"}
	}
	return nil
}

func ValidateIntegrationRequest(request IntegrationRequest) error {
	if request.ChangeSetRef.String() == "" || request.RepositoryRef.String() == "" || request.PrincipalRef.String() == "" || request.ProjectRef.String() == "" || !validWorkspaceLogicalRef(request.TargetRef) || !validGitObjectFormat(request.ObjectFormat) || !validGitOID(request.SourceOID, request.ObjectFormat) || !validGitOID(request.ExpectedTargetOID, request.ObjectFormat) || !validWorkspaceLogicalRef(request.IntentRef) || !validWorkspaceLogicalRef(request.AttemptRef) || !validWorkspaceLogicalRef(request.IdempotencyKey) || request.ActionFence == 0 || request.RequestedAt.IsZero() {
		return &VersionControlContractError{Code: "version_control.integration_request_invalid"}
	}
	return nil
}

func ValidateIntegrationResult(request IntegrationRequest, result IntegrationResult) error {
	if err := ValidateIntegrationRequest(request); err != nil {
		return err
	}
	if result.ChangeSetRef != request.ChangeSetRef || result.RepositoryRef != request.RepositoryRef || result.SourceOID != request.SourceOID || result.TargetRef != request.TargetRef || result.TargetBeforeOID != request.ExpectedTargetOID || result.ObjectFormat != request.ObjectFormat {
		return &VersionControlContractError{Code: "version_control.integration_result_invalid"}
	}
	return ValidateIntegrationResultFields(result)
}

// ValidateIntegrationResultFields validates a persisted VCS result without
// fabricating request-only authorization data. Application state links the
// result to its immutable effect intent, attempt and fence separately.
func ValidateIntegrationResultFields(result IntegrationResult) error {
	if result.ChangeSetRef.String() == "" || result.RepositoryRef.String() == "" ||
		!validGitObjectFormat(result.ObjectFormat) || !validGitOID(result.SourceOID, result.ObjectFormat) ||
		!validWorkspaceLogicalRef(result.TargetRef) || !validGitOID(result.TargetBeforeOID, result.ObjectFormat) ||
		!validIntegrationStatus(result.Status) || !validWorkspaceLogicalRef(result.AdapterRef) ||
		!validWorkspaceLogicalRef(result.ReceiptRef) || result.RecordedAt.IsZero() {
		return &VersionControlContractError{Code: "version_control.integration_result_invalid"}
	}
	if result.Status == IntegrationStatusIntegrated {
		if !validGitOID(result.TargetAfterOID, result.ObjectFormat) || !validGitOID(result.TreeOID, result.ObjectFormat) || !validWorkspaceLogicalRef(result.MarkerRef) || result.ConflictDigest != "" {
			return &VersionControlContractError{Code: "version_control.integration_clean_invalid"}
		}
	} else if result.TargetAfterOID != result.TargetBeforeOID || result.TreeOID != "" || result.MarkerRef != "" || !validWorkspaceDigest(result.ConflictDigest) {
		return &VersionControlContractError{Code: "version_control.integration_nonclean_invalid"}
	}
	return nil
}

func VersionControlContractErrorCode(err error) string {
	var contractErr *VersionControlContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func validMergeStatus(status MergeStatus) bool {
	return status == MergeStatusClean || status == MergeStatusConflicted || status == MergeStatusStale
}

func validIntegrationStatus(status IntegrationStatus) bool {
	return status == IntegrationStatusIntegrated || status == IntegrationStatusConflicted || status == IntegrationStatusStale
}

func withinWriteSet(paths, writeSet []string) bool {
	for _, path := range paths {
		allowed := false
		for _, scope := range writeSet {
			if path == scope || strings.HasPrefix(path, scope+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}
