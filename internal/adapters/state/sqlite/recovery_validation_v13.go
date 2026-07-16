package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func validateRecoveryV13Mailbox(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT ref FROM mailbox_envelopes ORDER BY ref`)
	if err != nil {
		return err
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, ref := range refs {
		record, err := readMailboxRecord(ctx, transaction, ref)
		if err != nil {
			return err
		}
		if record.Envelope.Ref.String() != ref || record.Action.Kind != application.ActionDeliverMailbox {
			return fmt.Errorf("sqlite.recovery_mailbox_record_invalid:%s", ref)
		}
		if err := requireMailboxArtifacts(ctx, transaction, record.Envelope); err != nil {
			return fmt.Errorf("sqlite.recovery_mailbox_artifact_invalid:%s:%w", ref, err)
		}
		if err := validateRecoveredMailboxTerminalCause(ctx, transaction, record); err != nil {
			return fmt.Errorf("sqlite.recovery_mailbox_terminal_invalid:%s:%w", ref, err)
		}
		for index, attempt := range record.Attempts {
			if attempt.Fence != uint64(index+1) || attempt.Recipient != record.Envelope.Recipient {
				return fmt.Errorf("sqlite.recovery_mailbox_attempt_invalid:%s", ref)
			}
		}
		var admissionEvents int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM events
WHERE kind = 'mailbox.admitted' AND goal_ref = ? AND work_item_ref = ?
  AND execution_ref = ? AND occurred_at = ?`,
			record.Envelope.GoalRef.String(), record.Envelope.ParentWorkItemRef.String(),
			record.Envelope.Recipient.ExecutionRef.String(), requiredTime(record.Envelope.AdmittedAt),
		).Scan(&admissionEvents); err != nil {
			return err
		}
		if admissionEvents == 0 {
			return fmt.Errorf("sqlite.recovery_mailbox_admission_event_missing:%s", ref)
		}
		var resolutions int
		if record.Acknowledgement == nil {
			if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM goal_child_handoff_resolutions WHERE mailbox_message_ref = ?`, ref,
			).Scan(&resolutions); err != nil {
				return err
			}
			if resolutions != 0 {
				return fmt.Errorf("sqlite.recovery_mailbox_resolution_without_ack:%s", ref)
			}
			continue
		}
		ack := record.Acknowledgement
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM goal_child_handoff_resolutions
WHERE goal_ref = ? AND parent_work_item_ref = ? AND child_work_item_ref = ?
  AND mailbox_message_ref = ? AND outcome = ? AND receipt_ref = ? AND resolved_at = ?`,
			ack.GoalRef.String(), ack.ParentWorkItemRef.String(), ack.ChildWorkItemRef.String(),
			ref, string(ack.Outcome), ack.Ref, requiredTime(ack.AcknowledgedAt),
		).Scan(&resolutions); err != nil {
			return err
		}
		if resolutions != 1 {
			return fmt.Errorf("sqlite.recovery_mailbox_ack_resolution_invalid:%s", ref)
		}
		var acknowledgementEvents int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM events
WHERE kind = ? AND goal_ref = ? AND work_item_ref = ?
  AND execution_ref = ? AND occurred_at = ?`,
			"mailbox."+string(ack.Outcome), ack.GoalRef.String(), ack.ParentWorkItemRef.String(),
			ack.Recipient.ExecutionRef.String(), requiredTime(ack.AcknowledgedAt),
		).Scan(&acknowledgementEvents); err != nil {
			return err
		}
		if acknowledgementEvents == 0 {
			return fmt.Errorf("sqlite.recovery_mailbox_ack_event_missing:%s", ref)
		}
	}
	return nil
}

func validateRecoveryOutboxV13(ctx context.Context, transaction *sql.Tx) error {
	if err := validateRecoveryOutboxLegacy(ctx, transaction); err != nil {
		return err
	}
	rows, err := transaction.QueryContext(ctx, `
