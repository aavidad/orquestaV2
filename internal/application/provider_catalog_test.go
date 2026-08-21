package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestProviderCatalogUsesCutoffObtainedAfterEachProbe(t *testing.T) {
	startedAt := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: startedAt}
	source := providerCatalogSourceStub{
		providerRef: "provider:local",
		observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
			clock.Advance(2 * time.Minute)
			observedAt := clock.Now()
			return providerCatalogObservationAt("provider:local", observedAt, observedAt.Add(time.Minute)), nil
		},
	}
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{source})

	catalog, err := orchestrator.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantCutoff := startedAt.Add(2 * time.Minute)
	observation, found := catalog.Provider("provider:local")
	if !found || catalog.ObservedAt() != wantCutoff || observation.ObservedAt != wantCutoff {
		t.Fatalf("catalog cutoff=%s observation=%+v found=%t", catalog.ObservedAt(), observation, found)
	}
	if failure, found := catalog.Failure("provider:local"); found {
		t.Fatalf("legitimate in-probe observation rejected: %+v", failure)
	}
}

func TestProviderCatalogRejectsObservationAfterPostProbeCutoff(t *testing.T) {
	startedAt := time.Date(2026, 8, 21, 18, 30, 0, 0, time.UTC)
	clock := &mutableClock{now: startedAt}
	source := providerCatalogSourceStub{
		providerRef: "provider:future",
		observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
			clock.Advance(time.Minute)
			cutoff := clock.Now()
			return providerCatalogObservationAt("provider:future", cutoff.Add(time.Nanosecond), cutoff.Add(time.Minute)), nil
		},
	}
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, []ProviderCatalogSource{source})

	catalog, err := orchestrator.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, found := catalog.Provider("provider:future"); found {
		t.Fatal("future observation entered the catalog")
	}
	failure, found := catalog.Failure("provider:future")
	if !found || failure.Code != ProviderCatalogObservationInvalid ||
		catalog.ObservedAt() != startedAt.Add(time.Minute) {
		t.Fatalf("failure=%+v found=%t cutoff=%s", failure, found, catalog.ObservedAt())
	}
}

func TestProviderCatalogLastCutoffCoversAllProbesAndRejectsClockRegression(t *testing.T) {
	startedAt := time.Date(2026, 8, 21, 19, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: startedAt}
	sources := []ProviderCatalogSource{
		providerCatalogSourceStub{
			providerRef: "provider:a",
			observation: providerCatalogObservationAt(
				"provider:a", startedAt, startedAt.Add(time.Minute),
			),
		},
		providerCatalogSourceStub{
			providerRef: "provider:b",
			observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
				clock.Advance(2 * time.Minute)
				observedAt := clock.Now()
				return providerCatalogObservationAt("provider:b", observedAt, observedAt.Add(time.Minute)), nil
			},
		},
	}
	orchestrator, _, _ := newProviderCatalogOrchestrator(t, clock, sources)
	catalog, err := orchestrator.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := DiscoverProviderAvailability(catalog, catalog.ObservedAt())
	if err != nil || len(discovery) != 2 || discovery[0].ProviderRef != "provider:a" ||
		discovery[0].Status != ProviderDiscoveryStale {
		t.Fatalf("post-probe discovery=%+v err=%v", discovery, err)
	}

	regressingClock := &mutableClock{now: startedAt}
	regressing := providerCatalogSourceStub{
		providerRef: "provider:regressing",
		observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
			regressingClock.Advance(-time.Nanosecond)
			return ports.ProviderCatalogObservation{}, errors.New("probe failed")
		},
	}
	orchestrator, _, _ = newProviderCatalogOrchestrator(t, regressingClock, []ProviderCatalogSource{regressing})
	if _, err := orchestrator.ObserveProviderCatalog(context.Background()); !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("regressing clock err=%v", err)
	}
}

func providerCatalogObservationAt(providerRef string, observedAt, expiresAt time.Time) ports.ProviderCatalogObservation {
	return ports.ProviderCatalogObservation{
		ProviderRef:  providerRef,
		Availability: ports.ProviderAvailabilityAvailable,
		Quota:        ports.ProviderQuotaAvailable,
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:   observedAt,
		ExpiresAt:    expiresAt,
	}
}
