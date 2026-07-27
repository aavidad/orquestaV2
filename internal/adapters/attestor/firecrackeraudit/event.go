//go:build linux

// Package firecrackeraudit stores a small, append-only audit stream for the
// privileged Firecracker boundary. Records deliberately contain no process or
// filesystem metadata.
package firecrackeraudit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const SchemaVersion = 1

const EmptyDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type Transition string

const (
	Started   Transition = "started"
	Succeeded Transition = "succeeded"
	Failed    Transition = "failed"
)

type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomeFailed    Outcome = "failed"
)

type DiagnosticCode string

const (
	DiagnosticNone           DiagnosticCode = "none"
	DiagnosticLaunchFailed   DiagnosticCode = "launch_failed"
	DiagnosticGuestFailed    DiagnosticCode = "guest_failed"
	DiagnosticTimeout        DiagnosticCode = "timeout"
	DiagnosticProtocolFailed DiagnosticCode = "protocol_failed"
	DiagnosticCleanupFailed  DiagnosticCode = "cleanup_failed"
	DiagnosticAuditFailed    DiagnosticCode = "audit_failed"
)

type Diagnostic struct {
	Code      DiagnosticCode `json:"code"`
	Bytes     uint64         `json:"bytes"`
	Digest    string         `json:"digest"`
	Truncated bool           `json:"truncated"`
}

// Event is intentionally closed: refs and digests are lowercase SHA-256 hex,
// diagnostics are machine codes, and no free-text extension map exists.
type Event struct {
	OperationRef string     `json:"operation_ref"`
	SubjectRef   string     `json:"subject_ref"`
	Transition   Transition `json:"transition"`
	Diagnostic   Diagnostic `json:"diagnostic"`
}

type Record struct {
	SchemaVersion uint32     `json:"schema_version"`
	Seq           uint64     `json:"seq"`
	PrevDigest    string     `json:"prev_digest"`
	EventDigest   string     `json:"event_digest"`
	StreamRef     string     `json:"stream_ref"`
	OperationRef  string     `json:"operation_ref"`
	SubjectRef    string     `json:"subject_ref"`
	Transition    Transition `json:"transition"`
	Diagnostic    Diagnostic `json:"diagnostic"`
}

type ErrorCode string

const (
	CodeConfig           ErrorCode = "firecracker_audit.config_invalid"
	CodeRef              ErrorCode = "firecracker_audit.ref_invalid"
	CodeEvent            ErrorCode = "firecracker_audit.event_invalid"
	CodeTransition       ErrorCode = "firecracker_audit.transition_invalid"
	CodeIO               ErrorCode = "firecracker_audit.io"
	CodeSecurity         ErrorCode = "firecracker_audit.security"
	CodeLimit            ErrorCode = "firecracker_audit.limit"
	CodeIntegrity        ErrorCode = "firecracker_audit.integrity"
	CodePartial          ErrorCode = "firecracker_audit.partial_record"
	CodeRecoveryRequired ErrorCode = "firecracker_audit.recovery_required"
	CodeNotOpen          ErrorCode = "firecracker_audit.not_open"
	CodeFinalExists      ErrorCode = "firecracker_audit.final_exists"
	CodeBlocked          ErrorCode = "firecracker_audit.blocked"
	CodeInUse            ErrorCode = "firecracker_audit.in_use"
)

type AuditError struct {
	Code  ErrorCode
	cause error
}

func (err *AuditError) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *AuditError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func ErrorCodeOf(err error) ErrorCode {
	var auditErr *AuditError
	if errors.As(err, &auditErr) {
		return auditErr.Code
	}
	return ""
}

func auditError(code ErrorCode, cause error) error {
	return &AuditError{Code: code, cause: cause}
}

func validHex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return false
	}
	return hex.EncodeToString(decoded) == value
}

func validTransition(value Transition) bool {
	return value == Started || value == Succeeded || value == Failed
}

func validDiagnosticCode(value DiagnosticCode) bool {
	switch value {
	case DiagnosticNone, DiagnosticLaunchFailed, DiagnosticGuestFailed,
		DiagnosticTimeout, DiagnosticProtocolFailed, DiagnosticCleanupFailed,
		DiagnosticAuditFailed:
		return true
	default:
		return false
	}
}

func validateEvent(event Event) error {
	if !validHex(event.OperationRef) || !validHex(event.SubjectRef) {
		return auditError(CodeRef, nil)
	}
	if !validTransition(event.Transition) || !validDiagnosticCode(event.Diagnostic.Code) ||
		!validHex(event.Diagnostic.Digest) {
		return auditError(CodeEvent, nil)
	}
	switch event.Transition {
	case Started:
		if event.Diagnostic.Code != DiagnosticNone || event.Diagnostic.Bytes != 0 ||
			event.Diagnostic.Digest != EmptyDigest || event.Diagnostic.Truncated {
			return auditError(CodeEvent, nil)
		}
	case Succeeded:
		if event.Diagnostic.Code != DiagnosticNone {
			return auditError(CodeEvent, nil)
		}
	case Failed:
		if event.Diagnostic.Code == DiagnosticNone {
			return auditError(CodeEvent, nil)
		}
	}
	return nil
}

func Digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
