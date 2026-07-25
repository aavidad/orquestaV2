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
	return insertRetirementReceipt(ctx, transaction, actionRef, row, governancePersisted, retirementCode, at)
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
       change_ref, plan_generation, work_item_generation, delivery_attempt, fence,
       claim_token, claimed_by, claimed_until, completed_at, retired_at, quarantined_at
FROM outbox WHERE ref = ?`
	destinations := []any{&row.kind, &row.goalRef, &row.itemRef, &row.executionRef,
		&row.changeRef, &row.planGeneration, &row.itemGeneration, &row.deliveryAttempt, &row.fence,
		&row.token, &row.worker, &row.leaseUntil, &row.completedAt, &row.retiredAt, &row.quarantinedAt}
	if governed {
		query = `
SELECT kind, goal_ref, work_item_ref, execution_ref,
       change_ref, plan_generation, work_item_generation, delivery_attempt, fence,
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
	governed bool, retirementCode string, at time.Time,
) error {
	if governed {
		_, err := transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref, change_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed', ?, ?)`,
			actionRef, row.governanceVersion, row.kind, row.goalRef, row.itemRef, row.executionRef,
			row.changeRef, row.planGeneration, row.itemGeneration, row.fence, row.deliveryAttempt,
			row.token.String, row.worker.String, retirementCode, requiredTime(at),
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed', ?, ?)`,
		actionRef, row.kind, row.goalRef, row.itemRef, row.executionRef,
		row.changeRef, row.planGeneration, row.itemGeneration, row.fence, row.deliveryAttempt,
		row.token.String, row.worker.String, retirementCode, requiredTime(at),
	)
	if err != nil {
		return mapDatabaseError(fmt.Errorf("sqlite.action_retirement_receipt: %w", err))
	}
	return nil
}
