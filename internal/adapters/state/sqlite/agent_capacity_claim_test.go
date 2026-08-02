package sqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestReclamoCapacidadReservaAtomicaYRepiteSinReseleccion(t *testing.T) {
	sistema := newSQLiteV15System(t, 70)
	sistema.capacidad = candidatoCapacidadSQLiteV15(t, sistema, 1, sistema.clock.Now())
	for indice := range 2 {
		sistema.submit(t, fmt.Sprintf("request:q4-ultimo-slot:%d", indice))
	}
	type resultado struct {
		reclamo application.ActionClaim
		hallado bool
		err     error
	}
	inicio, resultados := make(chan struct{}), make(chan resultado, 2)
	var grupo sync.WaitGroup
	for indice := range 2 {
		grupo.Add(1)
		go func(indice int) {
			defer grupo.Done()
			<-inicio
			reclamo, hallado, err := sistema.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef: "worker:q4", Token: fmt.Sprintf("claim:q4:%d", indice), LeaseDuration: time.Minute,
				Capabilities: sqliteTestCapabilities(), BudgetPolicy: sistema.policy, CapacityCandidates: sistema.capacidad,
			})
			resultados <- resultado{reclamo, hallado, err}
		}(indice)
	}
	close(inicio)
	grupo.Wait()
	close(resultados)
	var ganador application.ActionClaim
	for resultado := range resultados {
		if resultado.err != nil {
			t.Fatal(resultado.err)
		}
		if resultado.hallado {
			if ganador.Action.Ref != "" {
				t.Fatal("dos reclamos ganaron la última plaza física")
			}
			ganador = resultado.reclamo
		}
	}
	if ganador.ReferenciaColocacion != sistema.capacidad[0].PlacementRef || ganador.CapacityReservation.Ref == "" {
		t.Fatalf("reclamo sin ligadura física: %+v", ganador)
	}
	assertHechosCapacidadQ4(t, sistema, 1, 1, 1)
	sistema.clock.Advance(2 * time.Minute)
	repetido, hallado, err := sistema.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:q4-replay", Token: "claim:q4-replay", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: sistema.policy, CapacityCandidates: sistema.capacidad,
	})
	if err != nil || !hallado || repetido.Action.Ref != ganador.Action.Ref ||
		repetido.CapacityReservation.Ref != ganador.CapacityReservation.Ref ||
		repetido.CapacityReservation.Fence != ganador.CapacityReservation.Fence || repetido.Fence <= ganador.Fence ||
		repetido.ReferenciaColocacion != ganador.ReferenciaColocacion {
		t.Fatalf("replay reseleccionó o duplicó: antes=%+v después=%+v hallado=%v err=%v", ganador, repetido, hallado, err)
	}
	assertHechosCapacidadQ4(t, sistema, 1, 1, 1)
	prepareSQLiteV15Launch(t, sistema, repetido)
	intento := sqliteV15Attempt(repetido, sistema.clock.Now())
	_, creado, err := sistema.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: repetido, Attempt: intento, OperationAt: sistema.clock.Now(),
	})
	if err != nil || !creado {
		t.Fatalf("intento con fence posterior creado=%v err=%v", creado, err)
	}
	acceptAgentCapacityLaunch(t, sistema, repetido, intento)
	assertEstadoReservaQ4(t, sistema, repetido.CapacityReservation.Ref, "consumed", 2)
}

