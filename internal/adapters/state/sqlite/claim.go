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

func (repository *Repository) ClaimNextAction(
	ctx context.Context,
	request application.ClaimRequest,
) (application.ActionClaim, bool, error) {
	if !validText(request.WorkerRef) || !validText(request.Token) {
		return application.ActionClaim{}, false, invalid(errors.New("sqlite.claim_identity_invalid"))
	}
	leaseUntil, err := safeLeaseUntil(request.Now, request.LeaseDuration)
	if err != nil {
		return application.ActionClaim{}, false, invalid(err)
	}
	now := request.Now.Round(0).UTC()
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	var tokenExists int
	err = transaction.QueryRowContext(
		ctx,
		"SELECT 1 FROM outbox WHERE claim_token = ? LIMIT 1",
		request.Token,
	).Scan(&tokenExists)
	if err == nil {
		return application.ActionClaim{}, false, stateError(application.StateAlreadyClaimed, errors.New("sqlite.claim_token_reused"))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}

	var action application.ActionRecord
	var kind string
	var goalValue, itemValue, executionValue string
	var availableAt int64
	var previousAttempt int64
	err = transaction.QueryRowContext(ctx, `
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, available_at, attempt
FROM outbox
WHERE completed_at IS NULL
  AND quarantined_at IS NULL
  AND available_at <= ?
  AND (claim_token IS NULL OR claimed_until <= ?)
ORDER BY available_at, ref
LIMIT 1`, requiredTime(now), requiredTime(now)).Scan(
		&action.Ref,
		&kind,
		&goalValue,
		&itemValue,
		&executionValue,
		&availableAt,
		&previousAttempt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if err := commit(transaction); err != nil {
			return application.ActionClaim{}, false, err
		}
		return application.ActionClaim{}, false, nil
	}
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	if previousAttempt < 0 || previousAttempt == int64(^uint64(0)>>1) {
		return application.ActionClaim{}, false, invalid(fmt.Errorf("sqlite.action_attempt_invalid"))
	}
	var refErr error
	action.Kind = application.ActionKind(kind)
	if action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return application.ActionClaim{}, false, invalid(refErr)
	}
	if action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
		return application.ActionClaim{}, false, invalid(refErr)
	}
	if action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
		return application.ActionClaim{}, false, invalid(refErr)
	}
	action.AvailableAt = time.Unix(0, availableAt).UTC()
	attempt := previousAttempt + 1
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, attempt = ?
WHERE ref = ?
  AND completed_at IS NULL
  AND quarantined_at IS NULL
  AND available_at <= ?
  AND (claim_token IS NULL OR claimed_until <= ?)`,
		request.Token,
		request.WorkerRef,
		requiredTime(leaseUntil),
		attempt,
		action.Ref,
		requiredTime(now),
		requiredTime(now),
	)
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.ActionClaim{}, false, err
	}
	claim := application.ActionClaim{
		Action:     action,
		Token:      request.Token,
		WorkerRef:  request.WorkerRef,
		Attempt:    uint64(attempt),
		LeaseUntil: leaseUntil,
	}
	if err := commit(transaction); err != nil {
		return application.ActionClaim{}, false, err
	}
	return claim, true, nil
}

func requireClaim(ctx context.Context, transaction *sql.Tx, claim application.ActionClaim) error {
	if err := validateClaim(claim); err != nil {
		return invalid(err)
	}
	var kind, goalValue, itemValue, executionValue string
	var availableAt int64
	var token, worker sql.NullString
	var leaseUntil, completedAt, quarantinedAt sql.NullInt64
	var attempt int64
	err := transaction.QueryRowContext(ctx, `
SELECT kind, goal_ref, work_item_ref, execution_ref, available_at,
       claim_token, claimed_by, claimed_until, attempt, completed_at, quarantined_at
FROM outbox
WHERE ref = ?`, claim.Action.Ref).Scan(
		&kind,
		&goalValue,
		&itemValue,
		&executionValue,
		&availableAt,
		&token,
		&worker,
		&leaseUntil,
		&attempt,
		&completedAt,
		&quarantinedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict(err)
	}
	if err != nil {
		return mapDatabaseError(err)
	}
	if completedAt.Valid || quarantinedAt.Valid || !token.Valid || !worker.Valid || !leaseUntil.Valid ||
		token.String != claim.Token || worker.String != claim.WorkerRef || attempt != int64(claim.Attempt) ||
		leaseUntil.Int64 != requiredTime(claim.LeaseUntil) || kind != string(claim.Action.Kind) ||
		goalValue != claim.Action.GoalRef.String() || itemValue != claim.Action.WorkItemRef.String() ||
		executionValue != claim.Action.ExecutionRef.String() || availableAt != requiredTime(claim.Action.AvailableAt) {
		return conflict(errors.New("sqlite.claim_cas_conflict"))
	}
	return nil
}

func completeClaim(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
	at time.Time,
	errorCode string,
	quarantined bool,
) error {
	var quarantineAt any
	if quarantined {
		quarantineAt = requiredTime(at)
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET completed_at = ?, quarantined_at = ?, last_error_code = ?
WHERE ref = ? AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND attempt = ? AND completed_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(at),
		quarantineAt,
		errorCode,
		claim.Action.Ref,
		claim.Token,
		claim.WorkerRef,
		requiredTime(claim.LeaseUntil),
		int64(claim.Attempt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func releaseClaimForRetry(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ActionRequeuedState,
) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET available_at = ?, claim_token = NULL, claimed_by = NULL, claimed_until = NULL,
    last_error_code = ?
WHERE ref = ? AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND attempt = ? AND completed_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(state.AvailableAt),
		state.ErrorCode,
		state.Claim.Action.Ref,
		state.Claim.Token,
		state.Claim.WorkerRef,
		requiredTime(state.Claim.LeaseUntil),
		int64(state.Claim.Attempt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}
