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

func (repository *Repository) CurrentAgentQuotaObservation(ctx context.Context, placement ports.AgentPlacementRef) (application.AgentQuotaObservationRecord, bool, error) {
	if placement.String() == "" {
		return application.AgentQuotaObservationRecord{}, false, invalid(errors.New("sqlite.agent_quota_placement_invalid"))
	}
	database, err := repository.database()
	if err != nil {
		return application.AgentQuotaObservationRecord{}, false, err
	}
	return readAgentQuotaObservation(ctx, database, `SELECT ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,evidence_ref,idempotency_key,expected_revision,revision FROM agent_quota_observations WHERE placement_ref=? ORDER BY revision DESC LIMIT 1`, placement.String())
}

func (repository *Repository) AppendAgentQuotaObservation(ctx context.Context, record application.AgentQuotaObservationRecord) (application.AgentQuotaObservationRecord, bool, error) {
	record.ObservedAt, record.ExpiresAt = record.ObservedAt.Round(0).UTC(), record.ExpiresAt.Round(0).UTC()
	record.ResetAt, record.RetryAt = record.ResetAt.Round(0).UTC(), record.RetryAt.Round(0).UTC()
	materialized, err := application.MaterializeAgentQuotaObservation(record.AgentQuotaObservationSubmission, record.ExpectedRevision)
	if err != nil || materialized != record || record.ExpectedRevision > maxSQLiteInteger || record.Revision > maxSQLiteInteger {
		return application.AgentQuotaObservationRecord{}, false, invalid(errors.New("sqlite.agent_quota_observation_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.AgentQuotaObservationRecord{}, false, err
	}
	defer transaction.Rollback()
	_, err = transaction.ExecContext(ctx, `INSERT INTO agent_quota_observations(ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,evidence_ref,idempotency_key,expected_revision,revision) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.Ref, record.PlacementRef.String(), string(record.WindowRef), string(record.Status), string(record.Quality),
		requiredTime(record.ObservedAt), requiredTime(record.ExpiresAt), storedTime(record.ResetAt), storedTime(record.RetryAt),
		record.EvidenceRef.String(), record.IdempotencyKey, int64(record.ExpectedRevision), int64(record.Revision))
	if err == nil {
		if err = commit(transaction); err != nil {
			return application.AgentQuotaObservationRecord{}, false, err
		}
		return record, true, nil
	}
	if !isSQLiteConstraintError(err) {
		return application.AgentQuotaObservationRecord{}, false, mapDatabaseError(err)
	}
	_ = transaction.Rollback()
	database, databaseErr := repository.database()
	if databaseErr != nil {
		return application.AgentQuotaObservationRecord{}, false, databaseErr
	}
	replayed, found, readErr := readAgentQuotaObservation(ctx, database, `SELECT ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,evidence_ref,idempotency_key,expected_revision,revision FROM agent_quota_observations WHERE placement_ref=? AND idempotency_key=?`, record.PlacementRef.String(), record.IdempotencyKey)
	if readErr != nil {
		return application.AgentQuotaObservationRecord{}, false, readErr
	}
	if found && replayed == record {
		return replayed, false, nil
	}
	return application.AgentQuotaObservationRecord{}, false, conflict(errors.New("sqlite.agent_quota_observation_conflict"))
}

func readAgentQuotaObservation(ctx context.Context, source queryer, query string, arguments ...any) (application.AgentQuotaObservationRecord, bool, error) {
	var record application.AgentQuotaObservationRecord
	var placement, evidence string
	var observedAt, expiresAt int64
	var resetAt, retryAt sql.NullInt64
	err := source.QueryRowContext(ctx, query, arguments...).Scan(
		&record.Ref, &placement, &record.WindowRef, &record.Status, &record.Quality,
		&observedAt, &expiresAt, &resetAt, &retryAt, &evidence,
		&record.IdempotencyKey, &record.ExpectedRevision, &record.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return record, false, nil
	}
	if err != nil {
		return record, false, mapDatabaseError(err)
	}
	record.PlacementRef, err = ports.NewAgentPlacementRef(placement)
	if evidence != "" && err == nil {
		record.EvidenceRef, err = goal.NewArtifactRef(evidence)
	}
	record.ObservedAt, record.ExpiresAt = time.Unix(0, observedAt).UTC(), time.Unix(0, expiresAt).UTC()
	record.ResetAt, record.RetryAt = restoredTime(resetAt), restoredTime(retryAt)
	materialized, materializeErr := application.MaterializeAgentQuotaObservation(record.AgentQuotaObservationSubmission, record.ExpectedRevision)
	if err != nil || materializeErr != nil || materialized != record {
		return application.AgentQuotaObservationRecord{}, false, invalid(errors.New("sqlite.agent_quota_observation_corrupt"))
	}
	return record, true, nil
}