SELECT o.ref, o.goal_ref, o.work_item_ref, o.execution_ref, o.mailbox_message_ref,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
       o.completed_at, o.retired_at, o.quarantined_at,
       e.project_ref, e.recipient_principal_ref, e.parent_work_item_ref,
       e.recipient_execution_ref, e.plan_generation, e.recipient_work_item_generation,
       wi.state, wi.revision, wi.execution_ref, g.state, g.plan_generation,
       (SELECT COUNT(*) FROM mailbox_delivery_attempts a WHERE a.mailbox_message_ref = o.mailbox_message_ref),
       (SELECT MAX(a.consumed_at) FROM mailbox_delivery_attempts a WHERE a.mailbox_message_ref = o.mailbox_message_ref)
FROM outbox o
JOIN mailbox_envelopes e ON e.ref = o.mailbox_message_ref
JOIN work_items wi ON wi.goal_ref = o.goal_ref AND wi.ref = o.work_item_ref
JOIN goals g ON g.ref = o.goal_ref
WHERE o.kind = 'deliver_mailbox'
ORDER BY o.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var action application.ActionRecord
		var goalValue, itemValue, executionValue, messageValue string
		var envelopeProject, envelopePrincipal, envelopeItem, envelopeExecution string
		var itemState, goalState string
		var planGeneration, itemGeneration, availableAt, deliveryAttempt, fence int64
		var envelopePlan, envelopeItemGeneration, currentItemRevision, currentGoalPlan int64
		var attemptCount int64
		var token, worker, currentExecution sql.NullString
		var claimedUntil, completedAt, retiredAt, quarantinedAt, consumedAt sql.NullInt64
		if err := rows.Scan(
			&action.Ref, &goalValue, &itemValue, &executionValue, &messageValue,
			&planGeneration, &itemGeneration, &availableAt,
			&token, &worker, &claimedUntil, &deliveryAttempt, &fence,
			&completedAt, &retiredAt, &quarantinedAt,
			&envelopeProject, &envelopePrincipal, &envelopeItem, &envelopeExecution,
			&envelopePlan, &envelopeItemGeneration, &itemState, &currentItemRevision,
			&currentExecution, &goalState, &currentGoalPlan, &attemptCount, &consumedAt,
		); err != nil {
			return err
		}
		var refErr error
		action.Kind = application.ActionDeliverMailbox
		if action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
			return refErr
		}
		if action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
			return refErr
		}
		if action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
			return refErr
		}
		if planGeneration <= 0 || itemGeneration <= 0 || deliveryAttempt < 0 || fence < 0 ||
			attemptCount < 0 {
			return errors.New("sqlite.recovery_mailbox_outbox_generation_invalid")
		}
		action.PlanGeneration = goal.PlanGeneration(planGeneration)
		action.WorkItemGeneration = goal.Revision(itemGeneration)
		action.AvailableAt = time.Unix(0, availableAt).UTC()
		if err := validateAction(action); err != nil {
			return err
		}
		if itemValue != envelopeItem || executionValue != envelopeExecution ||
			planGeneration != envelopePlan || itemGeneration != envelopeItemGeneration ||
			currentItemRevision < itemGeneration || currentGoalPlan < planGeneration ||
			!validText(envelopeProject) || !validText(envelopePrincipal) {
			return errors.New("sqlite.recovery_mailbox_outbox_scope_invalid")
		}
		claimed := token.Valid || worker.Valid || claimedUntil.Valid
		if claimed != (token.Valid && worker.Valid && claimedUntil.Valid && deliveryAttempt > 0 && fence > 0) ||
			attemptCount != fence || deliveryAttempt != fence ||
			(claimed && worker.String != envelopePrincipal) || quarantinedAt.Valid {
			return errors.New("sqlite.recovery_mailbox_outbox_claim_invalid")
		}
		if attemptCount == 0 && (claimed || fence != 0 || deliveryAttempt != 0 || completedAt.Valid) {
			return errors.New("sqlite.recovery_mailbox_outbox_virgin_invalid")
		}
		if completedAt.Valid != consumedAt.Valid ||
			(completedAt.Valid && completedAt.Int64 != consumedAt.Int64) {
			return errors.New("sqlite.recovery_mailbox_outbox_consumption_invalid")
		}
		if !completedAt.Valid && !retiredAt.Valid {
			if goalState != string(goal.GoalStateRunning) ||
				itemState != string(goal.WorkItemStateRunning) ||
				!currentExecution.Valid || currentExecution.String != executionValue {
				return errors.New("sqlite.recovery_mailbox_outbox_active_state_invalid")
			}
		}
		_ = messageValue
	}
	return rows.Err()
}

