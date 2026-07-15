package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type directorLeaseRow struct {
	projectRef              string
	principalRef            string
	token                   string
	fence                   int64
	leaseUntil              int64
	claimRequestRef         string
	claimRequestFingerprint string
	claimAuthorizationRef   string
	renewRequestRef         sql.NullString
	renewRequestFingerprint sql.NullString
	renewAuthorizationRef   sql.NullString
	updatedAt               int64
}

type directorLeaseReceiptRow struct {
	ref                string
	action             string
	requestFingerprint string
	authorizationRef   string
	goalRef            string
	projectRef         string
	principalRef       string
	fence              int64
	leaseUntil         int64
	occurredAt         int64
}

func (repository *Repository) DirectorReplay(
	ctx context.Context,
	request application.DirectorReplayRequest,
) (application.DirectorReplayRecord, bool, error) {
	if !validText(request.RequestRef) || !validText(request.RequestFingerprint) ||
		request.PrincipalRef.String() == "" || request.ProjectRef.String() == "" ||
		request.GoalRef.String() == "" {
		return application.DirectorReplayRecord{}, false, invalid(errors.New("sqlite.director_replay_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.DirectorReplayRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	switch request.Kind {
	case application.DirectorMutationClaim, application.DirectorMutationRenew:
		receipt, found, readErr := readDirectorLeaseReceipt(
			ctx, transaction, request.PrincipalRef, request.ProjectRef, request.RequestRef,
		)
		if readErr != nil {
			return application.DirectorReplayRecord{}, false, readErr
		}
		if !found {
			if err := commit(transaction); err != nil {
				return application.DirectorReplayRecord{}, false, err
			}
			return application.DirectorReplayRecord{}, false, nil
		}
		lease, replayErr := repository.replayActiveDirectorLease(
			ctx, transaction, receipt, string(request.Kind), request.RequestFingerprint,
			request.GoalRef, request.ProjectRef, request.PrincipalRef, "", 0,
		)
		if replayErr != nil {
			return application.DirectorReplayRecord{}, false, replayErr
		}
		if err := commit(transaction); err != nil {
			return application.DirectorReplayRecord{}, false, err
		}
		return application.DirectorReplayRecord{Lease: lease}, true, nil
	case application.DirectorMutationPlan:
		decision, found, readErr := readDirectorDecisionByRequest(
			ctx, transaction, request.PrincipalRef, request.ProjectRef, request.GoalRef, request.RequestRef,
		)
		if readErr != nil {
			return application.DirectorReplayRecord{}, false, readErr
		}
		if !found {
			if err := commit(transaction); err != nil {
				return application.DirectorReplayRecord{}, false, err
			}
			return application.DirectorReplayRecord{}, false, nil
		}
		if decision.RequestFingerprint != request.RequestFingerprint {
			return application.DirectorReplayRecord{}, false,
				conflict(errors.New("sqlite.director_replay_fingerprint_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.DirectorReplayRecord{}, false, err
		}
		return application.DirectorReplayRecord{Decision: decision}, true, nil
	default:
		return application.DirectorReplayRecord{}, false, invalid(errors.New("sqlite.director_replay_kind_invalid"))
	}
}

func (repository *Repository) ClaimDirector(
	ctx context.Context,
	state application.ClaimDirectorState,
) (application.DirectorLeaseRecord, bool, error) {
	if err := validateClaimDirectorState(state); err != nil {
		return application.DirectorLeaseRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String(),
	); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if err := requireDirectorGoalScope(ctx, transaction, state.GoalRef, state.ProjectRef); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	receipt, replay, err := readDirectorLeaseReceipt(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.RequestRef,
	)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if replay {
		record, restoreErr := repository.replayActiveDirectorLease(
			ctx, transaction, receipt, "claim", state.RequestFingerprint,
			state.GoalRef, state.ProjectRef, state.PrincipalRef, "", 0,
		)
		if restoreErr != nil {
			return application.DirectorLeaseRecord{}, false, restoreErr
		}
		if err := commit(transaction); err != nil {
			return application.DirectorLeaseRecord{}, false, err
		}
		return record, false, nil
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if now.Before(state.AuthorizationReceipt.RecordedAt()) {
		return application.DirectorLeaseRecord{}, false,
			conflict(errors.New("sqlite.director_authorization_from_future"))
	}
	leaseUntil, err := safeLeaseUntil(now, state.LeaseDuration)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, invalid(err)
	}
	stored, found, err := readDirectorLease(ctx, transaction, state.GoalRef)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	changed := true
	receiptInserted := false
	if !found {
		_, err = transaction.ExecContext(ctx, `
INSERT INTO director_leases(
    goal_ref, project_ref, principal_ref, token, fence, lease_until,
    claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
    updated_at
) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?)`,
			state.GoalRef.String(), state.ProjectRef.String(), state.PrincipalRef.String(), state.Token,
			requiredTime(leaseUntil), state.RequestRef, state.RequestFingerprint,
			state.AuthorizationReceipt.Ref(), requiredTime(now),
		)
	} else {
		if now.Before(time.Unix(0, stored.leaseUntil).UTC()) {
			return application.DirectorLeaseRecord{}, false, stateError(
				application.StateAlreadyClaimed, errors.New("sqlite.director_lease_live"),
			)
		}
		if stored.fence <= 0 || uint64(stored.fence) >= maxSQLiteInteger {
			return application.DirectorLeaseRecord{}, false, conflict(errors.New("sqlite.director_fence_exhausted"))
		}
		if err := insertDirectorLeaseReceipt(
			ctx, transaction, "claim", state.RequestRef, state.RequestFingerprint,
			state.AuthorizationReceipt.Ref(), state.GoalRef, state.ProjectRef, state.PrincipalRef,
			stored.fence+1, requiredTime(leaseUntil), now,
		); err != nil {
			return application.DirectorLeaseRecord{}, false, err
		}
		receiptInserted = true
		result, updateErr := transaction.ExecContext(ctx, `
UPDATE director_leases
SET principal_ref = ?, token = ?, fence = fence + 1, lease_until = ?,
    claim_request_ref = ?, claim_request_fingerprint = ?,
    claim_authorization_receipt_ref = ?,
    renew_request_ref = NULL, renew_request_fingerprint = NULL,
    renew_authorization_receipt_ref = NULL, updated_at = ?
WHERE goal_ref = ? AND project_ref = ? AND fence = ? AND lease_until <= ?`,
			state.PrincipalRef.String(), state.Token, requiredTime(leaseUntil),
			state.RequestRef, state.RequestFingerprint, state.AuthorizationReceipt.Ref(),
			requiredTime(now), state.GoalRef.String(), state.ProjectRef.String(),
			stored.fence, requiredTime(now),
		)
		if updateErr != nil {
			err = updateErr
		} else {
			err = requireOneRow(result)
		}
	}
	if err != nil {
		return application.DirectorLeaseRecord{}, false, mapDatabaseError(err)
	}
	stored, found, err = readDirectorLease(ctx, transaction, state.GoalRef)
	if err != nil || !found {
		if err == nil {
			err = conflict(errors.New("sqlite.director_lease_missing_after_write"))
		}
		return application.DirectorLeaseRecord{}, false, err
	}
	record, err := restoreDirectorLease(state.GoalRef, stored)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if !receiptInserted {
		if err := insertDirectorLeaseReceipt(
			ctx, transaction, "claim", state.RequestRef, state.RequestFingerprint,
			state.AuthorizationReceipt.Ref(), state.GoalRef, state.ProjectRef, state.PrincipalRef,
			stored.fence, stored.leaseUntil, now,
		); err != nil {
			return application.DirectorLeaseRecord{}, false, err
		}
	}
	if err := commit(transaction); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	return record, changed, nil
}

func (repository *Repository) RenewDirector(
	ctx context.Context,
	state application.RenewDirectorState,
) (application.DirectorLeaseRecord, bool, error) {
	if err := validateRenewDirectorState(state); err != nil {
		return application.DirectorLeaseRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String(),
	); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if err := requireDirectorGoalScope(ctx, transaction, state.GoalRef, state.ProjectRef); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	receipt, replay, err := readDirectorLeaseReceipt(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.RequestRef,
	)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if replay {
		record, restoreErr := repository.replayActiveDirectorLease(
			ctx, transaction, receipt, "renew", state.RequestFingerprint,
			state.GoalRef, state.ProjectRef, state.PrincipalRef, state.Token, state.Fence,
		)
		if restoreErr != nil {
			return application.DirectorLeaseRecord{}, false, restoreErr
		}
		if err := commit(transaction); err != nil {
			return application.DirectorLeaseRecord{}, false, err
		}
		return record, false, nil
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if now.Before(state.AuthorizationReceipt.RecordedAt()) {
		return application.DirectorLeaseRecord{}, false,
			conflict(errors.New("sqlite.director_authorization_from_future"))
	}
	stored, found, err := readDirectorLease(ctx, transaction, state.GoalRef)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if !found {
		return application.DirectorLeaseRecord{}, false, conflict(errors.New("sqlite.director_lease_missing"))
	}
	if stored.projectRef != state.ProjectRef.String() || stored.principalRef != state.PrincipalRef.String() ||
		stored.token != state.Token || stored.fence <= 0 || uint64(stored.fence) != state.Fence ||
		!now.Before(time.Unix(0, stored.leaseUntil).UTC()) {
		return application.DirectorLeaseRecord{}, false, conflict(errors.New("sqlite.director_renew_fence_conflict"))
	}
	leaseUntil, err := safeLeaseUntil(now, state.LeaseDuration)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, invalid(err)
	}
	if !leaseUntil.After(time.Unix(0, stored.leaseUntil).UTC()) {
		return application.DirectorLeaseRecord{}, false, conflict(errors.New("sqlite.director_renew_not_extended"))
	}
	if err := insertDirectorLeaseReceipt(
		ctx, transaction, "renew", state.RequestRef, state.RequestFingerprint,
		state.AuthorizationReceipt.Ref(), state.GoalRef, state.ProjectRef, state.PrincipalRef,
		stored.fence, requiredTime(leaseUntil), now,
	); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE director_leases
SET lease_until = ?, renew_request_ref = ?, renew_request_fingerprint = ?,
    renew_authorization_receipt_ref = ?, updated_at = ?
WHERE goal_ref = ? AND project_ref = ? AND principal_ref = ?
  AND token = ? AND fence = ? AND lease_until = ? AND lease_until > ?`,
		requiredTime(leaseUntil), state.RequestRef, state.RequestFingerprint,
		state.AuthorizationReceipt.Ref(), requiredTime(now), state.GoalRef.String(),
		state.ProjectRef.String(), state.PrincipalRef.String(), state.Token,
		int64(state.Fence), stored.leaseUntil, requiredTime(now),
	)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	stored, found, err = readDirectorLease(ctx, transaction, state.GoalRef)
	if err != nil || !found {
		if err == nil {
			err = conflict(errors.New("sqlite.director_lease_missing_after_renew"))
		}
		return application.DirectorLeaseRecord{}, false, err
	}
	record, err := restoreDirectorLease(state.GoalRef, stored)
	if err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.DirectorLeaseRecord{}, false, err
	}
	return record, true, nil
}

func (repository *Repository) ApplyDirectorPlan(
	ctx context.Context,
	state application.ApplyDirectorPlanState,
) (application.DirectorDecisionRecord, bool, error) {
	if err := validateApplyDirectorPlanState(state); err != nil {
		return application.DirectorDecisionRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String(),
	); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}

	storedDecision, found, err := readDirectorDecisionByRequest(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.GoalRef, state.RequestRef,
	)
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if found {
		if !directorDecisionReplays(state, storedDecision) {
			return application.DirectorDecisionRecord{}, false,
				conflict(errors.New("sqlite.director_decision_replay_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.DirectorDecisionRecord{}, false, err
		}
		return storedDecision, false, nil
	}

	now, err := repository.transactionTime()
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := requireLiveDirectorLease(ctx, transaction, state, now); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	current, err := readGoalRecord(ctx, transaction, state.GoalRef.String())
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := validateDirectorPlanTransition(state, current); err != nil {
		return application.DirectorDecisionRecord{}, false, conflict(err)
	}
	if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := insertDirectorPlanDelta(ctx, transaction, current.Goal, state.Goal); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := insertScheduled(ctx, transaction, state.NewExecutions, state.NewActions); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := requireReadyExecutions(ctx, transaction, state.Goal); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := insertEvents(ctx, transaction, state.Events); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := insertDirectorDecision(ctx, transaction, state.Decision, state.ProjectRef); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	commitNow, err := repository.transactionTime()
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := requireLiveDirectorLease(ctx, transaction, state, commitNow); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	decision, found, err := readDirectorDecisionByRequest(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.GoalRef, state.RequestRef,
	)
	if err != nil || !found {
		if err == nil {
			err = conflict(errors.New("sqlite.director_decision_missing_after_write"))
		}
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	return decision, true, nil
}

func validateClaimDirectorState(state application.ClaimDirectorState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		!validText(state.Token) || state.LeaseDuration <= 0 || state.RequestedAt.IsZero() {
		return errors.New("sqlite.director_claim_invalid")
	}
	request := state.AuthorizationReceipt.Decision().Request()
	if request.Principal().Ref != state.PrincipalRef || request.ProjectRef() != state.ProjectRef ||
		request.Permission() != identity.PermissionGoalsDirect || request.ResourceRef() != state.GoalRef.String() ||
		state.RequestedAt.Before(state.AuthorizationReceipt.RecordedAt()) {
		return errors.New("sqlite.director_claim_authorization_invalid")
	}
	return nil
}

func validateRenewDirectorState(state application.RenewDirectorState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		!validText(state.Token) || state.Fence == 0 || state.Fence > maxSQLiteInteger ||
		state.LeaseDuration <= 0 || state.RequestedAt.IsZero() {
		return errors.New("sqlite.director_renew_invalid")
	}
	request := state.AuthorizationReceipt.Decision().Request()
	if request.Principal().Ref != state.PrincipalRef || request.ProjectRef() != state.ProjectRef ||
		request.Permission() != identity.PermissionGoalsDirect || request.ResourceRef() != state.GoalRef.String() ||
		state.RequestedAt.Before(state.AuthorizationReceipt.RecordedAt()) {
		return errors.New("sqlite.director_renew_authorization_invalid")
	}
	return nil
}

func validateApplyDirectorPlanState(state application.ApplyDirectorPlanState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		!validText(state.LeaseToken) || state.LeaseFence == 0 || state.LeaseFence > maxSQLiteInteger ||
		state.ExpectedGoalRevision == 0 || uint64(state.ExpectedGoalRevision) >= maxSQLiteInteger ||
		uint64(state.ExpectedPlanGeneration) >= maxSQLiteInteger || state.OperationAt.IsZero() {
		return errors.New("sqlite.director_plan_invalid")
	}
	snapshot := state.Goal.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return err
	}
	decision := state.Decision
	if !validText(decision.Ref) || decision.RequestRef != state.RequestRef ||
		decision.RequestFingerprint != state.RequestFingerprint || decision.GoalRef != state.GoalRef ||
		decision.PrincipalRef != state.PrincipalRef || decision.LeaseFence != state.LeaseFence ||
		decision.SourceGoalRevision != state.ExpectedGoalRevision ||
		decision.SourcePlanGeneration != state.ExpectedPlanGeneration ||
		decision.AppliedGoalRevision != state.Goal.Revision() ||
		decision.AppliedPlanGeneration != state.Goal.PlanGeneration() || !validText(decision.Reason) ||
		decision.DecidedAt.IsZero() || !decision.DecidedAt.Equal(state.OperationAt) ||
		!sameAuthorizationReceipt(decision.AuthorizationReceipt, state.AuthorizationReceipt) {
		return errors.New("sqlite.director_decision_invalid")
	}
	request := state.AuthorizationReceipt.Decision().Request()
	if request.Principal().Ref != state.PrincipalRef || request.ProjectRef() != state.ProjectRef ||
		request.Permission() != identity.PermissionGoalsDirect || request.ResourceRef() != state.GoalRef.String() ||
		state.OperationAt.Before(state.AuthorizationReceipt.RecordedAt()) {
		return errors.New("sqlite.director_plan_authorization_invalid")
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(state.Events))
	for _, event := range state.Events {
		if err := validateEvent(event); err != nil || event.GoalRef != state.Goal.Ref() {
			return errors.New("sqlite.director_event_invalid")
		}
		if _, duplicate := seen[event.Ref]; duplicate {
			return errors.New("sqlite.director_event_duplicate")
		}
		seen[event.Ref] = struct{}{}
	}
	return nil
}

func requireDirectorGoalScope(
	ctx context.Context,
	source queryer,
	goalRef goal.GoalRef,
	projectRef goal.ProjectRef,
) error {
	var exists int
	err := source.QueryRowContext(ctx, `
SELECT 1 FROM goals WHERE ref = ? AND project_ref = ?`,
		goalRef.String(), projectRef.String(),
	).Scan(&exists)
	return mapDatabaseError(err)
}

func readDirectorLease(
	ctx context.Context,
	source queryer,
	goalRef goal.GoalRef,
) (directorLeaseRow, bool, error) {
	var stored directorLeaseRow
	err := source.QueryRowContext(ctx, `
SELECT project_ref, principal_ref, token, fence, lease_until,
       claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
       renew_request_ref, renew_request_fingerprint, renew_authorization_receipt_ref,
       updated_at
FROM director_leases WHERE goal_ref = ?`, goalRef.String()).Scan(
		&stored.projectRef, &stored.principalRef, &stored.token, &stored.fence, &stored.leaseUntil,
		&stored.claimRequestRef, &stored.claimRequestFingerprint, &stored.claimAuthorizationRef,
		&stored.renewRequestRef, &stored.renewRequestFingerprint, &stored.renewAuthorizationRef,
		&stored.updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return directorLeaseRow{}, false, nil
	}
	if err != nil {
		return directorLeaseRow{}, false, mapDatabaseError(err)
	}
	return stored, true, nil
}

func restoreDirectorLease(
	goalRef goal.GoalRef,
	stored directorLeaseRow,
) (application.DirectorLeaseRecord, error) {
	principalRef, err := identity.NewPrincipalRef(stored.principalRef)
	if err != nil || stored.fence <= 0 || !validText(stored.token) || stored.leaseUntil == 0 {
		if err == nil {
			err = errors.New("sqlite.director_lease_record_invalid")
		}
		return application.DirectorLeaseRecord{}, invalid(err)
	}
	return application.DirectorLeaseRecord{
		GoalRef: goalRef, PrincipalRef: principalRef, Token: stored.token,
		Fence: uint64(stored.fence), LeaseUntil: time.Unix(0, stored.leaseUntil).UTC(),
	}, nil
}

func readDirectorLeaseReceipt(
	ctx context.Context,
	source queryer,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	requestRef string,
) (directorLeaseReceiptRow, bool, error) {
	var stored directorLeaseReceiptRow
	err := source.QueryRowContext(ctx, `
SELECT ref, action, request_fingerprint, authorization_receipt_ref,
       goal_ref, project_ref, principal_ref,
       fence, lease_until, occurred_at
FROM director_lease_receipts
WHERE principal_ref = ? AND project_ref = ? AND request_ref = ?`,
		principalRef.String(), projectRef.String(), requestRef).Scan(
		&stored.ref, &stored.action, &stored.requestFingerprint, &stored.authorizationRef,
		&stored.goalRef, &stored.projectRef, &stored.principalRef,
		&stored.fence, &stored.leaseUntil, &stored.occurredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return directorLeaseReceiptRow{}, false, nil
	}
	if err != nil {
		return directorLeaseReceiptRow{}, false, mapDatabaseError(err)
	}
	return stored, true, nil
}

func (repository *Repository) replayActiveDirectorLease(
	ctx context.Context,
	source queryer,
	stored directorLeaseReceiptRow,
	action string,
	requestFingerprint string,
	goalRef goal.GoalRef,
	projectRef goal.ProjectRef,
	principalRef identity.PrincipalRef,
	token string,
	fence uint64,
) (application.DirectorLeaseRecord, error) {
	if stored.action != action || stored.requestFingerprint != requestFingerprint ||
		stored.goalRef != goalRef.String() || stored.projectRef != projectRef.String() ||
		stored.principalRef != principalRef.String() ||
		stored.fence <= 0 || stored.leaseUntil == 0 || (fence != 0 && uint64(stored.fence) != fence) {
		return application.DirectorLeaseRecord{}, conflict(errors.New("sqlite.director_lease_replay_conflict"))
	}
	active, found, err := readDirectorLease(ctx, source, goalRef)
	if err != nil {
		return application.DirectorLeaseRecord{}, err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.DirectorLeaseRecord{}, err
	}
	if !found || active.projectRef != projectRef.String() || active.principalRef != principalRef.String() ||
		active.fence != stored.fence || (token != "" && active.token != token) ||
		!now.Before(time.Unix(0, active.leaseUntil).UTC()) {
		return application.DirectorLeaseRecord{}, conflict(errors.New("sqlite.director_lease_replay_inactive"))
	}
	return application.DirectorLeaseRecord{
		GoalRef: goalRef, PrincipalRef: principalRef, Token: active.token,
		Fence: uint64(stored.fence), LeaseUntil: time.Unix(0, stored.leaseUntil).UTC(),
	}, nil
}

func insertDirectorLeaseReceipt(
	ctx context.Context,
	transaction *sql.Tx,
	action string,
	requestRef string,
	requestFingerprint string,
	authorizationRef string,
	goalRef goal.GoalRef,
	projectRef goal.ProjectRef,
	principalRef identity.PrincipalRef,
	fence int64,
	leaseUntil int64,
	occurredAt time.Time,
) error {
	refFingerprint := canonicalFingerprint(
		"director-lease-receipt.v1", action, principalRef.String(), requestRef,
		requestFingerprint, goalRef.String(), projectRef.String(),
	)
	_, err := transaction.ExecContext(ctx, `
INSERT INTO director_lease_receipts(
    ref, action, request_ref, request_fingerprint, authorization_receipt_ref,
    goal_ref, project_ref, principal_ref,
    fence, lease_until, occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deterministicRef("director-lease-receipt", refFingerprint), action, requestRef,
		requestFingerprint, authorizationRef, goalRef.String(), projectRef.String(),
		principalRef.String(), fence, leaseUntil, requiredTime(occurredAt),
	)
	return mapDatabaseError(err)
}

func requireLiveDirectorLease(
	ctx context.Context,
	source queryer,
	state application.ApplyDirectorPlanState,
	now time.Time,
) error {
	var exists int
	err := source.QueryRowContext(ctx, `
SELECT 1
FROM director_leases
WHERE goal_ref = ? AND project_ref = ? AND principal_ref = ?
  AND token = ? AND fence = ? AND lease_until > ?`,
		state.GoalRef.String(), state.ProjectRef.String(), state.PrincipalRef.String(),
		state.LeaseToken, int64(state.LeaseFence), requiredTime(now),
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict(errors.New("sqlite.director_lease_fence_conflict"))
	}
	return mapDatabaseError(err)
}

func validateDirectorPlanTransition(
	state application.ApplyDirectorPlanState,
	current application.GoalRecord,
) error {
	before := current.Goal.Snapshot()
	after := state.Goal.Snapshot()
	if current.Goal.Ref() != state.GoalRef || current.Goal.Project() != state.ProjectRef ||
		current.Goal.Revision() != state.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != state.ExpectedPlanGeneration ||
		after.Ref != before.Ref || after.ActorRef != before.ActorRef || after.ProjectRef != before.ProjectRef ||
		!reflect.DeepEqual(after.AppSpec, before.AppSpec) || after.State != before.State ||
		!after.CreatedAt.Equal(before.CreatedAt) || !after.StartedAt.Equal(before.StartedAt) ||
		!after.ClosedAt.Equal(before.ClosedAt) || after.Revision != before.Revision+1 ||
		after.PlanGeneration != before.PlanGeneration+1 || len(after.Phases) < len(before.Phases) ||
		len(after.WorkItems) < len(before.WorkItems) ||
		!reflect.DeepEqual(after.Phases[:len(before.Phases)], before.Phases) ||
		!reflect.DeepEqual(after.WorkItems[:len(before.WorkItems)], before.WorkItems) {
		return errors.New("sqlite.director_plan_cas_conflict")
	}
	return nil
}

func insertDirectorPlanDelta(
	ctx context.Context,
	transaction *sql.Tx,
	before goal.Goal,
	after goal.Goal,
) error {
	beforeSnapshot := before.Snapshot()
	afterSnapshot := after.Snapshot()
	if err := insertGoalPhases(ctx, transaction, afterSnapshot, len(beforeSnapshot.Phases)); err != nil {
		return err
	}
	return insertWorkItems(ctx, transaction, afterSnapshot, len(beforeSnapshot.WorkItems))
}

func insertDirectorDecision(
	ctx context.Context,
	transaction *sql.Tx,
	decision application.DirectorDecisionRecord,
	projectRef goal.ProjectRef,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO director_decisions(
    ref, request_ref, request_fingerprint, authorization_receipt_ref,
    goal_ref, project_ref, principal_ref, lease_fence,
    source_goal_revision, source_plan_generation,
    applied_goal_revision, applied_plan_generation, reason, decided_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		decision.Ref, decision.RequestRef, decision.RequestFingerprint,
		decision.AuthorizationReceipt.Ref(), decision.GoalRef.String(), projectRef.String(),
		decision.PrincipalRef.String(), int64(decision.LeaseFence),
		int64(decision.SourceGoalRevision), int64(decision.SourcePlanGeneration),
		int64(decision.AppliedGoalRevision), int64(decision.AppliedPlanGeneration),
		decision.Reason, requiredTime(decision.DecidedAt),
	)
	return mapDatabaseError(err)
}

func readDirectorDecisionByRequest(
	ctx context.Context,
	source queryer,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	requestRef string,
) (application.DirectorDecisionRecord, bool, error) {
	var decision application.DirectorDecisionRecord
	var goalValue, principalValue, authorizationRef string
	var fence, sourceRevision, sourceGeneration, appliedRevision, appliedGeneration, decidedAt int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       goal_ref, principal_ref, lease_fence,
       source_goal_revision, source_plan_generation,
       applied_goal_revision, applied_plan_generation, reason, decided_at
FROM director_decisions
WHERE principal_ref = ? AND project_ref = ? AND goal_ref = ? AND request_ref = ?`,
		principalRef.String(), projectRef.String(), goalRef.String(), requestRef,
	).Scan(
		&decision.Ref, &decision.RequestRef, &decision.RequestFingerprint, &authorizationRef,
		&goalValue, &principalValue, &fence, &sourceRevision, &sourceGeneration,
		&appliedRevision, &appliedGeneration, &decision.Reason, &decidedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.DirectorDecisionRecord{}, false, nil
	}
	if err != nil {
		return application.DirectorDecisionRecord{}, false, mapDatabaseError(err)
	}
	if decision.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
		return application.DirectorDecisionRecord{}, false, invalid(err)
	}
	if decision.PrincipalRef, err = identity.NewPrincipalRef(principalValue); err != nil {
		return application.DirectorDecisionRecord{}, false, invalid(err)
	}
	if fence <= 0 || sourceRevision <= 0 || sourceGeneration < 0 || appliedRevision <= 0 || appliedGeneration <= 0 {
		return application.DirectorDecisionRecord{}, false, invalid(errors.New("sqlite.director_decision_generation_invalid"))
	}
	decision.LeaseFence = uint64(fence)
	decision.SourceGoalRevision = goal.Revision(sourceRevision)
	decision.SourcePlanGeneration = goal.PlanGeneration(sourceGeneration)
	decision.AppliedGoalRevision = goal.Revision(appliedRevision)
	decision.AppliedPlanGeneration = goal.PlanGeneration(appliedGeneration)
	decision.DecidedAt = time.Unix(0, decidedAt).UTC()
	decision.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, authorizationRef)
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	return decision, true, nil
}

func readAuthorizationReceipt(
	ctx context.Context,
	source queryer,
	ref string,
) (identity.AuthorizationReceipt, error) {
	var requestRef, principalValue, actorValue, kind, method string
	var projectValue, permission, resourceRef, outcome, role, reasonCode string
	var requestedAt, membershipRevision, decidedAt, recordedAt int64
	err := source.QueryRowContext(ctx, `
SELECT a.request_ref, p.ref, p.actor_ref, p.kind, p.authentication_method,
       a.project_ref, a.permission, a.resource_ref, a.requested_at,
       a.outcome, a.role, a.membership_revision, a.reason_code,
       a.decided_at, a.recorded_at
FROM authorization_receipts a
JOIN principals p ON p.ref = a.principal_ref
WHERE a.ref = ?`, ref).Scan(
		&requestRef, &principalValue, &actorValue, &kind, &method,
		&projectValue, &permission, &resourceRef, &requestedAt,
		&outcome, &role, &membershipRevision, &reasonCode, &decidedAt, &recordedAt,
	)
	if err != nil {
		return identity.AuthorizationReceipt{}, mapDatabaseError(err)
	}
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	actorRef, err := goal.NewActorRef(actorValue)
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(kind), method)
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: principal, ProjectRef: projectRef,
		Permission: identity.Permission(permission), ResourceRef: resourceRef,
		RequestedAt: time.Unix(0, requestedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationOutcome(outcome), Role: identity.Role(role),
		MembershipRevision: identity.MembershipRevision(membershipRevision), ReasonCode: reasonCode,
		DecidedAt: time.Unix(0, decidedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: ref, Decision: decision, RecordedAt: time.Unix(0, recordedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	return receipt, nil
}

func directorDecisionReplays(
	state application.ApplyDirectorPlanState,
	stored application.DirectorDecisionRecord,
) bool {
	return stored.RequestRef == state.RequestRef && stored.RequestFingerprint == state.RequestFingerprint &&
		stored.GoalRef == state.GoalRef && stored.PrincipalRef == state.PrincipalRef &&
		stored.LeaseFence == state.LeaseFence && stored.SourceGoalRevision == state.ExpectedGoalRevision &&
		stored.SourcePlanGeneration == state.ExpectedPlanGeneration &&
		stored.AppliedGoalRevision == state.Goal.Revision() &&
		stored.AppliedPlanGeneration == state.Goal.PlanGeneration() &&
		stored.Reason == state.Decision.Reason
}
