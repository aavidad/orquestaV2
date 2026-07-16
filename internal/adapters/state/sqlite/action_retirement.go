package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
)

// consumeRetiredAction records a deterministic, non-provider consumption for
// an action neutralized by a control or Director decision. It never deletes
// outbox history and preserves an already-issued claim identity.
func consumeRetiredAction(
	ctx context.Context,
	transaction *sql.Tx,
	actionRef string,
	authorityRef string,
	at time.Time,
) error {
	if !validText(actionRef) || !validText(authorityRef) || at.IsZero() {
		return invalid(errors.New("sqlite.action_retirement_invalid"))
	}
	var kind, goalRef, itemRef, executionRef string
	var planGeneration, itemGeneration, deliveryAttempt, fence int64
	var token, worker sql.NullString
	var leaseUntil, completedAt, retiredAt, quarantinedAt sql.NullInt64
	err := transaction.QueryRowContext(ctx, `
SELECT kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, delivery_attempt, fence,
       claim_token, claimed_by, claimed_until, completed_at, retired_at, quarantined_at
FROM outbox WHERE ref = ?`, actionRef).Scan(
		&kind, &goalRef, &itemRef, &executionRef,
		&planGeneration, &itemGeneration, &deliveryAttempt, &fence,
		&token, &worker, &leaseUntil, &completedAt, &retiredAt, &quarantinedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict(errors.New("sqlite.action_retirement_missing"))
	}
	if err != nil {
		return mapDatabaseError(err)
	}
	if completedAt.Valid {
		var receipt int
		if err := transaction.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref = ?`, actionRef,
		).Scan(&receipt); err != nil {
			return mapDatabaseError(err)
		}
		if receipt != 1 {
			return conflict(errors.New("sqlite.action_retirement_receipt_missing"))
		}
		return nil
	}
	if retiredAt.Valid || quarantinedAt.Valid || kind == string(application.ActionDeliverMailbox) ||
		planGeneration <= 0 || itemGeneration <= 0 || deliveryAttempt < 0 || fence < 0 {
		return conflict(errors.New("sqlite.action_retirement_state_invalid"))
	}
	claimed := token.Valid || worker.Valid || leaseUntil.Valid
	if claimed != (token.Valid && worker.Valid && leaseUntil.Valid && deliveryAttempt > 0 && fence > 0) {
		return conflict(errors.New("sqlite.action_retirement_claim_invalid"))
	}
	if !claimed {
		if deliveryAttempt >= int64(maxSQLiteInteger) {
			return invalid(errors.New("sqlite.action_retirement_attempt_overflow"))
		}
		deliveryAttempt++
		err = transaction.QueryRowContext(ctx, `
INSERT INTO work_item_fences(goal_ref, work_item_ref, fence)
VALUES (?, ?, 1)
ON CONFLICT(goal_ref, work_item_ref)
DO UPDATE SET fence = work_item_fences.fence + 1
RETURNING fence`, goalRef, itemRef).Scan(&fence)
		if err != nil {
			return mapDatabaseError(err)
		}
		token = sql.NullString{String: "retire:" + authorityRef + ":" + actionRef, Valid: true}
		worker = sql.NullString{String: "system:" + authorityRef, Valid: true}
		leaseUntil = sql.NullInt64{Int64: requiredTime(at), Valid: true}
	}
	const retirementCode = "application.action_retired"
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, delivery_attempt = ?, fence = ?,
    completed_at = ?, last_error_code = ?
WHERE ref = ? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		token.String, worker.String, leaseUntil.Int64, deliveryAttempt, fence,
		requiredTime(at), retirementCode, actionRef,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed', ?, ?)`,
		actionRef, kind, goalRef, itemRef, executionRef, planGeneration, itemGeneration,
		fence, deliveryAttempt, token.String, worker.String, retirementCode, requiredTime(at),
	)
	if err != nil {
		return mapDatabaseError(fmt.Errorf("sqlite.action_retirement_receipt: %w", err))
	}
	return nil
}