func validateRecoveredMailboxTerminalCause(
	ctx context.Context,
	source queryer,
	record application.MailboxRecord,
) error {
	var executionState, failureCode string
	var finishedAt sql.NullInt64
	var itemState, goalState, currentExecution string
	err := source.QueryRowContext(ctx, `
SELECT execution.state, execution.failure_code, execution.finished_at,
       item.state, COALESCE(item.execution_ref, ''), goal.state
FROM executions execution
JOIN work_items item
  ON item.goal_ref = execution.goal_ref AND item.ref = execution.work_item_ref
JOIN goals goal ON goal.ref = execution.goal_ref
WHERE execution.ref = ?
  AND execution.goal_ref = ?
  AND execution.work_item_ref = ?`,
		record.Envelope.Recipient.ExecutionRef.String(), record.Envelope.GoalRef.String(),
		record.Envelope.ParentWorkItemRef.String(),
	).Scan(&executionState, &failureCode, &finishedAt, &itemState, &currentExecution, &goalState)
	if err != nil {
		return err
	}
	if record.Retirement != nil {
		retirement := record.Retirement
		controlledSuccess := false
		if executionState == string(application.ExecutionSucceeded) &&
			retirement.FailureCode == controlledMailboxCancellationCode {
			var controls int
			if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM controls
WHERE goal_ref = ? AND operation = 'cancel' AND requested_at <= ?
  AND (target = 'goal' OR (target = 'work_item' AND work_item_ref = ?))`,
				record.Envelope.GoalRef.String(), requiredTime(retirement.RetiredAt),
				record.Envelope.ParentWorkItemRef.String(),
			).Scan(&controls); err == nil {
				controlledSuccess = controls > 0
			}
		}
		if (!controlledSuccess && executionState != string(application.ExecutionFailed) &&
			executionState != string(application.ExecutionStopped) &&
			executionState != string(application.ExecutionCanceled)) || !finishedAt.Valid ||
			(!controlledSuccess && failureCode != retirement.FailureCode) ||
			finishedAt.Int64 != retirement.RetiredAt.UnixNano() {
			return errors.New("sqlite.recovery_mailbox_retirement_cause_invalid")
		}
		return nil
	}
	if record.Acknowledgement != nil {
		return nil
	}
	if executionState != string(application.ExecutionRunning) ||
		itemState != string(goal.WorkItemStateRunning) || currentExecution != record.Envelope.Recipient.ExecutionRef.String() ||
		goalState != string(goal.GoalStateRunning) {
		return errors.New("sqlite.recovery_mailbox_unresolved_recipient_not_live")
	}
	return nil
}

func validateRecoveryReceiptBindingsV13(ctx context.Context, transaction *sql.Tx) error {
	if err := validateRecoveryReceiptBindingsLegacy(ctx, transaction); err != nil {
		return err
	}
	var invalid int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM action_consumption_receipts r
JOIN outbox o ON o.ref = r.action_ref
WHERE (r.kind = 'deliver_mailbox') <> (r.mailbox_message_ref IS NOT NULL)
   OR r.mailbox_message_ref IS NOT o.mailbox_message_ref
   OR (r.kind = 'deliver_mailbox' AND (
        r.outcome <> 'completed' OR r.error_code <> '' OR r.worker_ref <> o.claimed_by
   ))`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_mailbox_receipt_binding_invalid")
	}
	return nil
}
