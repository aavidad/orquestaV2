package application

import (
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var ErrProviderRouteInvalid = errors.New("application.provider_route_invalid")

type ProviderRouteCandidate struct {
	ProviderRef string
	ModelRef    string
}

// ProviderRouteRequest makes ordering and fallback an explicit application
// policy. Candidates after the first are never tried unless AllowFallback is
// true; providers cannot choose or silently change the route.
type ProviderRouteRequest struct {
	Candidates             []ProviderRouteCandidate
	RequiredCapabilityRefs []string
	ReasoningEffort        governance.ReasoningEffort
	AllowFallback          bool
}

type ProviderRouteReason string

const (
	ProviderRouteSelected            ProviderRouteReason = "selected"
	ProviderRouteNoRoute             ProviderRouteReason = "no_route"
	ProviderRouteFallbackDisabled    ProviderRouteReason = "fallback_disabled"
	ProviderRouteProviderAbsent      ProviderRouteReason = "provider_absent"
	ProviderRouteProviderFailed      ProviderRouteReason = "provider_failed"
	ProviderRouteProviderUnknown     ProviderRouteReason = "provider_unknown"
	ProviderRouteProviderUnavailable ProviderRouteReason = "provider_unavailable"
	ProviderRouteProviderStale       ProviderRouteReason = "provider_stale"
	ProviderRouteQuotaUnknown        ProviderRouteReason = "quota_unknown"
	ProviderRouteQuotaExhausted      ProviderRouteReason = "quota_exhausted"
	ProviderRouteModelAbsent         ProviderRouteReason = "model_absent"
	ProviderRouteCapabilityMissing   ProviderRouteReason = "capability_missing"
	ProviderRouteEffortUnsupported   ProviderRouteReason = "effort_unsupported"
)

type ProviderRouteRejection struct {
	Candidate ProviderRouteCandidate
	Reason    ProviderRouteReason
}

type ProviderRouteDecision struct {
	Selected      bool
	Candidate     ProviderRouteCandidate
	UsedFallback  bool
	Reason        ProviderRouteReason
	ObservedUsage governance.ResourceUsage
	Rejections    []ProviderRouteRejection
}

func RouteProviderModel(catalog ProviderCatalog, request ProviderRouteRequest) (ProviderRouteDecision, error) {
	decision := ProviderRouteDecision{Reason: ProviderRouteNoRoute}
	if err := validateProviderRouteRequest(catalog, request); err != nil {
		return decision, err
	}
	limit := len(request.Candidates)
	if !request.AllowFallback && limit > 1 {
		limit = 1
	}
	for index, candidate := range request.Candidates[:limit] {
		observation, model, reason := providerRouteCandidate(catalog, candidate, request, catalog.observedAt)
		if reason != ProviderRouteSelected {
			decision.Rejections = append(decision.Rejections, ProviderRouteRejection{
				Candidate: candidate,
				Reason:    reason,
			})
			continue
		}
		decision.Selected = true
		decision.Candidate = ProviderRouteCandidate{ProviderRef: model.ProviderRef, ModelRef: model.ModelRef}
		decision.UsedFallback = index > 0
		decision.Reason = ProviderRouteSelected
		decision.ObservedUsage = observation.Usage
		return decision, nil
	}
	if !request.AllowFallback && len(request.Candidates) > 1 {
		decision.Reason = ProviderRouteFallbackDisabled
	}
	return decision, nil
}

func providerRouteCandidate(
	catalog ProviderCatalog,
	candidate ProviderRouteCandidate,
	request ProviderRouteRequest,
	now time.Time,
) (ports.ProviderCatalogObservation, ports.ProviderModel, ProviderRouteReason) {
	if _, failed := catalog.Failure(candidate.ProviderRef); failed {
		return ports.ProviderCatalogObservation{}, ports.ProviderModel{}, ProviderRouteProviderFailed
	}
	observation, found := catalog.Provider(candidate.ProviderRef)
	if !found {
		return ports.ProviderCatalogObservation{}, ports.ProviderModel{}, ProviderRouteProviderAbsent
	}
	if !now.Before(observation.ExpiresAt) {
		return observation, ports.ProviderModel{}, ProviderRouteProviderStale
	}
	switch observation.Availability {
	case ports.ProviderAvailabilityUnknown:
		return observation, ports.ProviderModel{}, ProviderRouteProviderUnknown
	case ports.ProviderAvailabilityUnavailable:
		return observation, ports.ProviderModel{}, ProviderRouteProviderUnavailable
	}
	switch observation.Quota {
	case ports.ProviderQuotaUnknown:
		return observation, ports.ProviderModel{}, ProviderRouteQuotaUnknown
	case ports.ProviderQuotaExhausted:
		return observation, ports.ProviderModel{}, ProviderRouteQuotaExhausted
	}
	for _, model := range observation.Models {
		if model.ModelRef != candidate.ModelRef {
			continue
		}
		if !containsProviderCapabilities(model.CapabilityRefs, request.RequiredCapabilityRefs) {
			return observation, model, ProviderRouteCapabilityMissing
		}
		if !ports.ProviderModelSupports(model, request.RequiredCapabilityRefs, request.ReasoningEffort) {
			return observation, model, ProviderRouteEffortUnsupported
		}
		return observation, model, ProviderRouteSelected
	}
	return observation, ports.ProviderModel{}, ProviderRouteModelAbsent
}

func validateProviderRouteRequest(catalog ProviderCatalog, request ProviderRouteRequest) error {
	if catalog.observedAt.IsZero() || len(request.Candidates) == 0 ||
		governance.ValidateReasoningEffort(request.ReasoningEffort) != nil {
		return ErrProviderRouteInvalid
	}
	seenCandidates := make(map[ProviderRouteCandidate]struct{}, len(request.Candidates))
	for _, candidate := range request.Candidates {
		if !validProviderRouteRef(candidate.ProviderRef) || !validProviderRouteRef(candidate.ModelRef) {
			return ErrProviderRouteInvalid
		}
		if _, duplicate := seenCandidates[candidate]; duplicate {
			return ErrProviderRouteInvalid
		}
		seenCandidates[candidate] = struct{}{}
	}
	seenCapabilities := make(map[string]struct{}, len(request.RequiredCapabilityRefs))
	for _, value := range request.RequiredCapabilityRefs {
		capability, err := goal.NewCapabilityRef(value)
		if err != nil || capability.String() != value {
			return ErrProviderRouteInvalid
		}
		if _, duplicate := seenCapabilities[value]; duplicate {
			return ErrProviderRouteInvalid
		}
		seenCapabilities[value] = struct{}{}
	}
	return nil
}

func containsProviderCapabilities(available, required []string) bool {
	for _, wanted := range required {
		found := false
		for _, actual := range available {
			if actual == wanted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
