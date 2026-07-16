package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

const controlledMailboxCancellationCode = "application.execution_canceled"

func retireControlledRecipientMailboxes(
	ctx context.Context,
	transaction *sql.Tx,
	executionRef goal.ExecutionRef,
	at time.Time,
) (bool, error) {
	var execution application.ExecutionRecord
	var goalValue, itemValue, state, failureCode string
	var finishedAt sql.NullInt64
	err := transaction.QueryRowContext(ctx, `
SELECT goal_ref, work_item_ref, state, failure_code, finished_at
FROM executions WHERE ref = ?`, executionRef.String()).Scan(
		&goalValue, &itemValue, &state, &failureCode, &finishedAt,
	)
	if err != nil {
		return false, mapDatabaseError(err)
	}
	if state == string(application.ExecutionSucceeded) {
		failureCode = controlledMailboxCancellationCode
	}
	if (state != string(application.ExecutionSucceeded) && state != string(application.ExecutionFailed) &&
		state != string(application.ExecutionStopped) && state != string(application.ExecutionCanceled)) ||
		!validText(failureCode) || !finishedAt.Valid ||
		finishedAt.Int64 != requiredTime(at) {
		return false, conflict(errors.New("sqlite.controlled_mailbox_terminal_invalid"))
	}
	var refErr error
	if execution.Ref, refErr = goal.NewExecutionRef(executionRef.String()); refErr != nil {
		return false, invalid(refErr)
	}
	if execution.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return false, invalid(refErr)
	}
	if execution.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
		return false, invalid(refErr)
	}
	execution.State = application.ExecutionState(state)
	execution.FailureCode = failureCode
	execution.FinishedAt = at.UTC()
	refs, err := unresolvedRecipientMailboxRefs(ctx, transaction, executionRef)
	if err != nil {
		return false, err
	}
	for _, ref := range refs {
		record, err := readMailboxRecord(ctx, transaction, ref)
		if err != nil {
			return false, err
		}
		retirement := application.MailboxRetirement{
			MessageRef: record.Envelope.Ref, ActionRef: record.Action.Ref,
			RecipientExecutionRef: execution.Ref, FailureCode: failureCode, RetiredAt: at.UTC(),
		}
		candidate := record
		candidate.Retirement = &retirement
		candidate.State = application.MailboxStateRetired
		if err := application.ValidateMailboxRetirement(candidate); err != nil {
			return false, invalid(err)
		}
		if err := insertMailboxRetirement(ctx, transaction, retirement); err != nil {
			return false, err
		}
	}
	return len(refs) > 0, nil
}

func unresolvedRecipientMailboxRefs(
	ctx context.Context,
	transaction *sql.Tx,
	executionRef goal.ExecutionRef,
) ([]string, error) {
	rows, err := transaction.QueryContext(ctx, `
SELECT envelope.ref
FROM mailbox_envelopes envelope
LEFT JOIN mailbox_delivery_acks ack ON ack.mailbox_message_ref = envelope.ref
LEFT JOIN mailbox_retirements retirement ON retirement.mailbox_message_ref = envelope.ref
WHERE envelope.recipient_execution_ref = ?
  AND ack.mailbox_message_ref IS NULL AND retirement.mailbox_message_ref IS NULL
ORDER BY envelope.ref`, executionRef.String())
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return nil, mapDatabaseError(err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return refs, nil
}

func insertMailboxRetirement(
	ctx context.Context,
	transaction *sql.Tx,
	retirement application.MailboxRetirement,
) error {
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO mailbox_retirements(
    mailbox_message_ref, action_ref, recipient_execution_ref, failure_code, retired_at
) VALUES (?, ?, ?, ?, ?)`,
		retirement.MessageRef.String(), retirement.ActionRef,
		retirement.RecipientExecutionRef.String(), retirement.FailureCode,
		requiredTime(retirement.RetiredAt),
	); err != nil {
		return mapDatabaseError(err)
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox SET retired_at = ?
WHERE ref = ? AND kind = 'deliver_mailbox' AND mailbox_message_ref = ?
  AND execution_ref = ? AND retired_at IS NULL`,
		requiredTime(retirement.RetiredAt), retirement.ActionRef,
		retirement.MessageRef.String(), retirement.RecipientExecutionRef.String(),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

// requireNoUnresolvedRecipientMailbox serializes execution replacement with
// mailbox admission/resolution in the repository writer transaction. A
// consumed message remains unresolved until ACK or system retirement.
func requireNoUnresolvedRecipientMailbox(
	ctx context.Context,
	transaction *sql.Tx,
	executionRef goal.ExecutionRef,
) error {
	var count int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM mailbox_envelopes envelope
LEFT JOIN mailbox_delivery_acks ack
  ON ack.mailbox_message_ref = envelope.ref
LEFT JOIN mailbox_retirements retirement
  ON retirement.mailbox_message_ref = envelope.ref
WHERE envelope.recipient_execution_ref = ?
  AND ack.mailbox_message_ref IS NULL
  AND retirement.mailbox_message_ref IS NULL`, executionRef.String()).Scan(&count); err != nil {
		return mapDatabaseError(err)
	}
	if count != 0 {
		return stateError(
			application.StateRecipientMailboxActive,
			errors.New("sqlite.recipient_mailbox_active"),
		)
	}
	return nil
}
