package acceptance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type v25DiscoverySource struct {
	providerRef string
	observation ports.ProviderCatalogObservation
	err         error
}

func (source v25DiscoverySource) ProviderRef() string { return source.providerRef }

func (source v25DiscoverySource) ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error) {
	return source.observation, source.err
}

func TestAcceptanceV25ProviderDiscoveryIsNeutralFreshAndFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 21, 14, 0, 0, 0, time.UTC)
	future := v25ProviderDiscoverySource(
		now, "provider:future", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable, now.Add(time.Minute),
	)
	future.observation.ObservedAt = now.Add(time.Nanosecond)
	sources := []application.ProviderCatalogSource{
		v25ProviderDiscoverySource(now, "provider:unknown", ports.ProviderAvailabilityUnknown, ports.ProviderQuotaAvailable, now.Add(time.Minute)),
		v25ProviderDiscoverySource(now, "provider:available", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable, now.Add(time.Minute)),
		v25ProviderDiscoverySource(now, "provider:exhausted", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaExhausted, now.Add(time.Minute)),
		v25ProviderDiscoverySource(now, "provider:unavailable", ports.ProviderAvailabilityUnavailable, ports.ProviderQuotaAvailable, now.Add(time.Minute)),
		v25ProviderDiscoverySource(now, "provider:stale", ports.ProviderAvailabilityAvailable, ports.ProviderQuotaAvailable, now),
		v25DiscoverySource{providerRef: "provider:failed", err: errors.New("provider unavailable")},
		future,
	}
	catalog, err := application.ObserveProviderCatalog(context.Background(), now, sources)
	if err != nil {
		t.Fatal(err)
	}

	discovery, err := application.DiscoverProviderAvailability(catalog, now)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]application.ProviderDiscoveryStatus{
		"provider:available":   application.ProviderDiscoveryAvailable,
		"provider:exhausted":   application.ProviderDiscoveryExhausted,
		"provider:failed":      application.ProviderDiscoveryFailed,
		"provider:future":      application.ProviderDiscoveryFailed,
		"provider:stale":       application.ProviderDiscoveryStale,
		"provider:unavailable": application.ProviderDiscoveryUnavailable,
		"provider:unknown":     application.ProviderDiscoveryUnknown,
	}
	if len(discovery) != len(want) {
		t.Fatalf("discovery=%+v", discovery)
	}
	for index, fact := range discovery {
		if index > 0 && discovery[index-1].ProviderRef >= fact.ProviderRef {
			t.Fatalf("provider discovery is not deterministically ordered: %+v", discovery)
		}
		if fact.Status != want[fact.ProviderRef] {
			t.Fatalf("provider %q status=%q want=%q", fact.ProviderRef, fact.Status, want[fact.ProviderRef])
		}
	}
	if discovery[0].Observation.Usage.Quality != governance.UsageQualityExact ||
		discovery[0].Observation.Usage.Known != governance.ResourceTokens {
		t.Fatalf("observed usage/limit facts were not preserved: %+v", discovery[0].Observation.Usage)
	}

	discovery[0].Observation.Models[0].CapabilityRefs[0] = "capability:tampered"
	repeated, err := application.DiscoverProviderAvailability(catalog, now)
	if err != nil || repeated[0].Observation.Models[0].CapabilityRefs[0] != "capability:edit" {
		t.Fatalf("caller mutated discovery/catalog: discovery=%+v err=%v", repeated, err)
	}
	if _, found := catalog.Provider("provider:available"); !found {
		t.Fatal("query-only discovery removed a provider from the catalog")
	}
	late, err := application.DiscoverProviderAvailability(catalog, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range late {
		if fact.Status != application.ProviderDiscoveryFailed && fact.Status != application.ProviderDiscoveryStale {
			t.Fatalf("post-observation cutoff kept stale fact usable: %+v", fact)
		}
	}
}

func v25ProviderDiscoverySource(
	now time.Time,
	providerRef string,
	availability ports.ProviderAvailabilityStatus,
	quota ports.ProviderQuotaStatus,
	expiresAt time.Time,
) v25DiscoverySource {
	return v25DiscoverySource{
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
				Resources: governance.ResourceVector{Tokens: 12},
				Known:     governance.ResourceTokens,
				Quality:   governance.UsageQualityExact,
			},
			ObservedAt: now.Add(-time.Minute),
			ExpiresAt:  expiresAt,
		},
	}
}
