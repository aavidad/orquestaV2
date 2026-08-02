package sqlite

import "context"
import "database/sql"
import "errors"
import "time"
import "orquesta/internal/application"
import "orquesta/internal/goal"
import "orquesta/internal/ports"

func validateRecoveryV38AgentPlacement(ctx context.Context, tx *sql.Tx) error {
	if err := validarObservacionesFisicasRecuperacion(ctx, tx); err != nil {
		return errors.Join(err, errors.New("sqlite.recovery_v38_agent_placement_invalid"))
	}
	refs, err := readSingleColumn(ctx, tx, `SELECT ref FROM agent_quota_observations ORDER BY placement_ref,revision`)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		var record application.AgentQuotaObservationRecord
		var placement, evidence string
		var observedAt, expiresAt int64
		var resetAt, retryAt sql.NullInt64
		err = tx.QueryRowContext(ctx, `SELECT ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,evidence_ref,idempotency_key,expected_revision,revision FROM agent_quota_observations WHERE ref=?`, ref).Scan(&record.Ref, &placement, &record.WindowRef, &record.Status, &record.Quality, &observedAt, &expiresAt, &resetAt, &retryAt, &evidence, &record.IdempotencyKey, &record.ExpectedRevision, &record.Revision)
		var placementErr, evidenceErr error
		record.PlacementRef, placementErr = ports.NewAgentPlacementRef(placement)
		if evidence != "" {
			record.EvidenceRef, evidenceErr = goal.NewArtifactRef(evidence)
		}
		record.ObservedAt, record.ExpiresAt = time.Unix(0, observedAt).UTC(), time.Unix(0, expiresAt).UTC()
		record.ResetAt, record.RetryAt = restoredTime(resetAt), restoredTime(retryAt)
		materialized, materializeErr := application.MaterializeAgentQuotaObservation(record.AgentQuotaObservationSubmission, record.ExpectedRevision)
		if err != nil || placementErr != nil || evidenceErr != nil || materializeErr != nil || materialized != record {
			return errors.New("sqlite.recovery_v38_agent_placement_invalid")
		}
	}
	invalid, err := readSingleColumn(ctx, tx, `SELECT reservation_ref FROM agent_placement_bindings binding WHERE NOT EXISTS(SELECT 1 FROM agent_capacity_reservations reservation WHERE reservation.ref=binding.reservation_ref) OR NOT EXISTS(SELECT 1 FROM agent_quota_observations quota WHERE quota.ref=binding.quota_observation_ref AND quota.revision=binding.quota_observation_revision AND quota.placement_ref=binding.placement_ref)`)
	if err != nil || len(invalid) != 0 {
		return errors.Join(err, errors.New("sqlite.recovery_v38_agent_placement_invalid"))
	}
	if err := validarReservasFisicasRecuperacion(ctx, tx); err != nil {
		return errors.Join(err, errors.New("sqlite.recovery_v38_agent_placement_invalid"))
	}
	return nil
}

