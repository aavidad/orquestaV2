package intake

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable machine-readable intake error code.
type ErrorCode string

const (
	ErrorInvalidArgument      ErrorCode = "intake.invalid_argument"
	ErrorInvalidRef           ErrorCode = "intake.invalid_ref"
	ErrorInvalidOrigin        ErrorCode = "intake.invalid_origin"
	ErrorRevisionConflict     ErrorCode = "intake.revision_conflict"
	ErrorStateMismatch        ErrorCode = "intake.state_mismatch"
	ErrorChannelStateCreation ErrorCode = "intake.channel_state_creation_forbidden"
	ErrorDuplicateRef         ErrorCode = "intake.duplicate_ref"
	ErrorIssueNotFound        ErrorCode = "intake.issue_not_found"
	ErrorQuestionNotFound     ErrorCode = "intake.question_not_found"
	ErrorOptionNotFound       ErrorCode = "intake.option_not_found"
	ErrorRecommendationCount  ErrorCode = "intake.recommendation_count"
	ErrorMessageKeyInvalid    ErrorCode = "intake.message_key_invalid"
	ErrorRoundLimit           ErrorCode = "intake.round_limit"
	ErrorChoiceConflict       ErrorCode = "intake.choice_conflict"
	ErrorQuestionRound        ErrorCode = "intake.question_round_not_found"
	ErrorRecommendationsDone  ErrorCode = "intake.recommendations_already_resolved"
	ErrorDependencyCycle      ErrorCode = "intake.question_dependency_cycle"
	ErrorDependencyPending    ErrorCode = "intake.question_dependency_pending"
)

// DomainError deliberately contains no human-facing text. A public surface
// resolves Code through the canonical i18n catalog.
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
