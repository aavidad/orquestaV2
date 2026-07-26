package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
)

type retiredActionRow struct {
	kind, goalRef, itemRef, executionRef, changeRef        string
	effectIntentRef                                        string
	planGeneration, itemGeneration, deliveryAttempt, fence int64
	governanceVersion                                      int64
	token, worker                                          sql.NullString
	leaseUntil, completedAt, retiredAt, quarantinedAt      sql.NullInt64
}

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
	row, governancePersisted, err := readRetiredAction(ctx, transaction, actionRef)
	if err != nil {
		return err
	}
	if row.completedAt.Valid {
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
	if row.retiredAt.Valid || row.quarantinedAt.Valid || row.kind == string(application.ActionDeliverMailbox) ||
		row.planGeneration <= 0 || row.itemGeneration <= 0 || row.deliveryAttempt < 0 || row.fence < 0 {
		return conflict(errors.New("sqlite.action_retirement_state_invalid"))
	}
	claimed := row.token.Valid || row.worker.Valid || row.leaseUntil.Valid
	if claimed != (row.token.Valid && row.worker.Valid && row.leaseUntil.Valid &&
		row.deliveryAttempt > 0 && row.fence > 0) {
		return conflict(errors.New("sqlite.action_retirement_claim_invalid"))
	}
	started, err := retiredAttestationHasStartedEffect(ctx, transaction, actionRef, row)
	if err != nil {
		return err
	}
	if started {
		row, err = takeOverExpiredAttestationRetirement(
			ctx, transaction, actionRef, authorityRef, at, row,
		)
		if err != nil {
			return err
		}
		const quarantineCode = "application.effect_unknown_applied"
		result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET completed_at = ?, quarantined_at = ?, last_error_code = ?
WHERE ref = ? AND effect_intent_ref = ? AND claim_token = ? AND claimed_by = ?
  AND claimed_until = ? AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
  AND EXISTS (
      SELECT 1 FROM effect_attempts
      WHERE action_ref = outbox.ref AND intent_ref = outbox.effect_intent_ref
        AND action_fence <= outbox.fence
  )`,
			requiredTime(at), requiredTime(at), quarantineCode,
			actionRef, row.effectIntentRef, row.token.String, row.worker.String,
			row.leaseUntil.Int64, row.deliveryAttempt, row.fence,
		)
		if err != nil {
			return mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return err
		}
		return insertRetirementReceipt(
			ctx, transaction, actionRef, row, governancePersisted,
			application.ActionConsumedQuarantined, quarantineCode, at,
		)
	}
	if !claimed {
		if row.deliveryAttempt >= int64(maxSQLiteInteger) {
			return invalid(errors.New("sqlite.action_retirement_attempt_overflow"))
		}
		row.deliveryAttempt++
		err = transaction.QueryRowContext(ctx, `
INSERT INTO work_item_fences(goal_ref, work_item_ref, fence)
VALUES (?, ?, 1)
ON CONFLICT(goal_ref, work_item_ref)
DO UPDATE SET fence = work_item_fences.fence + 1
RETURNING fence`, row.goalRef, row.itemRef).Scan(&row.fence)
		if err != nil {
			return mapDatabaseError(err)
		}
		row.token = sql.NullString{String: "retire:" + authorityRef + ":" + actionRef, Valid: true}
		row.worker = sql.NullString{String: "system:" + authorityRef, Valid: true}
		row.leaseUntil = sql.NullInt64{Int64: requiredTime(at.Add(time.Nanosecond)), Valid: true}
	}
	const retirementCode = "application.action_retired"
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, delivery_attempt = ?, fence = ?,
    completed_at = ?, last_error_code = ?
WHERE ref = ? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		row.token.String, row.worker.String, row.leaseUntil.Int64, row.deliveryAttempt, row.fence,
		requiredTime(at), retirementCode, actionRef,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	return insertRetirementReceipt(
		ctx, transaction, actionRef, row, governancePersisted,
		application.ActionConsumedCompleted, retirementCode, at,
	)
}

func retiredAttestationHasStartedEffect(
	ctx context.Context,
	transaction *sql.Tx,
	actionRef string,
	row retiredActionRow,
) (bool, error) {
	if row.kind != string(application.ActionAttestTest) {
		return false, nil
	}
	if row.effectIntentRef == "" {
		return false, conflict(errors.New("sqlite.action_retirement_attestation_intent_missing"))
	}
	var total, causal int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*),
       COALESCE(SUM(CASE WHEN intent_ref = ? AND action_fence <= ? THEN 1 ELSE 0 END), 0)
FROM effect_attempts
WHERE action_ref = ?`,
		row.effectIntentRef, row.fence, actionRef,
	).Scan(&total, &causal); err != nil {
		return false, mapDatabaseError(err)
	}
	if total == 0 {
		return false, nil
	}
	if total != 1 || causal != 1 {
		return false, conflict(errors.New("sqlite.action_retirement_attestation_attempt_ambiguous"))
	}
	return true, nil
}

