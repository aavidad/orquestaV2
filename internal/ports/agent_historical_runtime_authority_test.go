package ports

import (
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func TestAgentHistoricalRuntimeAuthorityRequiresExactNeutralBinding(t *testing.T) {
	authority := historicalRuntimeAuthorityFixture(t)
	if err := ValidateAgentHistoricalRuntimeAuthority(authority); err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		mutate func(*AgentHistoricalRuntimeAuthority)
		code   string
	}{
		"fence":     {func(value *AgentHistoricalRuntimeAuthority) { value.Key.ActionFence = 0 }, "agent.historical_runtime_authority_action_fence_invalid"},
		"execution": {func(value *AgentHistoricalRuntimeAuthority) { value.Key.ExecutionRef = goal.ExecutionRef{} }, "agent.historical_runtime_authority_execution_ref_mismatch"},
		"subject":   {func(value *AgentHistoricalRuntimeAuthority) { value.Subject.GoalRef = goal.GoalRef{} }, "agent.historical_runtime_authority_subject_invalid"},
		"plan":      {func(value *AgentHistoricalRuntimeAuthority) { value.Digests.PlanSHA256 = "" }, "agent.historical_runtime_authority_digest_invalid"},
		"grant":     {func(value *AgentHistoricalRuntimeAuthority) { value.Digests.GrantSHA256 = strings.Repeat("A", 64) }, "agent.historical_runtime_authority_digest_invalid"},
		"kernel":    {func(value *AgentHistoricalRuntimeAuthority) { value.Digests.KernelSHA256 = strings.Repeat("0", 63) }, "agent.historical_runtime_authority_digest_invalid"},
		"initramfs": {func(value *AgentHistoricalRuntimeAuthority) { value.Digests.InitramfsSHA256 = strings.Repeat("g", 64) }, "agent.historical_runtime_authority_digest_invalid"},
		"profile":   {func(value *AgentHistoricalRuntimeAuthority) { value.Digests.ProfileSHA256 = "" }, "agent.historical_runtime_authority_digest_invalid"},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := authority
			testCase.mutate(&candidate)
			err := ValidateAgentHistoricalRuntimeAuthority(candidate)
			if got := AgentHistoricalRuntimeAuthorityErrorCode(err); got != testCase.code {
				t.Fatalf("error code=%q err=%v, want %q", got, err, testCase.code)
			}
		})
	}
}

func TestAgentHistoricalRuntimeAuthorityContainsNoMaterialOrTransport(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(AgentHistoricalRuntimeAuthority{}),
		reflect.TypeOf(AgentHistoricalRuntimeAuthorityKey{}),
		reflect.TypeOf(AgentHistoricalRuntimeDigests{}),
	} {
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if field.Type.Kind() == reflect.Slice {
				t.Fatalf("%s contains mutable material field %s", typ.Name(), field.Name)
			}
			lower := strings.ToLower(field.Name)
			for _, forbidden := range []string{"path", "content", "payload", "endpoint", "socket", "credential", "secret", "url"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("%s exposes forbidden field %s", typ.Name(), field.Name)
				}
			}
		}
	}
}

func historicalRuntimeAuthorityFixture(t *testing.T) AgentHistoricalRuntimeAuthority {
	t.Helper()
	executionRef, err := goal.NewExecutionRef("execution:b12-authority")
	if err != nil {
		t.Fatal(err)
	}
	goalRef, err := goal.NewGoalRef("goal:b12-authority")
	if err != nil {
		t.Fatal(err)
	}
	workItemRef, err := goal.NewWorkItemRef("work-item:b12-authority")
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	return AgentHistoricalRuntimeAuthority{
		Key: AgentHistoricalRuntimeAuthorityKey{ExecutionRef: executionRef, ActionFence: 7},
		Subject: AgentEnvironmentLifecycleSubject{
			ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
			PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
			SpecHash: digest, ProviderRef: "provider:local", ModelRef: "model:exact",
			AgentRef: "agent:exact", ExternalRef: "physical:opaque",
		},
		Digests: AgentHistoricalRuntimeDigests{
			PlanSHA256: digest, GrantSHA256: digest, KernelSHA256: digest,
			InitramfsSHA256: digest, ProfileSHA256: digest,
		},
	}
}
