package ports

import (
	"bytes"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestAgentProviderRequestValidatesPreparedStagesAndDetachedClone(t *testing.T) {
	launch := validAgentProviderRequest(AgentProviderRequestLaunch)
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
		request := validAgentProviderRequest(stage)
		if err := ValidatePreparedAgentProviderRequest(request); err != nil {
			t.Fatalf("stage %s: %v", stage, err)
		}
	}
}

func TestAgentProviderRequestRejectsCausalAndByteDrift(t *testing.T) {
	tests := map[string]struct {
		mutate func(*AgentProviderRequest)
		code   string
		stage  AgentProviderRequestStage
	}{
		"execution": {mutate: func(v *AgentProviderRequest) { v.Key.ExecutionRef = goal.ExecutionRef{} }, code: "execution_ref_invalid"},
		"fence":     {mutate: func(v *AgentProviderRequest) { v.Key.ActionFence = 0 }, code: "action_fence_invalid"},
		"stage":     {mutate: func(v *AgentProviderRequest) { v.Key.Stage = "other" }, code: "stage_invalid"},
		"attempt":   {mutate: func(v *AgentProviderRequest) { v.EffectAttemptRef = " attempt" }, code: "effect_attempt_ref_invalid"},
		"provider":  {mutate: func(v *AgentProviderRequest) { v.ProviderRef = "provider\nother" }, code: "provider_ref_invalid"},
		"key":       {mutate: func(v *AgentProviderRequest) { v.IdempotencyKey = "" }, code: "idempotency_key_invalid"},
		"empty body": {mutate: func(v *AgentProviderRequest) {
			v.Body = nil
			v.BodySHA256 = AgentProviderRequestBodySHA256(nil)
		}, code: "body_invalid"},
		"large body": {mutate: func(v *AgentProviderRequest) {
			v.Body = bytes.Repeat([]byte("x"), maxAgentProviderRequestBodyBytes+1)
			v.BodySHA256 = AgentProviderRequestBodySHA256(v.Body)
		}, code: "body_invalid"},
		"digest":        {mutate: func(v *AgentProviderRequest) { v.BodySHA256 = strings.Repeat("f", 64) }, code: "body_digest_invalid"},
		"launch target": {mutate: func(v *AgentProviderRequest) { v.TargetRef = "ejecucion:other" }, code: "launch_target_invalid"},
		"partial bind":  {mutate: func(v *AgentProviderRequest) { v.LaunchBindingRef = "ejecucion:physical" }, code: "launch_binding_partial"},
		"session target": {mutate: func(v *AgentProviderRequest) {
			v.TargetRef = ""
		}, code: "session_target_invalid", stage: AgentProviderRequestSessionStart},
		"session revision": {mutate: func(v *AgentProviderRequest) {
			v.ExpectedRevision = 0
		}, code: "session_target_invalid", stage: AgentProviderRequestSessionStart},
		"session binding": {mutate: func(v *AgentProviderRequest) {
			v.LaunchBindingRef = "ejecucion:physical"
		}, code: "session_target_invalid", stage: AgentProviderRequestSessionStart},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stage := test.stage
			if stage == "" {
				stage = AgentProviderRequestLaunch
			}
			candidate := validAgentProviderRequest(stage)
			test.mutate(&candidate)
			if code := AgentProviderRequestContractErrorCode(ValidateAgentProviderRequest(candidate)); code != "agent_provider_request."+test.code {
				t.Fatalf("code=%q want=%q", code, "agent_provider_request."+test.code)
			}
		})
	}
}

func validAgentProviderRequest(stage AgentProviderRequestStage) AgentProviderRequest {
	execution, _ := goal.NewExecutionRef("execution:provider-request")
	body := []byte(`{"schema":"request.v1"}`)
	request := AgentProviderRequest{
		Key:              AgentProviderRequestKey{ExecutionRef: execution, ActionFence: 7, Stage: stage},
		EffectAttemptRef: "effect-attempt:provider-request", ProviderRef: "provider:docker",
		IdempotencyKey: "provider-request:sha256:" + strings.Repeat("a", 64),
		Body:           body, BodySHA256: AgentProviderRequestBodySHA256(body),
	}
	if stage != AgentProviderRequestLaunch {
		request.TargetRef, request.ExpectedRevision = "ejecucion:physical", 3
	}
	return request
}
