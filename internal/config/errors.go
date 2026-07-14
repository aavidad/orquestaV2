package config

import "errors"

// ErrorCode identifies configuration failures without coupling callers to
// human text.
type ErrorCode string

const (
	ErrorRegistryInvalid         ErrorCode = "config_registry_invalid"
	ErrorFileRead                ErrorCode = "config_file_read_failed"
	ErrorFileInvalid             ErrorCode = "config_file_invalid"
	ErrorUnknownKey              ErrorCode = "config_unknown_key"
	ErrorValueInvalid            ErrorCode = "config_value_invalid"
	ErrorEffectiveInputForbidden ErrorCode = "config_effective_input_forbidden"
	ErrorEffectiveWrite          ErrorCode = "config_effective_write_failed"
	ErrorChildEnvironmentInvalid ErrorCode = "config_child_environment_invalid"
)

// Error carries a stable code and optional key. Translation belongs to the
// public interface, not this package.
type Error struct {
	Code  ErrorCode
	Key   Key
	Cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Key != "" {
		return string(e.Code) + ": " + string(e.Key)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// HasErrorCode reports whether err contains a configuration error with code.
func HasErrorCode(err error, code ErrorCode) bool {
	var configError *Error
	return errors.As(err, &configError) && configError.Code == code
}
