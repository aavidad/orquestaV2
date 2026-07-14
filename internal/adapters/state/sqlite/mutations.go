package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func (repository *Repository) RecordLaunchPrepared(ctx context.Context, state application.LaunchPreparedState) error {
	if err := validateLaunchPrepared(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, transaction, state.Goal); err != nil {
			return err
		}
		if err := updateExecutionCAS(ctx, transaction, state.Execution, application.ExecutionQueued); err != nil {
			return err
		}
		return insertEvent(ctx, transaction, state.Event)
	})
}

func (repository *Repository) RecordLaunchAccepted(ctx context.Context, state application.LaunchAcceptedState) error {
	if err := validateLaunchAccepted(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateExecutionCAS(ctx, transaction, state.Execution, application.ExecutionDispatching); err != nil {
			return err
		}
		// Consume first so the partial active-action index remains a hard
		// invariant even while launch and observe share one WorkItem.
		if err := completeClaim(ctx, transaction, state.Claim, state.Event.OccurredAt, "", false); err != nil {
			return err
		}
		if err := insertAction(ctx, transaction, state.NextAction); err != nil {
			return err
		}
		if err := insertEvent(ctx, transaction, state.Event); err != nil {
			return err
		}
		return nil
	})
}

func (repository *Repository) RequeueAction(ctx context.Context, state application.ActionRequeuedState) error {
	if err := validateRequeued(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateExecutionCAS(ctx, transaction, state.Execution, state.Execution.State); err != nil {
			return err
		}
		return releaseClaimForRetry(ctx, transaction, state)
	})
}

func (repository *Repository) QuarantineAction(ctx context.Context, state application.ActionQuarantinedState) error {
	if err := validateQuarantined(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
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

func (repository *Repository) RecordExecutionReplaced(
	ctx context.Context,
	state application.ExecutionReplacedState,
) error {
	if err := validateExecutionReplaced(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		item, _ := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
		if err := updateWorkItemCAS(ctx, transaction, item, state.ExpectedItemRevision); err != nil {
			return err
		}
		expectedExecutionState := application.ExecutionRunning
		if state.Claim.Action.Kind == application.ActionLaunchAgent {
			expectedExecutionState = application.ExecutionDispatching
		}
		// Retire the old active execution before inserting its replacement;
		// the partial unique index then proves at most one active attempt.
		if err := updateExecutionCAS(ctx, transaction, state.FailedExecution, expectedExecutionState); err != nil {
			return err
		}
		if err := completeClaim(
			ctx, transaction, state.Claim, state.OperationAt, state.ErrorCode, false,
		); err != nil {
			return err
		}
		if err := insertExecution(ctx, transaction, state.ReplacementExecution); err != nil {
			return err
		}
		if err := insertAction(ctx, transaction, state.NextAction); err != nil {
			return err
		}
		return insertEvents(ctx, transaction, state.Events)
	})
}

func (repository *Repository) RecordGoalSucceeded(ctx context.Context, state application.GoalSucceededState) error {
	if _, err := validateSucceeded(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, transaction, state.Goal); err != nil {
			return err
		}
		if err := updateExecutionCAS(ctx, transaction, state.Execution, application.ExecutionRunning); err != nil {
			return err
		}
		if err := insertArtifact(ctx, transaction, state.Artifact); err != nil {
			return err
		}
		if err := insertAttestation(ctx, transaction, state.Attestation); err != nil {
			return err
		}
		if err := completeClaim(ctx, transaction, state.Claim, state.Execution.FinishedAt, "", false); err != nil {
			return err
		}
		if err := insertScheduled(ctx, transaction, state.NewExecutions, state.NewActions); err != nil {
			return err
		}
		if err := requireReadyExecutions(ctx, transaction, state.Goal); err != nil {
			return err
		}
		if err := insertEvents(ctx, transaction, state.Events); err != nil {
			return err
		}
		return nil
	})
}

func (repository *Repository) RecordGoalFailed(ctx context.Context, state application.GoalFailedState) error {
	if _, err := validateFailed(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(transaction *sql.Tx) error {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, transaction, state.Goal); err != nil {
			return err
		}
		expectedExecutionState := application.ExecutionRunning
		if state.Claim.Action.Kind == application.ActionLaunchAgent {
			expectedExecutionState = application.ExecutionDispatching
		}
		if err := updateExecutionCAS(ctx, transaction, state.Execution, expectedExecutionState); err != nil {
			return err
		}
		if err := completeClaim(
			ctx, transaction, state.Claim, state.Execution.FinishedAt, state.Execution.FailureCode, false,
		); err != nil {
			return err
		}
		if err := insertScheduled(ctx, transaction, state.NewExecutions, state.NewActions); err != nil {
			return err
		}
		if err := requireReadyExecutions(ctx, transaction, state.Goal); err != nil {
			return err
		}
		if err := insertEvents(ctx, transaction, state.Events); err != nil {
			return err
		}
		return nil
	})
}

func requireReadyExecutions(ctx context.Context, transaction *sql.Tx, aggregate goal.Goal) error {
	for _, item := range aggregate.ReadyWorkItems() {
		var count int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM executions
WHERE goal_ref = ? AND work_item_ref = ?
  AND state IN ('queued', 'dispatching', 'running')`,
			aggregate.Ref().String(), item.Ref().String(),
		).Scan(&count); err != nil {
			return mapDatabaseError(err)
		}
		if count != 1 {
			return conflict(errors.New("sqlite.ready_execution_missing"))
		}
	}
	return nil
}

func insertScheduled(
	ctx context.Context,
	transaction *sql.Tx,
	executions []application.ExecutionRecord,
	actions []application.ActionRecord,
) error {
	for _, execution := range executions {
		if err := insertExecution(ctx, transaction, execution); err != nil {
			return err
		}
	}
	for _, action := range actions {
		if err := insertAction(ctx, transaction, action); err != nil {
			return err
		}
	}
	return nil
}

func (repository *Repository) mutate(
	ctx context.Context,
	claim application.ActionClaim,
	operationAt time.Time,
	mutation func(*sql.Tx) error,
) error {
	if operationAt.IsZero() {
		return invalid(errors.New("sqlite.operation_time_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	if err := repository.requireLiveLease(claim); err != nil {
		return err
	}
	if err := requireClaim(ctx, transaction, claim); err != nil {
		return err
	}
	if err := mutation(transaction); err != nil {
		return err
	}
	if err := repository.requireLiveLease(claim); err != nil {
		return err
	}
	return commit(transaction)
}

func (repository *Repository) requireLiveLease(claim application.ActionClaim) error {
	now, err := repository.transactionTime()
	if err != nil {
		return err
	}
	if !now.Before(claim.LeaseUntil) {
		return conflict(errors.New("sqlite.claim_lease_expired"))
	}
	return nil
}
