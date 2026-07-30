package application

import (
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentQuotaGateIsSeparateAndFailsClosed(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	base := placementQuotaRecord(now)
	exhausted, unknown, stale, unknownQuality := base, base, base, base
	exhausted.Status, unknown.Status, stale.ExpiresAt = AgentQuotaExhausted, AgentQuotaUnknown, now
	unknownQuality.Quality = "unknown"
	tests := []struct {
		record *AgentQuotaObservationRecord
		reason AgentCapacityAdmissionReason
	}{
		{nil, AgentCapacityAdmissionUnknown},
		{&base, AgentCapacityAdmissionAvailable},
		{&exhausted, AgentCapacityAdmissionExhausted},
		{&unknown, AgentCapacityAdmissionUnknown},
		{&unknownQuality, AgentCapacityAdmissionUnknown},
		{&stale, AgentCapacityAdmissionStale},
	}
	for _, test := range tests {
		reason, err := DecideAgentQuotaGate(now, test.record)
		if err != nil || reason != test.reason {
			t.Fatalf("DecideAgentQuotaGate() = %s, %v", reason, err)
		}
	}
	for _, invalidRef := range []string{" quota:one", "quota:\x00"} {
		malformed := base
		malformed.Ref = invalidRef
		if _, err := DecideAgentQuotaGate(now, &malformed); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("cuota no canónica aceptada: %v", err)
		}
	}
	for _, mutate := range []func(*AgentQuotaObservationRecord){
		func(value *AgentQuotaObservationRecord) { value.WindowRef = " window:one" },
		func(value *AgentQuotaObservationRecord) { value.Revision = 0 },
		func(value *AgentQuotaObservationRecord) { value.Status = "unavailable" },
		func(value *AgentQuotaObservationRecord) { value.Status = "stale" },
		func(value *AgentQuotaObservationRecord) { value.Quality = "invented" },
		func(value *AgentQuotaObservationRecord) { value.ObservedAt = time.Time{} },
		func(value *AgentQuotaObservationRecord) {
			value.ObservedAt, value.ExpiresAt = now.Add(time.Minute), now.Add(2*time.Minute)
		},
		func(value *AgentQuotaObservationRecord) { value.ExpiresAt = value.ObservedAt },
		func(value *AgentQuotaObservationRecord) { value.ResetAt = value.ObservedAt },
		func(value *AgentQuotaObservationRecord) { value.RetryAt = value.ObservedAt },
	} {
		invalid := base
		mutate(&invalid)
		if _, err := DecideAgentQuotaGate(now, &invalid); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("registro de cuota inválido aceptado: %+v", invalid)
		}
	}
}

func placementQuotaRecord(now time.Time) AgentQuotaObservationRecord {
	return AgentQuotaObservationRecord{Ref: "quota:one", WindowRef: "window:one", Revision: 7,
		Status: AgentQuotaAvailable, Quality: "exact",
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute)}
}

func TestAgentPlacementBindingUsesExactReservationAndQuota(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	project, _ := goal.NewProjectRef("project:one")
	goalRef, _ := goal.NewGoalRef("goal:one")
	workRef, _ := goal.NewWorkItemRef("work:one")
	executionRef, _ := goal.NewExecutionRef("execution:one")
	reservation := AgentCapacityReservation{Ref: "reservation:one", ObservationRef: "physical:one", ObservationRevision: 3,
		ActionRef: "action:one", EffectIntentRef: "intent:one", IdempotencyKey: "idempotency:one",
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: workRef, ExecutionRef: executionRef,
		PlanGeneration: 1, WorkItemGeneration: 1, Revision: 1, Fence: 1,
		Allocation: AgentCapacityAllocation{Slots: 1}, State: AgentCapacityReserved, ReservedAt: now, UpdatedAt: now}
	placementRef, _ := ports.NewAgentPlacementRef("placement:one")
	quota := placementQuotaRecord(now)
	candidate := AgentCapacityPlacementCandidate{PlacementRef: placementRef,
		Physical: AgentPlacementObservationPresentation{"physical:one", 3},
		Quota:    AgentPlacementObservationPresentation{"quota:one", 7}}
	binding, err := NewAgentPlacementBinding(candidate, reservation, quota)
	if err != nil {
		t.Fatal(err)
	}
	candidate.Physical.ObservationRevision, reservation.Ref, quota.Ref = 2, "reservation:changed", "quota:changed"
	quotaRef, quotaRevision := binding.QuotaObservation()
	if binding.PlacementRef() != placementRef || binding.ReservationRef() != "reservation:one" ||
		quotaRef != "quota:one" || quotaRevision != 7 {
		t.Fatalf("binding cambió tras mutar entradas: %+v", binding)
	}
	for _, mutate := range []func(*AgentCapacityPlacementCandidate){
		func(value *AgentCapacityPlacementCandidate) { value.PlacementRef = ports.AgentPlacementRef{} },
		func(value *AgentCapacityPlacementCandidate) { value.Physical.ObservationRevision = 2 },
		func(value *AgentCapacityPlacementCandidate) { value.Quota.ObservationRevision = 6 },
		func(value *AgentCapacityPlacementCandidate) { value.Physical.ObservationRef = " physical:one" },
		func(value *AgentCapacityPlacementCandidate) {
			value.Quota.ObservationRef = value.Physical.ObservationRef
		},
	} {
		invalid := AgentCapacityPlacementCandidate{PlacementRef: placementRef,
			Physical: AgentPlacementObservationPresentation{"physical:one", 3},
			Quota:    AgentPlacementObservationPresentation{"quota:one", 7}}
		mutate(&invalid)
		if _, err := NewAgentPlacementBinding(invalid, reservation, placementQuotaRecord(now)); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("candidato inválido aceptado: %+v", invalid)
		}
	}
	candidate.Physical.ObservationRevision, candidate.Quota.ObservationRevision = 3, 7
	invalidReservation, invalidQuota := reservation, placementQuotaRecord(now)
	invalidReservation.Fence, invalidQuota.WindowRef = 0, ""
	if _, err := NewAgentPlacementBinding(candidate, invalidReservation, placementQuotaRecord(now)); !errors.Is(err, ErrAgentCapacityInvalid) {
		t.Fatalf("reserva inválida aceptada: %v", err)
	}
	if _, err := NewAgentPlacementBinding(candidate, reservation, invalidQuota); !errors.Is(err, ErrAgentCapacityInvalid) {
		t.Fatalf("cuota inválida aceptada: %v", err)
	}
}
