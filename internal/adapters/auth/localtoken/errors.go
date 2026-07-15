package localtoken

import "errors"

type ErrorCode string

const (
	CodePathInvalid          ErrorCode = "localtoken.path_invalid"
	CodeDirectoryInvalid     ErrorCode = "localtoken.directory_invalid"
	CodeDirectoryPermissions ErrorCode = "localtoken.directory_permissions"
	CodeTokenFileInvalid     ErrorCode = "localtoken.token_file_invalid"
	CodeTokenFilePermissions ErrorCode = "localtoken.token_file_permissions"
	CodeTokenInvalid         ErrorCode = "localtoken.token_invalid"
	CodePrincipalInvalid     ErrorCode = "localtoken.principal_invalid"
	CodeAuthenticationFailed ErrorCode = "localtoken.authentication_failed"
	CodeContextInvalid       ErrorCode = "localtoken.context_invalid"
	CodeRandomFailed         ErrorCode = "localtoken.random_failed"
	CodePersistFailed        ErrorCode = "localtoken.persist_failed"
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

func IsError(err error, code ErrorCode) bool {
	var localErr *Error
	return errors.As(err, &localErr) && localErr.Code == code
}
