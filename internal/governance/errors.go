package governance

import (
	"errors"
	"fmt"
)

// ErrorCode is stable machine-readable governance failure information.
type ErrorCode string

const (
	ErrorInvalidArgument  ErrorCode = "governance.invalid_argument"
	ErrorInvalidRef       ErrorCode = "governance.invalid_ref"
	ErrorInvalidCurrency  ErrorCode = "governance.invalid_currency"
	ErrorCurrencyConflict ErrorCode = "governance.currency_conflict"
	ErrorOverflow         ErrorCode = "governance.resource_overflow"
)

// DomainError contains no human-facing copy. Interfaces translate its code.
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
