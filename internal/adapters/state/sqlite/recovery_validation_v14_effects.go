package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func validateRecoveryV14StopEffects(
	ctx context.Context,
	transaction *sql.Tx,
	controls map[string]application.ControlRecord,
) (map[string]recoveryControlEffects, error) {
	counts := make(map[string]recoveryControlEffects, len(controls))
	for ref := range controls {
		counts[ref] = recoveryControlEffects{}
	}
	if err := countRecoveryV14StopActions(ctx, transaction, counts); err != nil {
		return nil, err
	}
	if err := countRecoveryV14StopEffectReceipts(ctx, transaction, controls, counts); err != nil {
		return nil, err
	}
	if err := validateRecoveryV14StopEffectReceiptUniqueness(ctx, transaction); err != nil {
		return nil, err
	}
	return counts, nil
}

func countRecoveryV14StopActions(
	ctx context.Context,
	transaction *sql.Tx,
	counts map[string]recoveryControlEffects,
) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT control_ref, COUNT(*) FROM outbox WHERE kind = 'stop_agent' GROUP BY control_ref`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var ref string
		var count int
		if err := rows.Scan(&ref, &count); err != nil {
			_ = rows.Close()
			return err
		}
		current, found := counts[ref]
		if !found {
			_ = rows.Close()
			return errors.New("sqlite.recovery_stop_action_control_missing")
		}
		current.actions = count
		counts[ref] = current
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	return rows.Close()
}

func countRecoveryV14StopEffectReceipts(
	ctx context.Context,
	transaction *sql.Tx,
	controls map[string]application.ControlRecord,
	counts map[string]recoveryControlEffects,
) error {
	receiptColumn, statusColumn, timeColumn, effectJoin, err := recoveryV14EffectProjection(ctx, transaction)
	if err != nil {
		return err
	}
	rows, err := transaction.QueryContext(ctx, `
SELECT action.control_ref, receipt.action_ref, `+receiptColumn+`,
       `+statusColumn+`, `+timeColumn+`,
       execution.state, execution.finished_at, execution.failure_code
FROM action_consumption_receipts receipt
JOIN outbox action ON action.ref = receipt.action_ref
JOIN executions execution ON execution.ref = receipt.execution_ref
`+effectJoin+`
WHERE receipt.kind = 'stop_agent' AND receipt.effect_receipt_ref IS NOT NULL
ORDER BY receipt.action_ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := countRecoveryV14StopEffectReceipt(rows, controls, counts); err != nil {
			return err
		}
	}
	return rows.Err()
}

func recoveryV14EffectProjection(ctx context.Context, source queryer) (string, string, string, string, error) {
	legacy, err := sqliteTableHasColumn(ctx, source, "action_consumption_receipts", "legacy_effect_status")
	if err != nil {
		return "", "", "", "", err
	}
	if legacy {
		return "COALESCE(effect.external_ref,receipt.effect_receipt_ref)",
			"COALESCE(receipt.legacy_effect_status,effect.status)",
			"COALESCE(receipt.legacy_effect_confirmed_at,effect.confirmed_at)",
			"LEFT JOIN effect_receipts effect ON effect.ref=receipt.effect_receipt_ref", nil
	}
	return "receipt.effect_receipt_ref", "receipt.effect_status", "receipt.effect_confirmed_at", "", nil
}

func countRecoveryV14StopEffectReceipt(
	rows *sql.Rows,
	controls map[string]application.ControlRecord,
	counts map[string]recoveryControlEffects,
) error {
	var controlRef, actionRef, receiptRef, status, executionState, failureCode string
	var confirmedAt int64
	var finishedAt sql.NullInt64
	if err := rows.Scan(
		&controlRef, &actionRef, &receiptRef, &status, &confirmedAt,
		&executionState, &finishedAt, &failureCode,
	); err != nil {
		return err
	}
	control, found := controls[controlRef]
	if !found || !validText(receiptRef) || confirmedAt < requiredTime(control.RequestedAt) || actionRef == "" {
		return fmt.Errorf("sqlite.recovery_stop_effect_binding_invalid:%s", actionRef)
	}
	current := counts[controlRef]
	current.effects++
	confirmed := time.Unix(0, confirmedAt).UTC()
	if control.Operation == application.ControlStop &&
		(control.Status != application.ControlConfirmed || control.ReceiptRef != receiptRef ||
			!control.ConfirmedAt.Equal(confirmed)) {
		return fmt.Errorf("sqlite.recovery_stop_effect_control_invalid:%s", actionRef)
	}
	if control.Operation == application.ControlCancel && control.Target == application.ControlTargetWorkItem &&
		(status == string(ports.AgentStopped) || status == string(ports.AgentStopAlreadyStopped)) {
		if control.Status == application.ControlConfirmed &&
			control.ReceiptRef == receiptRef && control.ConfirmedAt.Equal(confirmed) {
			current.finalEffects++
		}
	}
	counts[controlRef] = current
	expectedFailureCode := "application.execution_stopped"
	if application.IsReviewCleanupControl(control) {
		expectedFailureCode = "review.round_aborted"
	} else if application.IsCouncilCleanupControl(control) {
		expectedFailureCode = "council.round_aborted"
	}
	if (status == string(ports.AgentStopped) || status == string(ports.AgentStopAlreadyStopped)) &&
		(executionState != string(application.ExecutionStopped) || !finishedAt.Valid ||
			finishedAt.Int64 != confirmedAt || failureCode != expectedFailureCode) {
		return fmt.Errorf("sqlite.recovery_stop_effect_execution_invalid:%s", actionRef)
	}
	return nil
}

func validateRecoveryV14StopEffectReceiptUniqueness(
	ctx context.Context,
	transaction *sql.Tx,
) error {
	var duplicates int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM (
    SELECT effect_receipt_ref FROM action_consumption_receipts
    WHERE effect_receipt_ref IS NOT NULL
    GROUP BY effect_receipt_ref HAVING COUNT(*) <> 1
)`).Scan(&duplicates); err != nil {
		return err
	}
	if duplicates != 0 {
		return errors.New("sqlite.recovery_stop_effect_receipt_duplicate")
	}
	return nil
}
