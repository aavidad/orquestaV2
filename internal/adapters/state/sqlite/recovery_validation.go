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
	if err := validateRecoveryVersion(ctx, transaction, current); err != nil {
		return "", "", invalid(err)
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

type recoveryValidator func(context.Context, *sql.Tx) error

func validateRecoveryVersion(ctx context.Context, tx *sql.Tx, version int) error {
	if version >= recoverySchemaV17 {
		// Classify broken ledgers at their owning schema boundary before the
		// hydrated read model rejects the same corruption more generically.
		governanceValidator := validateRecoveryV17Governance
		if version >= recoverySchemaV38RecoveryClaim {
			governanceValidator = validateRecoveryV28Governance
		}
		if version >= recoverySchemaV38StopNonApplication {
			governanceValidator = validateRecoveryV37Governance
		}
		validators := []recoveryValidator{
			validateRecoveryV10Identity,
			validateRecoveryV12Director,
			validateRecoveryV13Mailbox,
			validateRecoveryV14Controls,
			governanceValidator,
			validateRecoveryV16WorkspaceGit,
			validateRecoveryV17TestAttestor,
			validateRecoveryV18Reviews,
		}
		if version >= recoverySchemaV19 {
			validators = append(validators, validateRecoveryV19Council)
		}
		if version >= recoverySchemaV20 {
			validators = append(validators, validateRecoveryV20CommandAudit)
		}
		if version >= recoverySchemaV21 {
			validators = append(validators, validateRecoveryV21PostArtifactMailbox)
		}
		if version >= recoverySchemaV23Intake {
			validators = append(validators, validateRecoveryV23Intake)
		}
		if version >= recoverySchemaV23Dossier {
			validators = append(validators, validateRecoveryV23Dossiers)
		}
		if version >= recoverySchemaV23Confirmation {
			validators = append(validators, validateRecoveryV23DossierConfirmations)
		}
		if version >= recoverySchemaV23WizardGaps {
			validators = append(validators, validateRecoveryV23WizardGapsOutcomes)
		}
		if version >= recoverySchemaV23 {
			if version >= recoverySchemaV23WizardGapsSnapshot {
				validators = append(validators, validateRecoveryV23WizardGapsSnapshots)
			}
			validators = append(validators, validateRecoveryV23WizardGapsInputs)
		}
		if version >= recoverySchemaV38Capacity {
			validators = append(validators, validateRecoveryV38AgentPlacement)
			if version >= recoverySchemaV38Environment {
				validators = append(validators, validarRecuperacionPreservacionEntorno)
			}
		}
		if version >= recoverySchemaV38AttemptLease {
			validators = append(validators, validateRecoveryV27EffectAttemptClaimLease)
		}
		if version >= recoverySchemaV38RecoveryClaim {
			recoveryClaimValidator := validateRecoveryV28EffectRecoveryClaim
			if version >= recoverySchemaV38RecoveryRequeue {
				recoveryClaimValidator = validateRecoveryV30EffectRecoveryClaim
			}
			validators = append(validators, recoveryClaimValidator)
		}
		if version >= recoverySchemaV38MicroVMHostLaunch {
			validators = append(validators, validateRecoveryV38MicroVMHostLaunchAuthority)
		}
		if version >= recoverySchemaV38LaunchRuntimeDigests {
			validators = append(validators, validateRecoveryV39MicroVMHostLaunchRuntimeDigests)
		}
		if version >= recoverySchemaV38EnvironmentLifecycle {
			validators = append(validators, validarRecuperacionAgentEnvironmentLifecycles)
		}
		if version >= recoverySchemaV38StopNonApplication {
			validators = append(validators, validateRecoveryV37EffectNonApplication)
		}
		validators = append(validators, validateMigratedGoalRecords)
		for _, validate := range validators {
			if err := validate(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	}
	validators := []recoveryValidator{validateMigratedGoalRecords, validateRecoveryV10Identity}
	if version == recoverySchemaV09 {
		validators = []recoveryValidator{validateRecoveryV09GoalRecords}
	} else {
		if version >= recoverySchemaV12 {
			validators = append(validators, validateRecoveryV12Director)
		}
		if version >= recoverySchemaV13 {
			validators = append(validators, validateRecoveryV13Mailbox)
		}
		if version >= recoverySchemaV14 {
			validators = append(validators, validateRecoveryV14Controls)
		}
		if version >= recoverySchemaV15 {
			validators = append(validators, validateRecoveryV15Governance)
		}
		if version >= recoverySchemaV16 {
			validators = append(validators, validateRecoveryV16WorkspaceGit)
		}
	}
	for _, validate := range validators {
		if err := validate(ctx, tx); err != nil {
			return err
		}
	}
	return nil
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
	hasPurpose, err := sqliteTableHasColumn(ctx, transaction, "executions", "purpose")
	if err != nil {
		return err
	}
	executionPurpose, boundPurpose := "'work'", "'work'"
	if hasPurpose {
		executionPurpose, boundPurpose = "e.purpose", "bound.purpose"
	}
	hasRetiredActions, err := sqliteTableHasColumn(ctx, transaction, "outbox", "retired_at")
	if err != nil {
		return err
	}
	stopRetirementPredicate := ""
	if hasRetiredActions {
		stopRetirementPredicate = " AND stop.retired_at IS NULL"
	}
	hasControls, err := sqliteTableHasColumn(ctx, transaction, "controls", "operation")
	if err != nil {
		return err
	}
	stopCancelProof := "0"
	if hasControls {
		stopCancelProof = `(SELECT COUNT(*) FROM outbox stop
        JOIN controls cancel_control
         ON cancel_control.ref=stop.control_ref
         AND cancel_control.goal_ref=stop.goal_ref
        WHERE stop.kind='stop_agent' AND stop.goal_ref=o.goal_ref
         AND stop.work_item_ref=o.work_item_ref AND stop.execution_ref=o.execution_ref
         AND stop.quarantined_at IS NULL` + stopRetirementPredicate + `
         AND cancel_control.operation='cancel'
         AND cancel_control.status='requested'
         AND (cancel_control.target='goal'
          OR (cancel_control.target='work_item'
           AND cancel_control.work_item_ref=stop.work_item_ref))
         AND (stop.completed_at IS NULL OR EXISTS (
          SELECT 1 FROM action_consumption_receipts stop_receipt
          WHERE stop_receipt.action_ref=stop.ref
           AND stop_receipt.kind='stop_agent'
           AND stop_receipt.goal_ref=stop.goal_ref
           AND stop_receipt.work_item_ref=stop.work_item_ref
           AND stop_receipt.execution_ref=stop.execution_ref
           AND stop_receipt.outcome='completed'
           AND stop_receipt.error_code=''
           AND stop_receipt.consumed_at=stop.completed_at
         )))`
	}
	rows, err := transaction.QueryContext(ctx, fmt.Sprintf(`
SELECT o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
       o.completed_at, o.quarantined_at, wf.fence,
       e.plan_generation, e.state, e.attempt_no, %s,
       wi.revision, wi.state, wi.execution_ref,
       %s,
       %s, bound.state,
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
LEFT JOIN executions bound ON bound.goal_ref=wi.goal_ref AND bound.work_item_ref=wi.ref AND bound.ref=wi.execution_ref
LEFT JOIN work_item_fences wf ON wf.goal_ref = o.goal_ref AND wf.work_item_ref = o.work_item_ref
WHERE o.kind IN ('launch_agent', 'observe_agent')
ORDER BY o.ref`, executionPurpose, stopCancelProof, boundPurpose))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var action application.ActionRecord
		var kind, goalValue, itemValue, executionValue string
		var planGeneration, itemGeneration, availableAt, deliveryAttempt, fence int64
		var executionPlanGeneration, executionAttempt, currentItemRevision, currentGoalPlanGeneration int64
		var pendingStops int64
		var executionState, executionPurpose, currentItemState, currentGoalState string
		var boundPurpose, boundState sql.NullString
		var currentExecutionRef sql.NullString
		var currentFence sql.NullInt64
		var token, worker sql.NullString
		var claimedUntil, completedAt, quarantinedAt sql.NullInt64
		if err := rows.Scan(
			&action.Ref, &kind, &goalValue, &itemValue, &executionValue,
			&planGeneration, &itemGeneration, &availableAt, &token, &worker, &claimedUntil,
			&deliveryAttempt, &fence, &completedAt, &quarantinedAt, &currentFence,
			&executionPlanGeneration, &executionState, &executionAttempt, &executionPurpose,
			&currentItemRevision, &currentItemState, &currentExecutionRef,
			&pendingStops,
			&boundPurpose, &boundState,
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
			executionPurpose,
			pendingStops,
			boundPurpose,
			boundState,
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
			if err := validateClaimBase(claim); err != nil {
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
	executionPurpose string,
	pendingStops int64,
	boundPurpose sql.NullString,
	boundState sql.NullString,
) bool {
	if planGeneration > currentGoalPlanGeneration || currentGoalState != "running" {
		return false
	}
	exactOrCancelStop := itemGeneration == currentItemRevision ||
		pendingStops == 1 && itemGeneration <= currentItemRevision
	switch kind {
	case application.ActionLaunchAgent:
		if executionPurpose == string(application.ExecutionPurposePrimaryReview) ||
			executionPurpose == string(application.ExecutionPurposeAdversarialReview) ||
			executionPurpose == string(application.ExecutionPurposeCouncilProposer) ||
			executionPurpose == string(application.ExecutionPurposeCouncilCritic) ||
			executionPurpose == string(application.ExecutionPurposeCouncilArbiter) {
			return (executionState == "queued" || executionState == "dispatching") &&
				currentItemState == "running" &&
				currentExecutionRef.Valid && currentExecutionRef.String != executionRef && boundPurpose.String == "author" &&
				boundState.String == "awaiting_integration" && itemGeneration <= currentItemRevision
		}
		if executionState == "queued" {
			initial := currentItemState == "pending" && !currentExecutionRef.Valid
			replacement := currentItemState == "running" && currentExecutionRef.Valid &&
				currentExecutionRef.String == executionRef
			return itemGeneration <= currentItemRevision && (initial || replacement)
		}
		if executionState != "dispatching" || currentItemState != "running" ||
			!currentExecutionRef.Valid || currentExecutionRef.String != executionRef {
			return false
		}
		return itemGeneration <= currentItemRevision
	case application.ActionObserveAgent:
		if executionPurpose == string(application.ExecutionPurposePrimaryReview) ||
			executionPurpose == string(application.ExecutionPurposeAdversarialReview) ||
			executionPurpose == string(application.ExecutionPurposeCouncilProposer) ||
			executionPurpose == string(application.ExecutionPurposeCouncilCritic) ||
			executionPurpose == string(application.ExecutionPurposeCouncilArbiter) {
			return executionState == "running" && currentItemState == "running" && currentExecutionRef.Valid &&
				currentExecutionRef.String != executionRef && boundPurpose.String == "author" &&
				boundState.String == "awaiting_integration" &&
				exactOrCancelStop
		}
		return executionState == "running" && currentItemState == "running" &&
			currentExecutionRef.Valid && currentExecutionRef.String == executionRef &&
			exactOrCancelStop
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
