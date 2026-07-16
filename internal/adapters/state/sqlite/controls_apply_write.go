package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
)

func persistInitialControlState(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
	found bool,
) (string, error) {
	if !found {
		if err := insertControl(ctx, transaction, state.Control); err != nil {
			return "", err
		}
	}
	if state.SupersededControl == nil {
		return "", nil
	}
	if err := updateControl(ctx, transaction, *state.SupersededControl); err != nil {
		return "", err
	}
	preRetiredAction := state.RetireActionRefs[0]
	if err := consumeRetiredAction(
		ctx, transaction, preRetiredAction, state.Control.Ref, state.OperationAt,
	); err != nil {
		return "", err
	}
	return preRetiredAction, nil
}

func persistControlGoalAndExecutions(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
	current application.GoalRecord,
) error {
	if state.Goal.Revision() != current.Goal.Revision() {
		if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, transaction, state.Goal); err != nil {
			return err
		}
	}
	if state.RequireMailboxClearForExecutionRef.String() != "" {
		if err := requireNoUnresolvedRecipientMailbox(
			ctx, transaction, state.RequireMailboxClearForExecutionRef,
		); err != nil {
			return err
		}
	}
	for _, execution := range state.Executions {
		stored, exists := sqliteExecutionByRef(current.Executions, execution.Ref)
		if !exists {
			if err := insertExecution(ctx, transaction, execution); err != nil {
				return err
			}
			continue
		}
		if err := updateExecutionCAS(ctx, transaction, execution, stored.State); err != nil {
			return err
		}
	}
	return nil
}

func persistControlActionsAndMailboxes(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
) error {
	for _, action := range state.NewActions {
		if err := insertAction(ctx, transaction, action); err != nil {
			return err
		}
	}
	if state.RetireMailboxForExecutionRef.String() == "" {
		return nil
	}
	retired, err := retireControlledRecipientMailboxes(
		ctx, transaction, state.RetireMailboxForExecutionRef, state.OperationAt,
	)
	if err != nil {
		return err
	}
	if !retired {
		return nil
	}
	_, err = transaction.ExecContext(ctx, `
UPDATE executions SET recipient_mailbox_retired = 1
WHERE ref = ? AND state IN ('succeeded', 'failed', 'stopped', 'canceled')`,
		state.RetireMailboxForExecutionRef.String(),
	)
	return mapDatabaseError(err)
}

func (repository *Repository) settleControlApplication(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
	found bool,
	preRetiredAction string,
) error {
	if state.Claim.Action.Ref != "" {
		if err := repository.requireLiveLease(state.Claim); err != nil {
			return err
		}
		if err := requireClaim(ctx, transaction, state.Claim); err != nil {
			return err
		}
		if err := completeClaimWithEffect(
			ctx, transaction, state.Claim, state.OperationAt, "", false, state.StopReceipt,
		); err != nil {
			return err
		}
	}
	// The claimed control action owns the current WorkItem fence. Consume it
	// before retiring rival launch/observe actions: retirement advances the
	// fence and would otherwise invalidate the claim that authorized this
	// transaction. All writes remain atomic in this transaction.
	for _, actionRef := range state.RetireActionRefs {
		if actionRef == preRetiredAction ||
			(state.Claim.Action.Ref != "" && actionRef == state.Claim.Action.Ref) {
			continue
		}
		if err := consumeRetiredAction(
			ctx, transaction, actionRef, state.Control.Ref, state.OperationAt,
		); err != nil {
			return err
		}
	}
	if err := insertEvents(ctx, transaction, state.Events); err != nil {
		return err
	}
	if found {
		return updateControl(ctx, transaction, state.Control)
	}
	return nil
}
