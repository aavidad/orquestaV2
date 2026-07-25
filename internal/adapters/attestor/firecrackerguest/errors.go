//go:build linux

// Package firecrackerguest implements the isolated PID1-side test runner. It
// consumes only the versioned raw-drive protocol and never imports core ports.
package firecrackerguest

import (
	"errors"
)

const (
	CodeInputInvalid       = "test_attestor.firecracker.guest_input_invalid"
	CodeSnapshotInvalid    = "test_attestor.firecracker.guest_snapshot_invalid"
	CodeSnapshotLimit      = "test_attestor.firecracker.guest_snapshot_limit"
	CodeMaterializeFailed  = "test_attestor.firecracker.guest_materialize_failed"
	CodeToolUnsupported    = "test_attestor.firecracker.guest_tool_unsupported"
	CodeExecutionFailed    = "test_attestor.firecracker.guest_execution_failed"
	CodeExecutionTimeout   = "test_attestor.firecracker.guest_execution_timeout"
	CodeOutputLimit        = "test_attestor.firecracker.guest_output_limit"
	CodeNoTests            = "test_attestor.firecracker.guest_no_tests"
	CodeIdentityFailed     = "test_attestor.firecracker.guest_identity_failed"
	CodeNetworkAvailable   = "test_attestor.firecracker.guest_network_available"
	CodeOutputWriteFailed  = "test_attestor.firecracker.guest_output_write_failed"
	CodeScratchUnavailable = "test_attestor.firecracker.guest_scratch_unavailable"
	CodeScratchCapacity    = "test_attestor.firecracker.guest_scratch_capacity"
	CodeCleanupFailed      = "test_attestor.firecracker.guest_cleanup_failed"
)

const NoTestsExitCode uint8 = 254

type Error struct {
	Code  string
	Cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func (err *Error) CauseCode() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var guestErr *Error
	if errors.As(err, &guestErr) {
		return guestErr.Code
	}
	return ""
}

func guestError(code string, cause error) error { return &Error{Code: code, Cause: cause} }
