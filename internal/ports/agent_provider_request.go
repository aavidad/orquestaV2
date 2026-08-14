package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/goal"
)

const (
	maxAgentProviderRequestBodyBytes = 2 * 1024 * 1024
	maxAgentProviderRequestRefBytes  = 512
)

type AgentProviderRequestStage string

const (
	AgentProviderRequestLaunch       AgentProviderRequestStage = "launch"
	AgentProviderRequestSessionStart AgentProviderRequestStage = "session_start"
	AgentProviderRequestSessionInput AgentProviderRequestStage = "session_input"
)

// AgentProviderRequestKey identifies one physical mutation within the exact
// application-owned launch attempt. Stage is an ordered physical step, not a
// lifecycle or a second effect attempt.
type AgentProviderRequestKey struct {
	ExecutionRef goal.ExecutionRef
	ActionFence  uint64
	Stage        AgentProviderRequestStage
}

// AgentProviderRequest preserves the exact non-secret bytes sent through an
// isolation provider. LaunchBinding* is empty until the launch response is
// durably bound; later stages carry that binding as TargetRef/ExpectedRevision.
type AgentProviderRequest struct {
	Key                   AgentProviderRequestKey
	EffectAttemptRef      string
	ProviderRef           string
	IdempotencyKey        string
	TargetRef             string
	ExpectedRevision      uint64
	Body                  []byte
	BodySHA256            string
	LaunchBindingRef      string
	LaunchBindingRevision uint64
}

// AgentProviderRequestJournal is implemented by the active StateRepository.
// Record must be exact-replay idempotent; BindLaunch is set-once; Resolve must
// return detached bytes suitable for byte-identical recovery. Resolve reports
// absence explicitly so adapters do not need to depend on application errors.
type AgentProviderRequestJournal interface {
	RecordAgentProviderRequest(context.Context, AgentProviderRequest) (AgentProviderRequest, error)
	BindAgentProviderLaunch(context.Context, AgentProviderRequestKey, string, uint64) (AgentProviderRequest, error)
	ResolveAgentProviderRequest(context.Context, AgentProviderRequestKey) (AgentProviderRequest, bool, error)
}

type AgentProviderRequestContractError struct{ Code string }

func (err *AgentProviderRequestContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentProviderRequestContractErrorCode(err error) string {
	var contractErr *AgentProviderRequestContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func AgentProviderRequestBodySHA256(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func ValidateAgentProviderRequestKey(key AgentProviderRequestKey) error {
	if key.ExecutionRef.String() == "" {
		return agentProviderRequestError("execution_ref_invalid")
	}
	if parsed, err := goal.NewExecutionRef(key.ExecutionRef.String()); err != nil || parsed != key.ExecutionRef {
		return agentProviderRequestError("execution_ref_invalid")
	}
	if key.ActionFence == 0 {
		return agentProviderRequestError("action_fence_invalid")
	}
	switch key.Stage {
	case AgentProviderRequestLaunch, AgentProviderRequestSessionStart, AgentProviderRequestSessionInput:
		return nil
	default:
		return agentProviderRequestError("stage_invalid")
	}
}

func ValidateAgentProviderRequest(request AgentProviderRequest) error {
	if err := ValidateAgentProviderRequestKey(request.Key); err != nil {
		return err
	}
	if !validAgentProviderRequestRef(request.EffectAttemptRef) {
		return agentProviderRequestError("effect_attempt_ref_invalid")
	}
	if !validAgentProviderRequestRef(request.ProviderRef) {
		return agentProviderRequestError("provider_ref_invalid")
	}
	if !validAgentProviderRequestRef(request.IdempotencyKey) {
		return agentProviderRequestError("idempotency_key_invalid")
	}
	if len(request.Body) == 0 || len(request.Body) > maxAgentProviderRequestBodyBytes {
		return agentProviderRequestError("body_invalid")
	}
	if request.BodySHA256 != AgentProviderRequestBodySHA256(request.Body) {
		return agentProviderRequestError("body_digest_invalid")
	}
	switch request.Key.Stage {
	case AgentProviderRequestLaunch:
		if request.TargetRef != "" || request.ExpectedRevision != 0 {
			return agentProviderRequestError("launch_target_invalid")
		}
		if (request.LaunchBindingRef == "") != (request.LaunchBindingRevision == 0) {
			return agentProviderRequestError("launch_binding_partial")
		}
		if request.LaunchBindingRef != "" && !validAgentProviderRequestRef(request.LaunchBindingRef) {
			return agentProviderRequestError("launch_binding_invalid")
		}
	case AgentProviderRequestSessionStart, AgentProviderRequestSessionInput:
		if !validAgentProviderRequestRef(request.TargetRef) || request.ExpectedRevision == 0 ||
			request.LaunchBindingRef != "" || request.LaunchBindingRevision != 0 {
			return agentProviderRequestError("session_target_invalid")
		}
	}
	return nil
}

func ValidatePreparedAgentProviderRequest(request AgentProviderRequest) error {
	if err := ValidateAgentProviderRequest(request); err != nil {
		return err
	}
	if request.Key.Stage == AgentProviderRequestLaunch && request.LaunchBindingRef != "" {
		return agentProviderRequestError("prepared_launch_already_bound")
	}
	return nil
}

func CloneAgentProviderRequest(request AgentProviderRequest) AgentProviderRequest {
	request.Body = append([]byte(nil), request.Body...)
	return request
}

func validAgentProviderRequestRef(value string) bool {
	if value == "" || len(value) > maxAgentProviderRequestRefBytes ||
		strings.TrimSpace(value) != value || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func agentProviderRequestError(suffix string) error {
	return &AgentProviderRequestContractError{Code: "agent_provider_request." + suffix}
}
