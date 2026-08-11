package codex

import (
	"context"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

// ProviderCatalogSource exposes only facts the Codex adapter owns. ObservedAt
// comes from its configured clock; expiry is supplied explicitly by the caller
// so this adapter does not invent a provider freshness policy or add
// configuration outside the canonical registry.
type ProviderCatalogSource struct {
	observation ports.ProviderCatalogObservation
}

// NewProviderCatalogSource captures one immutable provider snapshot for an
// explicit observation window. Availability remains unknown while the local
// adapter is live: a resolved executable does not prove remote availability.
func NewProviderCatalogSource(
	adapter *Adapter,
	expiresAt time.Time,
) (*ProviderCatalogSource, error) {
	if adapter == nil || adapter.lifecycle == nil {
		return nil, &Error{Code: CodeStateInvalid}
	}
	observedAt := adapter.config.Now().Round(0).UTC()
	expiresAt = expiresAt.Round(0).UTC()
	if observedAt.IsZero() || !expiresAt.After(observedAt) {
		return nil, &Error{Code: CodeStateInvalid}
	}
	model := ports.ProviderModel{ProviderRef: ProviderRef, ModelRef: adapter.modelRef()}
	if err := ports.ValidateProviderModel(model); err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	adapter.mu.Lock()
	unavailable := adapter.closed || context.Cause(adapter.lifecycle) != nil
	adapter.mu.Unlock()
	availability := ports.ProviderAvailabilityUnknown
	if unavailable {
		availability = ports.ProviderAvailabilityUnavailable
	}
	observation := ports.ProviderCatalogObservation{
		ProviderRef:  model.ProviderRef,
		Models:       []ports.ProviderModel{model},
		Availability: availability,
		Quota:        ports.ProviderQuotaUnknown,
		Usage:        unknownCodexUsage(),
		ObservedAt:   observedAt,
		ExpiresAt:    expiresAt,
	}
	if err := ports.ValidateProviderCatalogObservation(observation); err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	return &ProviderCatalogSource{
		observation: observation,
	}, nil
}

func (source *ProviderCatalogSource) ProviderRef() string {
	if source == nil {
		return ""
	}
	return source.observation.ProviderRef
}

func (source *ProviderCatalogSource) ObserveProviderCatalog(
	ctx context.Context,
) (ports.ProviderCatalogObservation, error) {
	if source == nil || ctx == nil {
		return ports.ProviderCatalogObservation{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.ProviderCatalogObservation{}, err
	}
	observation := source.observation
	observation.Models = append([]ports.ProviderModel(nil), source.observation.Models...)
	for index := range observation.Models {
		observation.Models[index].CapabilityRefs = append(
			[]string(nil), source.observation.Models[index].CapabilityRefs...,
		)
		observation.Models[index].ReasoningEfforts = append(
			[]governance.ReasoningEffort(nil), source.observation.Models[index].ReasoningEfforts...,
		)
	}
	return observation, nil
}
