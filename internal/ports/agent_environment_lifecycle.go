package ports

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"orquesta/internal/goal"
)

const maxAgentEnvironmentOpaqueValueBytes = 512

// AgentPhysicalToken identifies the physical execution without exposing the
// provider's identifier format to application or domain code.
type AgentPhysicalToken struct{ value string }

func NewAgentPhysicalToken(value string) (AgentPhysicalToken, error) {
	if !validAgentEnvironmentOpaqueValue(value) {
		return AgentPhysicalToken{}, environmentLifecycleError("physical_token_invalid")
	}
	return AgentPhysicalToken{value: value}, nil
}

func (token AgentPhysicalToken) String() string { return token.value }

// AgentPhysicalRevision and AgentPhysicalFence are deliberately opaque. The
// adapter may translate them to another representation, but application must
// only carry back the exact values observed from the physical lifecycle.
type AgentPhysicalRevision struct{ value string }

func NewAgentPhysicalRevision(value string) (AgentPhysicalRevision, error) {
	if !validAgentEnvironmentOpaqueValue(value) {
		return AgentPhysicalRevision{}, environmentLifecycleError("physical_revision_invalid")
	}
	return AgentPhysicalRevision{value: value}, nil
}

func (revision AgentPhysicalRevision) String() string { return revision.value }

type AgentPhysicalFence struct{ value string }

func NewAgentPhysicalFence(value string) (AgentPhysicalFence, error) {
	if !validAgentEnvironmentOpaqueValue(value) {
		return AgentPhysicalFence{}, environmentLifecycleError("physical_fence_invalid")
	}
	return AgentPhysicalFence{value: value}, nil
}

func (fence AgentPhysicalFence) String() string { return fence.value }

type AgentEnvironmentLifecycleState string

const (
	AgentEnvironmentActive     AgentEnvironmentLifecycleState = "active"
	AgentEnvironmentQuiescing  AgentEnvironmentLifecycleState = "quiescing"
	AgentEnvironmentQuiesced   AgentEnvironmentLifecycleState = "quiesced"
	AgentEnvironmentPreserving AgentEnvironmentLifecycleState = "preserving"
	AgentEnvironmentPreserved  AgentEnvironmentLifecycleState = "preserved"
	AgentEnvironmentClosing    AgentEnvironmentLifecycleState = "closing"
	AgentEnvironmentClosed     AgentEnvironmentLifecycleState = "closed"
)

// AgentEnvironmentLifecycleSubject binds every physical transition to the
// exact launch identity. It contains neither credentials nor physical paths.
type AgentEnvironmentLifecycleSubject struct {
	ExecutionRef      goal.ExecutionRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	SpecHash          string
	ProviderRef       string
	ModelRef          string
	AgentRef          string
	ExternalRef       string
}

// AgentEnvironmentLifecycleToken is the only physical continuation authority
// carried between operations. Revision and fence are compared, never ordered.
type AgentEnvironmentLifecycleToken struct {
	PhysicalToken AgentPhysicalToken
	Revision      AgentPhysicalRevision
	Fence         AgentPhysicalFence
	State         AgentEnvironmentLifecycleState
}

// AgentEnvironmentInspectRequest obtains the first token and refreshes it
// after an ambiguous response. It is read-only and creates no lifecycle fact.
type AgentEnvironmentInspectRequest struct {
	Subject AgentEnvironmentLifecycleSubject
}

type AgentEnvironmentInspectReceipt struct {
	Subject AgentEnvironmentLifecycleSubject
	Token   AgentEnvironmentLifecycleToken
}

type AgentQuiesceRequest struct {
	Subject        AgentEnvironmentLifecycleSubject
	ExpectedToken  AgentEnvironmentLifecycleToken
	IdempotencyKey string
}

type AgentQuiesceReceipt struct {
	Subject        AgentEnvironmentLifecycleSubject
	PreviousToken  AgentEnvironmentLifecycleToken
	NextToken      AgentEnvironmentLifecycleToken
	IdempotencyKey string
	ReceiptRef     string
	ConfirmedAt    time.Time
}

type AgentPreserveRequest struct {
	Subject        AgentEnvironmentLifecycleSubject
	ExpectedToken  AgentEnvironmentLifecycleToken
	IdempotencyKey string
}

// AgentPhysicalPreservationManifest is physical input to
// ResultadoPreservacionEntornoAgente, never a second final preservation fact.
// Application still enriches and persists the existing result/receipt with
// workspace, configuration and CAS evidence.
type AgentPhysicalPreservationManifest struct {
	Ref          string
	SHA256       string
	Content      []byte
	ContentBytes uint64
	WorkRevision AgentPhysicalRevision
	Causality    AgentPhysicalPreservationCausality
	SealedAt     time.Time
}

