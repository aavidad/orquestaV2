package ports

import (
	"context"
	"errors"

	"orquesta/internal/goal"
)

// AgentProviderStopRequestKey binds one stop mutation to the accepted runtime
// generation and the application-owned stop attempt.
type AgentProviderStopRequestKey struct {
	ExecutionRef      goal.ExecutionRef
	LaunchActionFence uint64
	StopActionFence   uint64
}

// AgentProviderStopRequest preserves exact non-secret bytes for a bound runtime.
type AgentProviderStopRequest struct {
	Key                  AgentProviderStopRequestKey
	StopEffectAttemptRef string
	ProviderRef          string
	IdempotencyKey       string
	TargetRef            string
	ExpectedRevision     uint64
	Body                 []byte
	BodySHA256           string
}

// AgentProviderStopRequestJournal belongs to the active StateRepository;
// exact replay and detached reads prevent physical-request recompilation.
type AgentProviderStopRequestJournal interface {
	RecordAgentProviderStopRequest(context.Context, AgentProviderStopRequest) (AgentProviderStopRequest, error)
	ResolveAgentProviderStopRequest(context.Context, AgentProviderStopRequestKey) (AgentProviderStopRequest, bool, error)
}

type AgentProviderStopRequestContractError struct{ Code string }

func (err *AgentProviderStopRequestContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentProviderStopRequestContractErrorCode(err error) string {
	var contractErr *AgentProviderStopRequestContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentProviderStopRequestKey(key AgentProviderStopRequestKey) error {
	if parsed, err := goal.NewExecutionRef(key.ExecutionRef.String()); err != nil || parsed != key.ExecutionRef {
		return agentProviderStopRequestError("execution_ref_invalid")
	}
	if key.LaunchActionFence == 0 {
		return agentProviderStopRequestError("launch_action_fence_invalid")
	}
	if key.StopActionFence == 0 {
		return agentProviderStopRequestError("stop_action_fence_invalid")
	}
	return nil
}

func ValidateAgentProviderStopRequest(request AgentProviderStopRequest) error {
	if err := ValidateAgentProviderStopRequestKey(request.Key); err != nil {
		return err
	}
	for _, check := range []struct{ suffix, value string }{
		{"effect_attempt_ref_invalid", request.StopEffectAttemptRef},
		{"provider_ref_invalid", request.ProviderRef},
		{"idempotency_key_invalid", request.IdempotencyKey},
		{"target_ref_invalid", request.TargetRef},
	} {
		if !validAgentProviderRequestRef(check.value) {
			return agentProviderStopRequestError(check.suffix)
		}
	}
	if request.ExpectedRevision == 0 {
		return agentProviderStopRequestError("expected_revision_invalid")
	}
	if len(request.Body) == 0 || len(request.Body) > maxAgentProviderRequestBodyBytes {
		return agentProviderStopRequestError("body_invalid")
	}
	if request.BodySHA256 != AgentProviderRequestBodySHA256(request.Body) {
		return agentProviderStopRequestError("body_digest_invalid")
	}
	return nil
}

func CloneAgentProviderStopRequest(request AgentProviderStopRequest) AgentProviderStopRequest {
	request.Body = append([]byte(nil), request.Body...)
	return request
}

func agentProviderStopRequestError(suffix string) error {
	return &AgentProviderStopRequestContractError{Code: "agent_provider_stop_request." + suffix}
}
