package catalog

import (
	"errors"
	"fmt"
)

// ErrorCode is stable machine-readable catalog failure metadata.
type ErrorCode string

const (
	ErrorInvalidArgument       ErrorCode = "wizard_catalog.invalid_argument"
	ErrorInvalidRef            ErrorCode = "wizard_catalog.invalid_ref"
	ErrorInvalidMessageKey     ErrorCode = "wizard_catalog.invalid_message_key"
	ErrorDuplicateRef          ErrorCode = "wizard_catalog.duplicate_ref"
	ErrorConflictingDefinition ErrorCode = "wizard_catalog.conflicting_definition"
	ErrorPackNotFound          ErrorCode = "wizard_catalog.pack_not_found"
	ErrorTaxonomyNotFound      ErrorCode = "wizard_catalog.taxonomy_not_found"
	ErrorTermNotFound          ErrorCode = "wizard_catalog.term_not_found"
	ErrorRecommendationCount   ErrorCode = "wizard_catalog.recommendation_count"
)

// DomainError contains no human-facing copy. Interfaces resolve Code through
// the canonical i18n catalog.
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
