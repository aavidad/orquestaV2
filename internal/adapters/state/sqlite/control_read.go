package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func readControlByRequest(
	ctx context.Context,
	source queryer,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	requestRef string,
) (application.ControlRecord, bool, error) {
	var record application.ControlRecord
	var principalValue, projectValue, goalValue, authorizationRef string
	var workItemValue, executionValue, mode, receiptRef sql.NullString
	var supersedesControlRef, supersededByControlRef sql.NullString
	var goalRevision, workItemRevision, executionAttempt, planGeneration, appSpecGeneration, requestedAt int64
	var confirmedAt, supersededAt sql.NullInt64
	var operation, target, status string
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       principal_ref, project_ref, goal_ref, goal_revision, work_item_ref, work_item_revision,
       execution_ref, execution_attempt, operation, target, mode, reason,
       plan_generation, app_spec_generation, spec_hash, status, requested_at,
       confirmed_at, receipt_ref, supersedes_control_ref, superseded_at,
       superseded_by_control_ref
FROM controls
WHERE principal_ref = ? AND project_ref = ? AND request_ref = ?`,
		principalRef.String(), projectRef.String(), requestRef,
	).Scan(
		&record.Ref, &record.RequestRef, &record.RequestFingerprint, &authorizationRef,
		&principalValue, &projectValue, &goalValue, &goalRevision, &workItemValue, &workItemRevision,
		&executionValue, &executionAttempt, &operation, &target, &mode, &record.Reason,
		&planGeneration, &appSpecGeneration, &record.SpecHash, &status, &requestedAt,
		&confirmedAt, &receiptRef, &supersedesControlRef, &supersededAt,
		&supersededByControlRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ControlRecord{}, false, nil
	}
	if err != nil {
		return application.ControlRecord{}, false, mapDatabaseError(err)
	}
	var refErr error
	if record.PrincipalRef, refErr = identity.NewPrincipalRef(principalValue); refErr != nil {
		return application.ControlRecord{}, false, invalid(refErr)
	}
	if record.ProjectRef, refErr = goal.NewProjectRef(projectValue); refErr != nil {
		return application.ControlRecord{}, false, invalid(refErr)
	}
	if record.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return application.ControlRecord{}, false, invalid(refErr)
	}
	if workItemValue.Valid {
		if record.WorkItemRef, refErr = goal.NewWorkItemRef(workItemValue.String); refErr != nil {
			return application.ControlRecord{}, false, invalid(refErr)
		}
	}
	if executionValue.Valid {
		if record.ExecutionRef, refErr = goal.NewExecutionRef(executionValue.String); refErr != nil {
			return application.ControlRecord{}, false, invalid(refErr)
		}
	}
	if goalRevision <= 0 || workItemRevision < 0 || executionAttempt < 0 || planGeneration <= 0 || appSpecGeneration <= 0 {
		return application.ControlRecord{}, false, invalid(errors.New("sqlite.control_generation_invalid"))
	}
	record.GoalRevision = goal.Revision(goalRevision)
	record.WorkItemRevision = goal.Revision(workItemRevision)
	record.ExecutionAttempt = uint64(executionAttempt)
	record.Operation = application.ControlOperation(operation)
	record.Target = application.ControlTarget(target)
	if mode.Valid {
		record.Mode = ports.AgentStopMode(mode.String)
	}
	record.PlanGeneration = goal.PlanGeneration(planGeneration)
	record.AppSpecGeneration = goal.AppSpecGeneration(appSpecGeneration)
	record.Status = application.ControlStatus(status)
	record.RequestedAt = time.Unix(0, requestedAt).UTC()
	record.ConfirmedAt = restoredTime(confirmedAt)
	if receiptRef.Valid {
		record.ReceiptRef = receiptRef.String
	}
	if supersedesControlRef.Valid {
		record.SupersedesControlRef = supersedesControlRef.String
	}
	record.SupersededAt = restoredTime(supersededAt)
	if supersededByControlRef.Valid {
		record.SupersededByControlRef = supersededByControlRef.String
	}
	record.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, authorizationRef)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	if err := application.ValidatePersistedControlRecord(record); err != nil {
		return application.ControlRecord{}, false, invalid(err)
	}
	return record, true, nil
}

func readControlByRef(
	ctx context.Context,
	source queryer,
	ref string,
) (application.ControlRecord, bool, error) {
	var principalValue, projectValue, requestRef string
	err := source.QueryRowContext(ctx, `
SELECT principal_ref, project_ref, request_ref FROM controls WHERE ref = ?`, ref,
	).Scan(&principalValue, &projectValue, &requestRef)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ControlRecord{}, false, nil
	}
	if err != nil {
		return application.ControlRecord{}, false, mapDatabaseError(err)
	}
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		return application.ControlRecord{}, false, invalid(err)
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil {
		return application.ControlRecord{}, false, invalid(err)
	}
	return readControlByRequest(ctx, source, principalRef, projectRef, requestRef)
}

func readControls(ctx context.Context, source queryer, goalValue string) ([]application.ControlRecord, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "controls", "ref")
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	if !persisted {
		return nil, nil
	}
	rows, err := source.QueryContext(ctx, `SELECT principal_ref, project_ref, request_ref
FROM controls WHERE goal_ref = ? ORDER BY requested_at, ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	type key struct{ principal, project, request string }
	var keys []key
	for rows.Next() {
		var item key
		if err := rows.Scan(&item.principal, &item.project, &item.request); err != nil {
			return nil, mapDatabaseError(err)
		}
		keys = append(keys, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]application.ControlRecord, 0, len(keys))
	for _, item := range keys {
		principal, err := identity.NewPrincipalRef(item.principal)
		if err != nil {
			return nil, invalid(err)
		}
		project, err := goal.NewProjectRef(item.project)
		if err != nil {
			return nil, invalid(err)
		}
		record, found, err := readControlByRequest(ctx, source, principal, project, item.request)
		if err != nil || !found {
			if err == nil {
				err = errors.New("sqlite.control_projection_missing")
			}
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

func controlIdentityMatches(left, right application.ControlRecord) bool {
	return left.Ref == right.Ref && left.RequestedAt.Equal(right.RequestedAt) &&
		controlRequestIdentityMatches(left, right)
}

func controlRequestIdentityMatches(left, right application.ControlRecord) bool {
	return left.RequestRef == right.RequestRef && left.RequestFingerprint == right.RequestFingerprint &&
		left.AuthorizationReceipt.Ref() == right.AuthorizationReceipt.Ref() &&
		left.PrincipalRef == right.PrincipalRef && left.ProjectRef == right.ProjectRef &&
		left.GoalRef == right.GoalRef && left.GoalRevision == right.GoalRevision &&
		left.WorkItemRef == right.WorkItemRef && left.WorkItemRevision == right.WorkItemRevision &&
		left.ExecutionRef == right.ExecutionRef && left.ExecutionAttempt == right.ExecutionAttempt &&
		left.Operation == right.Operation && left.Target == right.Target && left.Mode == right.Mode &&
		left.Reason == right.Reason && left.PlanGeneration == right.PlanGeneration &&
		left.AppSpecGeneration == right.AppSpecGeneration && left.SpecHash == right.SpecHash &&
		left.SupersedesControlRef == right.SupersedesControlRef
}

func sqliteExecutionByRef(
	records []application.ExecutionRecord,
	ref goal.ExecutionRef,
) (application.ExecutionRecord, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return application.ExecutionRecord{}, false
}
