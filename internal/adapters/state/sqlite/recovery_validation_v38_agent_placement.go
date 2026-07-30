package sqlite

import "context"
import "database/sql"
import "errors"
import "time"
import "orquesta/internal/application"
import "orquesta/internal/goal"
import "orquesta/internal/ports"

func validateRecoveryV38AgentPlacement(ctx context.Context, tx *sql.Tx) error {
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
	return nil
}
