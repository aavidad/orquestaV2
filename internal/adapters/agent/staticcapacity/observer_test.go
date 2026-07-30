package staticcapacity

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestObserverContract(t *testing.T) {
	config := validConfig()
	observer, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	got, err := observer.ObserveCapacity(context.Background(), config.Observation.SourceRef, config.Observation.PoolRef)
	if err != nil {
		t.Fatal(err)
	}
	if got != config.Observation {
		t.Fatalf("configured facts lost: %#v", got)
	}
	nextConfig := config
	nextConfig.Observation.WindowRef = "window:next"
	next, err := New(nextConfig)
	if err != nil || next.observation.WindowRef == got.WindowRef {
		t.Fatalf("new window was not represented: %v", err)
	}
	invalid := []Config{validConfig(), validConfig(), validConfig()}
	invalid[0].Observation.Resources.Messages = applicable(1, 1)
	invalid[1].Observation.Resources.Credits = applicable(1, 1)
	invalid[2].Observation.ExpiresAt = invalid[2].Observation.ObservedAt
	for index, value := range invalid {
		if _, err = New(value); err != ErrInvalidConfig {
			t.Fatalf("invalid config %d error = %v", index, err)
		}
	}
	scopes := [][2]string{{"other", "pool:default"}, {"source:static", "other"}}
	for index, scope := range scopes {
		if _, err = observer.ObserveCapacity(context.Background(), application.AgentCapacitySourceRef(scope[0]), application.AgentCapacityPoolRef(scope[1])); err != ErrScopeMismatch {
			t.Fatalf("scope %d error = %v", index, err)
		}
	}
}

func validConfig() Config {
	observed := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	notApplicable := application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityNotApplicable}
	return Config{Observation: application.AgentCapacityObservation{
		SourceRef: "source:static", PoolRef: "pool:default", WindowRef: "window:first",
		Status: application.AgentCapacityAvailable, Quality: "exact",
		ObservedAt: observed, ExpiresAt: observed.Add(time.Minute),
		ResetAt: observed.Add(time.Hour), RetryAt: observed.Add(30 * time.Second),
		Resources: application.AgentCapacityResources{
			Slots: applicable(4, 0), Seconds: notApplicable, Messages: notApplicable,
			Tokens: notApplicable, Credits: notApplicable,
		},
	}}
}

func applicable(limit, remaining int64) application.AgentCapacityDimension {
	return application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityApplicable, Limit: application.AgentCapacityAmount{Present: true, Value: limit}, Remaining: application.AgentCapacityAmount{Present: true, Value: remaining}}
}