// AgentPhysicalPreservationCausality exposes only neutral cryptographic facts
// needed to enrich ResultadoPreservacionEntornoAgente. The isolation adapter
// validates them against its own manifest schema before returning the receipt.
type AgentPhysicalPreservationCausality struct {
	PlanSHA256      string
	GrantSHA256     string
	KernelSHA256    string
	InitramfsSHA256 string
	ProfileSHA256   string
}

func (manifest AgentPhysicalPreservationManifest) IsEmpty() bool {
	return manifest.Ref == "" && manifest.SHA256 == "" && len(manifest.Content) == 0 &&
		manifest.ContentBytes == 0 && manifest.WorkRevision.String() == "" &&
		manifest.Causality == (AgentPhysicalPreservationCausality{}) && manifest.SealedAt.IsZero()
}

type AgentPreserveReceipt struct {
	Subject        AgentEnvironmentLifecycleSubject
	PreviousToken  AgentEnvironmentLifecycleToken
	NextToken      AgentEnvironmentLifecycleToken
	IdempotencyKey string
	Manifest       AgentPhysicalPreservationManifest
	ReceiptRef     string
	ConfirmedAt    time.Time
}

type AgentPhysicalPreservationBinding struct {
	ManifestRef    string
	ManifestSHA256 string
}

// AgentPreservationBinding proves that application accepted and durably
// recorded the physical preservation before asking the isolation provider to
// close. ApplicationReceiptRef is the opaque Ref of
// application.ComprobantePreservacionEntornoAgente; the physical manifest is
// retained only as the exact provider-side subject of that durable fact.
type AgentPreservationBinding struct {
	ApplicationReceiptRef string
	PhysicalManifest      AgentPhysicalPreservationBinding
}

type AgentCloseRequest struct {
	Subject        AgentEnvironmentLifecycleSubject
	ExpectedToken  AgentEnvironmentLifecycleToken
	Preservation   AgentPreservationBinding
	IdempotencyKey string
}

type AgentCloseReceipt struct {
	Subject        AgentEnvironmentLifecycleSubject
	PreviousToken  AgentEnvironmentLifecycleToken
	NextToken      AgentEnvironmentLifecycleToken
	Preservation   AgentPreservationBinding
	IdempotencyKey string
	ReceiptRef     string
	ConfirmedAt    time.Time
}

// AgentEnvironmentLifecycle is the neutral successful-completion boundary.
// Quiesce is not AgentController.Stop: Stop remains control/cancellation.
// Close is logical and never authorizes deletion of preserved content.
type AgentEnvironmentLifecycle interface {
	Inspect(context.Context, AgentEnvironmentInspectRequest) (AgentEnvironmentInspectReceipt, error)
	Quiesce(context.Context, AgentQuiesceRequest) (AgentQuiesceReceipt, error)
	Preserve(context.Context, AgentPreserveRequest) (AgentPreserveReceipt, error)
	Close(context.Context, AgentCloseRequest) (AgentCloseReceipt, error)
}

func ValidateAgentEnvironmentInspectReceipt(
	request AgentEnvironmentInspectRequest,
	receipt AgentEnvironmentInspectReceipt,
) error {
	if err := validateAgentEnvironmentSubject(request.Subject); err != nil {
		return err
	}
	if receipt.Subject != request.Subject {
		return environmentLifecycleError("inspect_subject_mismatch")
	}
	return validateAgentEnvironmentToken(request.Subject, receipt.Token)
}

// ValidateAgentEnvironmentLifecycleTarget binds a physical lifecycle subject
// to the exact accepted launch. It mirrors the existing observe/stop target
// guards without changing either contract.
func ValidateAgentEnvironmentLifecycleTarget(
	launch AgentLaunchReceipt,
	subject AgentEnvironmentLifecycleSubject,
) error {
	if err := validateAgentEnvironmentSubject(subject); err != nil {
		return err
	}
	checks := []bool{
		subject.ExecutionRef == launch.ExecutionRef,
		subject.GoalRef == launch.GoalRef,
		subject.WorkItemRef == launch.WorkItemRef,
		subject.PlanGeneration == launch.PlanGeneration,
		subject.AppSpecGeneration == launch.AppSpecGeneration,
		subject.ExecutionAttempt == launch.ExecutionAttempt,
		subject.SpecHash == launch.SpecHash,
		subject.ProviderRef == launch.ProviderRef,
		subject.ModelRef == launch.ModelRef,
		subject.AgentRef == launch.AgentRef,
		subject.ExternalRef == launch.ExternalRef,
	}
	for _, matches := range checks {
		if !matches {
			return environmentLifecycleError("subject_launch_mismatch")
		}
	}
	return nil
}

