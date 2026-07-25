// Package firecrackerlauncher implements the privileged, narrow launcher
// boundary used by the Firecracker TestAttestor adapter.
package firecrackerlauncher

import "errors"

const (
	CodeUnavailable        = "test_attestor.firecracker_launcher_unavailable"
	CodeConfigInvalid      = "test_attestor.firecracker_launcher_config_invalid"
	CodePrivilegeRequired  = "test_attestor.firecracker_launcher_privilege_required"
	CodePeerUnauthorized   = "test_attestor.firecracker_launcher_peer_unauthorized"
	CodeServerUnauthorized = "test_attestor.firecracker_launcher_server_unauthorized"
	CodeProtocolInvalid    = "test_attestor.firecracker_launcher_protocol_invalid"
	CodeDescriptorInvalid  = "test_attestor.firecracker_launcher_descriptor_invalid"
	CodeInputInvalid       = "test_attestor.firecracker_launcher_input_invalid"
	CodeOutputInvalid      = "test_attestor.firecracker_launcher_output_invalid"
	CodeAssetsUnsafe       = "test_attestor.firecracker_launcher_assets_unsafe"
	CodeNetworkUnsafe      = "test_attestor.firecracker_launcher_network_unsafe"
	CodeRuntimeRootUnsafe  = "test_attestor.firecracker_launcher_runtime_root_unsafe"
	CodeResourceUnsafe     = "test_attestor.firecracker_launcher_resource_unsafe"
	CodeExecutionFailed    = "test_attestor.firecracker_launcher_execution_failed"
	CodeExecutionTimeout   = "test_attestor.firecracker_launcher_execution_timeout"
	CodeCleanupFailed      = "test_attestor.firecracker_launcher_cleanup_failed"
	CodeResponseInvalid    = "test_attestor.firecracker_launcher_response_invalid"
	responseCodeOK         = "ok"
)

type Error struct {
	Code string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) CauseCode() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

func launcherError(code string) error {
	return &Error{Code: code}
}
