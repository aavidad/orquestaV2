package application

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var ErrProviderCatalogInvalid = errors.New("application.provider_catalog_invalid")

// ProviderCatalogSource is deliberately read-only. Providers report facts;
// application remains the sole authority for routing and lifecycle writes.
type ProviderCatalogSource interface {
	ProviderRef() string
	ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error)
}

type ProviderCatalogFailureCode string

const (
	ProviderCatalogObservationFailed  ProviderCatalogFailureCode = "provider_observation_failed"
	ProviderCatalogObservationInvalid ProviderCatalogFailureCode = "provider_observation_invalid"
)

type ProviderCatalogFailure struct {
	ProviderRef string
	Code        ProviderCatalogFailureCode
}

// ProviderCatalog is an in-memory decision input, not durable provider state.
// Maps are private so callers cannot mutate observations across Goals.
type ProviderCatalog struct {
	observedAt time.Time
	providers  map[string]ports.ProviderCatalogObservation
	failures   map[string]ProviderCatalogFailure
}

func ObserveProviderCatalog(
	ctx context.Context,
	now time.Time,
	sources []ProviderCatalogSource,
) (ProviderCatalog, error) {
	if ctx == nil || now.IsZero() {
		return ProviderCatalog{}, ErrProviderCatalogInvalid
	}
	normalized, err := normalizeProviderCatalogSources(sources)
	if err != nil {
		return ProviderCatalog{}, err
	}
	return observeNormalizedProviderCatalog(ctx, now, normalized)
}

// ObserveProviderCatalog reads the sources bound at composition time using the
// orchestrator clock. It is a query only: it never claims work or writes Goal
// lifecycle state.
func (orchestrator *Orchestrator) ObserveProviderCatalog(ctx context.Context) (ProviderCatalog, error) {
	if orchestrator == nil || orchestrator.clock == nil {
		return ProviderCatalog{}, ErrProviderCatalogInvalid
	}
	return observeNormalizedProviderCatalog(
		ctx,
		orchestrator.clock.Now().Round(0).UTC(),
		orchestrator.providerCatalogSources,
	)
}

// RouteProviderModel resolves an explicit request against one fresh query. It
// deliberately does not bind the decision to an Action or launch an adapter.
func (orchestrator *Orchestrator) RouteProviderModel(
	ctx context.Context,
	request ProviderRouteRequest,
) (ProviderRouteDecision, error) {
	catalog, err := orchestrator.ObserveProviderCatalog(ctx)
	if err != nil {
		return ProviderRouteDecision{}, err
	}
	return RouteProviderModel(catalog, request)
}

func observeNormalizedProviderCatalog(
	ctx context.Context,
	now time.Time,
	normalized []normalizedProviderCatalogSource,
) (ProviderCatalog, error) {
	if ctx == nil || now.IsZero() {
		return ProviderCatalog{}, ErrProviderCatalogInvalid
	}
	catalog := ProviderCatalog{
		observedAt: now,
		providers:  make(map[string]ports.ProviderCatalogObservation, len(normalized)),
		failures:   make(map[string]ProviderCatalogFailure),
	}
	for _, normalizedSource := range normalized {
		if err := ctx.Err(); err != nil {
			return catalog, err
		}
		providerRef := normalizedSource.providerRef
		observation, observeErr := normalizedSource.source.ObserveProviderCatalog(ctx)
		if err := ctx.Err(); err != nil {
			return catalog, err
		}
		if observeErr != nil {
			catalog.failures[providerRef] = ProviderCatalogFailure{
				ProviderRef: providerRef,
				Code:        ProviderCatalogObservationFailed,
			}
			continue
		}
		if observation.ProviderRef != providerRef || observation.ObservedAt.After(now) ||
			ports.ValidateProviderCatalogObservation(observation) != nil {
			catalog.failures[providerRef] = ProviderCatalogFailure{
				ProviderRef: providerRef,
				Code:        ProviderCatalogObservationInvalid,
			}
			continue
		}
		catalog.providers[providerRef] = cloneProviderObservation(observation)
	}
	return catalog, nil
}

func (catalog ProviderCatalog) ObservedAt() time.Time { return catalog.observedAt }

func (catalog ProviderCatalog) Provider(providerRef string) (ports.ProviderCatalogObservation, bool) {
	observation, found := catalog.providers[providerRef]
	return cloneProviderObservation(observation), found
}

func (catalog ProviderCatalog) Failure(providerRef string) (ProviderCatalogFailure, bool) {
	failure, found := catalog.failures[providerRef]
	return failure, found
}

type normalizedProviderCatalogSource struct {
	providerRef string
	source      ProviderCatalogSource
}

func normalizeProviderCatalogSources(sources []ProviderCatalogSource) ([]normalizedProviderCatalogSource, error) {
	normalized := make([]normalizedProviderCatalogSource, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			return nil, ErrProviderCatalogInvalid
		}
		providerRef := source.ProviderRef()
		if !validProviderRouteRef(providerRef) {
			return nil, ErrProviderCatalogInvalid
		}
		normalized = append(normalized, normalizedProviderCatalogSource{providerRef: providerRef, source: source})
	}
	slices.SortFunc(normalized, func(left, right normalizedProviderCatalogSource) int {
		return cmp.Compare(left.providerRef, right.providerRef)
	})
	for index := 1; index < len(normalized); index++ {
		if normalized[index-1].providerRef == normalized[index].providerRef {
			return nil, ErrProviderCatalogInvalid
		}
	}
	return normalized, nil
}

func cloneProviderObservation(observation ports.ProviderCatalogObservation) ports.ProviderCatalogObservation {
	clone := observation
	clone.Models = append([]ports.ProviderModel(nil), observation.Models...)
	for index := range clone.Models {
		clone.Models[index].CapabilityRefs = append([]string(nil), observation.Models[index].CapabilityRefs...)
		clone.Models[index].ReasoningEfforts = append(
			[]governance.ReasoningEffort(nil), observation.Models[index].ReasoningEfforts...,
		)
	}
	return clone
}

func validProviderRouteRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}
