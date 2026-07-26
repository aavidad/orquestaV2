package gaps

import (
	"errors"
	"fmt"
)

// ErrorCode is stable machine-readable output. Human copy belongs to i18n.
type ErrorCode string

const (
	ErrorInvalidArgument  ErrorCode = "wizard_gaps.invalid_argument"
	ErrorUnknownDimension ErrorCode = "wizard_gaps.unknown_dimension"
	ErrorUnknownQuestion  ErrorCode = "wizard_gaps.unknown_question"
	ErrorUnknownOption    ErrorCode = "wizard_gaps.unknown_option"
	ErrorInvalidFreeText  ErrorCode = "wizard_gaps.invalid_free_text"
	ErrorUnknownPack      ErrorCode = "wizard_gaps.unknown_pack"
)

// DomainError deliberately carries no human-facing prose.
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
