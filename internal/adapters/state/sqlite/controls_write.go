package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
)

func insertControl(
	ctx context.Context,
	transaction *sql.Tx,
	record application.ControlRecord,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO controls(
    ref, request_ref, request_fingerprint, authorization_receipt_ref,
    principal_ref, project_ref, goal_ref, goal_revision, work_item_ref, work_item_revision,
    execution_ref, execution_attempt, operation, target, mode, reason,
    plan_generation, app_spec_generation, spec_hash, status, requested_at,
	    confirmed_at, receipt_ref, supersedes_control_ref, superseded_at,
	    superseded_by_control_ref
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Ref, record.RequestRef, record.RequestFingerprint, record.AuthorizationReceipt.Ref(),
		record.PrincipalRef.String(), record.ProjectRef.String(), record.GoalRef.String(),
		int64(record.GoalRevision), nullableString(record.WorkItemRef.String()), int64(record.WorkItemRevision),
		nullableString(record.ExecutionRef.String()), int64(record.ExecutionAttempt),
		string(record.Operation), string(record.Target), string(record.Mode), record.Reason,
		int64(record.PlanGeneration), int64(record.AppSpecGeneration), record.SpecHash,
		string(record.Status), requiredTime(record.RequestedAt), storedTime(record.ConfirmedAt),
		nullableString(record.ReceiptRef), nullableString(record.SupersedesControlRef),
		storedTime(record.SupersededAt), nullableString(record.SupersededByControlRef),
	)
	return mapDatabaseError(err)
}

func updateControl(
	ctx context.Context,
	transaction *sql.Tx,
	record application.ControlRecord,
) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE controls
	SET status = ?, confirmed_at = ?, receipt_ref = ?, superseded_at = ?, superseded_by_control_ref = ?
	WHERE ref = ? AND status = 'requested'`,
		string(record.Status), storedTime(record.ConfirmedAt), nullableString(record.ReceiptRef),
		storedTime(record.SupersededAt), nullableString(record.SupersededByControlRef),
		record.Ref,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}
