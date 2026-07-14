package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
)

func (repository *Repository) RecordLaunchAccepted(ctx context.Context, state application.LaunchAcceptedState) error {
	item, err := validateLaunchAccepted(state)
	if err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateWorkItemCAS(ctx, transaction, item, state.ExpectedItemRevision); err != nil {
			return err
		}
		if err := updateExecution(ctx, transaction, state.Execution); err != nil {
			return err
		}
		if err := insertAction(ctx, transaction, state.NextAction); err != nil {
			return err
		}
		if err := insertEvent(ctx, transaction, state.Event); err != nil {
			return err
		}
		return completeClaim(ctx, transaction, state.Claim, state.Event.OccurredAt, "", false)
	})
}

func (repository *Repository) RequeueAction(ctx context.Context, state application.ActionRequeuedState) error {
	if err := validateRequeued(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, func(transaction *sql.Tx) error {
		if err := updateExecution(ctx, transaction, state.Execution); err != nil {
			return err
		}
		return releaseClaimForRetry(ctx, transaction, state)
	})
}

func (repository *Repository) QuarantineAction(ctx context.Context, state application.ActionQuarantinedState) error {
	if err := validateQuarantined(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, func(transaction *sql.Tx) error {
		if err := insertEvent(ctx, transaction, state.Event); err != nil {
			return err
		}
		return completeClaim(
			ctx,
			transaction,
			state.Claim,
			state.Event.OccurredAt,
			state.ErrorCode,
			true,
		)
	})
}

func (repository *Repository) RecordGoalSucceeded(ctx context.Context, state application.GoalSucceededState) error {
	item, err := validateSucceeded(state)
	if err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateWorkItemCAS(ctx, transaction, item, state.ExpectedItemRevision); err != nil {
			return err
		}
		if err := updateExecution(ctx, transaction, state.Execution); err != nil {
			return err
		}
		if err := insertArtifact(ctx, transaction, state.Artifact); err != nil {
			return err
		}
		if err := insertAttestation(ctx, transaction, state.Attestation); err != nil {
			return err
		}
		if err := insertEvents(ctx, transaction, state.Events); err != nil {
			return err
		}
		return completeClaim(ctx, transaction, state.Claim, state.Execution.FinishedAt, "", false)
	})
}

func (repository *Repository) RecordGoalFailed(ctx context.Context, state application.GoalFailedState) error {
	item, err := validateFailed(state)
	if err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateWorkItemCAS(ctx, transaction, item, state.ExpectedItemRevision); err != nil {
			return err
		}
		if err := updateExecution(ctx, transaction, state.Execution); err != nil {
			return err
		}
		if err := insertEvents(ctx, transaction, state.Events); err != nil {
			return err
		}
		return completeClaim(ctx, transaction, state.Claim, state.Execution.FinishedAt, state.Execution.FailureCode, false)
	})
}

func (repository *Repository) mutate(
	ctx context.Context,
	claim application.ActionClaim,
	mutation func(*sql.Tx) error,
) error {
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	if err := requireClaim(ctx, transaction, claim); err != nil {
		return err
	}
	if err := mutation(transaction); err != nil {
		return err
	}
	return commit(transaction)
}
