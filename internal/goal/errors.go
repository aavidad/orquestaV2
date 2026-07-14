package goal

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable machine-readable domain error code.
type ErrorCode string

const (
	ErrorInvalidArgument      ErrorCode = "goal.invalid_argument"
	ErrorInvalidRef           ErrorCode = "goal.invalid_ref"
	ErrorInvalidPlan          ErrorCode = "goal.invalid_plan"
	ErrorInvalidTransition    ErrorCode = "goal.invalid_transition"
	ErrorRevisionConflict     ErrorCode = "goal.revision_conflict"
	ErrorDuplicateWorkItem    ErrorCode = "goal.duplicate_work_item"
	ErrorWorkItemNotFound     ErrorCode = "goal.work_item_not_found"
	ErrorWorkItemsRequired    ErrorCode = "goal.work_items_required"
	ErrorWorkItemsNotTerminal ErrorCode = "goal.work_items_not_terminal"
	ErrorOutcomeConflict      ErrorCode = "goal.outcome_conflict"
	ErrorScopeConflict        ErrorCode = "goal.scope_conflict"
	ErrorEvidenceRequired     ErrorCode = "goal.evidence_required"
	ErrorDuplicateEvidence    ErrorCode = "goal.duplicate_evidence"
	ErrorSnapshotInvalid      ErrorCode = "goal.snapshot_invalid"
	ErrorIntentHashMismatch   ErrorCode = "goal.intent_hash_mismatch"
	ErrorAppSpecHashMismatch  ErrorCode = "goal.app_spec_hash_mismatch"
	ErrorWorkItemNotReady     ErrorCode = "goal.work_item_not_ready"
)

// DomainError carries no human-facing copy. Interfaces translate Code through
// their i18n catalog.
type DomainError struct {
	Code  ErrorCode
	Field string
}

func (err *DomainError) Error() string {
	if err == nil {
		return ""
	}
	if err.Field == "" {
		return string(err.Code)
	}
	return fmt.Sprintf("%s:%s", err.Code, err.Field)
}

// ErrorCodeOf extracts a stable code without parsing error text.
func ErrorCodeOf(err error) ErrorCode {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return ""
}

func domainError(code ErrorCode, field string) error {
	return &DomainError{Code: code, Field: field}
}
