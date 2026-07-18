package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func insertWorkItemAuthorities(
	ctx context.Context,
	transaction *sql.Tx,
	goalRef string,
	authorities []application.WorkItemAuthority,
) error {
	persisted, err := sqliteTableHasColumn(ctx, transaction, "work_item_authorities", "work_item_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	if !persisted {
		return nil
	}
	for _, authority := range authorities {
		if authority.WorkItemRef.String() == "" || authority.PrincipalRef.String() == "" ||
			(authority.Permission != identity.PermissionGoalsCreate && authority.Permission != identity.PermissionGoalsDirect) ||
			(authority.Source != application.EffectApprovalSourceGoalConfirmation &&
				authority.Source != application.EffectApprovalSourceDirectorDecision) ||
			(authority.Source == application.EffectApprovalSourceGoalConfirmation) !=
				(authority.Permission == identity.PermissionGoalsCreate) ||
			authority.AuthorizationReceipt.Ref() == "" || authority.RecordedAt.IsZero() {
			return invalid(errors.New("sqlite.work_item_authority_invalid"))
		}
		persistedReceipt, err := readAuthorizationReceipt(ctx, transaction, authority.AuthorizationReceipt.Ref())
		if err != nil || !sameAuthorizationReceipt(persistedReceipt, authority.AuthorizationReceipt) {
			if err != nil {
				return err
			}
			return conflict(errors.New("sqlite.work_item_authority_receipt_conflict"))
		}
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_item_authorities(
    work_item_ref, goal_ref, principal_ref, permission, source,
    authorization_receipt_ref, recorded_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`, authority.WorkItemRef.String(), goalRef,
			authority.PrincipalRef.String(), string(authority.Permission), string(authority.Source),
			authority.AuthorizationReceipt.Ref(), requiredTime(authority.RecordedAt)); err != nil {
			return mapDatabaseError(err)
		}
	}
	return nil
}

func insertEffectAdmission(ctx context.Context, transaction *sql.Tx, action application.ActionRecord) error {
	intent := action.EffectIntent
	if action.EffectIntentRef != intent.Ref || intent.ActionRef != action.Ref ||
		intent.ActionKind != action.Kind || intent.Subject.GoalRef != action.GoalRef ||
		intent.Subject.WorkItemRef != action.WorkItemRef || intent.Subject.ExecutionRef != action.ExecutionRef ||
		intent.Subject.PlanGeneration != action.PlanGeneration {
		return invalid(errors.New("sqlite.effect_intent_action_mismatch"))
	}
	if err := application.ValidateEffectIntent(intent); err != nil {
		return invalid(err)
	}
	persistedAuthority, err := readAuthorizationReceipt(ctx, transaction, intent.Authority.Ref())
	if err != nil || !sameAuthorizationReceipt(persistedAuthority, intent.Authority) {
		if err != nil {
			return err
		}
		return conflict(errors.New("sqlite.effect_intent_authority_conflict"))
	}
	resources := intent.Demand.Resources
	_, err = transaction.ExecContext(ctx, `
INSERT INTO effect_intents(
    ref, request_ref, request_fingerprint, action_ref, action_kind, kind,
    project_ref, goal_ref, work_item_ref, execution_ref, plan_generation,
    app_spec_generation, spec_hash, actor_ref, proposed_by_ref, permission,
    authority_receipt_ref, demand_ref, demand_tokens, demand_money_micros,
    demand_currency, demand_active_time_ns, demand_process_slots, demand_disk_bytes,
    security_criticality, reasoning_effort, policy_hash, policy_revision,
    quota_retry_delay_ns, approval_ttl_ns, target_digest, idempotency_key, created_at, digest
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		intent.Ref, intent.RequestRef, intent.RequestFingerprint, intent.ActionRef,
		string(intent.ActionKind), string(intent.Kind), intent.Subject.ProjectRef.String(),
		intent.Subject.GoalRef.String(), intent.Subject.WorkItemRef.String(), intent.Subject.ExecutionRef.String(),
		int64(intent.Subject.PlanGeneration), int64(intent.Subject.AppSpecGeneration), intent.Subject.SpecHash,
		intent.Subject.ActorRef.String(), intent.ProposedBy.String(), string(intent.Permission), intent.Authority.Ref(),
		intent.Demand.Ref, resources.Tokens, resources.MoneyMicros, string(resources.Currency),
		resources.ActiveTimeNS, resources.ProcessSlots, resources.DiskBytes,
		string(intent.SecurityCriticality), string(intent.ReasoningEffort), intent.PolicyHash,
		int64(intent.PolicyRevision), int64(intent.QuotaRetryDelay), int64(intent.ApprovalTTL), intent.TargetDigest,
		intent.IdempotencyKey, requiredTime(intent.CreatedAt), intent.Digest,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if action.EffectApproval != nil {
		return insertEffectApproval(ctx, transaction, intent, *action.EffectApproval)
	}
	return nil
}

func insertEffectApproval(
	ctx context.Context,
	transaction *sql.Tx,
	intent application.EffectIntent,
	approval application.EffectApproval,
) error {
	if err := application.ValidateEffectApproval(intent, approval); err != nil {
		return invalid(err)
	}
	persistedAuthorization, err := readAuthorizationReceipt(ctx, transaction, approval.AuthorizationReceipt.Ref())
	if err != nil || !sameAuthorizationReceipt(persistedAuthorization, approval.AuthorizationReceipt) {
		if err != nil {
			return err
		}
		return conflict(errors.New("sqlite.effect_approval_authority_conflict"))
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO effect_approvals(
    ref, request_ref, request_fingerprint, intent_ref, intent_digest,
    project_ref, goal_ref, work_item_ref, execution_ref, plan_generation,
    app_spec_generation, spec_hash, actor_ref, proposed_by_ref, decided_by_ref,
    decision, source, security_criticality, reason, idempotency_key,
    policy_hash, policy_revision, target_digest, authorization_receipt_ref, decided_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		approval.Ref, approval.RequestRef, approval.RequestFingerprint, approval.IntentRef,
		approval.IntentDigest, approval.Subject.ProjectRef.String(), approval.Subject.GoalRef.String(),
		approval.Subject.WorkItemRef.String(), approval.Subject.ExecutionRef.String(),
		int64(approval.Subject.PlanGeneration), int64(approval.Subject.AppSpecGeneration),
		approval.Subject.SpecHash, approval.Subject.ActorRef.String(), approval.ProposedBy.String(),
		approval.DecidedBy.String(), string(approval.Decision), string(approval.Source),
		string(approval.SecurityCriticality), approval.Reason, approval.IdempotencyKey,
		approval.PolicyHash, int64(approval.PolicyRevision), approval.TargetDigest,
		approval.AuthorizationReceipt.Ref(), requiredTime(approval.DecidedAt), storedTime(approval.ExpiresAt),
	)
	return mapDatabaseError(err)
}

func (repository *Repository) EffectReplay(
	ctx context.Context,
	request application.EffectReplayRequest,
) (application.EffectApproval, bool, error) {
	if !validText(request.RequestRef) || !validText(request.RequestFingerprint) ||
		request.PrincipalRef.String() == "" || request.ProjectRef.String() == "" ||
		request.GoalRef.String() == "" || !validText(request.IntentRef) || !validText(request.IntentDigest) {
		return application.EffectApproval{}, false, invalid(errors.New("sqlite.effect_replay_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	defer transaction.Rollback()
	approval, found, err := readEffectApprovalByRequest(
		ctx, transaction, request.PrincipalRef.String(), request.RequestRef,
	)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	if found && (approval.RequestFingerprint != request.RequestFingerprint ||
		approval.Subject.ProjectRef != request.ProjectRef || approval.Subject.GoalRef != request.GoalRef ||
		approval.IntentRef != request.IntentRef || approval.IntentDigest != request.IntentDigest) {
		return application.EffectApproval{}, false, conflict(errors.New("sqlite.effect_replay_conflict"))
	}
	if err := commit(transaction); err != nil {
		return application.EffectApproval{}, false, err
	}
	return approval, found, nil
}

func (repository *Repository) DecideEffect(
	ctx context.Context,
	state application.DecideEffectState,
) (application.EffectApproval, bool, error) {
	if state.OperationAt.IsZero() || !state.Approval.DecidedAt.Equal(state.OperationAt.UTC()) ||
		state.RequestRef != state.Approval.RequestRef ||
		state.RequestFingerprint != state.Approval.RequestFingerprint ||
		state.IntentRef != state.Approval.IntentRef || state.IntentDigest != state.Approval.IntentDigest ||
		state.PrincipalRef != state.Approval.DecidedBy || state.ProjectRef != state.Approval.Subject.ProjectRef ||
		state.GoalRef != state.Approval.Subject.GoalRef ||
		!sameAuthorizationReceipt(state.AuthorizationReceipt, state.Approval.AuthorizationReceipt) ||
		application.ValidatePersistedEffectApproval(state.Approval) != nil {
		return application.EffectApproval{}, false, invalid(errors.New("sqlite.effect_decision_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	defer transaction.Rollback()
	now, err := repository.transactionTime()
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	if state.OperationAt.After(now) {
		return application.EffectApproval{}, false, invalid(errors.New("sqlite.effect_decision_time_future"))
	}
	if replay, found, readErr := readEffectApprovalByRequest(
		ctx, transaction, state.PrincipalRef.String(), state.RequestRef,
	); readErr != nil {
		return application.EffectApproval{}, false, readErr
	} else if found {
		if replay != state.Approval {
			return application.EffectApproval{}, false, conflict(errors.New("sqlite.effect_decision_replay_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.EffectApproval{}, false, err
		}
		return replay, false, nil
	}
	intent, err := readEffectIntent(ctx, transaction, state.IntentRef)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	if intent.Digest != state.IntentDigest || application.ValidateEffectApproval(intent, state.Approval) != nil {
		return application.EffectApproval{}, false, conflict(errors.New("sqlite.effect_decision_intent_conflict"))
	}
	request := state.AuthorizationReceipt.Decision().Request()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionEffectsApprove, request.ResourceRef(),
	); err != nil {
		return application.EffectApproval{}, false, err
	}
	if err := insertEffectApproval(ctx, transaction, intent, state.Approval); err != nil {
		return application.EffectApproval{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.EffectApproval{}, false, err
	}
	return state.Approval, true, nil
}

func (repository *Repository) RecordEffectAttempt(
	ctx context.Context,
	state application.RecordEffectAttemptState,
) (application.EffectAttempt, bool, error) {
	if state.OperationAt.IsZero() || !state.Attempt.StartedAt.Equal(state.OperationAt.UTC()) ||
		validateClaim(state.Claim) != nil ||
		!effectAttemptMatchesClaim(state.Attempt, state.Claim) {
		return application.EffectAttempt{}, false, invalid(errors.New("sqlite.effect_attempt_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.EffectAttempt{}, false, err
	}
	defer transaction.Rollback()
	if err := repository.requireLiveLease(state.Claim); err != nil {
		return application.EffectAttempt{}, false, err
	}
	if err := requireClaim(ctx, transaction, state.Claim); err != nil {
		return application.EffectAttempt{}, false, err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return application.EffectAttempt{}, false, err
	}
	if state.OperationAt.After(now) {
		return application.EffectAttempt{}, false, invalid(errors.New("sqlite.effect_attempt_time_future"))
	}
	candidate := claimCandidate{
		action: state.Claim.Action, projectRef: state.Claim.Action.EffectIntent.Subject.ProjectRef,
		governanceVersion: 1,
	}
	intent, approval, admitted, err := requireEffectAdmission(ctx, transaction, candidate, now)
	if err != nil {
		return application.EffectAttempt{}, false, err
	}
	if !admitted || intent != state.Claim.Action.EffectIntent || approval != state.Claim.EffectApproval {
		return application.EffectAttempt{}, false,
			conflict(errors.New("sqlite.effect_attempt_authority_stale"))
	}
	if state.Attempt.StartedAt.Before(intent.CreatedAt) || state.Attempt.StartedAt.Before(approval.DecidedAt) ||
		(approval.Source == application.EffectApprovalSourceExplicitDecision &&
			!state.Attempt.StartedAt.Before(approval.ExpiresAt)) {
		return application.EffectAttempt{}, false, invalid(errors.New("sqlite.effect_attempt_time_invalid"))
	}
	if err := requireEffectAttemptFrontier(ctx, transaction, state, candidate); err != nil {
		return application.EffectAttempt{}, false, err
	}
	stored, found, err := readEffectAttemptByFence(
		ctx, transaction, state.Attempt.ActionRef, state.Attempt.ActionFence,
	)
	if err != nil {
		return application.EffectAttempt{}, false, err
	}
	if found {
		if stored != state.Attempt {
			return application.EffectAttempt{}, false, conflict(errors.New("sqlite.effect_attempt_replay_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.EffectAttempt{}, false, err
		}
		return stored, false, nil
	}
	if err := insertEffectAttempt(ctx, transaction, state.Attempt); err != nil {
		return application.EffectAttempt{}, false, err
	}
	if err := repository.requireLiveLease(state.Claim); err != nil {
		return application.EffectAttempt{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.EffectAttempt{}, false, err
	}
	return state.Attempt, true, nil
}

func requireEffectAttemptFrontier(ctx context.Context, transaction *sql.Tx, state application.RecordEffectAttemptState, candidate claimCandidate) error {
	frontier, eventKind, reservationRef, intentRef := "running", "execution.accepted", "", ""
	if state.Claim.Action.Kind == application.ActionLaunchAgent {
		if err := requireClaimBudgetReservation(ctx, transaction, state.Claim, candidate); err != nil {
			return err
		}
		frontier, eventKind, reservationRef, intentRef = "dispatching", "execution.dispatching", state.Claim.BudgetReservationRef, state.Attempt.IntentRef
	}
	var bound int
	err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE ref=? AND state=? AND (?='' OR governance_version=1)
AND (?='' OR (budget_reservation_ref=? AND effect_intent_ref=?)) AND EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=? AND event.work_item_ref=? AND event.execution_ref=? AND event.kind=? AND event.occurred_at<=?)`,
		state.Attempt.Subject.ExecutionRef.String(), frontier, reservationRef, reservationRef, reservationRef, intentRef,
		state.Attempt.Subject.GoalRef.String(), state.Attempt.Subject.WorkItemRef.String(), state.Attempt.Subject.ExecutionRef.String(), eventKind, requiredTime(state.Attempt.StartedAt)).Scan(&bound)
	if err != nil {
		return mapDatabaseError(err)
	}
	if bound != 1 || (reservationRef != "" && state.Attempt.StartedAt.Before(state.Claim.BudgetReservation.ReservedAt)) {
		return conflict(errors.New("sqlite.effect_attempt_dispatch_frontier_invalid"))
	}
	return nil
}

func effectAttemptMatchesClaim(attempt application.EffectAttempt, claim application.ActionClaim) bool {
	return validText(attempt.Ref) && attempt.IntentRef == claim.Action.EffectIntentRef &&
		attempt.IntentDigest == claim.Action.EffectIntent.Digest &&
		attempt.ApprovalRef == claim.EffectApproval.Ref && attempt.Subject == claim.Action.EffectIntent.Subject &&
		attempt.ActionRef == claim.Action.Ref && attempt.ActionFence == claim.Fence &&
		attempt.WorkerRef == claim.WorkerRef && validText(attempt.IdempotencyKey) && !attempt.StartedAt.IsZero()
}

func insertEffectAttempt(ctx context.Context, transaction *sql.Tx, attempt application.EffectAttempt) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO effect_attempts(
    ref, intent_ref, intent_digest, approval_ref, project_ref, goal_ref,
    work_item_ref, execution_ref, plan_generation, app_spec_generation,
    spec_hash, actor_ref, action_ref, action_fence, worker_ref,
    idempotency_key, started_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attempt.Ref, attempt.IntentRef, attempt.IntentDigest, attempt.ApprovalRef,
		attempt.Subject.ProjectRef.String(), attempt.Subject.GoalRef.String(),
		attempt.Subject.WorkItemRef.String(), attempt.Subject.ExecutionRef.String(),
		int64(attempt.Subject.PlanGeneration), int64(attempt.Subject.AppSpecGeneration),
		attempt.Subject.SpecHash, attempt.Subject.ActorRef.String(), attempt.ActionRef,
		int64(attempt.ActionFence), attempt.WorkerRef, attempt.IdempotencyKey,
		requiredTime(attempt.StartedAt),
	)
	return mapDatabaseError(err)
}
