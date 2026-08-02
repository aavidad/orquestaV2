package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type seleccionCapacidadReclamo struct {
	candidato        application.AgentCapacityPlacementCandidate
	cuota            application.AgentQuotaObservationRecord
	revisionEsperada uint64
	insertar         bool
	activa           bool
	repetida         bool
	reserva          application.AgentCapacityReservation
	colocacion       ports.AgentPlacementRef
}

type relojCapacidadReclamo struct{ ahora time.Time }

func (reloj relojCapacidadReclamo) Now() time.Time { return reloj.ahora }

func seleccionarCapacidadReclamo(ctx context.Context, tx *sql.Tx, accion claimCandidate, candidatos []application.AgentCapacityPlacementCandidate, ahora time.Time) (seleccionCapacidadReclamo, bool, error) {
	if accion.action.Kind != application.ActionLaunchAgent {
		return seleccionCapacidadReclamo{}, true, nil
	}
	// Los lanzamientos históricos sin ledger de efectos son recuperables, pero no son lanzamientos nuevos V38.
	if accion.action.EffectIntentRef == "" {
		return seleccionCapacidadReclamo{}, true, nil
	}
	reserva, colocacion, encontrada, err := leerReservaCapacidadAccion(ctx, tx, accion.action.Ref)
	if err != nil || encontrada {
		return seleccionCapacidadReclamo{activa: encontrada, repetida: encontrada, reserva: reserva, colocacion: colocacion}, encontrada && reserva.State == application.AgentCapacityReserved, err
	}
	demanda, err := application.AgentCapacityDemandFromBudget(accion.action.EffectIntent.Demand)
	if err != nil {
		return seleccionCapacidadReclamo{}, false, invalid(err)
	}
	for _, candidato := range candidatos {
		cuota, vigente, err := readAgentQuotaObservation(ctx, tx, `SELECT ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,evidence_ref,idempotency_key,expected_revision,revision FROM agent_quota_observations WHERE ref=? AND revision=? AND placement_ref=? AND revision=(SELECT MAX(revision) FROM agent_quota_observations WHERE placement_ref=?)`, candidato.Quota.ObservationRef, candidato.Quota.ObservationRevision, candidato.PlacementRef.String(), candidato.PlacementRef.String())
		if err != nil {
			return seleccionCapacidadReclamo{}, false, err
		}
		razonCuota, falloCuota := application.DecideAgentQuotaGate(ahora, &cuota)
		if !vigente || falloCuota != nil || razonCuota != application.AgentCapacityAdmissionAvailable {
			continue
		}
		retenida, err := leerCapacidadRetenida(ctx, tx, candidato.Physical.Observation.SourceRef, candidato.Physical.Observation.PoolRef)
		if err != nil {
			return seleccionCapacidadReclamo{}, false, err
		}
		admision, falloDecision := application.DecideAgentCapacityAdmission(relojCapacidadReclamo{ahora}, candidato.Physical.Observation, nil, demanda, retenida)
		if falloDecision != nil || admision.Reason != application.AgentCapacityAdmissionAvailable {
			continue
		}
		esperada, insertar, vigente, err := versionObservacionCapacidad(ctx, tx, candidato.Physical)
		if err != nil {
			return seleccionCapacidadReclamo{}, false, err
		}
		if vigente {
			return seleccionCapacidadReclamo{candidato: candidato, cuota: cuota, revisionEsperada: esperada, insertar: insertar, activa: true}, true, nil
		}
	}
	return seleccionCapacidadReclamo{}, false, nil
}

