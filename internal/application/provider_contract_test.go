package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type providerCatalogSourceStub struct {
	providerRef string
	observation ports.ProviderCatalogObservation
	err         error
	observe     func(context.Context) (ports.ProviderCatalogObservation, error)
}

type changingProviderCatalogSource struct{ calls int }

func (source *changingProviderCatalogSource) ProviderRef() string {
	source.calls++
	if source.calls == 1 {
		return "provider:stable"
	}
	return "provider:changed"
}

func (source *changingProviderCatalogSource) ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error) {
	return ports.ProviderCatalogObservation{}, errors.New("unavailable")
}

func (source providerCatalogSourceStub) ProviderRef() string { return source.providerRef }

func (source providerCatalogSourceStub) ObserveProviderCatalog(ctx context.Context) (ports.ProviderCatalogObservation, error) {
	if source.observe != nil {
		return source.observe(ctx)
	}
	return source.observation, source.err
}

func TestProviderRoutingReportsAbsentProviderWithoutInventingAvailability(t *testing.T) {
	now := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	catalog, err := ObserveProviderCatalog(context.Background(), now, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := RouteProviderModel(catalog, ProviderRouteRequest{
		Candidates:             []ProviderRouteCandidate{{ProviderRef: "provider:missing", ModelRef: "model:missing"}},
		RequiredCapabilityRefs: []string{"capability:edit"},
		ReasoningEffort:        governance.ReasoningEffortMedium,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selected || decision.Reason != ProviderRouteNoRoute || len(decision.Rejections) != 1 ||
		decision.Rejections[0].Reason != ProviderRouteProviderAbsent {
		t.Fatalf("absent provider must fail closed: %+v", decision)
	}
}

func TestProviderRoutingQuotaUnknownOrExhaustedRequiresExplicitFallback(t *testing.T) {
	now := time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		quota     ports.ProviderQuotaStatus
		rejection ProviderRouteReason
	}{
		{name: "unknown", quota: ports.ProviderQuotaUnknown, rejection: ProviderRouteQuotaUnknown},
		{name: "exhausted", quota: ports.ProviderQuotaExhausted, rejection: ProviderRouteQuotaExhausted},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog := mustObserveProviderCatalog(t, now,
				providerSource(now, "provider:primary", test.quota, "model:primary", "capability:edit"),
				providerSource(now, "provider:fallback", ports.ProviderQuotaAvailable, "model:fallback", "capability:edit"),
			)
			request := ProviderRouteRequest{
				Candidates: []ProviderRouteCandidate{
					{ProviderRef: "provider:primary", ModelRef: "model:primary"},
					{ProviderRef: "provider:fallback", ModelRef: "model:fallback"},
				},
				RequiredCapabilityRefs: []string{"capability:edit"},
				ReasoningEffort:        governance.ReasoningEffortMedium,
			}

			blocked, err := RouteProviderModel(catalog, request)
			if err != nil {
				t.Fatal(err)
			}
			if blocked.Selected || blocked.Reason != ProviderRouteFallbackDisabled ||
				len(blocked.Rejections) != 1 || blocked.Rejections[0].Reason != test.rejection {
				t.Fatalf("fallback must be opt-in: %+v", blocked)
			}

			request.AllowFallback = true
			selected, err := RouteProviderModel(catalog, request)
			if err != nil {
				t.Fatal(err)
			}
			if !selected.Selected || !selected.UsedFallback ||
				selected.Candidate != (ProviderRouteCandidate{ProviderRef: "provider:fallback", ModelRef: "model:fallback"}) ||
				len(selected.Rejections) != 1 || selected.Rejections[0].Reason != test.rejection {
				t.Fatalf("explicit fallback must isolate quota state: %+v", selected)
			}
		})
	}
}