func takeOverExpiredAttestationRetirement(
	ctx context.Context,
	transaction *sql.Tx,
	actionRef string,
	authorityRef string,
	at time.Time,
	row retiredActionRow,
) (retiredActionRow, error) {
	if requiredTime(at) < row.leaseUntil.Int64 {
		return row, nil
	}
	if row.deliveryAttempt >= int64(maxSQLiteInteger) {
		return row, invalid(errors.New("sqlite.action_retirement_attempt_overflow"))
	}
	leaseUntil := at.Add(time.Nanosecond)
	if !leaseUntil.After(at) || requiredTime(leaseUntil) <= requiredTime(at) {
		return row, invalid(errors.New("sqlite.action_retirement_lease_overflow"))
	}
	token := "retire:" + authorityRef + ":" + actionRef
	worker := "system:" + authorityRef
	nextAttempt := row.deliveryAttempt + 1
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, delivery_attempt = ?
WHERE ref = ? AND effect_intent_ref = ?
  AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND delivery_attempt = ? AND fence = ? AND claimed_until <= ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
  AND EXISTS (
      SELECT 1 FROM work_item_fences current
      WHERE current.goal_ref = outbox.goal_ref
        AND current.work_item_ref = outbox.work_item_ref
        AND current.fence = outbox.fence
  )
  AND (SELECT COUNT(*) FROM effect_attempts WHERE action_ref = outbox.ref) = 1
  AND EXISTS (
      SELECT 1 FROM effect_attempts
      WHERE action_ref = outbox.ref AND intent_ref = outbox.effect_intent_ref
        AND action_fence <= outbox.fence
  )`,
		token, worker, requiredTime(leaseUntil), nextAttempt,
		actionRef, row.effectIntentRef,
		row.token.String, row.worker.String, row.leaseUntil.Int64,
		row.deliveryAttempt, row.fence, requiredTime(at),
	)
	if err != nil {
		return row, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return row, err
	}
	row.token = sql.NullString{String: token, Valid: true}
	row.worker = sql.NullString{String: worker, Valid: true}
	row.leaseUntil = sql.NullInt64{Int64: requiredTime(leaseUntil), Valid: true}
	row.deliveryAttempt = nextAttempt
	return row, nil
}

func readRetiredAction(
	ctx context.Context, transaction *sql.Tx, actionRef string,
) (retiredActionRow, bool, error) {
	var row retiredActionRow
	governed, err := sqliteTableHasColumn(ctx, transaction, "outbox", "governance_version")
	if err != nil {
		return row, false, mapDatabaseError(err)
	}
	query := `
SELECT kind, goal_ref, work_item_ref, execution_ref,
       change_ref, '', plan_generation, work_item_generation, delivery_attempt, fence,
       claim_token, claimed_by, claimed_until, completed_at, retired_at, quarantined_at
FROM outbox WHERE ref = ?`
	destinations := []any{&row.kind, &row.goalRef, &row.itemRef, &row.executionRef,
		&row.changeRef, &row.effectIntentRef,
		&row.planGeneration, &row.itemGeneration, &row.deliveryAttempt, &row.fence,
		&row.token, &row.worker, &row.leaseUntil, &row.completedAt, &row.retiredAt, &row.quarantinedAt}
	if governed {
		query = `
SELECT kind, goal_ref, work_item_ref, execution_ref,
       change_ref, COALESCE(effect_intent_ref, ''),
       plan_generation, work_item_generation, delivery_attempt, fence,
       claim_token, claimed_by, claimed_until, completed_at, retired_at, quarantined_at,
       governance_version
FROM outbox WHERE ref = ?`
		destinations = append(destinations, &row.governanceVersion)
	}
	err = transaction.QueryRowContext(ctx, query, actionRef).Scan(destinations...)
	if errors.Is(err, sql.ErrNoRows) {
		return row, governed, conflict(errors.New("sqlite.action_retirement_missing"))
	}
	return row, governed, mapDatabaseError(err)
}

func insertRetirementReceipt(
	ctx context.Context, transaction *sql.Tx, actionRef string, row retiredActionRow,
	governed bool, outcome application.ActionConsumptionOutcome, code string, at time.Time,
) error {
	if governed {
		_, err := transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref, change_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			actionRef, row.governanceVersion, row.kind, row.goalRef, row.itemRef, row.executionRef,
			row.changeRef, row.planGeneration, row.itemGeneration, row.fence, row.deliveryAttempt,
			row.token.String, row.worker.String, string(outcome), code, requiredTime(at),
		)
		if err != nil {
			return mapDatabaseError(fmt.Errorf("sqlite.action_retirement_receipt: %w", err))
		}
		return nil
	}
	_, err := transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref, change_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		actionRef, row.kind, row.goalRef, row.itemRef, row.executionRef,
		row.changeRef, row.planGeneration, row.itemGeneration, row.fence, row.deliveryAttempt,
		row.token.String, row.worker.String, string(outcome), code, requiredTime(at),
	)
	if err != nil {
		return mapDatabaseError(fmt.Errorf("sqlite.action_retirement_receipt: %w", err))
	}
	return nil
}
