package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

// ValidateAgentLaunchRecoveryClaim is the last read-only fence before an
// idempotent reconciliation may contact an external launch provider.
func (repository *Repository) ValidateAgentLaunchRecoveryClaim(
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
	if err := requireAgentLaunchRecoveryCurrentBindings(ctx, transaction, claim); err != nil {
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

// requireAgentLaunchRecoveryCurrentBindings runs inside the writer's
// transaction as well as the pre-provider read fence. The exact reservation
// revisions are authority: a concurrent capacity transition or budget
// settlement must win with StateConflict instead of being overwritten by a
// later retry release.
func requireAgentLaunchRecoveryCurrentBindings(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
) error {
	capacityReservation, placement, found, err := leerReservaCapacidadAccion(
		ctx, transaction, claim.Action.Ref,
	)
	if err != nil {
		return err
	}
	if !found || capacityReservation != claim.CapacityReservation ||
		placement != claim.ReferenciaColocacion {
		return conflict(errors.New("sqlite.claim_capacity_conflict"))
	}
	if err := requireClaimBudgetReservation(ctx, transaction, claim, claimCandidate{
		action: claim.Action, projectRef: claim.Action.EffectIntent.Subject.ProjectRef,
	}); err != nil {
		return err
	}
	activeReservation, found, err := readActiveBudgetReservation(ctx, transaction, claim.Action.Ref)
	if err != nil {
		return err
	}
	if !found || activeReservation != claim.BudgetReservation {
		return conflict(errors.New("sqlite.claim_budget_reservation_conflict"))
	}
	var effectIntentRef string
	if err := transaction.QueryRowContext(
		ctx, `SELECT COALESCE(effect_intent_ref,'') FROM outbox WHERE ref=?`, claim.Action.Ref,
	).Scan(&effectIntentRef); err != nil {
		return mapDatabaseError(err)
	}
	if effectIntentRef != claim.Action.EffectIntentRef {
		return conflict(errors.New("sqlite.claim_effect_intent_conflict"))
	}
	return nil
}
