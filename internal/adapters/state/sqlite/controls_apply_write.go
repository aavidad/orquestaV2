package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
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
	if status, err := supersededStopActionStatus(ctx, transaction, *state.SupersededControl); err != nil || status == "quarantined" {
		return "", err
	}
	preRetiredAction := state.RetireActionRefs[0]
	if err := settleRetiredLaunchReservation(
		ctx, transaction, preRetiredAction, state.OperationAt,
	); err != nil {
		return "", err
	}
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
		if applicationTerminalExecution(execution.State) {
			if err := scheduleExecutionSessionRevocation(
				ctx, transaction, execution.GoalRef, execution.WorkItemRef,
				execution.Ref, "", state.OperationAt,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func persistControlActionsAndMailboxes(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
) error {
	for _, control := range state.NewControls {
		if err := insertControl(ctx, transaction, control); err != nil {
			return err
		}
	}
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
			ctx, transaction, state.Claim, state.OperationAt, state.ClaimErrorCode, false, state.EffectReceipt,
		); err != nil {
			return err
		}
	}
	if state.BudgetSettlement != nil {
		if err := insertBudgetSettlement(ctx, transaction, *state.BudgetSettlement); err != nil {
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
		if err := settleRetiredLaunchReservation(
			ctx, transaction, actionRef, state.OperationAt,
		); err != nil {
			return err
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

func settleRetiredLaunchReservation(
	ctx context.Context,
	transaction *sql.Tx,
	actionRef string,
	at time.Time,
) error {
	var kind string
	var governanceVersion int64
	err := transaction.QueryRowContext(ctx, `
SELECT kind, governance_version FROM outbox WHERE ref = ?`, actionRef).Scan(&kind, &governanceVersion)
	if err != nil {
		return mapDatabaseError(err)
	}
	if kind != string(application.ActionLaunchAgent) || governanceVersion != 1 {
		return nil
	}
	reservation, found, err := readActiveBudgetReservation(ctx, transaction, actionRef)
	if err != nil || !found {
		return err
	}
	var attempts int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM effect_attempts WHERE action_ref = ?`, actionRef).Scan(&attempts); err != nil {
		return mapDatabaseError(err)
	}
	if attempts != 0 {
		return conflict(errors.New("sqlite.retired_launch_effect_attempt_exists"))
	}
	usage := governance.ResourceUsage{
		Resources: governance.ResourceVector{Currency: reservation.Resources.Currency},
		Known:     governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	}
	settlement, err := governance.Reconcile(reservation, usage)
	if err != nil {
		return invalid(err)
	}
	settlement.SettledAt = at
	return insertBudgetSettlement(ctx, transaction, settlement)
}
