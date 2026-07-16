package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func (repository *Repository) GetMailbox(
	ctx context.Context,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef application.MailboxMessageRef,
	recipient application.MailboxEndpoint,
) (application.MailboxRecord, error) {
	if projectRef.String() == "" || goalRef.String() == "" || messageRef.String() == "" ||
		recipient.PrincipalRef.String() == "" || recipient.WorkItemRef.String() == "" ||
		recipient.ExecutionRef.String() == "" {
		return application.MailboxRecord{}, invalid(errors.New("sqlite.mailbox_query_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, err := readMailboxRecord(ctx, transaction, messageRef.String())
	if err != nil {
		return application.MailboxRecord{}, err
	}
	if record.Envelope.ProjectRef != projectRef || record.Envelope.GoalRef != goalRef ||
		record.Envelope.Recipient != recipient {
		return application.MailboxRecord{}, stateError(application.StateNotFound, errors.New("sqlite.mailbox_not_found"))
	}
	if err := validateRecoveredMailboxTerminalCause(ctx, transaction, record); err != nil {
		return application.MailboxRecord{}, invalid(err)
	}
	if err := commit(transaction); err != nil {
		return application.MailboxRecord{}, err
	}
	return record, nil
}

func (repository *Repository) ListMailbox(
	ctx context.Context,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	recipient application.MailboxEndpoint,
	limit int,
) ([]application.MailboxRecord, error) {
	if projectRef.String() == "" || goalRef.String() == "" ||
		recipient.PrincipalRef.String() == "" || recipient.WorkItemRef.String() == "" ||
		recipient.ExecutionRef.String() == "" || limit <= 0 {
		return nil, invalid(errors.New("sqlite.mailbox_query_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	rows, err := transaction.QueryContext(ctx, `
SELECT ref FROM mailbox_envelopes
WHERE project_ref = ? AND goal_ref = ? AND recipient_principal_ref = ?
  AND parent_work_item_ref = ? AND recipient_execution_ref = ?
ORDER BY admitted_at ASC, ref ASC LIMIT ?`,
		projectRef.String(), goalRef.String(), recipient.PrincipalRef.String(),
		recipient.WorkItemRef.String(), recipient.ExecutionRef.String(), limit,
	)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return nil, mapDatabaseError(err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]application.MailboxRecord, 0, len(refs))
	for _, ref := range refs {
		record, err := readMailboxRecord(ctx, transaction, ref)
		if err != nil {
			return nil, err
		}
		if err := validateRecoveredMailboxTerminalCause(ctx, transaction, record); err != nil {
			return nil, invalid(err)
		}
		result = append(result, record)
	}
	if err := commit(transaction); err != nil {
		return nil, err
	}
	return result, nil
}
