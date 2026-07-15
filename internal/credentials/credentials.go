// Package credentials defines the transport-neutral credential boundary.
// Secret material is callback-scoped and never belongs in durable projections.
package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrorCode is stable machine-readable failure information.
type ErrorCode string

const (
	ErrorInvalidRequest      ErrorCode = "credentials.invalid_request"
	ErrorInvalidRef          ErrorCode = "credentials.invalid_ref"
	ErrorAlreadyExists       ErrorCode = "credentials.already_exists"
	ErrorNotFound            ErrorCode = "credentials.not_found"
	ErrorOwnerMismatch       ErrorCode = "credentials.owner_mismatch"
	ErrorScopeDenied         ErrorCode = "credentials.scope_denied"
	ErrorPurposeDenied       ErrorCode = "credentials.purpose_denied"
	ErrorVersionConflict     ErrorCode = "credentials.version_conflict"
	ErrorIdempotencyConflict ErrorCode = "credentials.idempotency_conflict"
	ErrorRevoked             ErrorCode = "credentials.revoked"
	ErrorUnsafeFile          ErrorCode = "credentials.unsafe_file"
	ErrorSecretLeak          ErrorCode = "credentials.secret_leak"
	ErrorChildEnvironment    ErrorCode = "credentials.child_environment"
	ErrorConsumerFailed      ErrorCode = "credentials.consumer_failed"
	ErrorStoreIO             ErrorCode = "credentials.store_io"
)

// Error contains no credential material. Cause is available to errors.Is/As,
// but deliberately omitted from Error so an unsafe dependency cannot leak it.
type Error struct {
	Code  ErrorCode `json:"code"`
	Field string    `json:"field,omitempty"`
	cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Field
}

func (e *Error) GoString() string { return e.Error() }

func (e *Error) Unwrap() error { return e.cause }

func NewError(code ErrorCode, field string) error { return &Error{Code: code, Field: field} }

func WrapError(code ErrorCode, field string, cause error) error {
	return &Error{Code: code, Field: field, cause: cause}
}

func HasErrorCode(err error, code ErrorCode) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

type CredentialRef string
type OwnerRef string
type ScopeRef string
type PurposeRef string
type Version uint64

func (ref CredentialRef) String() string { return string(ref) }
func (ref OwnerRef) String() string      { return string(ref) }
func (ref ScopeRef) String() string      { return string(ref) }
func (ref PurposeRef) String() string    { return string(ref) }

// Secret owns a private copy of credential material.
type Secret struct{ material []byte }

func NewSecret(material []byte) (Secret, error) {
	if len(material) == 0 {
		return Secret{}, NewError(ErrorInvalidRequest, "material")
	}
	return Secret{material: append([]byte(nil), material...)}, nil
}

// Bytes always returns a detached copy.
func (secret Secret) Bytes() []byte { return append([]byte(nil), secret.material...) }

// Destroy clears material shared by shallow value copies and invalidates this value.
func (secret *Secret) Destroy() {
	if secret != nil {
		clear(secret.material)
		secret.material = nil
	}
}
func (Secret) String() string               { return "[REDACTED]" }
func (Secret) GoString() string             { return "[REDACTED]" }
func (Secret) MarshalJSON() ([]byte, error) { return json.Marshal("[REDACTED]") }

type CreateRequest struct {
	ActorRef      string
	RequestRef    string
	CredentialRef CredentialRef
	OwnerRef      OwnerRef
	ScopeRefs     []ScopeRef
	PurposeRef    PurposeRef
	Material      Secret
}

type UseRequest struct {
	ActorRef      string
	RequestRef    string
	CredentialRef CredentialRef
	OwnerRef      OwnerRef
	ScopeRef      ScopeRef
	PurposeRef    PurposeRef
	Version       Version
}

type RotateRequest struct {
	ActorRef, RequestRef string
	CredentialRef        CredentialRef
	OwnerRef             OwnerRef
	ExpectedVersion      Version
	Material             Secret
}

type RevokeRequest struct {
	ActorRef, RequestRef string
	CredentialRef        CredentialRef
	OwnerRef             OwnerRef
	ExpectedVersion      Version
	Reason               string
}

// Metadata is the durable, material-free credential projection.
type Metadata struct {
	CredentialRef                   CredentialRef
	OwnerRef                        OwnerRef
	ScopeRefs                       []ScopeRef
	PurposeRef                      PurposeRef
	Version                         Version
	Revoked                         bool
	CreatedAt, RotatedAt, RevokedAt time.Time
}

