package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
)

const microVMSessionMaxRefBytes = 512

// MicroVMSessionRef is an opaque logical reference. It deliberately cannot
// carry a path, endpoint, command, payload or credential.
type MicroVMSessionRef struct{ value string }

func NewMicroVMSessionRef(value string) (MicroVMSessionRef, error) {
	if !validMicroVMSessionRef(value) {
		return MicroVMSessionRef{}, microVMSessionError("ref_invalid")
	}
	return MicroVMSessionRef{value: value}, nil
}

func (ref MicroVMSessionRef) String() string { return ref.value }

// MicroVMSessionOpenRequest binds one host-broker session to the existing
// execution, workspace, access, egress, effect and attestation authorities.
// The broker consumes ChallengeRef atomically once; an exact retry may only
// replay its receipt.
type MicroVMSessionOpenRequest struct {
	Session                  ExecutionSessionEnsureRequest
	SessionRef               ExecutionSessionRef
	ExecutionWorkspaceRef    ExecutionWorkspaceRef
	AccessAuthority          AgentLaunchAccessAuthority
	EgressAuthority          AgentLaunchEgressAuthority
	EffectAuthority          AgentLaunchEffectAuthority
	AttestationRef           goal.AttestationRef
	AttestationSubjectDigest string
	GuestImageDigest         string
	GuestKernelDigest        string
	EffectiveConfigDigest    string
	ChallengeRef             MicroVMSessionRef
	ChallengeExpiresAt       time.Time
	RequestedAt              time.Time
}

type MicroVMSessionOpenReceipt struct {
	SessionRef   ExecutionSessionRef
	OpenDigest   string
	ChallengeRef MicroVMSessionRef
	ReceiptRef   MicroVMSessionRef
	OpenedAt     time.Time
	Replayed     bool
}

type MicroVMSessionOperation string

const (
	MicroVMSessionArtifactRead    MicroVMSessionOperation = "artifact_read"
	MicroVMSessionArtifactPublish MicroVMSessionOperation = "artifact_publish"
	MicroVMSessionMCPInvoke       MicroVMSessionOperation = "mcp_invoke"
	MicroVMSessionMailboxReceive  MicroVMSessionOperation = "mailbox_receive"
	MicroVMSessionMailboxSend     MicroVMSessionOperation = "mailbox_send"
)

// MicroVMSessionRequest carries refs and a digest only. Progress and results
// travel through the existing mailbox/artifact authorities; no parallel
// progress protocol lives here. RequestRef is a durable idempotency identity:
// it shares one namespace per SessionRef with End; exact retries replay the
// original receipt and cross-operation or divergent reuse fails closed.
type MicroVMSessionRequest struct {
	SessionRef     ExecutionSessionRef
	OpenReceiptRef MicroVMSessionRef
	RequestRef     MicroVMSessionRef
	Operation      MicroVMSessionOperation
	AuthorityRef   string
	ResourceRef    MicroVMSessionRef
	RequestDigest  string
	RequestedAt    time.Time
}

type MicroVMSessionRequestReceipt struct {
	SessionRef           ExecutionSessionRef
	RequestRef           MicroVMSessionRef
	Operation            MicroVMSessionOperation
	RequestBindingDigest string
	ResultRef            MicroVMSessionRef
	ResultDigest         string
	ReceiptRef           MicroVMSessionRef
	CompletedAt          time.Time
	Replayed             bool
}

type MicroVMSessionEndMode string

const (
	MicroVMSessionClose  MicroVMSessionEndMode = "close"
	MicroVMSessionRevoke MicroVMSessionEndMode = "revoke"
)

// MicroVMSessionEndRequest uses the same per-session RequestRef namespace and
// exact-replay/divergent-conflict rule as Exchange. Ending remains available
// after the launch effect lease expires.
type MicroVMSessionEndRequest struct {
	SessionRef     ExecutionSessionRef
	OpenReceiptRef MicroVMSessionRef
	RequestRef     MicroVMSessionRef
	Mode           MicroVMSessionEndMode
	RequestedAt    time.Time
}

type MicroVMSessionEndReceipt struct {
	SessionRef           ExecutionSessionRef
	RequestRef           MicroVMSessionRef
	Mode                 MicroVMSessionEndMode
	RequestBindingDigest string
	ReceiptRef           MicroVMSessionRef
	EndedAt              time.Time
	Replayed             bool
}

