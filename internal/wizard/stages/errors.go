package stages

import "errors"

type ErrorCode string

const (
	ErrorInvalidArgument       ErrorCode = "wizard.stages.invalid_argument"
	ErrorInvalidDigest         ErrorCode = "wizard.stages.invalid_digest"
	ErrorInvalidRef            ErrorCode = "wizard.stages.invalid_ref"
	ErrorDuplicateRef          ErrorCode = "wizard.stages.duplicate_ref"
	ErrorReferenceNotFound     ErrorCode = "wizard.stages.reference_not_found"
	ErrorDependencyCycle       ErrorCode = "wizard.stages.dependency_cycle"
	ErrorDependencyOrder       ErrorCode = "wizard.stages.dependency_order"
	ErrorWriteSetConflict      ErrorCode = "wizard.stages.write_set_conflict"
	ErrorIncompleteCatalog     ErrorCode = "wizard.stages.incomplete_catalog"
	ErrorConflictingDefinition ErrorCode = "wizard.stages.conflicting_definition"
	ErrorCatalogVersionUnknown ErrorCode = "wizard.stages.catalog_version_unknown"
	ErrorCatalogDigestMismatch ErrorCode = "wizard.stages.catalog_digest_mismatch"
)

type DomainError struct {
	Code  ErrorCode
	Field string
}

func (value DomainError) Error() string {
	if value.Field == "" {
		return string(value.Code)
	}
	return string(value.Code) + ": " + value.Field
}

func domainError(code ErrorCode, field string) error {
	return DomainError{Code: code, Field: field}
}

func ErrorCodeOf(err error) ErrorCode {
	var target DomainError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}
