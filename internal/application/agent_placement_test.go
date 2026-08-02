package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type estadoCuotaPrueba struct {
	StateRepository
	vigente   AgentQuotaObservationRecord
	guardados int
}

func (estado *estadoCuotaPrueba) CurrentAgentQuotaObservation(context.Context, ports.AgentPlacementRef) (AgentQuotaObservationRecord, bool, error) {
	return estado.vigente, estado.guardados > 0, nil
}

func (estado *estadoCuotaPrueba) AppendAgentQuotaObservation(_ context.Context, registro AgentQuotaObservationRecord) (AgentQuotaObservationRecord, bool, error) {
	estado.vigente, estado.guardados = registro, estado.guardados+1
	return registro, true, nil
}

func TestRegistrarObservacionCuotaPersisteEvidenciaYRepiteExactamente(t *testing.T) {
	estado, artefactos := &estadoCuotaPrueba{}, newMemoryArtifactStore()
	observacion := placementQuotaSubmission(time.Unix(100, 0).UTC()).AgentQuotaObservation
	for range 2 {
		if err := RegistrarObservacionCuota(context.Background(), estado, artefactos, observacion, []byte(`{"usedPercent":25}`)); err != nil {
			t.Fatal(err)
		}
	}
	if estado.guardados != 1 || estado.vigente.Revision != 1 || estado.vigente.EvidenceRef.String() == "" {
		t.Fatalf("estado de cuota no idempotente: %+v guardados=%d", estado.vigente, estado.guardados)
	}
}

func TestOrdenarCandidatosColocacionDeduplicaYRechazaConflictos(t *testing.T) {
	candidato := func(ref string, revision uint64) AgentCapacityPlacementCandidate {
		colocacion, _ := ports.NewAgentPlacementRef(ref)
		return AgentCapacityPlacementCandidate{PlacementRef: colocacion,
			Physical: AgentPlacementObservationPresentation{"fisica:" + ref, revision},
			Quota:    AgentPlacementObservationPresentation{"cuota:" + ref, revision}}
	}
	primero, segundo := candidato("placement:b", 1), candidato("placement:a", 2)
	ordenados, err := OrdenarCandidatosColocacion([]AgentCapacityPlacementCandidate{primero, segundo, primero})
	if err != nil || len(ordenados) != 2 || ordenados[0] != segundo || ordenados[1] != primero {
		t.Fatalf("candidatos=%+v error=%v", ordenados, err)
	}
	conflicto := primero
	conflicto.Quota.ObservationRevision++
	if _, err := OrdenarCandidatosColocacion([]AgentCapacityPlacementCandidate{primero, conflicto}); !errors.Is(err, ErrAgentCapacityInvalid) {
		t.Fatalf("duplicado conflictivo aceptado: %v", err)
	}
}

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
		func(value *AgentQuotaObservationRecord) { value.ExpectedRevision = value.Revision },
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
	record, err := MaterializeAgentQuotaObservation(placementQuotaSubmission(now), 6)
	if err != nil {
		panic(err)
	}
	return record
}

func placementQuotaSubmission(now time.Time) AgentQuotaObservationSubmission {
	placementRef, _ := ports.NewAgentPlacementRef("placement:one")
	return AgentQuotaObservationSubmission{Ref: "quota:one", IdempotencyKey: "quota-observation:one",
		AgentQuotaObservation: AgentQuotaObservation{PlacementRef: placementRef, WindowRef: "window:one",
			Status: AgentQuotaAvailable, Quality: "exact",
			ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute)}}
}

func TestAgentQuotaObservationMaterializationIsCASDeterministic(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	first, err := MaterializeAgentQuotaObservation(placementQuotaSubmission(now), 0)
	if err != nil || first.ExpectedRevision != 0 || first.Revision != 1 {
		t.Fatalf("primera revisión = %d/%d, %v", first.ExpectedRevision, first.Revision, err)
	}
	next := placementQuotaSubmission(now)
	next.Ref, next.IdempotencyKey = "quota:two", "quota-observation:two"
	second, err := MaterializeAgentQuotaObservation(next, first.Revision)
	if err != nil || second.ExpectedRevision != 1 || second.Revision != 2 {
		t.Fatalf("segunda revisión = %d/%d, %v", second.ExpectedRevision, second.Revision, err)
	}
	badRef, badKey, badPlacement := placementQuotaSubmission(now), placementQuotaSubmission(now), placementQuotaSubmission(now)
	badRef.Ref, badKey.IdempotencyKey, badPlacement.PlacementRef = " quota:one", " idempotency", ports.AgentPlacementRef{}
	for _, invalid := range []AgentQuotaObservationSubmission{badRef, badKey, badPlacement} {
		if _, err := MaterializeAgentQuotaObservation(invalid, 0); !errors.Is(err, ErrAgentCapacityInvalid) {
			t.Fatalf("submission inválida aceptada: %+v", invalid)
		}
	}
	if _, err := MaterializeAgentQuotaObservation(placementQuotaSubmission(now), ^uint64(0)); !errors.Is(err, ErrAgentCapacityInvalid) {
		t.Fatalf("revisión máxima aceptada: %v", err)
	}
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
	if binding.placementRef != placementRef || binding.reservationRef != "reservation:one" ||
		binding.quotaObservationRef != "quota:one" || binding.quotaObservationRevision != 7 {
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
	crossPlacement, _ := ports.NewAgentPlacementRef("placement:other")
	invalidQuota = placementQuotaRecord(now)
	invalidQuota.PlacementRef = crossPlacement
	if _, err := NewAgentPlacementBinding(candidate, reservation, invalidQuota); !errors.Is(err, ErrAgentCapacityInvalid) {
		t.Fatalf("binding aceptó cuota de otro placement: %v", err)
	}
}