func leerCapacidadRetenida(ctx context.Context, tx *sql.Tx, fuente application.AgentCapacitySourceRef, pool application.AgentCapacityPoolRef) (application.AgentCapacityDemand, error) {
	valores := [5]int64{}
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(reservation.slots),0),COALESCE(SUM(reservation.seconds),0),COALESCE(SUM(reservation.messages),0),COALESCE(SUM(reservation.tokens),0),COALESCE(SUM(reservation.credits),0) FROM agent_capacity_reservations reservation JOIN agent_capacity_observations observation ON observation.ref=reservation.observation_ref WHERE observation.source_ref=? AND observation.pool_ref=? AND reservation.state IN ('reserved','consumed','quarantined')`, fuente, pool).Scan(&valores[0], &valores[1], &valores[2], &valores[3], &valores[4])
	presente := func(valor int64) application.AgentCapacityAmount {
		return application.AgentCapacityAmount{Present: true, Value: valor}
	}
	return application.AgentCapacityDemand{Slots: presente(valores[0]), Seconds: presente(valores[1]), Messages: presente(valores[2]), Tokens: presente(valores[3]), Credits: presente(valores[4])}, mapDatabaseError(err)
}

func versionObservacionCapacidad(ctx context.Context, tx *sql.Tx, entrega application.AgentCapacityObservationSubmission) (uint64, bool, bool, error) {
	var ultima, exacta int64
	var observada sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0),MAX(observed_at),COALESCE(MAX(CASE WHEN ref=? AND idempotency_key=? THEN revision ELSE 0 END),0) FROM agent_capacity_observations WHERE source_ref=? AND pool_ref=?`, entrega.Ref, entrega.IdempotencyKey, entrega.Observation.SourceRef, entrega.Observation.PoolRef).Scan(&ultima, &observada, &exacta)
	if err != nil {
		return 0, false, false, mapDatabaseError(err)
	}
	if ultima < 0 || uint64(ultima) > maxSQLiteInteger || exacta != 0 && exacta != ultima {
		return 0, false, false, nil
	}
	if exacta > 0 {
		registro, err := application.MaterializeAgentCapacityObservation(entrega, uint64(exacta-1))
		if err != nil {
			return 0, false, false, invalid(err)
		}
		coincide, err := coincideObservacionCapacidad(ctx, tx, registro)
		return uint64(exacta - 1), false, coincide, err
	}
	if observada.Valid && entrega.Observation.ObservedAt.UnixNano() <= observada.Int64 {
		return 0, false, false, nil
	}
	return uint64(ultima), true, true, nil
}

