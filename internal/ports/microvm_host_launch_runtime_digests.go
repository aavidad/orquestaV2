package ports

import (
	"context"
	"errors"
)

// MicroVMHostLaunchRuntimeDigestsV1 is the immutable physical-plan supplement
// for one v32 launch authority. It carries no image material or host paths.
type MicroVMHostLaunchRuntimeDigestsV1 struct {
	Key              MicroVMHostLaunchAuthorityKey
	PlanSHA256       string
	ConcessionSHA256 string
	KernelSHA256     string
	InitramfsSHA256  string
	ProfileSHA256    string
}

// MicroVMHostLaunchPreparationRegistry atomically prepares the v32 authority
// and its runtime-digest supplement before the external launch boundary.
type MicroVMHostLaunchPreparationRegistry interface {
	MicroVMHostLaunchAuthorityRegistry
	PrepareWithRuntime(context.Context, MicroVMHostLaunchAuthorityV1, MicroVMHostLaunchRuntimeDigestsV1) (MicroVMHostLaunchAuthorityV1, error)
	ResolveRuntime(context.Context, MicroVMHostLaunchAuthorityKey) (MicroVMHostLaunchRuntimeDigestsV1, error)
}

type MicroVMHostLaunchRuntimeDigestsContractError struct{ Code string }

func (err *MicroVMHostLaunchRuntimeDigestsContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func MicroVMHostLaunchRuntimeDigestsContractErrorCode(err error) string {
	var contractErr *MicroVMHostLaunchRuntimeDigestsContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateMicroVMHostLaunchRuntimeDigestsV1(value MicroVMHostLaunchRuntimeDigestsV1) error {
	if err := ValidateMicroVMHostLaunchAuthorityKey(value.Key); err != nil {
		return &MicroVMHostLaunchRuntimeDigestsContractError{Code: "microvm_host_launch_runtime_digests.key_invalid"}
	}
	for _, field := range []struct{ code, digest string }{
		{"plan", value.PlanSHA256}, {"concession", value.ConcessionSHA256},
		{"kernel", value.KernelSHA256}, {"initramfs", value.InitramfsSHA256}, {"profile", value.ProfileSHA256},
	} {
		if !validWorkspaceDigest(field.digest) {
			return &MicroVMHostLaunchRuntimeDigestsContractError{Code: "microvm_host_launch_runtime_digests." + field.code + "_invalid"}
		}
	}
	return nil
}
