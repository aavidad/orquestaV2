package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func validateRecoveryDatabase(ctx context.Context, database *sql.DB) (string, string, error) {
	transaction, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", "", mapDatabaseError(err)
	}
	defer transaction.Rollback()
	migrations, err := loadMigrations()
	if err != nil {
		return "", "", invalid(err)
	}
	var current int
	if err := transaction.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return "", "", mapDatabaseError(err)
	}
	prefix, err := recoveryMigrationPrefix(migrations, current)
	if err != nil {
		return "", "", invalid(err)
	}
	if err := verifyAppliedMigrations(ctx, transaction, prefix, current); err != nil {
		return "", "", err
	}
	if err := verifyForeignKeys(ctx, transaction); err != nil {
		return "", "", err
	}
	actualSchema, err := schemaInventoryDigest(ctx, transaction)
	if err != nil {
		return "", "", invalid(err)
	}
	expectedSchema, err := canonicalSchemaInventoryDigest(current)
	if err != nil || actualSchema != expectedSchema {
		return "", "", invalid(errors.New("sqlite.recovery_schema_inventory_invalid"))
	}
	var integrity string
	if err := transaction.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
		return "", "", invalid(fmt.Errorf("sqlite.integrity_check_failed: %w", err))
	}
	if integrity != "ok" {
		return "", "", invalid(fmt.Errorf("sqlite.integrity_check_failed:%s", integrity))
	}
	switch current {
	case recoverySchemaV09:
		if err := validateRecoveryV09GoalRecords(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
	case recoverySchemaV10:
		if err := validateMigratedGoalRecords(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV10Identity(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
	case recoverySchemaV12:
		if err := validateMigratedGoalRecords(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV10Identity(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV12Director(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
	case recoverySchemaV13:
		if err := validateMigratedGoalRecords(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV10Identity(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV12Director(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
		if err := validateRecoveryV13Mailbox(ctx, transaction); err != nil {
			return "", "", invalid(err)
		}
	}
	if err := validateRecoveryEvents(ctx, transaction); err != nil {
		return "", "", invalid(err)
	}
	if err := validateRecoveryOutbox(ctx, transaction); err != nil {
		return "", "", invalid(err)
	}
	if err := validateRecoveryReceiptBindings(ctx, transaction); err != nil {
		return "", "", invalid(err)
	}
	schemaRef := migrationSchemaRef(prefix)
	logicalDigest, err := logicalStateDigest(ctx, transaction)
	if err != nil {
		return "", "", invalid(err)
	}
	if err := transaction.Commit(); err != nil {
		return "", "", mapDatabaseError(err)
	}
	return schemaRef, logicalDigest, nil
}

func schemaInventoryDigest(ctx context.Context, source queryer) (string, error) {
	hash := sha256.New()
	rows, err := source.QueryContext(ctx, `
SELECT type, name, tbl_name, COALESCE(sql, '')
FROM sqlite_schema
WHERE name <> 'sqlite_schema'
ORDER BY type, name, tbl_name, sql`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var objectType, name, tableName, statement string
		if err := rows.Scan(&objectType, &name, &tableName, &statement); err != nil {
			return "", err
		}
		writeDigestField(hash, objectType)
		writeDigestField(hash, name)
		writeDigestField(hash, tableName)
		writeDigestField(hash, statement)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func validateRecoveryEvents(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at
FROM events ORDER BY ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var event application.EventRecord
		var goalValue string
		var workItemValue, executionValue sql.NullString
		var occurredAt int64
		if err := rows.Scan(&event.Ref, &event.Kind, &goalValue, &workItemValue, &executionValue, &occurredAt); err != nil {
			return err
		}
		if event.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
			return err
		}
		if workItemValue.Valid {
			if event.WorkItemRef, err = goal.NewWorkItemRef(workItemValue.String); err != nil {
				return err
			}
		}
		if executionValue.Valid {
			if event.ExecutionRef, err = goal.NewExecutionRef(executionValue.String); err != nil {
				return err
			}
		}
		event.OccurredAt = time.Unix(0, occurredAt).UTC()
		if err := validateEvent(event); err != nil {
			return err
		}
	}
	return rows.Err()
}

func validateRecoveryOutbox(ctx context.Context, transaction *sql.Tx) error {
	hasMailbox, err := sqliteTableHasColumn(ctx, transaction, "outbox", "mailbox_message_ref")
	if err != nil {
		return err
	}
	if hasMailbox {
		return validateRecoveryOutboxV13(ctx, transaction)
	}
	return validateRecoveryOutboxLegacy(ctx, transaction)
}

func validateRecoveryOutboxLegacy(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
       o.completed_at, o.quarantined_at, wf.fence,
       e.plan_generation, e.state, e.attempt_no,
       wi.revision, wi.state, wi.execution_ref,
       g.plan_generation, g.state
FROM outbox o
JOIN executions e
  ON e.goal_ref = o.goal_ref
 AND e.work_item_ref = o.work_item_ref
 AND e.ref = o.execution_ref
JOIN work_items wi
  ON wi.goal_ref = o.goal_ref
 AND wi.ref = o.work_item_ref
JOIN goals g ON g.ref = o.goal_ref
LEFT JOIN work_item_fences wf ON wf.goal_ref = o.goal_ref AND wf.work_item_ref = o.work_item_ref
WHERE o.kind IN ('launch_agent', 'observe_agent')
ORDER BY o.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var action application.ActionRecord
		var kind, goalValue, itemValue, executionValue string
		var planGeneration, itemGeneration, availableAt, deliveryAttempt, fence int64
		var executionPlanGeneration, executionAttempt, currentItemRevision, currentGoalPlanGeneration int64
		var executionState, currentItemState, currentGoalState string
		var currentExecutionRef sql.NullString
		var currentFence sql.NullInt64
		var token, worker sql.NullString
		var claimedUntil, completedAt, quarantinedAt sql.NullInt64
		if err := rows.Scan(
			&action.Ref, &kind, &goalValue, &itemValue, &executionValue,
			&planGeneration, &itemGeneration, &availableAt, &token, &worker, &claimedUntil,
			&deliveryAttempt, &fence, &completedAt, &quarantinedAt, &currentFence,
			&executionPlanGeneration, &executionState, &executionAttempt,
			&currentItemRevision, &currentItemState, &currentExecutionRef,
			&currentGoalPlanGeneration, &currentGoalState,
		); err != nil {
			return err
		}
		var refErr error
		action.Kind = application.ActionKind(kind)
		if action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
			return refErr
		}
		if action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
			return refErr
		}
		if action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
			return refErr
		}
		if planGeneration <= 0 || itemGeneration <= 0 || deliveryAttempt < 0 || fence < 0 {
			return fmt.Errorf(
				"sqlite.recovery_outbox_generation_invalid:%s:plan=%d:item=%d:delivery=%d:fence=%d:current=%v",
				action.Ref, planGeneration, itemGeneration, deliveryAttempt, fence, currentFence,
			)
		}
		action.PlanGeneration = goal.PlanGeneration(planGeneration)
		action.WorkItemGeneration = goal.Revision(itemGeneration)
		action.AvailableAt = time.Unix(0, availableAt).UTC()
		if err := validateAction(action); err != nil {
			return err
		}
		if planGeneration != executionPlanGeneration || itemGeneration > currentItemRevision {
			return fmt.Errorf(
				"sqlite.recovery_outbox_scope_invalid:%s:plan=%d:execution_plan=%d:item=%d:current_item=%d",
				action.Ref, planGeneration, executionPlanGeneration, itemGeneration, currentItemRevision,
			)
		}
		if !completedAt.Valid && !activeRecoveryActionState(
			action.Kind,
			executionValue,
			planGeneration,
			itemGeneration,
			executionState,
			executionAttempt,
			currentItemRevision,
			currentItemState,
			currentExecutionRef,
			currentGoalPlanGeneration,
			currentGoalState,
		) {
			return fmt.Errorf("sqlite.recovery_outbox_active_state_invalid:%s", action.Ref)
		}
		claimed := token.Valid || worker.Valid || claimedUntil.Valid
		if claimed != (token.Valid && worker.Valid && claimedUntil.Valid && deliveryAttempt > 0 && fence > 0) {
			return errors.New("sqlite.recovery_outbox_claim_invalid")
		}
		if !currentFence.Valid {
			if claimed || deliveryAttempt != 0 || fence != 0 || completedAt.Valid || quarantinedAt.Valid {
				return errors.New("sqlite.recovery_outbox_fence_missing")
			}
			continue
		}
		if currentFence.Int64 < fence {
			return errors.New("sqlite.recovery_outbox_fence_regressed")
		}
		if claimed {
			claim := application.ActionClaim{
				Action: action, Token: token.String, WorkerRef: worker.String,
				DeliveryAttempt: uint64(deliveryAttempt), Fence: uint64(fence),
				LeaseUntil: time.Unix(0, claimedUntil.Int64).UTC(),
			}
			if err := validateClaim(claim); err != nil {
				return err
			}
			if !completedAt.Valid && !quarantinedAt.Valid && currentFence.Int64 != fence {
				return errors.New("sqlite.recovery_outbox_claim_fenced")
			}
		}
		if quarantinedAt.Valid && !completedAt.Valid {
			return errors.New("sqlite.recovery_outbox_terminal_invalid")
		}
	}
	return rows.Err()
}

func activeRecoveryActionState(
	kind application.ActionKind,
	executionRef string,
	planGeneration int64,
	itemGeneration int64,
	executionState string,
	executionAttempt int64,
	currentItemRevision int64,
	currentItemState string,
	currentExecutionRef sql.NullString,
	currentGoalPlanGeneration int64,
	currentGoalState string,
) bool {
	if planGeneration > currentGoalPlanGeneration || currentGoalState != "running" {
		return false
	}
	switch kind {
	case application.ActionLaunchAgent:
		if executionState == "queued" {
			return currentItemState == "pending" && itemGeneration == currentItemRevision
		}
		if executionState != "dispatching" || currentItemState != "running" ||
			!currentExecutionRef.Valid || currentExecutionRef.String != executionRef {
			return false
		}
		return itemGeneration == currentItemRevision ||
			(executionAttempt == 1 && itemGeneration+1 == currentItemRevision)
	case application.ActionObserveAgent:
		return executionState == "running" && currentItemState == "running" &&
			currentExecutionRef.Valid && currentExecutionRef.String == executionRef &&
			itemGeneration == currentItemRevision
	default:
		return false
	}
}

func validateRecoveryReceiptBindings(ctx context.Context, transaction *sql.Tx) error {
	hasMailbox, err := sqliteTableHasColumn(ctx, transaction, "action_consumption_receipts", "mailbox_message_ref")
	if err != nil {
		return err
	}
	if hasMailbox {
		return validateRecoveryReceiptBindingsV13(ctx, transaction)
	}
	return validateRecoveryReceiptBindingsLegacy(ctx, transaction)
}

func validateRecoveryReceiptBindingsLegacy(ctx context.Context, transaction *sql.Tx) error {
	var invalidBindings int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM outbox o
LEFT JOIN action_consumption_receipts r ON r.action_ref = o.ref
WHERE
    (o.completed_at IS NULL AND r.action_ref IS NOT NULL)
 OR (o.completed_at IS NOT NULL AND (o.claim_token IS NULL OR r.action_ref IS NULL))
 OR (r.action_ref IS NOT NULL AND NOT (
        r.kind = o.kind
    AND r.goal_ref = o.goal_ref
    AND r.work_item_ref = o.work_item_ref
    AND r.execution_ref = o.execution_ref
    AND r.plan_generation = o.plan_generation
    AND r.work_item_generation = o.work_item_generation
    AND r.fence = o.fence
    AND r.delivery_attempt = o.delivery_attempt
    AND r.claim_token = o.claim_token
    AND r.worker_ref = o.claimed_by
    AND r.error_code = o.last_error_code
    AND r.consumed_at = o.completed_at
    AND (
          (r.outcome = 'completed' AND o.quarantined_at IS NULL)
       OR (r.outcome = 'quarantined' AND o.quarantined_at = o.completed_at)
    )
 ))`).Scan(&invalidBindings); err != nil {
		return err
	}
	if invalidBindings != 0 {
		return errors.New("sqlite.recovery_receipt_binding_invalid")
	}
	return nil
}

func sqliteTableHasColumn(ctx context.Context, source queryer, table, column string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM pragma_table_xinfo(" + quoteSQLiteIdentifier(table) + ") WHERE name = ?"
	if err := source.QueryRowContext(ctx, query, column).Scan(&count); err != nil {
		return false, err
	}
	return count == 1, nil
}

// logicalStateDigest hashes schema plus every persisted value as a canonical
// unordered row multiset per table. It detects semantic tampering independently
// of SQLite page layout and is recomputed by verify and restore.
func logicalStateDigest(ctx context.Context, transaction *sql.Tx) (string, error) {
	hash := sha256.New()
	rows, err := transaction.QueryContext(ctx, `
SELECT type, name, tbl_name, COALESCE(sql, '')
FROM sqlite_schema
WHERE name <> 'sqlite_schema'
ORDER BY type, name, tbl_name, sql`)
	if err != nil {
		return "", err
	}
	var tables []string
	for rows.Next() {
		var objectType, name, tableName, statement string
		if err := rows.Scan(&objectType, &name, &tableName, &statement); err != nil {
			_ = rows.Close()
			return "", err
		}
		writeDigestField(hash, objectType)
		writeDigestField(hash, name)
		writeDigestField(hash, tableName)
		writeDigestField(hash, statement)
		if objectType == "table" && name != "sqlite_schema" {
			tables = append(tables, name)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return "", err
	}
	if err := rows.Close(); err != nil {
		return "", err
	}
	for _, table := range tables {
		columns, err := recoveryTableColumns(ctx, transaction, table)
		if err != nil {
			return "", err
		}
		writeDigestField(hash, "table:"+table)
		for _, column := range columns {
			writeDigestField(hash, column)
		}
		queryColumns := make([]string, len(columns))
		for index, column := range columns {
			queryColumns[index] = quoteSQLiteIdentifier(column)
		}
		query := "SELECT " + strings.Join(queryColumns, ",") + " FROM " + quoteSQLiteIdentifier(table) +
			" ORDER BY " + strings.Join(queryColumns, ",")
		dataRows, err := transaction.QueryContext(ctx, query)
		if err != nil {
			return "", err
		}
		for dataRows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := dataRows.Scan(destinations...); err != nil {
				_ = dataRows.Close()
				return "", err
			}
			var encoded strings.Builder
			for _, value := range values {
				encodeSQLiteValue(&encoded, value)
			}
			writeDigestField(hash, encoded.String())
		}
		if err := dataRows.Err(); err != nil {
			_ = dataRows.Close()
			return "", err
		}
		if err := dataRows.Close(); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func recoveryTableColumns(ctx context.Context, transaction *sql.Tx, table string) ([]string, error) {
	rows, err := transaction.QueryContext(ctx, "PRAGMA table_xinfo("+quoteSQLiteIdentifier(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var cid, notNull, primaryKey, hidden int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey, &hidden); err != nil {
			return nil, err
		}
		if hidden == 0 || hidden == 2 || hidden == 3 {
			columns = append(columns, name)
		}
	}
	return columns, rows.Err()
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func encodeSQLiteValue(builder *strings.Builder, value any) {
	switch typed := value.(type) {
	case nil:
		builder.WriteString("N;")
	case int64:
		fmt.Fprintf(builder, "I%d;", typed)
	case float64:
		builder.WriteString("F" + strconv.FormatFloat(typed, 'x', -1, 64) + ";")
	case bool:
		builder.WriteString("B" + strconv.FormatBool(typed) + ";")
	case []byte:
		fmt.Fprintf(builder, "X%d:%s;", len(typed), hex.EncodeToString(typed))
	case string:
		fmt.Fprintf(builder, "T%d:%s;", len(typed), typed)
	default:
		fmt.Fprintf(builder, "U%T:%v;", value, value)
	}
}

func writeDigestField(writer io.Writer, value string) {
	fmt.Fprintf(writer, "%d:%s\n", len(value), value)
}
