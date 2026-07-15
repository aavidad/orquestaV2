// Package jsonimport converts the bounded, historical orquesta_config.v0 JSON
// document into canonical configuration changes. It is an explicit one-shot
// migration tool: runtime bootstrap never reads legacy JSON through this
// package.
package jsonimport

import (
	"context"
	"errors"

	"orquesta/internal/config"
)

// Disposition states how one exact legacy leaf is handled by the product-owned
// migration table.
type Disposition string

const (
	DispositionMapped         Disposition = "mapped"
	DispositionSchema         Disposition = "schema"
	DispositionDeferred       Disposition = "deferred"
	DispositionSecretRequired Disposition = "secret_required"
)

// SourceKind declares the only accepted JSON shape for one legacy leaf. It is
// part of the canonical mapping row so parsing has no second path-to-type
// authority.
type SourceKind string

const (
	SourceKindString      SourceKind = "string"
	SourceKindBool        SourceKind = "bool"
	SourceKindInteger     SourceKind = "integer"
	SourceKindStringArray SourceKind = "string_array"
)

// Mapping is one immutable row in the canonical orquesta_config.v0 table.
// TargetKey and Transform are populated only for directly supported values.
type Mapping struct {
	LegacyPath  string      `json:"legacy_path"`
	SourceKind  SourceKind  `json:"source_kind"`
	TargetKey   config.Key  `json:"target_key,omitempty"`
	Transform   string      `json:"transform,omitempty"`
	Disposition Disposition `json:"disposition"`
	Reason      string      `json:"reason,omitempty"`
}

// Entry is one source leaf accounted for by exactly one canonical mapping.
// Value is exported only for non-sensitive mapped values. Deferred and secret
// leaves deliberately carry no source value.
type Entry struct {
	LegacyPath  string      `json:"legacy_path"`
	TargetKey   config.Key  `json:"target_key,omitempty"`
	Transform   string      `json:"transform,omitempty"`
	Disposition Disposition `json:"disposition"`
	Reason      string      `json:"reason,omitempty"`
	Value       any         `json:"value,omitempty"`
}

// Plan is a deterministic dry-run. SourceSHA256 binds exact input only when no
// secret leaf exists. Secret-blocked plans omit it, preventing low-entropy
// secret guessing; their PlanSHA256 binds only redacted structure and mapping.
type Plan struct {
	Ready               bool     `json:"ready"`
	SourceSHA256        string   `json:"source_sha256"`
	MappingSHA256       string   `json:"mapping_sha256"`
	PlanSHA256          string   `json:"plan_sha256"`
	Entries             []Entry  `json:"entries"`
	UnresolvedPaths     []string `json:"unresolved_paths"`
	SecretRequiredPaths []string `json:"secret_required_paths"`
	changes             []config.Change
}

// Options binds the importer to the sole V07 configuration writer.
type Options struct {
	Manager *config.Manager
}

// ApplyRequest confirms one exact dry-run against one expected TOML revision.
type ApplyRequest struct {
	Source             []byte
	ExpectedPlanSHA256 string
	ExpectedRevision   config.Revision
	ActorRef           string
	RequestRef         string
	Confirm            bool
}

// ErrorCode is stable machine-readable migration failure data.
type ErrorCode string

const (
	ErrorDependenciesRequired ErrorCode = "json_import_dependencies_required"
	ErrorContextInvalid       ErrorCode = "json_import_context_invalid"
	ErrorSourceInvalid        ErrorCode = "json_import_source_invalid"
	ErrorSourceTooLarge       ErrorCode = "json_import_source_too_large"
	ErrorSchemaRequired       ErrorCode = "json_import_schema_required"
	ErrorSchemaUnsupported    ErrorCode = "json_import_schema_unsupported"
	ErrorUnknownPath          ErrorCode = "json_import_unknown_path"
	ErrorValueInvalid         ErrorCode = "json_import_value_invalid"
	ErrorPlanNotReady         ErrorCode = "json_import_plan_not_ready"
	ErrorPlanMismatch         ErrorCode = "json_import_plan_mismatch"
	ErrorConfirmationRequired ErrorCode = "json_import_confirmation_required"
	ErrorMappingInvalid       ErrorCode = "json_import_mapping_invalid"
)

// Error contains only stable code and non-sensitive path. Source values are
// never interpolated into errors.
type Error struct {
	Code  ErrorCode
	Path  string
	Cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Path != "" {
		return string(err.Code) + ": " + err.Path
	}
	return string(err.Code)
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// HasErrorCode reports whether err contains the requested stable code.
func HasErrorCode(err error, code ErrorCode) bool {
	var importError *Error
	return errors.As(err, &importError) && importError.Code == code
}

func readyContext(ctx context.Context) error {
	if ctx == nil {
		return &Error{Code: ErrorContextInvalid}
	}
	select {
	case <-ctx.Done():
		return &Error{Code: ErrorContextInvalid, Cause: ctx.Err()}
	default:
		return nil
	}
}
