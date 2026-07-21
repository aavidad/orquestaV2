package gitlocal

import "errors"

type ErrorCode string

const (
	CodeConfigInvalid     ErrorCode = "gitlocal.config_invalid"
	CodeUnavailable       ErrorCode = "gitlocal.unavailable"
	CodeRepositoryInvalid ErrorCode = "gitlocal.repository_invalid"
	CodeWorkspaceUnsafe   ErrorCode = "gitlocal.workspace_unsafe"
	CodeWorkspaceConflict ErrorCode = "gitlocal.workspace_conflict"
	CodeWorkspaceNotFound ErrorCode = "gitlocal.workspace_not_found"
	CodeWorkspaceDirty    ErrorCode = "gitlocal.workspace_dirty"
	CodeWriteSetViolation ErrorCode = "gitlocal.write_set_violation"
	CodeBaseStale         ErrorCode = "gitlocal.base_stale"
	CodeNoChanges         ErrorCode = "gitlocal.no_changes"
	CodeChangeConflict    ErrorCode = "gitlocal.change_conflict"
	CodeGitFailed         ErrorCode = "gitlocal.git_failed"
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
func ErrorCodeOf(err error) ErrorCode {
	var adapterErr *Error
	if errors.As(err, &adapterErr) {
		return adapterErr.Code
	}
	return ""
}
