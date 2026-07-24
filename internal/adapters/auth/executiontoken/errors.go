package executiontoken

import "errors"

type ErrorCode string

const (
	CodeDependenciesRequired  ErrorCode = "executiontoken.dependencies_required"
	CodeContextInvalid        ErrorCode = "executiontoken.context_invalid"
	CodeRequestInvalid        ErrorCode = "executiontoken.request_invalid"
	CodeAuthorityUnavailable  ErrorCode = "executiontoken.authority_unavailable"
	CodeCredentialUnavailable ErrorCode = "executiontoken.credential_unavailable"
	CodeAuthenticationFailed  ErrorCode = "executiontoken.authentication_failed"
	CodeRandomFailed          ErrorCode = "executiontoken.random_failed"
)

type Error struct {
	Code  ErrorCode
	Cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func executionError(code ErrorCode, cause ...error) error {
	if len(cause) == 0 || cause[0] == nil {
		return &Error{Code: code}
	}
	return &Error{Code: code, Cause: cause[0]}
}

func IsError(err error, code ErrorCode) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}
