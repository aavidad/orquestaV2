package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

type agentCapacityClock struct{ now time.Time }

func (clock agentCapacityClock) Now() time.Time { return clock.now }

type agentCapacityObserverFake struct{ observation AgentCapacityObservation }

func (fake agentCapacityObserverFake) ObserveCapacity(
	context.Context,
	AgentCapacitySourceRef,
	AgentCapacityPoolRef,
) (AgentCapacityObservation, error) {
	return fake.observation, nil
}

func TestAgentCapacityObserverContract(t *testing.T) {
	observation := validAgentCapacityObservation(t)
	observation.Resources.Seconds = capacityDimension(60, 50)
	observation.Resources.Messages = capacityDimension(20, 10)
	observation.Resources.Tokens = capacityDimension(1_000, 700)
	observation.Resources.Credits = capacityDimension(12, 2)
	var observer AgentCapacityObserver = agentCapacityObserverFake{observation: observation}
	got, err := observer.ObserveCapacity(
		context.Background(), observation.SourceRef, observation.PoolRef,
	)
	if err != nil || got.WindowRef != observation.WindowRef ||
		got.ArtifactRef.String() == "" || got.Resources.Credits.Remaining.Value != 2 {
		t.Fatalf("fake contractual incoherente: got=%+v err=%v", got, err)
	}
	if err := ValidateAgentCapacityObservation(got); err != nil {
		t.Fatalf("observación tipada inválida: %v", err)
	}
}

func TestAgentCapacityAdmissionFailsClosedWithoutBlockingControl(t *testing.T) {
	base := agentCapacityBaseTime()
	cases := []struct {
		name      string
		mutate    func(*AgentCapacityObservation)
		now       time.Time
		sourceErr error
		reason    AgentCapacityAdmissionReason
		admit     int64
	}{
		{name: "disponible", reason: AgentCapacityAdmissionAvailable, admit: 3},
		{name: "cero presente", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Slots = capacityDimension(3, 0)
		}, reason: AgentCapacityAdmissionExhausted},
		{name: "restante de slots ausente", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Slots.Remaining = AgentCapacityAmount{}
		}, reason: AgentCapacityAdmissionUnknown},
		{name: "restante sin límite", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Slots.Limit = AgentCapacityAmount{}
			value.Resources.Slots.Remaining = AgentCapacityAmount{Present: true, Value: 2}
		}, reason: AgentCapacityAdmissionAvailable, admit: 2},
		{name: "límite sin restante", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Slots.Limit = AgentCapacityAmount{Present: true, Value: 5}
			value.Resources.Slots.Remaining = AgentCapacityAmount{}
		}, reason: AgentCapacityAdmissionUnknown},
		{name: "slots no aplicables", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Slots = notApplicableCapacityDimension()
		}, reason: AgentCapacityAdmissionUnknown},
		{name: "aplicabilidad desconocida", mutate: func(value *AgentCapacityObservation) {
			value.Resources.Messages.Applicability = AgentCapacityApplicabilityUnknown
		}, reason: AgentCapacityAdmissionUnknown},
		{name: "calidad desconocida", mutate: func(value *AgentCapacityObservation) {
			value.Quality = governance.UsageQualityUnknown
		}, reason: AgentCapacityAdmissionUnknown},
		{name: "stale inclusivo", now: base.Add(5 * time.Minute), reason: AgentCapacityAdmissionStale},
		{name: "fuente no disponible", mutate: func(value *AgentCapacityObservation) {
			value.Status = AgentCapacityUnavailable
		}, reason: AgentCapacityAdmissionUnavailable},
		{name: "error de fuente", sourceErr: errors.New("credential=must-not-leak"),
			reason: AgentCapacityAdmissionUnavailable},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			observation := validAgentCapacityObservation(t)
			if test.mutate != nil {
				test.mutate(&observation)
			}
			now := test.now
			if now.IsZero() {
				now = base.Add(time.Minute)
			}
			got, err := DecideAgentCapacityAdmission(
				agentCapacityClock{now: now}, observation, test.sourceErr,
			)
			if err != nil || got.Reason != test.reason || got.NewAdmissions != test.admit ||
				!got.ControlAllowed {
				t.Fatalf("decisión inesperada: got=%+v err=%v", got, err)
			}
			if strings.Contains(fmt.Sprint(got), "must-not-leak") {
				t.Fatal("el error de fuente filtró un secreto")
			}
		})
	}
}

