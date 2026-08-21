package application

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestProviderDiscoveryClassifiesFactsDeterministicallyAndFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	sources := []ProviderCatalogSource{
		discoverySource(now, "provider:unknown-quota", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaUnknown),
		providerCatalogSourceStub{providerRef: "provider:failed", err: errors.New("observation failed")},
		discoverySource(now, "provider:unavailable", ports.ProviderAvailabilityUnavailable, ports.ProviderQuotaAvailable),
		discoverySource(now, "provider:available", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable),
		discoverySource(now, "provider:unknown", ports.ProviderAvailabilityUnknown, ports.ProviderQuotaAvailable),
		discoverySource(now, "provider:exhausted", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaExhausted),
		discoverySource(now.Add(-2*time.Minute), "provider:stale", ports.ProviderAvailabilityUnavailable, ports.ProviderQuotaExhausted),
	}
	catalog := mustObserveProviderCatalog(t, now, sources...)

	got, err := DiscoverProviderAvailability(catalog, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		providerRef string
		status      ProviderDiscoveryStatus
	}{
		{"provider:available", ProviderDiscoveryAvailable},
		{"provider:exhausted", ProviderDiscoveryExhausted},
		{"provider:failed", ProviderDiscoveryFailed},
		{"provider:stale", ProviderDiscoveryStale},
		{"provider:unavailable", ProviderDiscoveryUnavailable},
		{"provider:unknown", ProviderDiscoveryUnknown},
		{"provider:unknown-quota", ProviderDiscoveryUnknown},
	}
	if len(got) != len(want) {
		t.Fatalf("discovery count=%d want=%d: %+v", len(got), len(want), got)
	}
	for index := range want {
		if got[index].ProviderRef != want[index].providerRef || got[index].Status != want[index].status {
			t.Fatalf("discovery[%d]=%+v want provider=%q status=%q", index, got[index], want[index].providerRef, want[index].status)
		}
	}
	if got[2].Failure.Code != ProviderCatalogObservationFailed ||
		!reflect.DeepEqual(got[2].Observation, ports.ProviderCatalogObservation{}) {
		t.Fatalf("failed observation leaked or lost failure fact: %+v", got[2])
	}
	if got[0].Observation.Usage != (governance.ResourceUsage{
		Resources: governance.ResourceVector{Tokens: 40},
		Known:     governance.ResourceTokens,
		Quality:   governance.UsageQualityMeasured,
	}) {
		t.Fatalf("observed limits/usage changed: %+v", got[0].Observation.Usage)
	}
}

func TestProviderDiscoveryClonesEveryResultAndPreservesCatalog(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	catalog := mustObserveProviderCatalog(t, now,
		discoverySource(now, "provider:one", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable),
	)

	first, err := DiscoverProviderAvailability(catalog, now)
	if err != nil {
		t.Fatal(err)
	}
	first[0].ProviderRef = "provider:tampered"
	first[0].Observation.Models[0].CapabilityRefs[0] = "capability:tampered"
	first[0].Observation.Models[0].ReasoningEfforts[0] = governance.ReasoningEffortHigh

	second, err := DiscoverProviderAvailability(catalog, now)
	if err != nil {
		t.Fatal(err)
	}
	if second[0].ProviderRef != "provider:one" ||
		second[0].Observation.Models[0].CapabilityRefs[0] != "capability:edit" ||
		second[0].Observation.Models[0].ReasoningEfforts[0] != governance.ReasoningEffortMedium {
		t.Fatalf("discovery leaked mutable output: %+v", second)
	}
}

