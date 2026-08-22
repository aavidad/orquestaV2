package application

import (
	"context"
	"testing"
	"time"
)

func TestAgentEnvironmentLifecycleCompositionRejectsEmptyAndPartial(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	for _, test := range []struct {
		name        string
		composition *AgentEnvironmentLifecycleComposition
	}{
		{name: "empty", composition: &AgentEnvironmentLifecycleComposition{}},
		{name: "store only", composition: &AgentEnvironmentLifecycleComposition{Store: newFakeAgentEnvironmentLifecycleStore(GoalRecord{})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies, _, _ := providerCatalogOrchestratorDependencies(clock, nil)
			dependencies.AgentLifecycle = test.composition
			orchestrator, err := New(dependencies)
			if orchestrator != nil || err == nil || err.Error() != "application.agent_environment_lifecycle_composition_invalid" {
				t.Fatalf("partial composition accepted: orchestrator=%v err=%v", orchestrator, err)
			}
		})
	}
}

func TestAgentEnvironmentLifecycleMethodsFailClosedWithoutComposition(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	dependencies, repository, agent := providerCatalogOrchestratorDependencies(clock, nil)
	orchestrator, err := New(dependencies)
	if err != nil {
		t.Fatal(err)
	}

	initial, created, initializeErr := orchestrator.InitializeAgentEnvironmentLifecycle(
		context.Background(), InitializeAgentEnvironmentLifecycleRequest{},
	)
	advanced, advanceErr := orchestrator.AdvanceAgentEnvironmentLifecycle(
		context.Background(), AdvanceAgentEnvironmentLifecycleRequest{},
	)
	if initializeErr != ErrAgentEnvironmentLifecycleUnavailable || advanceErr != ErrAgentEnvironmentLifecycleUnavailable {
		t.Fatalf("unexpected fail-closed errors: initialize=%v advance=%v", initializeErr, advanceErr)
	}
	if initial != (AgentEnvironmentLifecycleSnapshot{}) || created || advanced != (AgentEnvironmentLifecycleServiceResult{}) {
		t.Fatalf("unavailable lifecycle returned state: initial=%+v created=%t advanced=%+v", initial, created, advanced)
	}
	if len(repository.records) != 0 || len(repository.actions) != 0 || agent.launches != 0 || clock.now.IsZero() {
		t.Fatalf("unavailable lifecycle caused effects: records=%d actions=%d launches=%d",
			len(repository.records), len(repository.actions), agent.launches)
	}
}
