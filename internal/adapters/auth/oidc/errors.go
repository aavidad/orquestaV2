package oidc

import "errors"

type ErrorCode string

const (
	CodeConfigInvalid     ErrorCode = "oidc.config_invalid"
	CodeContextInvalid    ErrorCode = "oidc.context_invalid"
	CodeDiscoveryFailed   ErrorCode = "oidc.discovery_failed"
	CodeCredentialInvalid ErrorCode = "oidc.credential_invalid"
	CodeTokenInvalid      ErrorCode = "oidc.token_invalid"
	CodeAudienceInvalid   ErrorCode = "oidc.audience_invalid"
	CodeSubjectInvalid    ErrorCode = "oidc.subject_invalid"
	CodeTimeInvalid       ErrorCode = "oidc.time_invalid"
	CodeGroupDenied       ErrorCode = "oidc.group_denied"
	CodeIdentityInvalid   ErrorCode = "oidc.identity_invalid"
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
	var oidcErr *Error
	return errors.As(err, &oidcErr) && oidcErr.Code == code
}
