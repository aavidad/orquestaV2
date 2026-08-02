package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
)

func consumirCapacidadLanzamiento(ctx context.Context, tx *sql.Tx, estado application.LaunchAcceptedState) error {
	if estado.Claim.Action.Kind != application.ActionLaunchAgent || estado.Claim.Action.EffectIntentRef == "" {
		return nil
	}
	return persistirTransicionCapacidad(ctx, tx, estado.Claim.CapacityReservation,
		application.AgentCapacityConsumed, application.AgentCapacityCauseEffectReceipt,
		estado.EffectReceipt.Ref, estado.EffectReceipt.AttemptRef, estado.EffectReceipt.Ref, estado.OperationAt)
}

func liquidarCapacidadLanzamiento(ctx context.Context, tx *sql.Tx, estado application.ActionQuarantinedState) error {
	if estado.Claim.Action.Kind != application.ActionLaunchAgent || estado.Claim.Action.EffectIntentRef == "" {
		return nil
	}
	resultado, causa := application.AgentCapacityQuarantined, application.AgentCapacityCauseUnknownApplied
	if estado.ClearEffectBinding {
		resultado, causa = application.AgentCapacityReleased, application.AgentCapacityCauseDefinitelyNotApplied
	}
	return persistirTransicionCapacidad(ctx, tx, estado.Claim.CapacityReservation,
		resultado, causa, estado.Claim.Action.Ref, "", "", estado.OperationAt)
}

func liberarCapacidadEjecucionTerminal(ctx context.Context, tx *sql.Tx, ejecucion application.ExecutionRecord) error {
	if !applicationTerminalExecution(ejecucion.State) {
		return nil
	}
	var accion sql.NullString
	var cantidad int
	err := tx.QueryRowContext(ctx, `SELECT MIN(action_ref),COUNT(*) FROM agent_capacity_reservations WHERE execution_ref=?`, ejecucion.Ref.String()).Scan(&accion, &cantidad)
	if err != nil {
		return mapDatabaseError(err)
	}
	if cantidad == 0 {
		return nil
	}
	if cantidad != 1 {
		return conflict(errors.New("sqlite.agent_capacity_execution_ambiguous"))
	}
	reserva, _, encontrada, err := leerReservaCapacidadAccion(ctx, tx, accion.String)
	if err != nil || !encontrada || reserva.ExecutionRef != ejecucion.Ref {
		if err != nil {
			return err
		}
		return conflict(errors.New("sqlite.agent_capacity_execution_invalid"))
	}
	if reserva.State == application.AgentCapacityReleased {
		return nil
	}
	causa := application.AgentCapacityCauseExecutionTerminal
	if reserva.State == application.AgentCapacityReserved {
		causa = application.AgentCapacityCauseDefinitelyNotApplied
	} else if reserva.State == application.AgentCapacityQuarantined {
		causa = application.AgentCapacityCauseReconciliation
	}
	return persistirTransicionCapacidad(ctx, tx, reserva, application.AgentCapacityReleased,
		causa, ejecucion.Ref.String(), "", "", ejecucion.FinishedAt)
}

func persistirTransicionCapacidad(ctx context.Context, tx *sql.Tx, reserva application.AgentCapacityReservation,
	resultado application.AgentCapacityReservationState, causa application.AgentCapacityTransitionCause,
	causaRef, intentoRef, comprobanteRef string, registrada time.Time,
) error {
	siguienteRevision := reserva.Revision + 1
	huella := canonicalFingerprint(reserva.Ref, fmt.Sprint(siguienteRevision), string(resultado), causaRef)
	transicion := application.AgentCapacityTransition{
		Ref: deterministicRef("capacity-transition", huella), ReservationRef: reserva.Ref,
		ProjectRef: reserva.ProjectRef, Fence: reserva.Fence, ExpectedRevision: reserva.Revision,
		Revision: siguienteRevision, Outcome: resultado, Cause: causa, CauseRef: causaRef,
		EffectAttemptRef: intentoRef, EffectReceiptRef: comprobanteRef,
		IdempotencyKey: deterministicRef("capacity-transition-idempotency", huella), RecordedAt: registrada,
	}
	if err := application.ValidateAgentCapacityTransition(reserva, transicion); err != nil {
		return invalid(err)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO agent_capacity_transitions(ref,reservation_ref,project_ref,fence,expected_revision,revision,outcome,cause_kind,cause_ref,effect_attempt_ref,effect_receipt_ref,idempotency_key,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		transicion.Ref, transicion.ReservationRef, transicion.ProjectRef.String(), transicion.Fence,
		transicion.ExpectedRevision, transicion.Revision, transicion.Outcome, transicion.Cause,
		transicion.CauseRef, nullableString(transicion.EffectAttemptRef), nullableString(transicion.EffectReceiptRef),
		transicion.IdempotencyKey, requiredTime(transicion.RecordedAt))
	if err != nil {
		return mapDatabaseError(err)
	}
	liquidada := any(nil)
	if resultado == application.AgentCapacityReleased {
		liquidada = requiredTime(registrada)
	}
	actualizada, err := tx.ExecContext(ctx, `UPDATE agent_capacity_reservations SET state=?,revision=?,last_transition_ref=?,last_cause_ref=?,updated_at=?,settled_at=? WHERE ref=? AND project_ref=? AND fence=? AND revision=?`,
		resultado, siguienteRevision, transicion.Ref, causaRef, requiredTime(registrada), liquidada,
		reserva.Ref, reserva.ProjectRef.String(), reserva.Fence, reserva.Revision)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(actualizada)
}
