package application

import (
	"context"
	"sort"
	"time"

	"orquesta/internal/ports"
)

// ProviderDiscoveryStatus classifies one provider observation without making
// it eligible for routing. Unknown is deliberately distinct from available.
type ProviderDiscoveryStatus string

const (
	ProviderDiscoveryAvailable   ProviderDiscoveryStatus = "available"
	ProviderDiscoveryUnavailable ProviderDiscoveryStatus = "unavailable"
	ProviderDiscoveryUnknown     ProviderDiscoveryStatus = "unknown"
	ProviderDiscoveryExhausted   ProviderDiscoveryStatus = "exhausted"
	ProviderDiscoveryStale       ProviderDiscoveryStatus = "stale"
	ProviderDiscoveryFailed      ProviderDiscoveryStatus = "failed"
)

// ProviderDiscovery is a neutral projection of facts already present in one
// ProviderCatalog query. Observation is empty only when source observation
// failed; it is cloned so callers cannot mutate the catalog or another result.
// Usage carries only observed dimensions and never implies a remaining limit.
type ProviderDiscovery struct {
	ProviderRef string
	Status      ProviderDiscoveryStatus
	Observation ports.ProviderCatalogObservation
	Failure     ProviderCatalogFailure
}

// DiscoverProviderAvailability deterministically projects a catalog snapshot
// at one explicit cutoff taken after observation. It does not route, reserve
// capacity, retry a source, or write lifecycle state.
func DiscoverProviderAvailability(catalog ProviderCatalog, asOf time.Time) ([]ProviderDiscovery, error) {
	if catalog.observedAt.IsZero() || asOf.IsZero() || asOf.Before(catalog.observedAt) {
		return nil, ErrProviderCatalogInvalid
	}

	providerRefs := make([]string, 0, len(catalog.providers)+len(catalog.failures))
	seen := make(map[string]struct{}, len(catalog.providers)+len(catalog.failures))
	for providerRef := range catalog.providers {
		if !validProviderRouteRef(providerRef) {
			return nil, ErrProviderCatalogInvalid
		}
		seen[providerRef] = struct{}{}
		providerRefs = append(providerRefs, providerRef)
	}
	for providerRef := range catalog.failures {
		if !validProviderRouteRef(providerRef) {
			return nil, ErrProviderCatalogInvalid
		}
		if _, duplicate := seen[providerRef]; duplicate {
			return nil, ErrProviderCatalogInvalid
		}
		providerRefs = append(providerRefs, providerRef)
	}
	sort.Strings(providerRefs)

	discovery := make([]ProviderDiscovery, 0, len(providerRefs))
	for _, providerRef := range providerRefs {
		if failure, failed := catalog.failures[providerRef]; failed {
			if failure.ProviderRef != providerRef {
				return nil, ErrProviderCatalogInvalid
			}
			discovery = append(discovery, ProviderDiscovery{
				ProviderRef: providerRef,
				Status:      ProviderDiscoveryFailed,
				Failure:     failure,
			})
			continue
		}

		observation := catalog.providers[providerRef]
		if observation.ProviderRef != providerRef ||
			observation.ObservedAt.After(catalog.observedAt) || observation.ObservedAt.After(asOf) ||
			ports.ValidateProviderCatalogObservation(observation) != nil {
			return nil, ErrProviderCatalogInvalid
		}
		discovery = append(discovery, ProviderDiscovery{
			ProviderRef: providerRef,
			Status:      classifyProviderDiscovery(asOf, observation),
			Observation: cloneProviderObservation(observation),
		})
	}
	return discovery, nil
}

func classifyProviderDiscovery(now time.Time, observation ports.ProviderCatalogObservation) ProviderDiscoveryStatus {
	switch {
	case !now.Before(observation.ExpiresAt):
		return ProviderDiscoveryStale
	case observation.Availability == ports.ProviderAvailabilityUnavailable:
		return ProviderDiscoveryUnavailable
	case observation.Availability == ports.ProviderAvailabilityUnknown:
		return ProviderDiscoveryUnknown
	case observation.Quota == ports.ProviderQuotaUnknown:
		return ProviderDiscoveryUnknown
	case observation.Quota == ports.ProviderQuotaExhausted:
		return ProviderDiscoveryExhausted
	default:
		return ProviderDiscoveryAvailable
	}
}

// DiscoverProviderAvailability observes the sources using the canonical clock
// and returns only their neutral discovery projection.
func (orchestrator *Orchestrator) DiscoverProviderAvailability(ctx context.Context) ([]ProviderDiscovery, error) {
	catalog, err := orchestrator.ObserveProviderCatalog(ctx)
	if err != nil {
		return nil, err
	}
	asOf := orchestrator.clock.Now().Round(0).UTC()
	return DiscoverProviderAvailability(catalog, asOf)
}
