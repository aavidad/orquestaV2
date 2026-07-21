package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// ExecutionWorkspaceRef identifies the private workspace of one execution.
// It is intentionally opaque: filesystem paths never cross this port.
type ExecutionWorkspaceRef struct{ value string }

func NewExecutionWorkspaceRef(value string) (ExecutionWorkspaceRef, error) {
	if !validWorkspaceOpaqueRef(value) {
		return ExecutionWorkspaceRef{}, &WorkspaceContractError{Code: "workspace.execution_ref_invalid"}
	}
	return ExecutionWorkspaceRef{value: value}, nil
}

func (ref ExecutionWorkspaceRef) String() string { return ref.value }

type GitObjectFormat string

const (
	GitObjectFormatSHA1   GitObjectFormat = "sha1"
	GitObjectFormatSHA256 GitObjectFormat = "sha256"
)

type WorkspacePrepareRequest struct {
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
	WriteSet          []string
	WriteSetDigest    string
	TargetRef         string
	IntentRef         string
	AttemptRef        string
	ActionFence       uint64
	IdempotencyKey    string
	PreparedAt        time.Time
}

// WorkspacePrepared is an adapter result. It carries no physical location;
// composition may resolve WorkspaceRef locally for the process it launches.
type WorkspacePrepared struct {
	WorkspaceRef   ExecutionWorkspaceRef
	RepositoryRef  identity.RepositoryRef
	ExecutionRef   goal.ExecutionRef
	TargetRef      string
	BaseOID        string
	ObjectFormat   GitObjectFormat
	WriteSetDigest string
	AdapterRef     string
	ReceiptRef     string
	PreparedAt     time.Time
}

type WorkspaceInspectRequest struct {
	WorkspaceRef   ExecutionWorkspaceRef
	RepositoryRef  identity.RepositoryRef
	ExecutionRef   goal.ExecutionRef
	IdempotencyKey string
}

type WorkspaceInspection struct {
	WorkspaceRef   ExecutionWorkspaceRef
	RepositoryRef  identity.RepositoryRef
	ExecutionRef   goal.ExecutionRef
	BaseOID        string
	HeadOID        string
	TreeOID        string
	WriteSetDigest string
	Dirty          bool
	AdapterRef     string
	InspectedAt    time.Time
}

type WorkspaceReleaseRequest struct {
	WorkspaceRef   ExecutionWorkspaceRef
	RepositoryRef  identity.RepositoryRef
	ExecutionRef   goal.ExecutionRef
	IdempotencyKey string
	RequestedAt    time.Time
}

type WorkspaceReleaseReceipt struct {
	WorkspaceRef  ExecutionWorkspaceRef
	RepositoryRef identity.RepositoryRef
	ExecutionRef  goal.ExecutionRef
	Released      bool
	ReceiptRef    string
	ReleasedAt    time.Time
}

type WorkspaceContractError struct{ Code string }