func validarObservacionesFisicasRecuperacion(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT ref,source_ref,pool_ref,window_ref,revision,expected_revision,status,quality,observed_at,expires_at,reset_at,retry_at,artifact_ref,slots_applicability,slots_limit,slots_remaining,seconds_applicability,seconds_limit,seconds_remaining,messages_applicability,messages_limit,messages_remaining,tokens_applicability,tokens_limit,tokens_remaining,credits_applicability,credits_limit,credits_remaining,idempotency_key FROM agent_capacity_observations ORDER BY source_ref,pool_ref,revision`)
	if err != nil {
		return err
	}
	defer rows.Close()
	ultimas := make(map[string]time.Time)
	for rows.Next() {
		var registro application.AgentCapacityObservationRecord
		var observada, expira int64
		var reinicio, reintento sql.NullInt64
		var artefacto string
		dimensiones := []*application.AgentCapacityDimension{&registro.Observation.Resources.Slots, &registro.Observation.Resources.Seconds, &registro.Observation.Resources.Messages, &registro.Observation.Resources.Tokens, &registro.Observation.Resources.Credits}
		aplicabilidades := make([]string, 5)
		limites, restantes := make([]sql.NullInt64, 5), make([]sql.NullInt64, 5)
		err = rows.Scan(&registro.Ref, &registro.Observation.SourceRef, &registro.Observation.PoolRef,
			&registro.Observation.WindowRef, &registro.Revision, &registro.ExpectedRevision,
			&registro.Observation.Status, &registro.Observation.Quality, &observada, &expira, &reinicio,
			&reintento, &artefacto, &aplicabilidades[0], &limites[0], &restantes[0],
			&aplicabilidades[1], &limites[1], &restantes[1], &aplicabilidades[2], &limites[2],
			&restantes[2], &aplicabilidades[3], &limites[3], &restantes[3], &aplicabilidades[4],
			&limites[4], &restantes[4], &registro.IdempotencyKey)
		if err != nil {
			return err
		}
		for indice, dimension := range dimensiones {
			dimension.Applicability = application.AgentCapacityApplicability(aplicabilidades[indice])
			dimension.Limit = application.AgentCapacityAmount{Present: limites[indice].Valid, Value: limites[indice].Int64}
			dimension.Remaining = application.AgentCapacityAmount{Present: restantes[indice].Valid, Value: restantes[indice].Int64}
		}
		registro.Observation.ObservedAt, registro.Observation.ExpiresAt = time.Unix(0, observada).UTC(), time.Unix(0, expira).UTC()
		registro.Observation.ResetAt, registro.Observation.RetryAt = restoredTime(reinicio), restoredTime(reintento)
		if artefacto != "" {
			registro.Observation.ArtifactRef, err = goal.NewArtifactRef(artefacto)
			if err != nil {
				return err
			}
		}
		entrega, err := application.NuevaEntregaObservacionCapacidad(registro.Observation)
		clave := string(registro.Observation.SourceRef) + "\x00" + string(registro.Observation.PoolRef)
		if err != nil || entrega.Ref != registro.Ref || entrega.IdempotencyKey != registro.IdempotencyKey ||
			registro.Revision != registro.ExpectedRevision+1 || !registro.Observation.ObservedAt.After(ultimas[clave]) {
			return errors.New("sqlite.recovery_v38_agent_capacity_observation_invalid")
		}
		ultimas[clave] = registro.Observation.ObservedAt
	}
	return rows.Err()
}

func validarReservasFisicasRecuperacion(ctx context.Context, tx *sql.Tx) error {
	acciones, err := readSingleColumn(ctx, tx, `SELECT action_ref FROM agent_capacity_reservations ORDER BY action_ref`)
	if err != nil {
		return err
	}
	for _, accion := range acciones {
		reserva, _, encontrada, err := leerReservaCapacidadAccion(ctx, tx, accion)
		if err != nil || !encontrada {
			return errors.Join(err, errors.New("sqlite.recovery_v38_agent_capacity_reservation_invalid"))
		}
		inicial := reserva
		inicial.State, inicial.Revision = application.AgentCapacityReserved, 1
		inicial.LastTransitionRef, inicial.LastCauseRef = "", ""
		inicial.UpdatedAt, inicial.SettledAt = inicial.ReservedAt, time.Time{}
		if application.ValidateAgentCapacityReservation(inicial) != nil {
			return errors.New("sqlite.recovery_v38_agent_capacity_reservation_invalid")
		}
		rows, err := tx.QueryContext(ctx, `SELECT ref,reservation_ref,project_ref,fence,expected_revision,revision,outcome,cause_kind,cause_ref,effect_attempt_ref,effect_receipt_ref,idempotency_key,recorded_at FROM agent_capacity_transitions WHERE reservation_ref=? ORDER BY revision`, reserva.Ref)
		if err != nil {
			return err
		}
		actual := inicial
		for rows.Next() {
			var transicion application.AgentCapacityTransition
			var proyecto string
			var intento, comprobante sql.NullString
			var registrada int64
			if err := rows.Scan(&transicion.Ref, &transicion.ReservationRef, &proyecto, &transicion.Fence,
				&transicion.ExpectedRevision, &transicion.Revision, &transicion.Outcome, &transicion.Cause,
				&transicion.CauseRef, &intento, &comprobante, &transicion.IdempotencyKey, &registrada); err != nil {
				rows.Close()
				return err
			}
			transicion.ProjectRef, err = goal.NewProjectRef(proyecto)
			transicion.EffectAttemptRef, transicion.EffectReceiptRef = intento.String, comprobante.String
			transicion.RecordedAt = time.Unix(0, registrada).UTC()
			if err != nil || application.ValidateAgentCapacityTransition(actual, transicion) != nil {
				rows.Close()
				return errors.New("sqlite.recovery_v38_agent_capacity_transition_invalid")
			}
			actual.State, actual.Revision = transicion.Outcome, transicion.Revision
			actual.LastTransitionRef, actual.LastCauseRef = transicion.Ref, transicion.CauseRef
			actual.UpdatedAt = transicion.RecordedAt
			if transicion.Outcome == application.AgentCapacityReleased {
				actual.SettledAt = transicion.RecordedAt
			}
		}
		if err := rows.Close(); err != nil || actual != reserva {
			return errors.Join(err, errors.New("sqlite.recovery_v38_agent_capacity_transition_invalid"))
		}
	}
	return nil
}
