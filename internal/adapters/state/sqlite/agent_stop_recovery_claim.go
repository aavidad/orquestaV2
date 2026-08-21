package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

// ValidateAgentStopRecoveryClaim is the last durable read fence before Stop
// reconciliation may contact the external runtime.
func (repository *Repository) ValidateAgentStopRecoveryClaim(
	ctx context.Context,
	claim application.ActionClaim,
) error {
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	if err := requireClaim(ctx, transaction, claim); err != nil {
		return err
	}
	if err := requireAgentStopRecoveryCurrentBindings(ctx, transaction, claim); err != nil {
		return err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return err
	}
	if !now.Before(claim.LeaseUntil) {
		return conflict(errors.New("sqlite.claim_lease_expired"))
	}
	return commit(transaction)
}

func requireAgentStopRecoveryCurrentBindings(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
) error {
	var effectIntentRef string
	if err := transaction.QueryRowContext(
		ctx, `SELECT COALESCE(effect_intent_ref,'') FROM outbox WHERE ref=?`, claim.Action.Ref,
	).Scan(&effectIntentRef); err != nil {
		return mapDatabaseError(err)
	}
	if effectIntentRef != claim.Action.EffectIntentRef {
		return conflict(errors.New("sqlite.claim_effect_intent_conflict"))
	}
	record, err := readGoalRecord(ctx, transaction, claim.Action.GoalRef.String())
	if err != nil {
		return err
	}
	if _, err := application.SelectAgentStopRecoveryAttempt(record, claim); err != nil {
		return conflict(errors.New("sqlite.claim_stop_recovery_effect_selection_conflict"))
	}
	return nil
}
