package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
)

// ValidateAgentStopRecoveryClaim is the last read-only state fence before an
// adapter may inspect the exact durable Stop operation. It never authorizes a
// second Stop or a terminal write.
func (repository *Repository) ValidateAgentStopRecoveryClaim(
	ctx context.Context,
	claim application.ActionClaim,
) error {
	if claim.Action.Kind != application.ActionStopAgent {
		return conflict(errors.New("sqlite.stop_recovery_kind_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	if err := requireClaim(ctx, transaction, claim); err != nil {
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
