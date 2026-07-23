package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
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
	decision, err := repository.applyDirectorPlanChange(ctx, transaction, state)
	if err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.DirectorDecisionRecord{}, false, err
	}
	return decision, true, nil
}

func (repository *Repository) applyDirectorPlanChange(
	ctx context.Context, transaction *sql.Tx, state application.ApplyDirectorPlanState,
) (application.DirectorDecisionRecord, error) {
	now, err := repository.transactionTime()
	if err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := requireLiveDirectorLease(ctx, transaction, state, now); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	current, err := readGoalRecord(ctx, transaction, state.GoalRef.String())
	if err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := validateDirectorPlanTransition(state, current); err != nil {
		return application.DirectorDecisionRecord{}, conflict(err)
	}
	if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := updateDirectorExistingWorkItems(ctx, transaction, current.Goal, state.Goal); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := insertDirectorPlanDelta(
		ctx, transaction, current.Goal, state.Goal, len(state.NewWorkItemAuthorities) > 0,
	); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := insertWorkItemAuthorities(
		ctx, transaction, state.GoalRef.String(), state.NewWorkItemAuthorities,
	); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	for _, execution := range state.UpdatedExecutions {
		stored, found := sqliteExecutionByRef(current.Executions, execution.Ref)
		if !found {
			return application.DirectorDecisionRecord{},
				conflict(errors.New("sqlite.director_execution_missing"))
		}
		if err := updateExecutionCAS(ctx, transaction, execution, stored.State); err != nil {
			return application.DirectorDecisionRecord{}, err
		}
	}
	for _, actionRef := range state.RetireActionRefs {
		if err := consumeRetiredAction(ctx, transaction, actionRef, state.Decision.Ref, state.OperationAt); err != nil {
			return application.DirectorDecisionRecord{}, err
		}
	}
	if err := insertScheduled(ctx, transaction, state.NewExecutions, state.NewActions); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := requireReadyExecutions(ctx, transaction, state.Goal); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := insertEvents(ctx, transaction, state.Events); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := insertDirectorDecision(ctx, transaction, state.Decision, state.ProjectRef); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	commitNow, err := repository.transactionTime()
	if err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	if err := requireLiveDirectorLease(ctx, transaction, state, commitNow); err != nil {
		return application.DirectorDecisionRecord{}, err
	}
	decision, found, err := readDirectorDecisionByRequest(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.GoalRef, state.RequestRef,
	)
	if err != nil || !found {
		if err == nil {
			err = conflict(errors.New("sqlite.director_decision_missing_after_write"))
		}
		return application.DirectorDecisionRecord{}, err
	}
	return decision, nil
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
		decision.SourceWorkItemRevision != state.ExpectedWorkItemRevision ||
		decision.AppliedGoalRevision != state.Goal.Revision() ||
		decision.AppliedPlanGeneration != state.Goal.PlanGeneration() || !validText(decision.Reason) ||
		decision.DecidedAt.IsZero() || !decision.DecidedAt.Equal(state.OperationAt) ||
		!sameAuthorizationReceipt(decision.AuthorizationReceipt, state.AuthorizationReceipt) {
		return errors.New("sqlite.director_decision_invalid")
	}
	if decision.Cause == "" {
		if decision.SourceWorkItemRef.String() != "" || decision.SourceWorkItemRevision != 0 ||
			decision.SourceExecutionRef.String() != "" || decision.SourceExecutionAttempt != 0 ||
			len(state.UpdatedExecutions) != 0 || len(state.RetireActionRefs) != 0 {
			return errors.New("sqlite.director_extension_source_invalid")
		}
	} else if decision.SourceWorkItemRef.String() == "" || decision.SourceExecutionRef.String() == "" ||
		decision.SourceExecutionAttempt == 0 || state.ExpectedWorkItemRevision == 0 {
		return errors.New("sqlite.director_replan_source_invalid")
	}
	if decision.Cause == goal.ReplanCauseGovernanceDecision {
		if !validReviewDigest(string(decision.CouncilSubjectDigest)) ||
			!validText(decision.CouncilDecisionRef) ||
			!validReviewDigest(string(decision.CouncilDecisionDigest)) {
			return errors.New("sqlite.director_council_fence_invalid")
		}
	} else if decision.CouncilSubjectDigest != "" || decision.CouncilDecisionRef != "" ||
		decision.CouncilDecisionDigest != "" {
		return errors.New("sqlite.director_council_fence_unexpected")
	}
	if err := validateDirectorReplanDelta(state); err != nil {
		return err
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

func validateDirectorReplanDelta(state application.ApplyDirectorPlanState) error {
	switch state.Decision.Cause {
	case "", goal.ReplanCauseExecutionStopped, goal.ReplanCauseExecutionFailed,
		goal.ReplanCauseReviewChangesRequested:
		if len(state.UpdatedExecutions) != 0 || len(state.RetireActionRefs) != 0 {
			return errors.New("sqlite.director_replan_delta_unexpected")
		}
	case goal.ReplanCauseSplitPending:
		if !validDirectorCanceledExecutionDelta(state, true) {
			return errors.New("sqlite.director_split_delta_invalid")
		}
	case goal.ReplanCauseGovernanceDecision:
		if !validDirectorCanceledExecutionDelta(state, false) {
			return errors.New("sqlite.director_council_delta_invalid")
		}
	default:
		return errors.New("sqlite.director_replan_cause_invalid")
	}
	return nil
}

func validDirectorCanceledExecutionDelta(state application.ApplyDirectorPlanState, retireLaunch bool) bool {
	if len(state.UpdatedExecutions) != 1 || len(state.RetireActionRefs) != 0 && !retireLaunch ||
		len(state.RetireActionRefs) != 1 && retireLaunch {
		return false
	}
	execution := state.UpdatedExecutions[0]
	return execution.Ref == state.Decision.SourceExecutionRef &&
		execution.WorkItemRef == state.Decision.SourceWorkItemRef &&
		execution.AttemptNo == state.Decision.SourceExecutionAttempt &&
		execution.State == application.ExecutionCanceled &&
		execution.FailureCode == "application.execution_superseded" &&
		execution.FinishedAt.Equal(state.OperationAt) &&
		(!retireLaunch || state.RetireActionRefs[0] == "action:launch:"+execution.Ref.String())
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
		!reflect.DeepEqual(after.Phases[:len(before.Phases)], before.Phases) {
		return errors.New("sqlite.director_plan_cas_conflict")
	}
	if state.Decision.Cause == "" {
		if !reflect.DeepEqual(after.WorkItems[:len(before.WorkItems)], before.WorkItems) {
			return errors.New("sqlite.director_plan_cas_conflict")
		}
		return nil
	}
	beforeSource, found := current.Goal.WorkItem(state.Decision.SourceWorkItemRef)
	if !found || beforeSource.Revision() != state.ExpectedWorkItemRevision {
		return errors.New("sqlite.director_replan_item_conflict")
	}
	afterSource, found := state.Goal.WorkItem(state.Decision.SourceWorkItemRef)
	if !found || afterSource.State() != goal.WorkItemStateSuperseded ||
		afterSource.Revision() != beforeSource.Revision()+1 {
		return errors.New("sqlite.director_replan_transition_invalid")
	}
	causalExecution, found := sqliteExecutionByRef(current.Executions, state.Decision.SourceExecutionRef)
	if !found || causalExecution.WorkItemRef != beforeSource.Ref() ||
		causalExecution.AttemptNo != state.Decision.SourceExecutionAttempt {
		return errors.New("sqlite.director_replan_execution_conflict")
	}
	switch state.Decision.Cause {
	case goal.ReplanCauseSplitPending:
		if causalExecution.State != application.ExecutionQueued {
			return errors.New("sqlite.director_replan_execution_conflict")
		}
	case goal.ReplanCauseExecutionStopped:
		if causalExecution.State != application.ExecutionStopped {
			return errors.New("sqlite.director_replan_execution_conflict")
		}
	case goal.ReplanCauseExecutionFailed:
		if causalExecution.State != application.ExecutionFailed {
			return errors.New("sqlite.director_replan_execution_conflict")
		}
	case goal.ReplanCauseReviewChangesRequested:
		change, changeFound := directorChangeForExecution(current, causalExecution)
		interrupt, interrupted := beforeSource.InterruptCause()
		if causalExecution.State != application.ExecutionFailed ||
			causalExecution.FailureCode != string(goal.ReplanCauseReviewChangesRequested) ||
			!changeFound || !interrupted || interrupt != goal.WorkItemInterruptExecutionFailed ||
			!application.FailedReviewPreservesCandidate(current, beforeSource, causalExecution, change) {
			return errors.New("sqlite.director_review_replan_causality_invalid")
		}
	case goal.ReplanCauseGovernanceDecision:
		if !validDirectorCouncilReplan(current, beforeSource, causalExecution, state.Decision) {
			return errors.New("sqlite.director_council_replan_causality_invalid")
		}
	default:
		return errors.New("sqlite.director_replan_cause_invalid")
	}
	for index := range before.WorkItems {
		if before.WorkItems[index].Ref == state.Decision.SourceWorkItemRef.String() {
			continue
		}
		if !reflect.DeepEqual(after.WorkItems[index], before.WorkItems[index]) {
			return errors.New("sqlite.director_replan_unrelated_mutation")
		}
	}
	return nil
}

func validDirectorCouncilReplan(current application.GoalRecord, source goal.WorkItem,
	author application.ExecutionRecord, decision application.DirectorDecisionRecord,
) bool {
	change, changeFound := directorChangeForExecution(current, author)
	round, roundFound := councilRoundFor(current.CouncilRounds, decision.CouncilSubjectDigest)
	if author.State != application.ExecutionAwaitingIntegration || !changeFound || !roundFound ||
		round.WorkItemRef != source.Ref() || round.ChangeSetRef != change.Ref.String() ||
		round.Subject.SpecHash != author.SpecHash ||
		application.ValidatePersistedCouncilSubject(current, round.Subject) != nil {
		return false
	}
	for _, intent := range current.EffectIntents {
		if intent.ActionKind == application.ActionIntegrateChange && intent.Subject.ExecutionRef == author.Ref {
			return false
		}
	}
	for _, receipt := range current.IntegrationReceipts {
		if receipt.ChangeRef == change.Ref {
			return false
		}
	}
	for _, persisted := range current.CouncilDecisions {
		if persisted.Ref != decision.CouncilDecisionRef ||
			persisted.SubjectDigest != decision.CouncilSubjectDigest ||
			persisted.DecisionDigest != decision.CouncilDecisionDigest ||
			persisted.Decision.SubjectDigest != string(decision.CouncilSubjectDigest) ||
			persisted.Decision.Digest != string(decision.CouncilDecisionDigest) {
			continue
		}
		switch persisted.Decision.Outcome {
		case council.OutcomeRejected, council.OutcomeNoConsensus, council.OutcomeBlockedSecurity:
			return true
		}
	}
	return false
}

func directorChangeForExecution(record application.GoalRecord,
	execution application.ExecutionRecord,
) (application.ChangeSet, bool) {
	var result application.ChangeSet
	found := false
	for _, change := range record.ChangeSets {
		if change.ExecutionRef != execution.Ref {
			continue
		}
		if found {
			return application.ChangeSet{}, false
		}
		result, found = change, true
	}
	return result, found
}

func updateDirectorExistingWorkItems(
	ctx context.Context,
	transaction *sql.Tx,
	before goal.Goal,
	after goal.Goal,
) error {
	for _, previous := range before.WorkItems() {
		current, found := after.WorkItem(previous.Ref())
		if !found {
			return conflict(errors.New("sqlite.director_work_item_removed"))
		}
		if current.Revision() == previous.Revision() {
			continue
		}
		if err := updateWorkItemCAS(ctx, transaction, current, previous.Revision()); err != nil {
			return err
		}
	}
	return nil
}

func insertDirectorPlanDelta(
	ctx context.Context,
	transaction *sql.Tx,
	before goal.Goal,
	after goal.Goal,
	governed bool,
) error {
	beforeSnapshot := before.Snapshot()
	afterSnapshot := after.Snapshot()
	if err := insertGoalPhases(ctx, transaction, afterSnapshot, len(beforeSnapshot.Phases)); err != nil {
		return err
	}
	return insertWorkItems(
		ctx, transaction, afterSnapshot, len(beforeSnapshot.WorkItems), governed,
	)
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
    source_goal_revision, source_plan_generation, cause,
    source_work_item_ref, source_work_item_revision,
    source_execution_ref, source_execution_attempt,
    applied_goal_revision, applied_plan_generation, reason, decided_at,
    council_subject_digest,council_decision_ref,council_decision_digest
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		decision.Ref, decision.RequestRef, decision.RequestFingerprint,
		decision.AuthorizationReceipt.Ref(), decision.GoalRef.String(), projectRef.String(),
		decision.PrincipalRef.String(), int64(decision.LeaseFence),
		int64(decision.SourceGoalRevision), int64(decision.SourcePlanGeneration),
		string(decision.Cause), nullableString(decision.SourceWorkItemRef.String()),
		int64(decision.SourceWorkItemRevision), nullableString(decision.SourceExecutionRef.String()),
		int64(decision.SourceExecutionAttempt),
		int64(decision.AppliedGoalRevision), int64(decision.AppliedPlanGeneration),
		decision.Reason, requiredTime(decision.DecidedAt),
		string(decision.CouncilSubjectDigest), nullableString(decision.CouncilDecisionRef),
		nullableString(string(decision.CouncilDecisionDigest)),
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
	var fence, sourceRevision, sourceGeneration, sourceItemRevision, sourceExecutionAttempt int64
	var appliedRevision, appliedGeneration, decidedAt int64
	var cause string
	var sourceItem, sourceExecution sql.NullString
	var councilSubject string
	var councilDecisionRef, councilDecisionDigest sql.NullString
	councilPersisted, err := sqliteTableHasColumn(ctx, source, "director_decisions", "council_subject_digest")
	if err != nil {
		return application.DirectorDecisionRecord{}, false, mapDatabaseError(err)
	}
	councilProjection := "'',NULL,NULL"
	if councilPersisted {
		councilProjection = "council_subject_digest,council_decision_ref,council_decision_digest"
	}
	err = source.QueryRowContext(ctx, fmt.Sprintf(`
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       goal_ref, principal_ref, lease_fence,
       source_goal_revision, source_plan_generation, cause,
       source_work_item_ref, source_work_item_revision,
       source_execution_ref, source_execution_attempt,
       applied_goal_revision, applied_plan_generation, reason, decided_at,%s
FROM director_decisions
WHERE principal_ref = ? AND project_ref = ? AND goal_ref = ? AND request_ref = ?`, councilProjection),
		principalRef.String(), projectRef.String(), goalRef.String(), requestRef,
	).Scan(
		&decision.Ref, &decision.RequestRef, &decision.RequestFingerprint, &authorizationRef,
		&goalValue, &principalValue, &fence, &sourceRevision, &sourceGeneration,
		&cause, &sourceItem, &sourceItemRevision, &sourceExecution, &sourceExecutionAttempt,
		&appliedRevision, &appliedGeneration, &decision.Reason, &decidedAt,
		&councilSubject, &councilDecisionRef, &councilDecisionDigest,
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
	decision.Cause = goal.ReplanCause(cause)
	decision.SourceWorkItemRevision = goal.Revision(sourceItemRevision)
	decision.SourceExecutionAttempt = uint64(sourceExecutionAttempt)
	if sourceItem.Valid {
		if decision.SourceWorkItemRef, err = goal.NewWorkItemRef(sourceItem.String); err != nil {
			return application.DirectorDecisionRecord{}, false, invalid(err)
		}
	}
	if sourceExecution.Valid {
		if decision.SourceExecutionRef, err = goal.NewExecutionRef(sourceExecution.String); err != nil {
			return application.DirectorDecisionRecord{}, false, invalid(err)
		}
	}
	decision.AppliedGoalRevision = goal.Revision(appliedRevision)
	decision.AppliedPlanGeneration = goal.PlanGeneration(appliedGeneration)
	decision.CouncilSubjectDigest = application.CouncilSubjectDigest(councilSubject)
	if councilDecisionRef.Valid {
		decision.CouncilDecisionRef = councilDecisionRef.String
	}
	if councilDecisionDigest.Valid {
		decision.CouncilDecisionDigest = application.CouncilSubjectDigest(councilDecisionDigest.String)
	}
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
		stored.Cause == state.Decision.Cause && stored.SourceWorkItemRef == state.Decision.SourceWorkItemRef &&
		stored.SourceWorkItemRevision == state.ExpectedWorkItemRevision &&
		stored.SourceExecutionRef == state.Decision.SourceExecutionRef &&
		stored.SourceExecutionAttempt == state.Decision.SourceExecutionAttempt &&
		stored.AppliedGoalRevision == state.Goal.Revision() &&
		stored.AppliedPlanGeneration == state.Goal.PlanGeneration() &&
		stored.Reason == state.Decision.Reason
}
