package ports

import (
	"bytes"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestAgentProviderRequestValidatesPreparedStagesAndDetachedClone(t *testing.T) {
	launch := validAgentProviderRequest(t, AgentProviderRequestLaunch)
	if err := ValidatePreparedAgentProviderRequest(launch); err != nil {
		t.Fatal(err)
	}
	bound := CloneAgentProviderRequest(launch)
	bound.LaunchBindingRef = "ejecucion:physical"
	bound.LaunchBindingRevision = 3
	if err := ValidateAgentProviderRequest(bound); err != nil {
		t.Fatal(err)
	}
	if code := AgentProviderRequestContractErrorCode(ValidatePreparedAgentProviderRequest(bound)); code != "agent_provider_request.prepared_launch_already_bound" {
		t.Fatalf("bound prepared code=%q", code)
	}
	clone := CloneAgentProviderRequest(launch)
	clone.Body[0] ^= 0xff
	if bytes.Equal(clone.Body, launch.Body) {
		t.Fatal("CloneAgentProviderRequest aliases body")
	}
	for _, stage := range []AgentProviderRequestStage{AgentProviderRequestSessionStart, AgentProviderRequestSessionInput} {
		request := validAgentProviderRequest(t, stage)
		if err := ValidatePreparedAgentProviderRequest(request); err != nil {
			t.Fatalf("stage %s: %v", stage, err)
		}
	}
}

func TestAgentProviderRequestRejectsCausalAndByteDrift(t *testing.T) {
	base := validAgentProviderRequest(t, AgentProviderRequestLaunch)
	tests := map[string]struct {
		mutate func(*AgentProviderRequest)
		code   string
	}{
		"execution": {func(v *AgentProviderRequest) { v.Key.ExecutionRef = goal.ExecutionRef{} }, "execution_ref_invalid"},
		"fence":     {func(v *AgentProviderRequest) { v.Key.ActionFence = 0 }, "action_fence_invalid"},
		"stage":     {func(v *AgentProviderRequest) { v.Key.Stage = "other" }, "stage_invalid"},
		"attempt":   {func(v *AgentProviderRequest) { v.EffectAttemptRef = " attempt" }, "effect_attempt_ref_invalid"},
		"provider":  {func(v *AgentProviderRequest) { v.ProviderRef = "provider\nother" }, "provider_ref_invalid"},
		"key":       {func(v *AgentProviderRequest) { v.IdempotencyKey = "" }, "idempotency_key_invalid"},
		"empty body": {func(v *AgentProviderRequest) {
			v.Body = nil
			v.BodySHA256 = AgentProviderRequestBodySHA256(nil)
		}, "body_invalid"},
		"large body": {func(v *AgentProviderRequest) {
			v.Body = bytes.Repeat([]byte("x"), maxAgentProviderRequestBodyBytes+1)
			v.BodySHA256 = AgentProviderRequestBodySHA256(v.Body)
		}, "body_invalid"},
		"digest":        {func(v *AgentProviderRequest) { v.BodySHA256 = strings.Repeat("f", 64) }, "body_digest_invalid"},
		"launch target": {func(v *AgentProviderRequest) { v.TargetRef = "ejecucion:other" }, "launch_target_invalid"},
		"partial bind":  {func(v *AgentProviderRequest) { v.LaunchBindingRef = "ejecucion:physical" }, "launch_binding_partial"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := CloneAgentProviderRequest(base)
			test.mutate(&candidate)
			if code := AgentProviderRequestContractErrorCode(ValidateAgentProviderRequest(candidate)); code != "agent_provider_request."+test.code {
				t.Fatalf("code=%q want=%q", code, "agent_provider_request."+test.code)
			}
		})
	}
}

func TestAgentProviderSessionRequestRequiresExactLaunchTarget(t *testing.T) {
	base := validAgentProviderRequest(t, AgentProviderRequestSessionStart)
	for name, mutate := range map[string]func(*AgentProviderRequest){
		"target":   func(v *AgentProviderRequest) { v.TargetRef = "" },
		"revision": func(v *AgentProviderRequest) { v.ExpectedRevision = 0 },
		"binding":  func(v *AgentProviderRequest) { v.LaunchBindingRef = "ejecucion:physical" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := CloneAgentProviderRequest(base)
			mutate(&candidate)
			if code := AgentProviderRequestContractErrorCode(ValidateAgentProviderRequest(candidate)); code != "agent_provider_request.session_target_invalid" {
				t.Fatalf("code=%q", code)
			}
		})
	}
}

func validAgentProviderRequest(t *testing.T, stage AgentProviderRequestStage) AgentProviderRequest {
	t.Helper()
	execution, err := goal.NewExecutionRef("execution:provider-request")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"schema":"request.v1"}`)
	request := AgentProviderRequest{
		Key:              AgentProviderRequestKey{ExecutionRef: execution, ActionFence: 7, Stage: stage},
		EffectAttemptRef: "effect-attempt:provider-request", ProviderRef: "provider:docker",
		IdempotencyKey: "provider-request:sha256:" + strings.Repeat("a", 64),
		Body:           body, BodySHA256: AgentProviderRequestBodySHA256(body),
	}
	if stage != AgentProviderRequestLaunch {
		request.TargetRef = "ejecucion:physical"
		request.ExpectedRevision = 3
	}
	return request
}
