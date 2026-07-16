package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (repository *Repository) MailboxReplay(
	ctx context.Context,
	request application.MailboxReplayRequest,
) (application.MailboxReplayRecord, bool, error) {
	if err := validateMailboxReplayRequest(request); err != nil {
		return application.MailboxReplayRecord{}, false, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.MailboxReplayRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	replay, found, err := readMailboxReplay(ctx, transaction, request)
	if err != nil {
		return application.MailboxReplayRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.MailboxReplayRecord{}, false, err
	}
	return replay, found, nil
}

func (repository *Repository) AdmitMailbox(
	ctx context.Context,
	state application.AdmitMailboxState,
) (application.MailboxRecord, bool, error) {
	if err := validateAdmitMailboxState(state); err != nil {
		return application.MailboxRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.Envelope.Source.PrincipalRef,
		state.Envelope.ProjectRef, identity.PermissionGoalsDirect, state.Envelope.GoalRef.String(),
	); err != nil {
		return application.MailboxRecord{}, false, err
	}
	replay, found, err := readMailboxReplay(ctx, transaction, application.MailboxReplayRequest{
		Kind: application.MailboxMutationAdmit, RequestRef: state.RequestRef,
		RequestFingerprint: state.RequestFingerprint,
		PrincipalRef:       state.Envelope.Source.PrincipalRef, ProjectRef: state.Envelope.ProjectRef,
		GoalRef: state.Envelope.GoalRef,
	})
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.MailboxRecord{}, false, err
		}
		return replay.Record, false, nil
	}
	var goalProject, goalState string
	var planGeneration int64
	if err := transaction.QueryRowContext(ctx, `
SELECT project_ref, state, plan_generation FROM goals WHERE ref = ?`,
		state.Envelope.GoalRef.String(),
	).Scan(&goalProject, &goalState, &planGeneration); err != nil {
		return application.MailboxRecord{}, false, mapDatabaseError(err)
	}
	if goalProject != state.Envelope.ProjectRef.String() || goalState != string(goal.GoalStateRunning) ||
		planGeneration != int64(state.Envelope.TargetPlanGeneration) {
		return application.MailboxRecord{}, false, conflict(errors.New("sqlite.mailbox_goal_scope_conflict"))
	}
	parentRevision, childRevision, err := requireMailboxLineage(ctx, transaction, state)
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if parentRevision != int64(state.Action.WorkItemGeneration) {
		return application.MailboxRecord{}, false, conflict(errors.New("sqlite.mailbox_parent_generation_conflict"))
	}
	if err := requireMailboxArtifacts(ctx, transaction, state.Envelope); err != nil {
		return application.MailboxRecord{}, false, err
	}
	var existing string
	err = transaction.QueryRowContext(ctx, `
SELECT ref FROM mailbox_envelopes
WHERE goal_ref = ? AND parent_work_item_ref = ? AND child_work_item_ref = ?
	`,
		state.Envelope.GoalRef.String(), state.Envelope.ParentWorkItemRef.String(),
		state.Envelope.ChildWorkItemRef.String(),
	).Scan(&existing)
	switch {
	case err == nil:
		return application.MailboxRecord{}, false,
			conflict(errors.New("sqlite.mailbox_child_delivery_duplicate"))
	case !errors.Is(err, sql.ErrNoRows):
		return application.MailboxRecord{}, false, mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO mailbox_envelopes(
    ref, request_ref, request_fingerprint, project_ref, goal_ref, plan_generation,
    kind, summary, content_hash,
    source_principal_ref, child_work_item_ref, source_execution_ref, source_work_item_generation,
    recipient_principal_ref, parent_work_item_ref, recipient_execution_ref,
    recipient_work_item_generation, admitted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.Envelope.Ref.String(), state.RequestRef, state.RequestFingerprint,
		state.Envelope.ProjectRef.String(), state.Envelope.GoalRef.String(),
		int64(state.Envelope.TargetPlanGeneration), string(state.Envelope.Kind),
		state.Envelope.Summary, state.Envelope.ContentHash,
		state.Envelope.Source.PrincipalRef.String(), state.Envelope.ChildWorkItemRef.String(),
		state.Envelope.Source.ExecutionRef.String(), childRevision,
		state.Envelope.Recipient.PrincipalRef.String(), state.Envelope.ParentWorkItemRef.String(),
		state.Envelope.Recipient.ExecutionRef.String(), parentRevision,
		requiredTime(state.Envelope.AdmittedAt),
	)
	if err != nil {
		return application.MailboxRecord{}, false, mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO mailbox_admission_receipts(
    ref, mailbox_message_ref, request_ref, request_fingerprint,
    authorization_receipt_ref, project_ref, goal_ref, source_principal_ref, admitted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.Admission.Ref, state.Envelope.Ref.String(), state.RequestRef,
		state.RequestFingerprint, state.AuthorizationReceipt.Ref(),
		state.Envelope.ProjectRef.String(), state.Envelope.GoalRef.String(),
		state.Envelope.Source.PrincipalRef.String(), requiredTime(state.Envelope.AdmittedAt),
	)
	if err != nil {
		return application.MailboxRecord{}, false, mapDatabaseError(err)
	}
	for position, artifactRef := range state.Envelope.ArtifactRefs {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO mailbox_artifact_refs(mailbox_message_ref, goal_ref, artifact_ref, position)
VALUES (?, ?, ?, ?)`, state.Envelope.Ref.String(), state.Envelope.GoalRef.String(),
			artifactRef.String(), position,
		); err != nil {
			return application.MailboxRecord{}, false, mapDatabaseError(err)
		}
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, available_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.Action.Ref, string(state.Action.Kind), state.Action.GoalRef.String(),
		state.Action.WorkItemRef.String(), state.Action.ExecutionRef.String(),
		int64(state.Action.PlanGeneration), int64(state.Action.WorkItemGeneration),
		state.Envelope.Ref.String(), requiredTime(state.Action.AvailableAt),
	); err != nil {
		return application.MailboxRecord{}, false, mapDatabaseError(err)
	}
	if err := insertEvent(ctx, transaction, state.Event); err != nil {
		return application.MailboxRecord{}, false, err
	}
	record, err := readMailboxRecord(ctx, transaction, state.Envelope.Ref.String())
	if err != nil {
		return application.MailboxRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.MailboxRecord{}, false, err
	}
	return record, true, nil
}

func (repository *Repository) ClaimMailbox(
	ctx context.Context,
	state application.ClaimMailboxState,
) (application.MailboxClaim, bool, error) {
	if err := validateClaimMailboxState(state); err != nil {
		return application.MailboxClaim{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.MailboxClaim{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsGet, state.MessageRef.String(),
	); err != nil {
		return application.MailboxClaim{}, false, err
	}
	replay, found, err := readMailboxReplay(ctx, transaction, mailboxReplayRequestFromClaim(state))
	if err != nil {
		return application.MailboxClaim{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.MailboxClaim{}, false, err
		}
		return replay.Claim, false, nil
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.MailboxClaim{}, false, err
	}
	if now.Before(state.AuthorizationReceipt.RecordedAt()) {
		return application.MailboxClaim{}, false, conflict(errors.New("sqlite.mailbox_authorization_from_future"))
	}
	leaseUntil, err := safeLeaseUntil(now, state.LeaseDuration)
	if err != nil {
		return application.MailboxClaim{}, false, invalid(err)
	}
	stored, err := readMailboxRecord(ctx, transaction, state.MessageRef.String())
	if err != nil {
		return application.MailboxClaim{}, false, err
	}
	if err := requireMailboxRecipient(stored, state.ProjectRef, state.GoalRef, state.PrincipalRef, state.RecipientExecutionRef); err != nil {
		return application.MailboxClaim{}, false, err
	}
	if stored.Retirement != nil {
		return application.MailboxClaim{}, false, conflict(errors.New("sqlite.mailbox_retired"))
	}
	if err := requireMailboxCurrentRecipient(ctx, transaction, stored); err != nil {
		return application.MailboxClaim{}, false, err
	}
	var claimedUntil, completedAt, retiredAt sql.NullInt64
	var deliveryAttempt, currentFence int64
	if err := transaction.QueryRowContext(ctx, `
SELECT claimed_until, delivery_attempt, fence, completed_at, retired_at
FROM outbox WHERE ref = ? AND kind = 'deliver_mailbox'`, stored.Action.Ref,
	).Scan(&claimedUntil, &deliveryAttempt, &currentFence, &completedAt, &retiredAt); err != nil {
		return application.MailboxClaim{}, false, mapDatabaseError(err)
	}
	if completedAt.Valid || retiredAt.Valid || stored.Acknowledgement != nil {
		return application.MailboxClaim{}, false, conflict(errors.New("sqlite.mailbox_terminal"))
	}
	if claimedUntil.Valid && now.Before(time.Unix(0, claimedUntil.Int64).UTC()) {
		return application.MailboxClaim{}, false, stateError(
			application.StateAlreadyClaimed, errors.New("sqlite.mailbox_claim_live"),
		)
	}
	if deliveryAttempt < 0 || currentFence < 0 || deliveryAttempt != currentFence ||
		uint64(currentFence) >= maxSQLiteInteger {
		return application.MailboxClaim{}, false, conflict(errors.New("sqlite.mailbox_attempt_exhausted"))
	}
	fence := currentFence + 1
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, delivery_attempt = ?, fence = ?
WHERE ref = ? AND kind = 'deliver_mailbox' AND mailbox_message_ref = ?
  AND completed_at IS NULL AND quarantined_at IS NULL
  AND retired_at IS NULL
  AND delivery_attempt = ? AND fence = ?
  AND (claim_token IS NULL OR claimed_until <= ?)`,
		state.Token, state.PrincipalRef.String(), requiredTime(leaseUntil), fence, fence,
		stored.Action.Ref, state.MessageRef.String(), currentFence, currentFence, requiredTime(now),
	)
	if err != nil {
		return application.MailboxClaim{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.MailboxClaim{}, false, err
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO mailbox_delivery_attempts(
    mailbox_message_ref, action_ref, project_ref, recipient_principal_ref,
    fence, claim_token,
    claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
    claimed_at, lease_until
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.MessageRef.String(), stored.Action.Ref, state.ProjectRef.String(),
		state.PrincipalRef.String(), fence, state.Token,
		state.RequestRef, state.RequestFingerprint, state.AuthorizationReceipt.Ref(),
		requiredTime(now), requiredTime(leaseUntil),
	)
	if err != nil {
		return application.MailboxClaim{}, false, mapDatabaseError(err)
	}
	record, err := readMailboxRecord(ctx, transaction, state.MessageRef.String())
	if err != nil {
		return application.MailboxClaim{}, false, err
	}
	if len(record.Attempts) == 0 {
		return application.MailboxClaim{}, false, conflict(errors.New("sqlite.mailbox_claim_missing_after_write"))
	}
	claim := application.MailboxClaim{Record: record, Attempt: record.Attempts[len(record.Attempts)-1]}
	if err := commit(transaction); err != nil {
		return application.MailboxClaim{}, false, err
	}
	return claim, true, nil
}
