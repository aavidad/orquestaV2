package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestResolveAgentHistoricalRuntimeAuthorityUsesOnlyCausalKey(t *testing.T) {
	authority := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	resolver := &historicalRuntimeAuthorityResolverStub{authority: authority}
	got, err := ResolveAgentHistoricalRuntimeAuthority(context.Background(), resolver, authority.Key)
	if err != nil || got != authority {
		t.Fatalf("resolved=%+v err=%v", got, err)
	}
	if resolver.key != authority.Key {
		t.Fatalf("lookup key=%+v, want %+v", resolver.key, authority.Key)
	}
}

func TestResolveAgentHistoricalRuntimeAuthorityFailsClosed(t *testing.T) {
	authority := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	lookupErr := errors.New("lookup failed")
	tests := map[string]struct {
		ctx      context.Context
		resolver AgentHistoricalRuntimeAuthorityResolver
		key      ports.AgentHistoricalRuntimeAuthorityKey
		want     error
	}{
		"nil context":    {nil, &historicalRuntimeAuthorityResolverStub{authority: authority}, authority.Key, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"nil resolver":   {context.Background(), nil, authority.Key, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"invalid key":    {context.Background(), &historicalRuntimeAuthorityResolverStub{authority: authority}, ports.AgentHistoricalRuntimeAuthorityKey{}, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"lookup error":   {context.Background(), &historicalRuntimeAuthorityResolverStub{err: lookupErr}, authority.Key, lookupErr},
		"crossed key":    {context.Background(), &historicalRuntimeAuthorityResolverStub{authority: applicationHistoricalRuntimeAuthorityFixture(t, "b", 12)}, authority.Key, ErrAgentHistoricalRuntimeAuthorityMismatch},
		"invalid result": {context.Background(), &historicalRuntimeAuthorityResolverStub{authority: mutateHistoricalAuthority(authority, func(value *ports.AgentHistoricalRuntimeAuthority) { value.Digests.KernelSHA256 = "" })}, authority.Key, ErrAgentHistoricalRuntimeAuthorityMismatch},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveAgentHistoricalRuntimeAuthority(testCase.ctx, testCase.resolver, testCase.key)
			if !errors.Is(err, testCase.want) || got != (ports.AgentHistoricalRuntimeAuthority{}) {
				t.Fatalf("resolved=%+v err=%v, want %v", got, err, testCase.want)
			}
		})
	}
}

type historicalRuntimeAuthorityResolverStub struct {
	authority ports.AgentHistoricalRuntimeAuthority
	err       error
	key       ports.AgentHistoricalRuntimeAuthorityKey
}

func (stub *historicalRuntimeAuthorityResolverStub) ResolveAgentHistoricalRuntimeAuthority(_ context.Context, key ports.AgentHistoricalRuntimeAuthorityKey) (ports.AgentHistoricalRuntimeAuthority, error) {
	stub.key = key
	return stub.authority, stub.err
}

func mutateHistoricalAuthority(value ports.AgentHistoricalRuntimeAuthority, mutate func(*ports.AgentHistoricalRuntimeAuthority)) ports.AgentHistoricalRuntimeAuthority {
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