func TestProviderCatalogIsolatesOneSourceFailure(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	healthy := providerSource(now, "provider:healthy", ports.ProviderQuotaAvailable, "model:healthy", "capability:review")
	healthy.observation.Usage = governance.ResourceUsage{
		Resources: governance.ResourceVector{Tokens: 7},
		Known:     governance.ResourceTokens,
		Quality:   governance.UsageQualityMeasured,
	}
	catalog, err := ObserveProviderCatalog(context.Background(), now, []ProviderCatalogSource{
		providerCatalogSourceStub{providerRef: "provider:failed", err: errors.New("remote unavailable")},
		healthy,
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, found := catalog.Failure("provider:failed")
	if !found || failed.Code != ProviderCatalogObservationFailed {
		t.Fatalf("isolated source failure missing: %+v found=%t", failed, found)
	}
	decision, err := RouteProviderModel(catalog, ProviderRouteRequest{
		Candidates: []ProviderRouteCandidate{
			{ProviderRef: "provider:failed", ModelRef: "model:failed"},
			{ProviderRef: "provider:healthy", ModelRef: "model:healthy"},
		},
		RequiredCapabilityRefs: []string{"capability:review"},
		ReasoningEffort:        governance.ReasoningEffortMedium,
		AllowFallback:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Selected || decision.Candidate.ProviderRef != "provider:healthy" ||
		len(decision.Rejections) != 1 || decision.Rejections[0].Reason != ProviderRouteProviderFailed ||
		decision.ObservedUsage != healthy.observation.Usage {
		t.Fatalf("one source failure corrupted the healthy route: %+v", decision)
	}
}

func TestProviderCatalogStopsObservingAfterContextCancellation(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 30, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, err := ObserveProviderCatalog(ctx, now, []ProviderCatalogSource{
		providerCatalogSourceStub{
			providerRef: "provider:a",
			observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
				calls++
				cancel()
				return ports.ProviderCatalogObservation{}, nil
			},
		},
		providerCatalogSourceStub{
			providerRef: "provider:b",
			observe: func(context.Context) (ports.ProviderCatalogObservation, error) {
				calls++
				return ports.ProviderCatalogObservation{}, nil
			},
		},
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("cancelled catalog continued observing: err=%v calls=%d", err, calls)
	}
}

func TestProviderRoutingNeverInfersCapabilityOrModelParity(t *testing.T) {
	now := time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)
	source := providerSource(now, "provider:review-like", ports.ProviderQuotaAvailable, "model:review-like")
	catalog := mustObserveProviderCatalog(t, now, source)
	decision, err := RouteProviderModel(catalog, ProviderRouteRequest{
		Candidates:             []ProviderRouteCandidate{{ProviderRef: "provider:review-like", ModelRef: "model:review-like"}},
		RequiredCapabilityRefs: []string{"capability:review"},
		ReasoningEffort:        governance.ReasoningEffortMedium,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selected || len(decision.Rejections) != 1 ||
		decision.Rejections[0].Reason != ProviderRouteCapabilityMissing {
		t.Fatalf("names must not manufacture advertised parity: %+v", decision)
	}
}

func TestProviderCatalogRejectsAmbiguousSourcesAndCopiesObservedCapabilities(t *testing.T) {
	now := time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)
	source := providerSource(now, "provider:one", ports.ProviderQuotaAvailable, "model:one", "capability:edit")
	if _, err := ObserveProviderCatalog(context.Background(), now, []ProviderCatalogSource{source, source}); !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("duplicate provider source err=%v", err)
	}

	catalog := mustObserveProviderCatalog(t, now, source)
	first, found := catalog.Provider("provider:one")
	if !found {
		t.Fatal("provider observation missing")
	}
	first.Models[0].CapabilityRefs[0] = "capability:tampered"
	second, _ := catalog.Provider("provider:one")
	if second.Models[0].CapabilityRefs[0] != "capability:edit" {
		t.Fatalf("catalog leaked mutable provider state: %+v", second)
	}
}

func TestProviderCatalogSnapshotsEachSourceIdentityExactlyOnce(t *testing.T) {
	now := time.Date(2026, 8, 11, 14, 30, 0, 0, time.UTC)
	source := &changingProviderCatalogSource{}
	catalog, err := ObserveProviderCatalog(context.Background(), now, []ProviderCatalogSource{source})
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 {
		t.Fatalf("provider identity reads=%d want 1", source.calls)
	}
	if failure, found := catalog.Failure("provider:stable"); !found || failure.ProviderRef != "provider:stable" {
		t.Fatalf("snapshotted provider failure=%+v found=%t", failure, found)
	}
}

func mustObserveProviderCatalog(t *testing.T, now time.Time, sources ...ProviderCatalogSource) ProviderCatalog {
	t.Helper()
	catalog, err := ObserveProviderCatalog(context.Background(), now, sources)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func providerSource(
	now time.Time,
	providerRef string,
	quota ports.ProviderQuotaStatus,
	modelRef string,
	capabilityRefs ...string,
) providerCatalogSourceStub {
	return providerCatalogSourceStub{
		providerRef: providerRef,
		observation: ports.ProviderCatalogObservation{
			ProviderRef:  providerRef,
			Availability: ports.ProviderAvailabilityAvailable,
			Quota:        quota,
			Usage: governance.ResourceUsage{
				Quality: governance.UsageQualityUnknown,
			},
			ObservedAt: now.Add(-time.Minute),
			ExpiresAt:  now.Add(time.Minute),
			Models: []ports.ProviderModel{{
				ProviderRef:      providerRef,
				ModelRef:         modelRef,
				CapabilityRefs:   append([]string(nil), capabilityRefs...),
				ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium},
			}},
		},
	}
}