func TestReclamoCapacidadReutilizaObservacionYRechazaRetroceso(t *testing.T) {
	sistema := newSQLiteV15System(t, 70)
	sistema.capacidad = candidatoCapacidadSQLiteV15(t, sistema, 2, sistema.clock.Now())
	for indice := range 2 {
		sistema.submit(t, fmt.Sprintf("request:q4-dos-slots:%d", indice))
		reclamo, hallado, err := sistema.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
			WorkerRef: "worker:q4-dos", Token: fmt.Sprintf("claim:q4-dos:%d", indice), LeaseDuration: time.Minute,
			Capabilities: sqliteTestCapabilities(), BudgetPolicy: sistema.policy, CapacityCandidates: sistema.capacidad,
		})
		if err != nil || !hallado || reclamo.CapacityReservation.Ref == "" {
			t.Fatalf("reclamo %d hallado=%v err=%v", indice, hallado, err)
		}
	}
	assertHechosCapacidadQ4(t, sistema, 1, 2, 2)
	sistema.submit(t, "request:q4-retroceso")
	anterior := candidatoCapacidadSQLiteV15(t, sistema, 3, sistema.clock.Now().Add(-time.Second))
	_, hallado, err := sistema.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:q4-retroceso", Token: "claim:q4-retroceso", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: sistema.policy, CapacityCandidates: anterior,
	})
	if err != nil || hallado {
		t.Fatalf("observación anterior admitida: hallado=%v err=%v", hallado, err)
	}
	assertHechosCapacidadQ4(t, sistema, 1, 2, 2)
}

func TestReclamoCapacidadRechazaAlteracionDurable(t *testing.T) {
	sistema := newSQLiteV15System(t, 2)
	sistema.submit(t, "request:q4-cas")
	reclamo := claimSQLiteV15(t, sistema, "claim:q4-cas")
	prepareSQLiteV15Launch(t, sistema, reclamo)
	alterado := reclamo
	alterado.ReferenciaColocacion, _ = ports.NewAgentPlacementRef("placement:q4-alterada")
	_, _, err := sistema.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: alterado, Attempt: sqliteV15Attempt(reclamo, sistema.clock.Now()), OperationAt: sistema.clock.Now(),
	})
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("colocación alterada aceptada: %v", err)
	}
	alterado = reclamo
	alterado.CapacityReservation.Allocation.Slots++
	_, _, err = sistema.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: alterado, Attempt: sqliteV15Attempt(reclamo, sistema.clock.Now()), OperationAt: sistema.clock.Now(),
	})
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("reserva alterada aceptada: %v", err)
	}
}

func TestReclamoCapacidadCuarentenaOReliberaAtomicamente(t *testing.T) {
	t.Run("efecto desconocido retiene la plaza", func(t *testing.T) {
		sistema := newSQLiteV15System(t, 2)
		sistema.capacidad = candidatoCapacidadSQLiteV15(t, sistema, 1, sistema.clock.Now())
		sistema.submit(t, "request:q4-cuarentena")
		reclamo := claimSQLiteV15(t, sistema, "claim:q4-cuarentena")
		prepareSQLiteV15Launch(t, sistema, reclamo)
		ahora := sistema.clock.Now()
		err := sistema.repository.QuarantineAction(context.Background(), application.ActionQuarantinedState{
			Claim: reclamo, ErrorCode: "application.effect_unknown_applied", OperationAt: ahora,
			Event: application.EventRecord{Ref: "event:q4-cuarentena", Kind: "action.quarantined",
				GoalRef: reclamo.Action.GoalRef, WorkItemRef: reclamo.Action.WorkItemRef,
				ExecutionRef: reclamo.Action.ExecutionRef, OccurredAt: ahora},
		})
		sqliteTestNoError(t, err)
		assertEstadoReservaQ4(t, sistema, reclamo.CapacityReservation.Ref, "quarantined", 2)
		sistema.submit(t, "request:q4-bloqueada")
		_, hallado, err := sistema.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
			WorkerRef: "worker:q4-bloqueada", Token: "claim:q4-bloqueada", LeaseDuration: time.Minute,
			Capabilities: sqliteTestCapabilities(), BudgetPolicy: sistema.policy, CapacityCandidates: sistema.capacidad,
		})
		if err != nil || hallado {
			t.Fatalf("la cuarentena liberó la plaza: hallado=%v err=%v", hallado, err)
		}
	})

	t.Run("efecto no aplicado libera la plaza", func(t *testing.T) {
		sistema := newSQLiteV15System(t, 2)
		sistema.capacidad = candidatoCapacidadSQLiteV15(t, sistema, 1, sistema.clock.Now())
		sistema.submit(t, "request:q4-liberada")
		reclamo := claimSQLiteV15(t, sistema, "claim:q4-liberada")
		prepareSQLiteV15Launch(t, sistema, reclamo)
		cero := governance.ResourceVector{Currency: reclamo.BudgetReservation.Resources.Currency}
		liquidacion, err := governance.Reconcile(reclamo.BudgetReservation, governance.ResourceUsage{
			Resources: cero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
		})
		sqliteTestNoError(t, err)
		liquidacion.SettledAt = sistema.clock.Now()
		err = sistema.repository.QuarantineAction(context.Background(), application.ActionQuarantinedState{
			Claim: reclamo, ErrorCode: "test.effect_definitely_not_applied", OperationAt: sistema.clock.Now(),
			BudgetSettlement: &liquidacion, ClearEffectBinding: true,
			Event: application.EventRecord{Ref: "event:q4-liberada", Kind: "action.quarantined",
				GoalRef: reclamo.Action.GoalRef, WorkItemRef: reclamo.Action.WorkItemRef,
				ExecutionRef: reclamo.Action.ExecutionRef, OccurredAt: sistema.clock.Now()},
		})
		sqliteTestNoError(t, err)
		assertEstadoReservaQ4(t, sistema, reclamo.CapacityReservation.Ref, "released", 2)
		sistema.submit(t, "request:q4-reutilizada")
		siguiente := claimSQLiteV15(t, sistema, "claim:q4-reutilizada")
		if siguiente.CapacityReservation.Ref == reclamo.CapacityReservation.Ref {
			t.Fatal("el siguiente lanzamiento reutilizó la reserva liquidada")
		}
	})
}