func coincideObservacionCapacidad(ctx context.Context, tx *sql.Tx, registro application.AgentCapacityObservationRecord) (bool, error) {
	o, recursos := registro.Observation, registro.Observation.Resources
	valor := func(cantidad application.AgentCapacityAmount) any {
		if cantidad.Present {
			return cantidad.Value
		}
		return nil
	}
	var coincidencias int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_capacity_observations WHERE ref=? AND source_ref=? AND pool_ref=? AND window_ref=? AND revision=? AND expected_revision=? AND status=? AND quality=? AND observed_at=? AND expires_at=? AND reset_at IS ? AND retry_at IS ? AND artifact_ref=? AND slots_applicability=? AND slots_limit IS ? AND slots_remaining IS ? AND seconds_applicability=? AND seconds_limit IS ? AND seconds_remaining IS ? AND messages_applicability=? AND messages_limit IS ? AND messages_remaining IS ? AND tokens_applicability=? AND tokens_limit IS ? AND tokens_remaining IS ? AND credits_applicability=? AND credits_limit IS ? AND credits_remaining IS ? AND idempotency_key=?`,
		registro.Ref, o.SourceRef, o.PoolRef, o.WindowRef, registro.Revision, registro.ExpectedRevision, o.Status, o.Quality, requiredTime(o.ObservedAt), requiredTime(o.ExpiresAt), storedTime(o.ResetAt), storedTime(o.RetryAt), o.ArtifactRef.String(),
		recursos.Slots.Applicability, valor(recursos.Slots.Limit), valor(recursos.Slots.Remaining), recursos.Seconds.Applicability, valor(recursos.Seconds.Limit), valor(recursos.Seconds.Remaining), recursos.Messages.Applicability, valor(recursos.Messages.Limit), valor(recursos.Messages.Remaining), recursos.Tokens.Applicability, valor(recursos.Tokens.Limit), valor(recursos.Tokens.Remaining), recursos.Credits.Applicability, valor(recursos.Credits.Limit), valor(recursos.Credits.Remaining), registro.IdempotencyKey).Scan(&coincidencias)
	return coincidencias == 1, mapDatabaseError(err)
}

func reservarCapacidadReclamo(ctx context.Context, tx *sql.Tx, seleccion seleccionCapacidadReclamo, accion claimCandidate, cerca uint64, ahora time.Time) (application.AgentCapacityReservation, ports.AgentPlacementRef, error) {
	if seleccion.repetida {
		return seleccion.reserva, seleccion.colocacion, nil
	}
	if !seleccion.activa {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, nil
	}
	observacion, err := application.MaterializeAgentCapacityObservation(seleccion.candidato.Physical, seleccion.revisionEsperada)
	if err != nil {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, invalid(err)
	}
	if seleccion.insertar {
		if err := insertarObservacionCapacidadReclamo(ctx, tx, observacion); err != nil {
			return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, err
		}
	}
	asignacion, err := application.AgentCapacityDemandFromBudget(accion.action.EffectIntent.Demand)
	reserva := application.AgentCapacityReservation{Ref: deterministicRef("capacity-reservation", canonicalFingerprint(accion.action.Ref)), ObservationRef: observacion.Ref, ObservationRevision: observacion.Revision,
		ActionRef: accion.action.Ref, EffectIntentRef: accion.action.EffectIntent.Ref, IdempotencyKey: deterministicRef("capacity-reserve", canonicalFingerprint(accion.action.Ref)), ProjectRef: accion.projectRef,
		GoalRef: accion.action.GoalRef, WorkItemRef: accion.action.WorkItemRef, ExecutionRef: accion.action.ExecutionRef, PlanGeneration: accion.action.PlanGeneration,
		WorkItemGeneration: accion.action.WorkItemGeneration, Revision: 1, Fence: cerca, Allocation: application.AgentCapacityAllocation{Slots: asignacion.Slots.Value, Seconds: asignacion.Seconds.Value,
			Messages: asignacion.Messages.Value, Tokens: asignacion.Tokens.Value, Credits: asignacion.Credits.Value}, State: application.AgentCapacityReserved, ReservedAt: ahora, UpdatedAt: ahora}
	if err != nil || application.ValidateAgentCapacityReservation(reserva) != nil {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, invalid(errors.Join(err, application.ErrAgentCapacityInvalid))
	}
	if _, err = application.NewAgentPlacementBinding(seleccion.candidato, reserva, seleccion.cuota); err != nil {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, invalid(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_capacity_reservations(ref,observation_ref,observation_revision,effect_intent_ref,project_ref,goal_ref,work_item_ref,execution_ref,action_ref,plan_generation,work_item_generation,fence,state,revision,slots,seconds,messages,tokens,credits,idempotency_key,reserved_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?, 'reserved',1,?,?,?,?,?,?,?,?)`, reserva.Ref, reserva.ObservationRef, reserva.ObservationRevision, reserva.EffectIntentRef, reserva.ProjectRef.String(), reserva.GoalRef.String(), reserva.WorkItemRef.String(), reserva.ExecutionRef.String(), reserva.ActionRef, reserva.PlanGeneration, reserva.WorkItemGeneration, reserva.Fence, reserva.Allocation.Slots, reserva.Allocation.Seconds, reserva.Allocation.Messages, reserva.Allocation.Tokens, reserva.Allocation.Credits, reserva.IdempotencyKey, requiredTime(ahora), requiredTime(ahora))
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO agent_placement_bindings(reservation_ref,placement_ref,quota_observation_ref,quota_observation_revision) VALUES(?,?,?,?)`, reserva.Ref, seleccion.candidato.PlacementRef.String(), seleccion.cuota.Ref, seleccion.cuota.Revision)
	}
	if err != nil {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, mapDatabaseError(err)
	}
	return reserva, seleccion.candidato.PlacementRef, nil
}

func leerReservaCapacidadAccion(ctx context.Context, fuente queryer, accion string) (application.AgentCapacityReservation, ports.AgentPlacementRef, bool, error) {
	var reserva application.AgentCapacityReservation
	var proyecto, objetivo, item, ejecucion, colocacion string
	var revisionObservacion, plan, generacion, cerca, revision, reservada, actualizada int64
	var liquidada sql.NullInt64
	err := fuente.QueryRowContext(ctx, `SELECT r.ref,r.observation_ref,r.observation_revision,r.effect_intent_ref,r.project_ref,r.goal_ref,r.work_item_ref,r.execution_ref,r.action_ref,r.plan_generation,r.work_item_generation,r.fence,r.state,r.revision,r.last_transition_ref,r.last_cause_ref,r.slots,r.seconds,r.messages,r.tokens,r.credits,r.idempotency_key,r.reserved_at,r.updated_at,r.settled_at,b.placement_ref FROM agent_capacity_reservations r JOIN agent_placement_bindings b ON b.reservation_ref=r.ref WHERE r.action_ref=?`, accion).Scan(&reserva.Ref, &reserva.ObservationRef, &revisionObservacion, &reserva.EffectIntentRef, &proyecto, &objetivo, &item, &ejecucion, &reserva.ActionRef, &plan, &generacion, &cerca, &reserva.State, &revision, &reserva.LastTransitionRef, &reserva.LastCauseRef, &reserva.Allocation.Slots, &reserva.Allocation.Seconds, &reserva.Allocation.Messages, &reserva.Allocation.Tokens, &reserva.Allocation.Credits, &reserva.IdempotencyKey, &reservada, &actualizada, &liquidada, &colocacion)
	if errors.Is(err, sql.ErrNoRows) {
		return reserva, ports.AgentPlacementRef{}, false, nil
	}
	if err != nil {
		return reserva, ports.AgentPlacementRef{}, false, mapDatabaseError(err)
	}
	var errores []error
	reserva.ProjectRef, err = goal.NewProjectRef(proyecto)
	errores = append(errores, err)
	reserva.GoalRef, err = goal.NewGoalRef(objetivo)
	errores = append(errores, err)
	reserva.WorkItemRef, err = goal.NewWorkItemRef(item)
	errores = append(errores, err)
	reserva.ExecutionRef, err = goal.NewExecutionRef(ejecucion)
	errores = append(errores, err)
	referencia, err := ports.NewAgentPlacementRef(colocacion)
	errores = append(errores, err)
	if revisionObservacion <= 0 || plan <= 0 || generacion <= 0 || cerca <= 0 || revision <= 0 {
		errores = append(errores, application.ErrAgentCapacityInvalid)
	}
	reserva.ObservationRevision, reserva.PlanGeneration, reserva.WorkItemGeneration = uint64(revisionObservacion), goal.PlanGeneration(plan), goal.Revision(generacion)
	reserva.Fence, reserva.Revision = uint64(cerca), uint64(revision)
	reserva.ReservedAt, reserva.UpdatedAt, reserva.SettledAt = time.Unix(0, reservada).UTC(), time.Unix(0, actualizada).UTC(), restoredTime(liquidada)
	errores = append(errores, application.ValidateAgentCapacityReservation(reserva))
	if errors.Join(errores...) != nil {
		return application.AgentCapacityReservation{}, ports.AgentPlacementRef{}, false, invalid(errors.Join(errores...))
	}
	return reserva, referencia, true, nil
}

func coincideReservaCapacidadReclamo(persistida, reclamada application.AgentCapacityReservation) bool {
	return reclamada.Revision == 1 && reclamada.State == application.AgentCapacityReserved &&
		persistida.Ref == reclamada.Ref && persistida.ObservationRef == reclamada.ObservationRef &&
		persistida.ObservationRevision == reclamada.ObservationRevision && persistida.ActionRef == reclamada.ActionRef &&
		persistida.EffectIntentRef == reclamada.EffectIntentRef && persistida.IdempotencyKey == reclamada.IdempotencyKey &&
		persistida.ProjectRef == reclamada.ProjectRef && persistida.GoalRef == reclamada.GoalRef &&
		persistida.WorkItemRef == reclamada.WorkItemRef && persistida.ExecutionRef == reclamada.ExecutionRef &&
		persistida.PlanGeneration == reclamada.PlanGeneration && persistida.WorkItemGeneration == reclamada.WorkItemGeneration &&
		persistida.Fence == reclamada.Fence && persistida.Allocation == reclamada.Allocation &&
		persistida.ReservedAt.Equal(reclamada.ReservedAt)
}

func insertarObservacionCapacidadReclamo(ctx context.Context, tx *sql.Tx, registro application.AgentCapacityObservationRecord) error {
	o, recursos := registro.Observation, registro.Observation.Resources
	valor := func(cantidad application.AgentCapacityAmount) any {
		if !cantidad.Present {
			return nil
		}
		return cantidad.Value
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO agent_capacity_observations(ref,source_ref,pool_ref,window_ref,revision,expected_revision,status,quality,observed_at,expires_at,reset_at,retry_at,artifact_ref,slots_applicability,slots_limit,slots_remaining,seconds_applicability,seconds_limit,seconds_remaining,messages_applicability,messages_limit,messages_remaining,tokens_applicability,tokens_limit,tokens_remaining,credits_applicability,credits_limit,credits_remaining,idempotency_key) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, registro.Ref, o.SourceRef, o.PoolRef, o.WindowRef, registro.Revision, registro.ExpectedRevision, o.Status, o.Quality, requiredTime(o.ObservedAt), requiredTime(o.ExpiresAt), storedTime(o.ResetAt), storedTime(o.RetryAt), o.ArtifactRef.String(), recursos.Slots.Applicability, valor(recursos.Slots.Limit), valor(recursos.Slots.Remaining), recursos.Seconds.Applicability, valor(recursos.Seconds.Limit), valor(recursos.Seconds.Remaining), recursos.Messages.Applicability, valor(recursos.Messages.Limit), valor(recursos.Messages.Remaining), recursos.Tokens.Applicability, valor(recursos.Tokens.Limit), valor(recursos.Tokens.Remaining), recursos.Credits.Applicability, valor(recursos.Credits.Limit), valor(recursos.Credits.Remaining), registro.IdempotencyKey)
	return mapDatabaseError(err)
}
