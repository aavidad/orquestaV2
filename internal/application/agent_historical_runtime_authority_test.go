package application

import (
	"errors"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestResolveAgentHistoricalRuntimeAuthorityIsExactAndFailClosed(t *testing.T) {
	expected := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	peer := applicationHistoricalRuntimeAuthorityFixture(t, "b", 12)

	got, err := ResolveAgentHistoricalRuntimeAuthority([]ports.AgentHistoricalRuntimeAuthority{peer, expected}, expected)
	if err != nil || got != expected {
		t.Fatalf("resolved=%+v err=%v", got, err)
	}

	tests := map[string]struct {
		history []ports.AgentHistoricalRuntimeAuthority
		want    error
	}{
		"absent":          {[]ports.AgentHistoricalRuntimeAuthority{peer}, ErrAgentHistoricalRuntimeAuthorityNotFound},
		"duplicate exact": {[]ports.AgentHistoricalRuntimeAuthority{expected, expected}, ErrAgentHistoricalRuntimeAuthorityAmbiguous},
		"crossed subject": {[]ports.AgentHistoricalRuntimeAuthority{mutateHistoricalAuthority(expected, func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.ProviderRef = "provider:peer"
		})}, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"crossed runtime": {[]ports.AgentHistoricalRuntimeAuthority{mutateHistoricalAuthority(expected, func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.ProfileSHA256 = strings.Repeat("b", 64)
		})}, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"invalid matching history": {[]ports.AgentHistoricalRuntimeAuthority{mutateHistoricalAuthority(expected, func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.KernelSHA256 = ""
		})}, ErrAgentHistoricalRuntimeAuthorityMismatch},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveAgentHistoricalRuntimeAuthority(testCase.history, expected)
			if !errors.Is(err, testCase.want) || got != (ports.AgentHistoricalRuntimeAuthority{}) {
				t.Fatalf("resolved=%+v err=%v, want %v", got, err, testCase.want)
			}
		})
	}
}

func TestResolveAgentHistoricalRuntimeAuthorityDoesNotOrderFences(t *testing.T) {
	expected := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	laterRecovery := expected
	laterRecovery.Key.ActionFence++
	if _, err := ResolveAgentHistoricalRuntimeAuthority([]ports.AgentHistoricalRuntimeAuthority{laterRecovery}, expected); !errors.Is(err, ErrAgentHistoricalRuntimeAuthorityNotFound) {
		t.Fatalf("later recovery fence crossed historical lookup: %v", err)
	}
}

func TestResolveAgentHistoricalRuntimeAuthorityRejectsEveryCrossedField(t *testing.T) {
	expected := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	peer := applicationHistoricalRuntimeAuthorityFixture(t, "b", 11)
	mutations := map[string]func(*ports.AgentHistoricalRuntimeAuthority){
		"subject execution": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.ExecutionRef = peer.Subject.ExecutionRef
		},
		"goal": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.GoalRef = peer.Subject.GoalRef },
		"work item": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.WorkItemRef = peer.Subject.WorkItemRef
		},
		"plan generation": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.PlanGeneration++ },
		"spec generation": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.AppSpecGeneration++ },
		"attempt":         func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.ExecutionAttempt++ },
		"spec":            func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.SpecHash = peer.Subject.SpecHash },
		"provider": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.ProviderRef = peer.Subject.ProviderRef
		},
		"model": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.ModelRef = peer.Subject.ModelRef },
		"agent": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Subject.AgentRef = peer.Subject.AgentRef },
		"external": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.ExternalRef = peer.Subject.ExternalRef
		},
		"plan digest": func(value *ports.AgentHistoricalRuntimeAuthority) { value.Digests.PlanSHA256 = peer.Digests.PlanSHA256 },
		"grant digest": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.GrantSHA256 = peer.Digests.GrantSHA256
		},
		"kernel digest": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.KernelSHA256 = peer.Digests.KernelSHA256
		},
		"initramfs digest": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.InitramfsSHA256 = peer.Digests.InitramfsSHA256
		},
		"profile digest": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Digests.ProfileSHA256 = peer.Digests.ProfileSHA256
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			crossed := mutateHistoricalAuthority(expected, mutate)
			if _, err := ResolveAgentHistoricalRuntimeAuthority([]ports.AgentHistoricalRuntimeAuthority{crossed}, expected); !errors.Is(err, ErrAgentHistoricalRuntimeAuthorityMismatch) {
				t.Fatalf("crossed field accepted: %v", err)
			}
		})
	}
}

func mutateHistoricalAuthority(
	value ports.AgentHistoricalRuntimeAuthority,
	mutate func(*ports.AgentHistoricalRuntimeAuthority),
) ports.AgentHistoricalRuntimeAuthority {
	mutate(&value)
	return value
}

func applicationHistoricalRuntimeAuthorityFixture(t *testing.T, suffix string, fence uint64) ports.AgentHistoricalRuntimeAuthority {
	t.Helper()
	executionRef, err := goal.NewExecutionRef("execution:b12-" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	goalRef, err := goal.NewGoalRef("goal:b12-" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	workItemRef, err := goal.NewWorkItemRef("work-item:b12-" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat(suffix, 64)
	return ports.AgentHistoricalRuntimeAuthority{
		Key: ports.AgentHistoricalRuntimeAuthorityKey{ExecutionRef: executionRef, ActionFence: fence},
		Subject: ports.AgentEnvironmentLifecycleSubject{
			ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
			PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
			SpecHash: digest, ProviderRef: "provider:" + suffix, ModelRef: "model:" + suffix,
			AgentRef: "agent:" + suffix, ExternalRef: "physical:" + suffix,
		},
		Digests: ports.AgentHistoricalRuntimeDigests{
			PlanSHA256: digest, GrantSHA256: digest, KernelSHA256: digest,
			InitramfsSHA256: digest, ProfileSHA256: digest,
		},
	}
}