func (err *WorkspaceContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func WorkspaceContractErrorCode(err error) string {
	var contractErr *WorkspaceContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func WorkspaceWriteSetDigest(writeSet []string) string {
	digest := sha256.New()
	for _, scope := range writeSet {
		digest.Write([]byte(scope))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func ValidateWorkspacePrepareRequest(request WorkspacePrepareRequest) error {
	switch {
	case request.WorkspaceRef.String() == "":
		return &WorkspaceContractError{Code: "workspace.execution_ref_required"}
	case request.PrincipalRef.String() == "" || request.ActorRef.String() == "" || request.ProjectRef.String() == "" || request.RepositoryRef.String() == "":
		return &WorkspaceContractError{Code: "workspace.scope_required"}
	case request.GoalRef.String() == "" || request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" || request.ExecutionAttempt == 0 || request.PlanGeneration == 0 || request.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.AppSpecHash):
		return &WorkspaceContractError{Code: "workspace.causality_invalid"}
	case !validWorkspaceWriteSet(request.WriteSet) || request.WriteSetDigest != WorkspaceWriteSetDigest(request.WriteSet):
		return &WorkspaceContractError{Code: "workspace.write_set_invalid"}
	case request.TargetRef != "" && !validWorkspaceLogicalRef(request.TargetRef):
		return &WorkspaceContractError{Code: "workspace.target_ref_invalid"}
	case !validWorkspaceLogicalRef(request.IntentRef) || !validWorkspaceLogicalRef(request.AttemptRef) || !validWorkspaceLogicalRef(request.IdempotencyKey) || request.ActionFence == 0 || request.PreparedAt.IsZero():
		return &WorkspaceContractError{Code: "workspace.effect_identity_invalid"}
	default:
		return nil
	}
}

func ValidateWorkspacePrepared(request WorkspacePrepareRequest, prepared WorkspacePrepared) error {
	if err := ValidateWorkspacePrepareRequest(request); err != nil {
		return err
	}
	if prepared.WorkspaceRef != request.WorkspaceRef || prepared.RepositoryRef != request.RepositoryRef || prepared.ExecutionRef != request.ExecutionRef || (request.TargetRef != "" && prepared.TargetRef != request.TargetRef) || !validWorkspaceLogicalRef(prepared.TargetRef) || !validGitObjectFormat(prepared.ObjectFormat) || !validGitOID(prepared.BaseOID, prepared.ObjectFormat) || prepared.WriteSetDigest != request.WriteSetDigest || !validWorkspaceLogicalRef(prepared.AdapterRef) || !validWorkspaceLogicalRef(prepared.ReceiptRef) || prepared.PreparedAt.IsZero() {
		return &WorkspaceContractError{Code: "workspace.prepared_mismatch"}
	}
	return nil
}

func ValidateWorkspaceInspection(request WorkspaceInspectRequest, inspection WorkspaceInspection) error {
	if request.WorkspaceRef.String() == "" || request.RepositoryRef.String() == "" || request.ExecutionRef.String() == "" || !validWorkspaceLogicalRef(request.IdempotencyKey) {
		return &WorkspaceContractError{Code: "workspace.inspect_request_invalid"}
	}
	if inspection.WorkspaceRef != request.WorkspaceRef || inspection.RepositoryRef != request.RepositoryRef || inspection.ExecutionRef != request.ExecutionRef || !validWorkspaceLogicalRef(inspection.AdapterRef) || inspection.InspectedAt.IsZero() || inspection.BaseOID == "" || inspection.HeadOID == "" || inspection.TreeOID == "" || !validWorkspaceDigest(inspection.WriteSetDigest) {
		return &WorkspaceContractError{Code: "workspace.inspection_invalid"}
	}
	return nil
}

func ValidateWorkspaceReleaseReceipt(request WorkspaceReleaseRequest, receipt WorkspaceReleaseReceipt) error {
	if request.WorkspaceRef.String() == "" || request.RepositoryRef.String() == "" || request.ExecutionRef.String() == "" || !validWorkspaceLogicalRef(request.IdempotencyKey) || request.RequestedAt.IsZero() {
		return &WorkspaceContractError{Code: "workspace.release_request_invalid"}
	}
	if receipt.WorkspaceRef != request.WorkspaceRef || receipt.RepositoryRef != request.RepositoryRef || receipt.ExecutionRef != request.ExecutionRef || !validWorkspaceLogicalRef(receipt.ReceiptRef) || receipt.ReleasedAt.IsZero() || receipt.ReleasedAt.Before(request.RequestedAt) {
		return &WorkspaceContractError{Code: "workspace.release_receipt_invalid"}
	}
	return nil
}

func validWorkspaceOpaqueRef(value string) bool {
	return validWorkspaceLogicalRef(value) && !strings.ContainsRune(value, '\x00')
}

func validWorkspaceLogicalRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func validWorkspaceWriteSet(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		scope, err := goal.NewWriteScope(value)
		if err != nil || scope.String() != value {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validGitObjectFormat(format GitObjectFormat) bool {
	return format == GitObjectFormatSHA1 || format == GitObjectFormatSHA256
}

func validGitOID(oid string, format GitObjectFormat) bool {
	wantLength := 0
	switch format {
	case GitObjectFormatSHA1:
		wantLength = 40
	case GitObjectFormatSHA256:
		wantLength = 64
	default:
		return false
	}
	if len(oid) != wantLength || strings.ToLower(oid) != oid {
		return false
	}
	_, err := hex.DecodeString(oid)
	return err == nil
}

// ValidateGitOID validates an object identifier without resolving or reading a
// repository. Application boundaries use it to reject malformed effect
// requests before authorization or durable admission.
func ValidateGitOID(oid string, format GitObjectFormat) error {
	if !validGitOID(oid, format) {
		return &WorkspaceContractError{Code: "workspace.git_oid_invalid"}
	}
	return nil
}

func validWorkspaceDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