func ValidateAgentQuiesceRequest(request AgentQuiesceRequest) error {
	if err := validateAgentEnvironmentSubject(request.Subject); err != nil {
		return err
	}
	if err := validateAgentEnvironmentToken(request.Subject, request.ExpectedToken); err != nil {
		return err
	}
	if request.ExpectedToken.State != AgentEnvironmentActive && request.ExpectedToken.State != AgentEnvironmentQuiescing {
		return environmentLifecycleError("quiesce_state_invalid")
	}
	return validateAgentEnvironmentIdempotency(request.IdempotencyKey)
}

func ValidateAgentQuiesceReceipt(request AgentQuiesceRequest, receipt AgentQuiesceReceipt) error {
	if err := ValidateAgentQuiesceRequest(request); err != nil {
		return err
	}
	return validateAgentEnvironmentTransition(request.Subject, request.ExpectedToken, request.IdempotencyKey,
		receipt.Subject, receipt.PreviousToken, receipt.NextToken, receipt.IdempotencyKey,
		receipt.ReceiptRef, receipt.ConfirmedAt,
		AgentEnvironmentQuiescing, AgentEnvironmentQuiesced)
}

func ValidateAgentPreserveRequest(request AgentPreserveRequest) error {
	if err := validateAgentEnvironmentSubject(request.Subject); err != nil {
		return err
	}
	if err := validateAgentEnvironmentToken(request.Subject, request.ExpectedToken); err != nil {
		return err
	}
	if request.ExpectedToken.State != AgentEnvironmentQuiesced && request.ExpectedToken.State != AgentEnvironmentPreserving {
		return environmentLifecycleError("preserve_state_invalid")
	}
	return validateAgentEnvironmentIdempotency(request.IdempotencyKey)
}

func ValidateAgentPreserveReceipt(request AgentPreserveRequest, receipt AgentPreserveReceipt) error {
	if err := ValidateAgentPreserveRequest(request); err != nil {
		return err
	}
	if err := validateAgentEnvironmentTransition(request.Subject, request.ExpectedToken, request.IdempotencyKey,
		receipt.Subject, receipt.PreviousToken, receipt.NextToken, receipt.IdempotencyKey,
		receipt.ReceiptRef, receipt.ConfirmedAt,
		AgentEnvironmentPreserving, AgentEnvironmentPreserved); err != nil {
		return err
	}
	if receipt.NextToken.State == AgentEnvironmentPreserving {
		if !receipt.Manifest.IsEmpty() {
			return environmentLifecycleError("preserve_pending_has_manifest")
		}
		return nil
	}
	if err := ValidateAgentPhysicalPreservationManifest(receipt.Manifest); err != nil {
		return err
	}
	if receipt.ConfirmedAt.Before(receipt.Manifest.SealedAt) {
		return environmentLifecycleError("preserve_confirmation_before_seal")
	}
	return nil
}

func ValidateAgentPhysicalPreservationManifest(manifest AgentPhysicalPreservationManifest) error {
	if !validAgentEnvironmentOpaqueValue(manifest.Ref) || !resumenEntornoValido(manifest.SHA256) ||
		manifest.ContentBytes != uint64(len(manifest.Content)) || manifest.ContentBytes == 0 ||
		manifest.WorkRevision.String() == "" || !validAgentPhysicalPreservationCausality(manifest.Causality) ||
		manifest.SealedAt.IsZero() {
		return environmentLifecycleError("manifest_invalid")
	}
	digest := sha256.Sum256(manifest.Content)
	if manifest.SHA256 != fmt.Sprintf("%x", digest) {
		return environmentLifecycleError("manifest_digest_mismatch")
	}
	return nil
}

func validAgentPhysicalPreservationCausality(causality AgentPhysicalPreservationCausality) bool {
	return resumenEntornoValido(causality.PlanSHA256) && resumenEntornoValido(causality.GrantSHA256) &&
		resumenEntornoValido(causality.KernelSHA256) && resumenEntornoValido(causality.InitramfsSHA256) &&
		(causality.ProfileSHA256 == "" || resumenEntornoValido(causality.ProfileSHA256))
}

func ValidateAgentCloseRequest(request AgentCloseRequest) error {
	if err := validateAgentEnvironmentSubject(request.Subject); err != nil {
		return err
	}
	if err := validateAgentEnvironmentToken(request.Subject, request.ExpectedToken); err != nil {
		return err
	}
	if request.ExpectedToken.State != AgentEnvironmentPreserved && request.ExpectedToken.State != AgentEnvironmentClosing {
		return environmentLifecycleError("close_state_invalid")
	}
	if err := validateAgentPreservationBinding(request.Preservation); err != nil {
		return err
	}
	return validateAgentEnvironmentIdempotency(request.IdempotencyKey)
}

