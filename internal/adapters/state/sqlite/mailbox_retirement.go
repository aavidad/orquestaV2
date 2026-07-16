package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

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

func retireFailedRecipientMailboxes(
	ctx context.Context,
	transaction *sql.Tx,
	failed application.ExecutionRecord,
) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT envelope.ref
FROM mailbox_envelopes envelope
LEFT JOIN mailbox_delivery_acks ack
  ON ack.mailbox_message_ref = envelope.ref
LEFT JOIN mailbox_retirements retirement
  ON retirement.mailbox_message_ref = envelope.ref
WHERE envelope.recipient_execution_ref = ?
  AND ack.mailbox_message_ref IS NULL
  AND retirement.mailbox_message_ref IS NULL
ORDER BY envelope.ref`, failed.Ref.String())
	if err != nil {
		return mapDatabaseError(err)
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return mapDatabaseError(err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return mapDatabaseError(err)
	}
	for _, ref := range refs {
		record, err := readMailboxRecord(ctx, transaction, ref)
		if err != nil {
			return err
		}
		retirement, err := application.BuildMailboxRetirement(record, failed)
		if err != nil {
			return invalid(err)
		}
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
UPDATE outbox
SET retired_at = ?
WHERE ref = ? AND kind = 'deliver_mailbox' AND mailbox_message_ref = ?
  AND execution_ref = ? AND retired_at IS NULL`,
			requiredTime(retirement.RetiredAt), retirement.ActionRef,
			retirement.MessageRef.String(), retirement.RecipientExecutionRef.String(),
		)
		if err != nil {
			return mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return err
		}
	}
	return nil
}
