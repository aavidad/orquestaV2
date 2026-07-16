package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (repository *Repository) MarkMailboxDelivered(
	ctx context.Context,
	state application.MarkMailboxDeliveredState,
) (application.MailboxRecord, bool, error) {
	if err := validateMarkMailboxDeliveredState(state); err != nil {
		return application.MailboxRecord{}, false, invalid(err)
	}
	return repository.advanceMailbox(ctx, mailboxAdvance{
		kind: application.MailboxMutationDeliver, requestRef: state.RequestRef,
		requestFingerprint: state.RequestFingerprint, authorization: state.AuthorizationReceipt,
		principalRef: state.PrincipalRef, projectRef: state.ProjectRef, goalRef: state.GoalRef,
		messageRef: state.MessageRef, executionRef: state.RecipientExecutionRef,
		claimToken: state.ClaimToken, fence: state.Fence,
		factRef: state.DeliveryRef,
	})
}

func (repository *Repository) ConsumeMailbox(
	ctx context.Context,
	state application.ConsumeMailboxState,
) (application.MailboxRecord, bool, error) {
	if err := validateConsumeMailboxState(state); err != nil {
		return application.MailboxRecord{}, false, invalid(err)
	}
	return repository.advanceMailbox(ctx, mailboxAdvance{
		kind: application.MailboxMutationConsume, requestRef: state.RequestRef,
		requestFingerprint: state.RequestFingerprint, authorization: state.AuthorizationReceipt,
		principalRef: state.PrincipalRef, projectRef: state.ProjectRef, goalRef: state.GoalRef,
		messageRef: state.MessageRef, executionRef: state.RecipientExecutionRef,
		claimToken: state.ClaimToken, fence: state.Fence,
		factRef: state.ConsumptionRef, consumptionReceipt: state.ConsumptionReceipt,
	})
}

type mailboxAdvance struct {
	kind               application.MailboxMutationKind
	requestRef         string
	requestFingerprint string
	authorization      identity.AuthorizationReceipt
	principalRef       identity.PrincipalRef
	projectRef         goal.ProjectRef
	goalRef            goal.GoalRef
	messageRef         application.MailboxMessageRef
	executionRef       goal.ExecutionRef
	claimToken         string
	fence              uint64
	factRef            string
	consumptionReceipt application.ActionConsumptionReceipt
}

func (repository *Repository) advanceMailbox(
	ctx context.Context,
	advance mailboxAdvance,
) (application.MailboxRecord, bool, error) {
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, advance.authorization, advance.principalRef,
		advance.projectRef, identity.PermissionGoalsGet, advance.messageRef.String(),
	); err != nil {
		return application.MailboxRecord{}, false, err
	}
	replayRequest := application.MailboxReplayRequest{
		Kind: advance.kind, RequestRef: advance.requestRef,
		RequestFingerprint: advance.requestFingerprint, PrincipalRef: advance.principalRef,
		ProjectRef: advance.projectRef, GoalRef: advance.goalRef, MessageRef: advance.messageRef,
	}
	replay, found, err := readMailboxReplay(ctx, transaction, replayRequest)
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.MailboxRecord{}, false, err
		}
		return replay.Record, false, nil
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if now.Before(advance.authorization.RecordedAt()) {
		return application.MailboxRecord{}, false, conflict(errors.New("sqlite.mailbox_authorization_from_future"))
	}
	record, err := readMailboxRecord(ctx, transaction, advance.messageRef.String())
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if err := requireMailboxRecipient(record, advance.projectRef, advance.goalRef, advance.principalRef, advance.executionRef); err != nil {
		return application.MailboxRecord{}, false, err
	}
	if record.Retirement != nil {
		return application.MailboxRecord{}, false, conflict(errors.New("sqlite.mailbox_retired"))
	}
	if advance.kind == application.MailboxMutationDeliver {
		result, err := transaction.ExecContext(ctx, `
UPDATE mailbox_delivery_attempts
SET delivery_request_ref = ?, delivery_request_fingerprint = ?,
    delivery_authorization_receipt_ref = ?, delivery_ref = ?, delivered_at = ?
WHERE mailbox_message_ref = ? AND recipient_principal_ref = ?
  AND claim_token = ? AND fence = ?
  AND delivered_at IS NULL AND consumed_at IS NULL
  AND lease_until > ?`,
			advance.requestRef, advance.requestFingerprint, advance.authorization.Ref(),
			advance.factRef, requiredTime(now), advance.messageRef.String(),
			advance.principalRef.String(), advance.claimToken,
			int64(advance.fence), requiredTime(now),
		)
		if err != nil {
			return application.MailboxRecord{}, false, mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return application.MailboxRecord{}, false, err
		}
	} else {
		if err := validateMailboxConsumptionReceipt(record, advance, now); err != nil {
			return application.MailboxRecord{}, false, invalid(err)
		}
		result, err := transaction.ExecContext(ctx, `
UPDATE mailbox_delivery_attempts
SET consumption_request_ref = ?, consumption_request_fingerprint = ?,
    consumption_authorization_receipt_ref = ?, consumption_ref = ?, consumed_at = ?
WHERE mailbox_message_ref = ? AND recipient_principal_ref = ?
  AND claim_token = ? AND fence = ?
  AND delivered_at IS NOT NULL AND consumed_at IS NULL
  AND lease_until > ?`,
			advance.requestRef, advance.requestFingerprint, advance.authorization.Ref(),
			advance.factRef, requiredTime(now), advance.messageRef.String(),
			advance.principalRef.String(), advance.claimToken,
			int64(advance.fence), requiredTime(now),
		)
		if err != nil {
			return application.MailboxRecord{}, false, mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return application.MailboxRecord{}, false, err
		}
		result, err = transaction.ExecContext(ctx, `
UPDATE outbox
SET completed_at = ?, last_error_code = ''
WHERE ref = ? AND kind = 'deliver_mailbox' AND mailbox_message_ref = ?
  AND claim_token = ? AND claimed_by = ? AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
			requiredTime(now), record.Action.Ref, advance.messageRef.String(), advance.claimToken,
			advance.principalRef.String(), int64(advance.fence), int64(advance.fence),
		)
		if err != nil {
			return application.MailboxRecord{}, false, mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return application.MailboxRecord{}, false, err
		}
		receipt := advance.consumptionReceipt
		_, err = transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref, mailbox_message_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			receipt.ActionRef, string(receipt.Kind), receipt.GoalRef.String(),
			receipt.WorkItemRef.String(), receipt.ExecutionRef.String(), advance.messageRef.String(),
			int64(receipt.PlanGeneration), int64(receipt.WorkItemGeneration), int64(receipt.Fence),
			int64(receipt.Fence), receipt.ClaimToken, advance.principalRef.String(),
			string(receipt.Outcome), receipt.ErrorCode, requiredTime(now),
		)
		if err != nil {
			return application.MailboxRecord{}, false, mapDatabaseError(err)
		}
	}
	persisted, err := readMailboxRecord(ctx, transaction, advance.messageRef.String())
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.MailboxRecord{}, false, err
	}
	return persisted, true, nil
}