func TestAgentCapacityValidationRejectsIncoherentObservations(t *testing.T) {
	base := agentCapacityBaseTime()
	cases := []struct {
		name   string
		mutate func(*AgentCapacityObservation)
	}{
		{"source ref", func(value *AgentCapacityObservation) { value.SourceRef = " source:one" }},
		{"pool ref", func(value *AgentCapacityObservation) { value.PoolRef = "" }},
		{"window ref", func(value *AgentCapacityObservation) { value.WindowRef = "window:\x00" }},
		{"status", func(value *AgentCapacityObservation) { value.Status = "maybe" }},
		{"quality", func(value *AgentCapacityObservation) { value.Quality = "trusted" }},
		{"observed at", func(value *AgentCapacityObservation) { value.ObservedAt = time.Time{} }},
		{"expiry", func(value *AgentCapacityObservation) { value.ExpiresAt = value.ObservedAt }},
		{"reset at", func(value *AgentCapacityObservation) { value.ResetAt = value.ObservedAt }},
		{"retry at", func(value *AgentCapacityObservation) { value.RetryAt = value.ObservedAt }},
		{"applicability", func(value *AgentCapacityObservation) {
			value.Resources.Messages.Applicability = "conditional"
		}},
		{"unknown with limit", capacityPresence(AgentCapacityApplicabilityUnknown, true, false)},
		{"unknown with remaining", capacityPresence(AgentCapacityApplicabilityUnknown, false, true)},
		{"not applicable with limit", capacityPresence(AgentCapacityApplicabilityNotApplicable, true, false)},
		{"not applicable with remaining", capacityPresence(AgentCapacityApplicabilityNotApplicable, false, true)},
		{"negative limit", func(value *AgentCapacityObservation) {
			value.Resources.Messages = capacityDimension(-1, 0)
		}},
		{"negative remaining", func(value *AgentCapacityObservation) {
			value.Resources.Messages = capacityDimension(1, -1)
		}},
		{"remaining above limit", func(value *AgentCapacityObservation) {
			value.Resources.Tokens = capacityDimension(1, 2)
		}},
		{"absent nonzero", func(value *AgentCapacityObservation) {
			value.Resources.Seconds.Limit.Value = 1
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			observation := validAgentCapacityObservation(t)
			test.mutate(&observation)
			if err := ValidateAgentCapacityObservation(observation); !errors.Is(err, ErrAgentCapacityInvalid) {
				t.Fatalf("observación inválida aceptada: %v", err)
			}
		})
	}
	t.Run("clock regression", func(t *testing.T) {
		observation := validAgentCapacityObservation(t)
		got, err := DecideAgentCapacityAdmission(
			agentCapacityClock{now: base.Add(-time.Nanosecond)}, observation, nil,
		)
		if !errors.Is(err, ErrAgentCapacityInvalid) || !got.ControlAllowed {
			t.Fatalf("reloj regresivo aceptado o control bloqueado: got=%+v err=%v", got, err)
		}
	})
	t.Run("retry before independent reset", func(t *testing.T) {
		observation := validAgentCapacityObservation(t)
		observation.ResetAt = base.Add(4 * time.Minute)
		observation.RetryAt = base.Add(3 * time.Minute)
		if err := ValidateAgentCapacityObservation(observation); err != nil {
			t.Fatalf("hechos futuros independientes rechazados: %v", err)
		}
	})
}

func TestAgentCapacityNewWindowReopensOnlyFromObservation(t *testing.T) {
	base := agentCapacityBaseTime()
	exhausted := validAgentCapacityObservation(t)
	exhausted.ExpiresAt = base.Add(10 * time.Minute)
	exhausted.ResetAt = base.Add(2 * time.Minute)
	exhausted.RetryAt = exhausted.ResetAt
	exhausted.Resources.Slots = capacityDimension(4, 0)

	afterTimer, err := DecideAgentCapacityAdmission(
		agentCapacityClock{now: base.Add(3 * time.Minute)}, exhausted, nil,
	)
	if err != nil || afterTimer.Reason != AgentCapacityAdmissionExhausted {
		t.Fatalf("un timer local reabrió capacidad: got=%+v err=%v", afterTimer, err)
	}

	reopened := exhausted
	reopened.WindowRef = "window:two"
	reopened.ObservedAt = base.Add(3 * time.Minute)
	reopened.ExpiresAt = base.Add(8 * time.Minute)
	reopened.ResetAt, reopened.RetryAt = time.Time{}, time.Time{}
	reopened.Resources.Slots = capacityDimension(4, 4)
	got, err := DecideAgentCapacityAdmission(
		agentCapacityClock{now: base.Add(4 * time.Minute)}, reopened, nil,
	)
	if err != nil || got.Reason != AgentCapacityAdmissionAvailable ||
		got.NewAdmissions != 4 || got.WindowRef == afterTimer.WindowRef {
		t.Fatalf("una nueva ventana válida no reabrió capacidad: got=%+v err=%v", got, err)
	}
}

func validAgentCapacityObservation(t *testing.T) AgentCapacityObservation {
	t.Helper()
	artifact, err := goal.NewArtifactRef("artifact:capacity-observation")
	if err != nil {
		t.Fatal(err)
	}
	base := agentCapacityBaseTime()
	return AgentCapacityObservation{
		SourceRef: "source:provider", PoolRef: "pool:default", WindowRef: "window:one",
		Status: AgentCapacityAvailable, Quality: governance.UsageQualityMeasured,
		ObservedAt: base, ExpiresAt: base.Add(5 * time.Minute),
		Resources: AgentCapacityResources{
			Slots:    capacityDimension(5, 3),
			Seconds:  notApplicableCapacityDimension(),
			Messages: notApplicableCapacityDimension(),
			Tokens:   notApplicableCapacityDimension(),
			Credits:  notApplicableCapacityDimension(),
		},
		ArtifactRef: artifact,
	}
}

func capacityDimension(limit, remaining int64) AgentCapacityDimension {
	return AgentCapacityDimension{
		Applicability: AgentCapacityApplicabilityApplicable,
		Limit:         AgentCapacityAmount{Present: true, Value: limit},
		Remaining:     AgentCapacityAmount{Present: true, Value: remaining},
	}
}

func notApplicableCapacityDimension() AgentCapacityDimension {
	return AgentCapacityDimension{Applicability: AgentCapacityApplicabilityNotApplicable}
}

func capacityPresence(app AgentCapacityApplicability, limit, remaining bool) func(*AgentCapacityObservation) {
	return func(value *AgentCapacityObservation) {
		value.Resources.Messages = AgentCapacityDimension{
			Applicability: app, Limit: AgentCapacityAmount{Present: limit}, Remaining: AgentCapacityAmount{Present: remaining},
		}
	}
}

func agentCapacityBaseTime() time.Time {
	return time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
}
