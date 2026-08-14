package ports

import (
	"bytes"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestAgentProviderStopRequestValidatesAndClonesDetachedBytes(t *testing.T) {
	request := validAgentProviderStopRequest(t)
	if err := ValidateAgentProviderStopRequest(request); err != nil {
		t.Fatal(err)
	}
	clone := CloneAgentProviderStopRequest(request)
	clone.Body[0] ^= 0xff
	if bytes.Equal(clone.Body, request.Body) {
		t.Fatal("CloneAgentProviderStopRequest aliases body")
	}
}

func TestAgentProviderStopRequestRejectsEveryCausalAndByteMutation(t *testing.T) {
	tests := map[string]struct {
		mutate func(*AgentProviderStopRequest)
		code   string
	}{
		"execution": {func(v *AgentProviderStopRequest) { v.Key.ExecutionRef = goal.ExecutionRef{} }, "execution_ref_invalid"},
		"launch fence": {func(v *AgentProviderStopRequest) { v.Key.LaunchActionFence = 0 }, "launch_action_fence_invalid"},
		"stop fence": {func(v *AgentProviderStopRequest) { v.Key.StopActionFence = 0 }, "stop_action_fence_invalid"},
		"attempt": {func(v *AgentProviderStopRequest) { v.StopEffectAttemptRef = " attempt" }, "effect_attempt_ref_invalid"},
		"provider": {func(v *AgentProviderStopRequest) { v.ProviderRef = "provider\nother" }, "provider_ref_invalid"},
		"idempotency": {func(v *AgentProviderStopRequest) { v.IdempotencyKey = "" }, "idempotency_key_invalid"},
		"target": {func(v *AgentProviderStopRequest) { v.TargetRef = " target" }, "target_ref_invalid"},
		"revision": {func(v *AgentProviderStopRequest) { v.ExpectedRevision = 0 }, "expected_revision_invalid"},
		"empty body": {func(v *AgentProviderStopRequest) {
			v.Body = nil
			v.BodySHA256 = AgentProviderRequestBodySHA256(nil)
		}, "body_invalid"},
		"large body": {func(v *AgentProviderStopRequest) {
			v.Body = bytes.Repeat([]byte("x"), maxAgentProviderRequestBodyBytes+1)
			v.BodySHA256 = AgentProviderRequestBodySHA256(v.Body)
		}, "body_invalid"},
		"digest": {func(v *AgentProviderStopRequest) { v.BodySHA256 = strings.Repeat("f", 64) }, "body_digest_invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentProviderStopRequest(t)
			test.mutate(&request)
			if code := AgentProviderStopRequestContractErrorCode(ValidateAgentProviderStopRequest(request));
				code != "agent_provider_stop_request."+test.code {
				t.Fatalf("code=%q request=%+v", code, request)
			}
		})
	}
}

func validAgentProviderStopRequest(t *testing.T) AgentProviderStopRequest {
	t.Helper()
	execution, err := goal.NewExecutionRef("execution:provider-stop")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"schema":"provider.stop.v1"}`)
	return AgentProviderStopRequest{
		Key: AgentProviderStopRequestKey{
			ExecutionRef: execution, LaunchActionFence: 7, StopActionFence: 11,
		},
		StopEffectAttemptRef: "effect-attempt:provider-stop", ProviderRef: "provider:isolated",
		IdempotencyKey: "provider-stop:1", TargetRef: "runtime:physical:1", ExpectedRevision: 3,
		Body: body, BodySHA256: AgentProviderRequestBodySHA256(body),
	}
}
