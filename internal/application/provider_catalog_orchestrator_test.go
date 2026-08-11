package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestProviderCatalogOrchestratorNormalizesOnceAndUsesCanonicalClock(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	source := &changingProviderCatalogSource{}
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{source})
	if source.calls != 1 {
		t.Fatalf("provider identity reads during composition=%d want 1", source.calls)
	}

	catalog, err := orchestrator.ObserveProviderCatalog(context.Background())
	if err != nil || catalog.ObservedAt() != now {
		t.Fatalf("catalog=%+v observed_at=%s err=%v", catalog, catalog.ObservedAt(), err)
	}
	if failure, found := catalog.Failure("provider:stable"); !found ||
		failure.Code != ProviderCatalogObservationFailed {
		t.Fatalf("normalized provider failure=%+v found=%t", failure, found)
	}

	clock.Advance(time.Minute)
	catalog, err = orchestrator.ObserveProviderCatalog(context.Background())
	if err != nil || catalog.ObservedAt() != now.Add(time.Minute) || source.calls != 1 {
		t.Fatalf("second query observed_at=%s identity_reads=%d err=%v", catalog.ObservedAt(), source.calls, err)
	}
	if _, err := orchestrator.ObserveProviderCatalog(nil); !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("nil query context err=%v", err)
	}
}

func TestProviderCatalogOrchestratorRoutesExplicitFallbackWithoutLifecycleWrites(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	healthy := providerSource(
		now, "provider:healthy", ports.ProviderQuotaAvailable, "model:healthy", "capability:review",
	)
	orchestrator, repository, agent := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{
		providerCatalogSourceStub{providerRef: "provider:failed", err: errors.New("remote unavailable")},
		healthy,
	})

	decision, err := orchestrator.RouteProviderModel(context.Background(), ProviderRouteRequest{
		Candidates: []ProviderRouteCandidate{
			{ProviderRef: "provider:failed", ModelRef: "model:failed"},
			{ProviderRef: "provider:healthy", ModelRef: "model:healthy"},
		},
		RequiredCapabilityRefs: []string{"capability:review"},
		ReasoningEffort:        governance.ReasoningEffortMedium,
		AllowFallback:          true,
	})
	if err != nil || !decision.Selected || !decision.UsedFallback ||
		decision.Candidate != (ProviderRouteCandidate{ProviderRef: "provider:healthy", ModelRef: "model:healthy"}) ||
		len(decision.Rejections) != 1 || decision.Rejections[0].Reason != ProviderRouteProviderFailed {
		t.Fatalf("query-only route=%+v err=%v", decision, err)
	}

	repository.mu.Lock()
	goalCount, actionCount, eventCount := len(repository.records), len(repository.actions), len(repository.events)
	repository.mu.Unlock()
	agent.mu.Lock()
	launchCount, observationCount, stopCount := agent.launches, agent.observationCalls, agent.stopCalls
	agent.mu.Unlock()
	if goalCount != 0 || actionCount != 0 || eventCount != 0 ||
		launchCount != 0 || observationCount != 0 || stopCount != 0 {
		t.Fatalf("provider query wrote lifecycle: goals=%d actions=%d events=%d launch=%d observe=%d stop=%d",
			goalCount, actionCount, eventCount, launchCount, observationCount, stopCount)
	}
}

func TestProviderCatalogOrchestratorRejectsAmbiguousComposition(t *testing.T) {
	now := time.Date(2026, 8, 11, 17, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	source := providerSource(now, "provider:one", ports.ProviderQuotaAvailable, "model:one")
	for name, sources := range map[string][]ProviderCatalogSource{
		"nil":       {nil},
		"duplicate": {source, source},
	} {
		t.Run(name, func(t *testing.T) {
			dependencies, _, _ := providerCatalogOrchestratorDependencies(clock, sources)
			orchestrator, err := New(dependencies)
			if orchestrator != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
				t.Fatalf("orchestrator=%v err=%v", orchestrator, err)
			}
		})
	}
}

func TestProviderCatalogOrchestratorRechecksFreshnessAndSupportsConcurrentQueries(t *testing.T) {
	now := time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	source := providerSource(now, "provider:one", ports.ProviderQuotaAvailable, "model:one")
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{source})
	request := ProviderRouteRequest{
		Candidates:      []ProviderRouteCandidate{{ProviderRef: "provider:one", ModelRef: "model:one"}},
		ReasoningEffort: governance.ReasoningEffortMedium,
	}

	var wait sync.WaitGroup
	errorsFound := make(chan error, 32)
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			decision, err := orchestrator.RouteProviderModel(context.Background(), request)
			if err != nil || !decision.Selected {
				errorsFound <- errors.New("concurrent provider route failed")
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}

	clock.Advance(2 * time.Minute)
	decision, err := orchestrator.RouteProviderModel(context.Background(), request)
	if err != nil || decision.Selected || len(decision.Rejections) != 1 ||
		decision.Rejections[0].Reason != ProviderRouteProviderStale {
		t.Fatalf("stale route=%+v err=%v", decision, err)
	}
}

func newProviderCatalogOrchestrator(
	t *testing.T,
	clock *mutableClock,
	sources []ProviderCatalogSource,
) (*Orchestrator, *memoryRepository, *scriptedAgent) {
	t.Helper()
	dependencies, repository, agent := providerCatalogOrchestratorDependencies(clock, sources)
	orchestrator, err := New(dependencies)
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator, repository, agent
}

func providerCatalogOrchestratorDependencies(
	clock *mutableClock,
	sources []ProviderCatalogSource,
) (Dependencies, *memoryRepository, *scriptedAgent) {
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	return Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities:      testAgentCapabilities(),
		ProviderCatalogSources: sources,
	}, repository, agent
}