func candidatoCapacidadSQLiteV15(t *testing.T, sistema *sqliteV15System, plazas int64, observada time.Time) []application.AgentCapacityPlacementCandidate {
	t.Helper()
	candidato := sistema.capacidad[0]
	candidato.Physical.Observation.ObservedAt = observada
	candidato.Physical.Observation.ExpiresAt = sistema.clock.Now().Add(24 * time.Hour)
	candidato.Physical.Observation.Resources.Slots.Limit.Value = plazas
	candidato.Physical.Observation.Resources.Slots.Remaining.Value = plazas
	entrega, err := application.NuevaEntregaObservacionCapacidad(candidato.Physical.Observation)
	sqliteTestNoError(t, err)
	candidato.Physical = entrega
	return []application.AgentCapacityPlacementCandidate{candidato}
}

func assertHechosCapacidadQ4(t *testing.T, sistema *sqliteV15System, observaciones, reservas, bindings int) {
	t.Helper()
	var gotObservaciones, gotReservas, gotBindings int
	err := sistema.repository.db.QueryRow(`SELECT (SELECT COUNT(*) FROM agent_capacity_observations),(SELECT COUNT(*) FROM agent_capacity_reservations),(SELECT COUNT(*) FROM agent_placement_bindings)`).Scan(&gotObservaciones, &gotReservas, &gotBindings)
	if err != nil || gotObservaciones != observaciones || gotReservas != reservas || gotBindings != bindings {
		t.Fatalf("hechos físicos=%d/%d/%d esperados=%d/%d/%d err=%v", gotObservaciones, gotReservas, gotBindings, observaciones, reservas, bindings, err)
	}
}

func assertEstadoReservaQ4(t *testing.T, sistema *sqliteV15System, referencia, estado string, revision int) {
	t.Helper()
	var obtenido string
	var obtenida int
	err := sistema.repository.db.QueryRow(`SELECT state,revision FROM agent_capacity_reservations WHERE ref=?`, referencia).Scan(&obtenido, &obtenida)
	if err != nil || obtenido != estado || obtenida != revision {
		t.Fatalf("reserva %s estado=%s revisión=%d err=%v", referencia, obtenido, obtenida, err)
	}
}