func ValidateAgentCloseReceipt(request AgentCloseRequest, receipt AgentCloseReceipt) error {
	if err := ValidateAgentCloseRequest(request); err != nil {
		return err
	}
	if receipt.Preservation != request.Preservation {
		return environmentLifecycleError("close_preservation_mismatch")
	}
	return validateAgentEnvironmentTransition(request.Subject, request.ExpectedToken, request.IdempotencyKey,
		receipt.Subject, receipt.PreviousToken, receipt.NextToken, receipt.IdempotencyKey,
		receipt.ReceiptRef, receipt.ConfirmedAt,
		AgentEnvironmentClosing, AgentEnvironmentClosed)
}

func validateAgentEnvironmentSubject(subject AgentEnvironmentLifecycleSubject) error {
	switch {
	case subject.ExecutionRef.String() == "", subject.GoalRef.String() == "", subject.WorkItemRef.String() == "",
		subject.PlanGeneration == 0, subject.AppSpecGeneration == 0, subject.ExecutionAttempt == 0,
		!goal.IsCanonicalAppSpecHash(subject.SpecHash), !validAgentIdentityRef(subject.ProviderRef),
		!validAgentIdentityRef(subject.ModelRef), !validAgentIdentityRef(subject.AgentRef),
		!validAgentEnvironmentOpaqueValue(subject.ExternalRef):
		return environmentLifecycleError("subject_invalid")
	default:
		return nil
	}
}

func validateAgentEnvironmentToken(subject AgentEnvironmentLifecycleSubject, token AgentEnvironmentLifecycleToken) error {
	if token.PhysicalToken.String() != subject.ExternalRef || token.Revision.String() == "" || token.Fence.String() == "" ||
		!validAgentEnvironmentState(token.State) {
		return environmentLifecycleError("token_invalid")
	}
	return nil
}

func validateAgentEnvironmentTransition(
	wantSubject AgentEnvironmentLifecycleSubject,
	wantPrevious AgentEnvironmentLifecycleToken,
	wantIdempotency string,
	gotSubject AgentEnvironmentLifecycleSubject,
	gotPrevious, gotNext AgentEnvironmentLifecycleToken,
	gotIdempotency string,
	receiptRef string,
	confirmedAt time.Time,
	pending, completed AgentEnvironmentLifecycleState,
) error {
	if gotSubject != wantSubject {
		return environmentLifecycleError("receipt_subject_mismatch")
	}
	if gotPrevious != wantPrevious {
		return environmentLifecycleError("receipt_previous_token_mismatch")
	}
	if gotIdempotency != wantIdempotency {
		return environmentLifecycleError("receipt_idempotency_mismatch")
	}
	if err := validateAgentEnvironmentToken(wantSubject, gotNext); err != nil {
		return err
	}
	if gotNext.Revision == gotPrevious.Revision || gotNext.Fence != gotPrevious.Fence ||
		(gotNext.State != pending && gotNext.State != completed) {
		return environmentLifecycleError("receipt_next_token_invalid")
	}
	if gotNext.State == pending {
		if receiptRef != "" || !confirmedAt.IsZero() {
			return environmentLifecycleError("pending_has_confirmation")
		}
		return nil
	}
	if !validAgentReceiptRef(receiptRef) || confirmedAt.IsZero() {
		return environmentLifecycleError("terminal_confirmation_invalid")
	}
	return nil
}

func validateAgentPreservationBinding(binding AgentPreservationBinding) error {
	if !validAgentEnvironmentOpaqueValue(binding.ApplicationReceiptRef) ||
		!validAgentEnvironmentOpaqueValue(binding.PhysicalManifest.ManifestRef) ||
		!resumenEntornoValido(binding.PhysicalManifest.ManifestSHA256) {
		return environmentLifecycleError("preservation_binding_invalid")
	}
	return nil
}

func validateAgentEnvironmentIdempotency(value string) error {
	if !validAgentEnvironmentOpaqueValue(value) {
		return environmentLifecycleError("idempotency_key_invalid")
	}
	return nil
}

func validAgentEnvironmentState(state AgentEnvironmentLifecycleState) bool {
	switch state {
	case AgentEnvironmentActive, AgentEnvironmentQuiescing, AgentEnvironmentQuiesced,
		AgentEnvironmentPreserving, AgentEnvironmentPreserved, AgentEnvironmentClosing, AgentEnvironmentClosed:
		return true
	default:
		return false
	}
}

func validAgentEnvironmentOpaqueValue(value string) bool {
	return value != "" && len(value) <= maxAgentEnvironmentOpaqueValueBytes && strings.TrimSpace(value) == value &&
		!strings.ContainsRune(value, '\x00') && utf8.ValidString(value)
}

func environmentLifecycleError(suffix string) error {
	return &AgentContractError{Code: "agent.environment_lifecycle_" + suffix}
}
