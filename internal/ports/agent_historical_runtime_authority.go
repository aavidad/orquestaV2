package ports

import (
	"errors"

	"orquesta/internal/goal"
)

// AgentHistoricalRuntimeAuthorityKey identifies the launch attempt whose
// immutable inputs must be used by later preservation. ActionFence is the
// historical launch-effect fence, never a later recovery fence.
type AgentHistoricalRuntimeAuthorityKey struct {
	ExecutionRef goal.ExecutionRef
	ActionFence  uint64
}

// AgentHistoricalRuntimeDigests binds identities only. It deliberately
// carries no kernel, initramfs, profile, credential, endpoint, or path data.
type AgentHistoricalRuntimeDigests struct {
	PlanSHA256      string
	GrantSHA256     string
	KernelSHA256    string
	InitramfsSHA256 string
	ProfileSHA256   string
}

// AgentHistoricalRuntimeAuthority is the neutral launch-time fact needed to
// reject preservation under a different lifecycle subject or runtime input.
// Persistence and physical-provider translation are separate boundaries.
type AgentHistoricalRuntimeAuthority struct {
	Key     AgentHistoricalRuntimeAuthorityKey
	Subject AgentEnvironmentLifecycleSubject
	Digests AgentHistoricalRuntimeDigests
}

type AgentHistoricalRuntimeAuthorityError struct{ Code string }

func (err *AgentHistoricalRuntimeAuthorityError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentHistoricalRuntimeAuthorityErrorCode(err error) string {
	var contractErr *AgentHistoricalRuntimeAuthorityError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentHistoricalRuntimeAuthority(authority AgentHistoricalRuntimeAuthority) error {
	if authority.Key.ActionFence == 0 {
		return historicalRuntimeAuthorityError("action_fence_invalid")
	}
	if err := validateAgentEnvironmentSubject(authority.Subject); err != nil {
		return historicalRuntimeAuthorityError("subject_invalid")
	}
	if authority.Key.ExecutionRef != authority.Subject.ExecutionRef {
		return historicalRuntimeAuthorityError("execution_ref_mismatch")
	}
	for _, digest := range []string{
		authority.Digests.PlanSHA256,
		authority.Digests.GrantSHA256,
		authority.Digests.KernelSHA256,
		authority.Digests.InitramfsSHA256,
		authority.Digests.ProfileSHA256,
	} {
		if !resumenEntornoValido(digest) {
			return historicalRuntimeAuthorityError("digest_invalid")
		}
	}
	return nil
}

func historicalRuntimeAuthorityError(reason string) error {
	return &AgentHistoricalRuntimeAuthorityError{Code: "agent.historical_runtime_authority_" + reason}
}