// Receipt is durable audit evidence and contains references only.
type Receipt struct {
	CredentialRef                   CredentialRef
	OwnerRef                        OwnerRef
	ScopeRef                        ScopeRef
	PurposeRef                      PurposeRef
	Version                         Version
	RequestRef, ActorRef, Operation string
	OccurredAt                      time.Time
}

type MutationResult struct {
	Metadata Metadata
	Receipt  Receipt
	Replayed bool
}

// Store is the only credential capability exposed to consumers.
type Store interface {
	Create(context.Context, CreateRequest) (MutationResult, error)
	Use(context.Context, UseRequest, func(Secret) error) (Receipt, error)
	Rotate(context.Context, RotateRequest) (MutationResult, error)
	Revoke(context.Context, RevokeRequest) (MutationResult, error)
}

func ValidateCreateRequest(request CreateRequest) error {
	if err := firstError(validateCommon(request.ActorRef, request.RequestRef, request.CredentialRef, request.OwnerRef), ValidatePurposeRef(request.PurposeRef)); err != nil {
		return err
	}
	if len(request.Material.material) == 0 || len(request.ScopeRefs) == 0 {
		return NewError(ErrorInvalidRequest, "material_or_scopes")
	}
	seen := make(map[ScopeRef]struct{}, len(request.ScopeRefs))
	for _, ref := range request.ScopeRefs {
		if err := ValidateScopeRef(ref); err != nil {
			return err
		}
		if _, duplicate := seen[ref]; duplicate {
			return NewError(ErrorInvalidRequest, "duplicate_scope_ref")
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func ValidateUseRequest(request UseRequest) error {
	return firstError(validateCommon(request.ActorRef, request.RequestRef, request.CredentialRef, request.OwnerRef),
		ValidateScopeRef(request.ScopeRef), ValidatePurposeRef(request.PurposeRef))
}

func ValidateRotateRequest(request RotateRequest) error {
	if err := validateCommon(request.ActorRef, request.RequestRef, request.CredentialRef, request.OwnerRef); err != nil {
		return err
	}
	if request.ExpectedVersion == 0 || len(request.Material.material) == 0 {
		return NewError(ErrorInvalidRequest, "version_or_material")
	}
	return nil
}

func ValidateRevokeRequest(request RevokeRequest) error {
	if err := validateCommon(request.ActorRef, request.RequestRef, request.CredentialRef, request.OwnerRef); err != nil {
		return err
	}
	if request.ExpectedVersion == 0 || !safeText(request.Reason, 256) {
		return NewError(ErrorInvalidRequest, "version_or_reason")
	}
	return nil
}

func ValidateCredentialRef(ref CredentialRef) error {
	const prefix = "credential:"
	value := string(ref)
	if !strings.HasPrefix(value, prefix) {
		return NewError(ErrorInvalidRef, "ref")
	}
	identifier := strings.TrimPrefix(value, prefix)
	if len(identifier) == 0 || len(identifier) > 128 {
		return NewError(ErrorInvalidRef, "ref")
	}
	for index, character := range identifier {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			index > 0 && (character == '.' || character == '_' || character == '-') {
			continue
		}
		return NewError(ErrorInvalidRef, "ref")
	}
	return nil
}
func ValidateOwnerRef(ref OwnerRef) error     { return validateOpaqueRef(string(ref)) }
func ValidateScopeRef(ref ScopeRef) error     { return validateOpaqueRef(string(ref)) }
func ValidatePurposeRef(ref PurposeRef) error { return validateOpaqueRef(string(ref)) }

func validateCommon(actor, request string, credential CredentialRef, owner OwnerRef) error {
	return firstError(validateOpaqueRef(actor), validateRef(request, "request:"),
		ValidateCredentialRef(credential), ValidateOwnerRef(owner))
}

func firstError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func validateRef(value, prefix string) error {
	if !strings.HasPrefix(value, prefix) {
		return NewError(ErrorInvalidRef, "ref")
	}
	return validateOpaqueRef(value)
}

func validateOpaqueRef(value string) error {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\x00') {
		return NewError(ErrorInvalidRef, "ref")
	}
	return nil
}

func safeText(value string, max int) bool {
	return value == strings.TrimSpace(value) && value != "" && len(value) <= max && !strings.ContainsAny(value, "\x00\r\n")
}