func TestProviderDiscoveryRejectsInvalidCatalogWithoutRepairingIt(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC)
	valid := discoverySource(now, "provider:one", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable).observation
	future := valid
	future.ObservedAt = now.Add(time.Nanosecond)
	future.ExpiresAt = now.Add(time.Minute)
	for name, catalog := range map[string]ProviderCatalog{
		"zero": {},
		"overlap": {
			observedAt: now,
			providers:  map[string]ports.ProviderCatalogObservation{"provider:one": valid},
			failures: map[string]ProviderCatalogFailure{"provider:one": {
				ProviderRef: "provider:one", Code: ProviderCatalogObservationFailed,
			}},
		},
		"mismatched failure": {
			observedAt: now,
			failures: map[string]ProviderCatalogFailure{"provider:one": {
				ProviderRef: "provider:other", Code: ProviderCatalogObservationFailed,
			}},
		},
		"future observation": {
			observedAt: now,
			providers:  map[string]ports.ProviderCatalogObservation{"provider:one": future},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := DiscoverProviderAvailability(catalog, now); got != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
}

func TestProviderDiscoveryRequiresFreshPostObservationCutoff(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 30, 0, 0, time.UTC)
	catalog := mustObserveProviderCatalog(t, now,
		discoverySource(now, "provider:one", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable),
	)
	if got, err := DiscoverProviderAvailability(catalog, time.Time{}); got != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("zero cutoff got=%+v err=%v", got, err)
	}
	if got, err := DiscoverProviderAvailability(catalog, now.Add(-time.Nanosecond)); got != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("pre-observation cutoff got=%+v err=%v", got, err)
	}
	got, err := DiscoverProviderAvailability(catalog, now.Add(time.Minute))
	if err != nil || len(got) != 1 || got[0].Status != ProviderDiscoveryStale {
		t.Fatalf("post-observation cutoff got=%+v err=%v", got, err)
	}
}

type advancingDiscoverySource struct {
	providerCatalogSourceStub
	clock *mutableClock
}

func (source advancingDiscoverySource) ObserveProviderCatalog(ctx context.Context) (ports.ProviderCatalogObservation, error) {
	source.clock.Advance(2 * time.Minute)
	observation, err := source.providerCatalogSourceStub.ObserveProviderCatalog(ctx)
	observedAt := source.clock.Now()
	observation.ObservedAt = observedAt
	observation.ExpiresAt = observedAt.Add(time.Minute)
	return observation, err
}

func TestProviderDiscoveryOrchestratorUsesPostProbeCutoff(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 45, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	base := discoverySource(now, "provider:one", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable)
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{
		advancingDiscoverySource{providerCatalogSourceStub: base, clock: clock},
	})
	got, err := orchestrator.DiscoverProviderAvailability(context.Background())
	if err != nil || len(got) != 1 || got[0].Status != ProviderDiscoveryAvailable {
		t.Fatalf("post-probe discovery=%+v err=%v", got, err)
	}
}

func TestProviderDiscoveryOrchestratorIsQueryOnlyAndRaceSafe(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	orchestrator, repository, agent := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{
		discoverySource(now, "provider:one", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable),
	})

	want, err := orchestrator.DiscoverProviderAvailability(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errorsFound := make(chan error, 24)
	for index := 0; index < 24; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			got, queryErr := orchestrator.DiscoverProviderAvailability(context.Background())
			if queryErr != nil || !reflect.DeepEqual(got, want) {
				errorsFound <- errors.New("concurrent discovery changed projection")
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}

	repository.mu.Lock()
	goalCount, actionCount, eventCount := len(repository.records), len(repository.actions), len(repository.events)
	repository.mu.Unlock()
	agent.mu.Lock()
	launchCount, observationCount, stopCount := agent.launches, agent.observationCalls, agent.stopCalls
	agent.mu.Unlock()
	if goalCount != 0 || actionCount != 0 || eventCount != 0 || launchCount != 0 || observationCount != 0 || stopCount != 0 {
		t.Fatalf("discovery mutated lifecycle or agents: goals=%d actions=%d events=%d launches=%d observations=%d stops=%d",
			goalCount, actionCount, eventCount, launchCount, observationCount, stopCount)
	}
}

func discoverySource(
	now time.Time,
	providerRef string,
	availability ports.ProviderAvailabilityStatus,
	quota ports.ProviderQuotaStatus,
) providerCatalogSourceStub {
	return providerCatalogSourceStub{
		providerRef: providerRef,
		observation: ports.ProviderCatalogObservation{
			ProviderRef: providerRef,
			Models: []ports.ProviderModel{{
				ProviderRef:      providerRef,
				ModelRef:         "model:default",
				CapabilityRefs:   []string{"capability:edit"},
				ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium},
			}},
			Availability: availability,
			Quota:        quota,
			Usage: governance.ResourceUsage{
				Resources: governance.ResourceVector{Tokens: 40},
				Known:     governance.ResourceTokens,
				Quality:   governance.UsageQualityMeasured,
			},
			ObservedAt: now.Add(-time.Minute),
			ExpiresAt:  now.Add(time.Minute),
		},
	}
}
