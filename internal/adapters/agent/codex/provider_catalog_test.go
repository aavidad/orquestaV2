package codex

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var _ application.ProviderCatalogSource = (*ProviderCatalogSource)(nil)

func TestCodexProviderCatalogReportsOnlyOwnedFactsAndFailsClosed(t *testing.T) {
	config := testConfig(t)
	config.Model = "codex-configured"
	observedAt := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	config.Now = func() time.Time { return observedAt }
	adapter := openTestAdapter(t, config)
	source, err := NewProviderCatalogSource(adapter, observedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	observation, err := source.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if source.ProviderRef() != ProviderRef || observation.ProviderRef != ProviderRef ||
		observation.Availability != ports.ProviderAvailabilityUnknown ||
		observation.Quota != ports.ProviderQuotaUnknown || observation.Usage != unknownCodexUsage() ||
		!observation.ObservedAt.Equal(observedAt) || !observation.ExpiresAt.Equal(observedAt.Add(time.Minute)) ||
		len(observation.Models) != 1 || observation.Models[0].ModelRef != config.Model ||
		len(observation.Models[0].CapabilityRefs) != 0 || len(observation.Models[0].ReasoningEfforts) != 0 {
		t.Fatalf("invented Codex provider fact: %+v", observation)
	}
	if err := ports.ValidateProviderCatalogObservation(observation); err != nil {
		t.Fatalf("provider observation invalid: %v", err)
	}

	observation.Models[0].ModelRef = "tampered"
	repeated, err := source.ObserveProviderCatalog(context.Background())
	if err != nil || repeated.Models[0].ModelRef != config.Model {
		t.Fatalf("snapshot identity changed: %+v error=%v", repeated, err)
	}
	catalog, err := application.ObserveProviderCatalog(
		context.Background(), observedAt, []application.ProviderCatalogSource{source},
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := application.RouteProviderModel(catalog, application.ProviderRouteRequest{
		Candidates:      []application.ProviderRouteCandidate{{ProviderRef: ProviderRef, ModelRef: config.Model}},
		ReasoningEffort: governance.ReasoningEffortMedium,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selected || len(decision.Rejections) != 1 ||
		decision.Rejections[0].Reason != application.ProviderRouteProviderUnknown {
		t.Fatalf("unknown Codex provider did not fail closed: %+v", decision)
	}
}

func TestCodexProviderCatalogMarksStoppedAdapterUnavailable(t *testing.T) {
	observedAt := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	config := testConfig(t)
	config.Now = func() time.Time { return observedAt }
	adapter := openTestAdapter(t, config)
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	source, err := NewProviderCatalogSource(adapter, observedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := source.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if observation.Availability != ports.ProviderAvailabilityUnavailable ||
		observation.Quota != ports.ProviderQuotaUnknown || observation.Models[0].ModelRef != DefaultModelRef {
		t.Fatalf("stopped adapter observation = %+v", observation)
	}

	catalog, err := application.ObserveProviderCatalog(
		context.Background(), observedAt, []application.ProviderCatalogSource{source},
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := application.RouteProviderModel(catalog, application.ProviderRouteRequest{
		Candidates:      []application.ProviderRouteCandidate{{ProviderRef: ProviderRef, ModelRef: DefaultModelRef}},
		ReasoningEffort: governance.ReasoningEffortMedium,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selected || len(decision.Rejections) != 1 ||
		decision.Rejections[0].Reason != application.ProviderRouteProviderUnavailable {
		t.Fatalf("unavailable Codex provider did not stay isolated: %+v", decision)
	}
}

func TestCodexProviderCatalogRejectsInvalidSourceAndHonorsCancellation(t *testing.T) {
	now := time.Date(2026, 8, 11, 17, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		adapter   *Adapter
		expiresAt time.Time
	}{
		{name: "nil adapter", expiresAt: now.Add(time.Minute)},
		{name: "zero adapter", adapter: &Adapter{}, expiresAt: now.Add(time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := NewProviderCatalogSource(test.adapter, test.expiresAt)
			if source != nil || ErrorCode(err) != CodeStateInvalid {
				t.Fatalf("source=%v error=%v code=%q", source, err, ErrorCode(err))
			}
		})
	}

	config := testConfig(t)
	config.Now = func() time.Time { return now }
	adapter := openTestAdapter(t, config)
	if source, err := NewProviderCatalogSource(adapter, now); source != nil || ErrorCode(err) != CodeStateInvalid {
		t.Fatalf("non-increasing window source=%v error=%v", source, err)
	}
	source, err := NewProviderCatalogSource(adapter, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.ObserveProviderCatalog(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled observation error=%v", err)
	}
	var nilSource *ProviderCatalogSource
	if nilSource.ProviderRef() != "" {
		t.Fatalf("nil source identity=%q", nilSource.ProviderRef())
	}
	if _, err := nilSource.ObserveProviderCatalog(context.Background()); ErrorCode(err) != CodeUnavailable {
		t.Fatalf("nil source error=%v", err)
	}
}

func TestCodexProviderCatalogConcurrentReadsKeepImmutableSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC)
	config := testConfig(t)
	config.Now = func() time.Time { return now }
	adapter := openTestAdapter(t, config)
	source, err := NewProviderCatalogSource(adapter, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	const readers = 16
	start := make(chan struct{})
	errorsFound := make(chan error, readers)
	var wait sync.WaitGroup
	for index := 0; index < readers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			for attempt := 0; attempt < 20; attempt++ {
				observation, observeErr := source.ObserveProviderCatalog(context.Background())
				if observeErr != nil {
					errorsFound <- observeErr
					return
				}
				if observation.ProviderRef != ProviderRef || len(observation.Models) != 1 ||
					observation.Models[0].ModelRef != DefaultModelRef ||
					observation.Availability != ports.ProviderAvailabilityUnknown {
					errorsFound <- errors.New("unstable provider snapshot")
					return
				}
			}
		}()
	}
	close(start)
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	wait.Wait()
	close(errorsFound)
	for found := range errorsFound {
		t.Fatal(found)
	}
}