// MicroVMSessionBroker is a replaceable host boundary, not a lifecycle or
// persistence authority. Its adapter owns durable challenge consumption,
// RequestRef bindings and revocation. It exposes no transport-specific address
// or VM identifier.
type MicroVMSessionBroker interface {
	Open(context.Context, MicroVMSessionOpenRequest) (MicroVMSessionOpenReceipt, error)
	Exchange(context.Context, MicroVMSessionRequest) (MicroVMSessionRequestReceipt, error)
	End(context.Context, MicroVMSessionEndRequest) (MicroVMSessionEndReceipt, error)
}

type MicroVMSessionContractError struct{ Code string }

func (err *MicroVMSessionContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func MicroVMSessionContractErrorCode(err error) string {
	var contractErr *MicroVMSessionContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateMicroVMSessionOpenRequest(request MicroVMSessionOpenRequest) error {
	if !validMicroVMSessionIdentity(request.Session) ||
		len(request.SessionRef.String()) > microVMSessionMaxRefBytes {
		return microVMSessionError("identity_invalid")
	}
	if _, err := NewExecutionSessionRef(request.SessionRef.String()); err != nil {
		return microVMSessionError("identity_invalid")
	}
	if request.ExecutionWorkspaceRef.String() != "" {
		if _, err := NewExecutionWorkspaceRef(request.ExecutionWorkspaceRef.String()); err != nil {
			return microVMSessionError("workspace_authority_invalid")
		}
	}
	if request.AccessAuthority.ArtifactAccessRef.String() == "" ||
		request.AccessAuthority.MCPAccessRef.String() == "" ||
		request.AccessAuthority.MailboxEndpointRef.String() == "" ||
		ValidateAgentLaunchAccessAuthority(request.SessionRef, request.AccessAuthority) != nil {
		return microVMSessionError("access_authority_invalid")
	}
	if ValidateAgentLaunchEgressAuthority(request.EgressAuthority) != nil {
		return microVMSessionError("egress_authority_invalid")
	}
	if ValidateAgentLaunchEffectAuthority(request.EffectAuthority) != nil {
		return microVMSessionError("effect_authority_invalid")
	}
	if _, err := goal.NewAttestationRef(request.AttestationRef.String()); err != nil {
		return microVMSessionError("attestation_ref_invalid")
	}
	for _, digest := range []string{request.AttestationSubjectDigest, request.GuestImageDigest,
		request.GuestKernelDigest, request.EffectiveConfigDigest} {
		if !validWorkspaceDigest(digest) {
			return microVMSessionError("digest_invalid")
		}
	}
	if !validMicroVMSessionDigestRef(request.ChallengeRef, "challenge:sha256:") {
		return microVMSessionError("challenge_ref_invalid")
	}
	if request.RequestedAt.IsZero() || request.RequestedAt.Before(request.EffectAuthority.StartedAt) ||
		!request.RequestedAt.Before(request.EffectAuthority.ClaimLeaseUntil) ||
		!request.ChallengeExpiresAt.After(request.RequestedAt) ||
		request.ChallengeExpiresAt.After(request.EffectAuthority.ClaimLeaseUntil) {
		return microVMSessionError("open_window_invalid")
	}
	if expires := request.EffectAuthority.ApprovalExpiresAt; !expires.IsZero() &&
		(!request.RequestedAt.Before(expires) || request.ChallengeExpiresAt.After(expires)) {
		return microVMSessionError("open_window_invalid")
	}
	return nil
}

func ValidateMicroVMSessionOpenReceipt(request MicroVMSessionOpenRequest, receipt MicroVMSessionOpenReceipt) error {
	if err := ValidateMicroVMSessionOpenRequest(request); err != nil {
		return err
	}
	if receipt.SessionRef != request.SessionRef || receipt.OpenDigest != MicroVMSessionOpenDigest(request) ||
		receipt.ChallengeRef != request.ChallengeRef ||
		!validMicroVMSessionDigestRef(receipt.ReceiptRef, "receipt:sha256:") ||
		receipt.OpenedAt.Before(request.RequestedAt) || !receipt.OpenedAt.Before(request.ChallengeExpiresAt) {
		return microVMSessionError("open_receipt_invalid")
	}
	return nil
}

func ValidateMicroVMSessionRequest(open MicroVMSessionOpenRequest, opened MicroVMSessionOpenReceipt, request MicroVMSessionRequest) error {
	if err := ValidateMicroVMSessionOpenReceipt(open, opened); err != nil {
		return err
	}
	wantAuthority, allowed := microVMSessionAuthority(open.AccessAuthority, request.Operation)
	if request.SessionRef != open.SessionRef || request.OpenReceiptRef != opened.ReceiptRef {
		return microVMSessionError("request_session_mismatch")
	}
	if !allowed {
		return microVMSessionError("operation_not_allowed")
	}
	if request.AuthorityRef != wantAuthority {
		return microVMSessionError("request_authority_mismatch")
	}
	if !validMicroVMSessionRef(request.RequestRef.String()) ||
		!validMicroVMSessionRef(request.ResourceRef.String()) || !validWorkspaceDigest(request.RequestDigest) ||
		request.RequestedAt.Before(opened.OpenedAt) {
		return microVMSessionError("request_invalid")
	}
	return nil
}

func ValidateMicroVMSessionRequestReceipt(open MicroVMSessionOpenRequest, opened MicroVMSessionOpenReceipt, request MicroVMSessionRequest, receipt MicroVMSessionRequestReceipt) error {
	if err := ValidateMicroVMSessionRequest(open, opened, request); err != nil {
		return err
	}
	if receipt.SessionRef != request.SessionRef || receipt.RequestRef != request.RequestRef ||
		receipt.Operation != request.Operation ||
		receipt.RequestBindingDigest != MicroVMSessionRequestBindingDigest(request) ||
		!validMicroVMSessionRef(receipt.ResultRef.String()) ||
		!validWorkspaceDigest(receipt.ResultDigest) ||
		!validMicroVMSessionDigestRef(receipt.ReceiptRef, "receipt:sha256:") ||
		receipt.CompletedAt.Before(request.RequestedAt) {
		return microVMSessionError("request_receipt_invalid")
	}
	return nil
}

func ValidateMicroVMSessionEndRequest(open MicroVMSessionOpenRequest, opened MicroVMSessionOpenReceipt, request MicroVMSessionEndRequest) error {
	if err := ValidateMicroVMSessionOpenReceipt(open, opened); err != nil {
		return err
	}
	if request.SessionRef != open.SessionRef || request.OpenReceiptRef != opened.ReceiptRef ||
		!validMicroVMSessionRef(request.RequestRef.String()) ||
		(request.Mode != MicroVMSessionClose && request.Mode != MicroVMSessionRevoke) ||
		request.RequestedAt.Before(opened.OpenedAt) {
		return microVMSessionError("end_request_invalid")
	}
	return nil
}

func ValidateMicroVMSessionEndReceipt(open MicroVMSessionOpenRequest, opened MicroVMSessionOpenReceipt, request MicroVMSessionEndRequest, receipt MicroVMSessionEndReceipt) error {
	if err := ValidateMicroVMSessionEndRequest(open, opened, request); err != nil {
		return err
	}
	if receipt.SessionRef != request.SessionRef || receipt.RequestRef != request.RequestRef ||
		receipt.Mode != request.Mode || receipt.RequestBindingDigest != MicroVMSessionEndBindingDigest(request) ||
		!validMicroVMSessionDigestRef(receipt.ReceiptRef, "receipt:sha256:") ||
		receipt.EndedAt.Before(request.RequestedAt) {
		return microVMSessionError("end_receipt_invalid")
	}
	return nil
}

func MicroVMSessionOpenDigest(request MicroVMSessionOpenRequest) string {
	values := []string{"orquesta.microvm-session-open.v2", request.Session.ProjectRef.String(),
		request.Session.GoalRef.String(), request.Session.WorkItemRef.String(), request.Session.ExecutionRef.String(),
		strconv.FormatUint(request.Session.ExecutionAttempt, 10), request.Session.ReplacesExecutionRef.String(),
		strconv.FormatUint(uint64(request.Session.PlanGeneration), 10), strconv.FormatUint(uint64(request.Session.AppSpecGeneration), 10),
		request.Session.SpecHash, request.SessionRef.String(), request.ExecutionWorkspaceRef.String(),
		request.AccessAuthority.ArtifactAccessRef.String(),
		request.AccessAuthority.MCPAccessRef.String(), request.AccessAuthority.MailboxEndpointRef.String(),
		request.EgressAuthority.PolicyRef, request.EgressAuthority.PayloadSHA256,
		request.EffectAuthority.AuthorizationReceiptRef, request.EffectAuthority.EffectApprovalRef,
		request.EffectAuthority.EffectAttemptRef, strconv.FormatUint(request.EffectAuthority.ActionFence, 10),
		request.EffectAuthority.StartedAt.UTC().Format(time.RFC3339Nano),
		request.EffectAuthority.ClaimLeaseUntil.UTC().Format(time.RFC3339Nano),
		request.EffectAuthority.ApprovalExpiresAt.UTC().Format(time.RFC3339Nano), request.AttestationRef.String(),
		request.AttestationSubjectDigest, request.GuestImageDigest, request.GuestKernelDigest,
		request.EffectiveConfigDigest, request.ChallengeRef.String(), request.ChallengeExpiresAt.UTC().Format(time.RFC3339Nano),
		request.RequestedAt.UTC().Format(time.RFC3339Nano)}
	return microVMSessionDigest(values...)
}

func MicroVMSessionRequestBindingDigest(request MicroVMSessionRequest) string {
	return microVMSessionDigest("orquesta.microvm-session-request.v1", request.SessionRef.String(),
		request.OpenReceiptRef.String(), request.RequestRef.String(), string(request.Operation), request.AuthorityRef,
		request.ResourceRef.String(), request.RequestDigest, request.RequestedAt.UTC().Format(time.RFC3339Nano))
}

func MicroVMSessionEndBindingDigest(request MicroVMSessionEndRequest) string {
	return microVMSessionDigest("orquesta.microvm-session-end.v1", request.SessionRef.String(),
		request.OpenReceiptRef.String(), request.RequestRef.String(), string(request.Mode),
		request.RequestedAt.UTC().Format(time.RFC3339Nano))
}

func validMicroVMSessionIdentity(request ExecutionSessionEnsureRequest) bool {
	_, projectErr := goal.NewProjectRef(request.ProjectRef.String())
	_, goalErr := goal.NewGoalRef(request.GoalRef.String())
	_, workErr := goal.NewWorkItemRef(request.WorkItemRef.String())
	_, executionErr := goal.NewExecutionRef(request.ExecutionRef.String())
	return projectErr == nil && goalErr == nil && workErr == nil && executionErr == nil &&
		request.ExecutionAttempt > 0 && request.PlanGeneration > 0 && request.AppSpecGeneration > 0 &&
		goal.IsCanonicalAppSpecHash(request.SpecHash) &&
		((request.ExecutionAttempt == 1 && request.ReplacesExecutionRef.String() == "") ||
			(request.ExecutionAttempt > 1 && validCanonicalExecutionRef(request.ReplacesExecutionRef)))
}

func validCanonicalExecutionRef(ref goal.ExecutionRef) bool {
	_, err := goal.NewExecutionRef(ref.String())
	return err == nil
}

func microVMSessionAuthority(authority AgentLaunchAccessAuthority, operation MicroVMSessionOperation) (string, bool) {
	switch operation {
	case MicroVMSessionArtifactRead, MicroVMSessionArtifactPublish:
		return authority.ArtifactAccessRef.String(), true
	case MicroVMSessionMCPInvoke:
		return authority.MCPAccessRef.String(), true
	case MicroVMSessionMailboxReceive, MicroVMSessionMailboxSend:
		return authority.MailboxEndpointRef.String(), true
	default:
		return "", false
	}
}

func validMicroVMSessionDigestRef(ref MicroVMSessionRef, prefix string) bool {
	value := ref.String()
	return strings.HasPrefix(value, prefix) && validWorkspaceDigest(strings.TrimPrefix(value, prefix))
}

func validMicroVMSessionRef(value string) bool {
	if value == "" || len(value) > microVMSessionMaxRefBytes || strings.TrimSpace(value) != value || !strings.ContainsRune(value, ':') {
		return false
	}
	for _, character := range value {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune(":-_.", character)) {
			return false
		}
	}
	return true
}

func microVMSessionDigest(values ...string) string {
	digest := sha256.New()
	for _, value := range values {
		_, _ = digest.Write([]byte(value))
		_, _ = digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func microVMSessionError(suffix string) error {
	return &MicroVMSessionContractError{Code: "microvm_session." + suffix}
}
