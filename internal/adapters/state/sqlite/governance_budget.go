package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func requireEffectAdmission(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
	now time.Time,
) (application.EffectIntent, application.EffectApproval, bool, error) {
	intent, found, err := requireCandidateEffectIntent(ctx, transaction, candidate)
	if err != nil || !found {
		return application.EffectIntent{}, application.EffectApproval{}, false, err
	}
	var approvalRef string
	err = transaction.QueryRowContext(ctx, `
SELECT ref FROM effect_approvals
WHERE intent_ref = ?
ORDER BY decided_at DESC,
         CASE source WHEN 'explicit_decision' THEN 0 ELSE 1 END,
         CASE decision WHEN 'denied' THEN 0 ELSE 1 END,
         ref DESC
LIMIT 1`, intent.Ref).Scan(&approvalRef)
	if errors.Is(err, sql.ErrNoRows) {
		return intent, application.EffectApproval{}, false, nil
	}
	if err != nil {
		return application.EffectIntent{}, application.EffectApproval{}, false, mapDatabaseError(err)
	}
	approval, found, err := readEffectApprovalByRef(ctx, transaction, approvalRef)
	if err != nil || !found {
		if err == nil {
			err = invalid(errors.New("sqlite.claim_effect_approval_missing"))
		}
		return application.EffectIntent{}, application.EffectApproval{}, false, err
	}
	if application.ValidateEffectApproval(intent, approval) != nil ||
		(approval.Source == application.EffectApprovalSourceExplicitDecision &&
			application.ValidatePersistedEffectApproval(approval) != nil) ||
		approval.Decision != application.EffectApproved || approval.DecidedAt.After(now) ||
		(approval.Source == application.EffectApprovalSourceExplicitDecision && !now.Before(approval.ExpiresAt)) {
		return intent, approval, false, nil
	}
	proposerCurrent, err := authorizationMembershipCurrent(ctx, transaction, intent.Authority)
	if err != nil {
		return application.EffectIntent{}, application.EffectApproval{}, false, err
	}
	approverCurrent, err := authorizationMembershipCurrent(ctx, transaction, approval.AuthorizationReceipt)
	if err != nil {
		return application.EffectIntent{}, application.EffectApproval{}, false, err
	}
	if !proposerCurrent || !approverCurrent {
		return intent, approval, false, nil
	}
	return intent, approval, true, nil
}

func requireCandidateEffectIntent(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
) (application.EffectIntent, bool, error) {
	intent, err := readEffectIntent(ctx, transaction, candidate.action.EffectIntentRef)
	if err != nil {
		if application.IsStateError(err, application.StateNotFound) {
			return application.EffectIntent{}, false, nil
		}
		return application.EffectIntent{}, false, err
	}
	if intent.ActionRef != candidate.action.Ref || intent.ActionKind != candidate.action.Kind ||
		intent.Subject.ProjectRef != candidate.projectRef || intent.Subject.GoalRef != candidate.action.GoalRef ||
		intent.Subject.WorkItemRef != candidate.action.WorkItemRef ||
		intent.Subject.ExecutionRef != candidate.action.ExecutionRef ||
		intent.Subject.PlanGeneration != candidate.action.PlanGeneration {
		return application.EffectIntent{}, false,
			invalid(errors.New("sqlite.claim_effect_intent_causal_invalid"))
	}
	return intent, true, nil
}

func authorizationMembershipCurrent(
	ctx context.Context,
	transaction *sql.Tx,
	receipt identity.AuthorizationReceipt,
) (bool, error) {
	if receipt.Ref() == "" || receipt.Decision().Outcome() != identity.AuthorizationAllowed {
		return false, nil
	}
	persisted, err := readAuthorizationReceipt(ctx, transaction, receipt.Ref())
	if err != nil {
		if application.IsStateError(err, application.StateNotFound) {
			return false, nil
		}
		return false, err
	}
	if !sameAuthorizationReceipt(persisted, receipt) {
		return false, invalid(errors.New("sqlite.claim_authorization_tampered"))
	}
	request := receipt.Decision().Request()
	if !identity.RoleAllows(receipt.Decision().Role(), request.Permission()) {
		return false, nil
	}
	if receipt.Decision().Role() == identity.RolePlatformAdmin &&
		receipt.Decision().MembershipRevision() == 0 {
		return true, nil
	}
	membership, err := readMembership(ctx, transaction, request.Principal().Ref, request.ProjectRef())
	if err != nil {
		if application.IsStateError(err, application.StateNotFound) {
			return false, nil
		}
		return false, err
	}
	return membership.IsActive() && membership.Role() == receipt.Decision().Role() &&
		membership.Revision() == receipt.Decision().MembershipRevision(), nil
}

func prepareBudgetAdmission(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
) (governance.BudgetReservation, bool, bool, error) {
	intent := candidate.action.EffectIntent
	envelopes, err := requireBudgetEnvelopes(
		ctx, transaction, candidate.projectRef, candidate.action.GoalRef, intent.PolicyHash, intent.PolicyRevision,
	)
	if err != nil {
		return governance.BudgetReservation{}, false, false, err
	}
	reservation, found, err := readActiveBudgetReservation(ctx, transaction, candidate.action.Ref)
	if err != nil {
		return governance.BudgetReservation{}, false, false, err
	}
	if found {
		if !reservationMatchesCandidate(reservation, candidate, intent.PolicyHash) {
			return governance.BudgetReservation{}, false, false,
				invalid(errors.New("sqlite.budget_reservation_replay_invalid"))
		}
		// A reservation is bound to the logical effect, not a disposable lease.
		// Reuse it after lease expiry so crash replay cannot reserve or charge the
		// same idempotent external call twice.
		return reservation, true, true, nil
	}
	overrun, err := goalHasBudgetOverrun(ctx, transaction, candidate.action.GoalRef)
	if err != nil {
		return governance.BudgetReservation{}, false, false, err
	}
	if overrun {
		return governance.BudgetReservation{}, false, false, nil
	}
	for _, envelope := range envelopes {
		usage, err := readBudgetUsage(ctx, transaction, envelope.Scope, envelope.SubjectRef)
		if err != nil {
			return governance.BudgetReservation{}, false, false, err
		}
		requested, err := governance.Add(usage, intent.Demand.Resources)
		if err != nil {
			return governance.BudgetReservation{}, false, false, invalid(err)
		}
		fits, err := governance.Fits(envelope.Limit, requested)
		if err != nil {
			return governance.BudgetReservation{}, false, false, invalid(err)
		}
		if !fits {
			return governance.BudgetReservation{}, false, false, nil
		}
	}
	return governance.BudgetReservation{}, false, true, nil
}

func goalHasBudgetOverrun(ctx context.Context, source queryer, goalRef goal.GoalRef) (bool, error) {
	var found int
	err := source.QueryRowContext(ctx, `
SELECT 1 FROM budget_settlements settlement
JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref
WHERE reservation.goal_ref=? AND (settlement.overrun_tokens<>0 OR settlement.overrun_money_micros<>0
 OR settlement.overrun_active_time_ns<>0 OR settlement.overrun_process_slots<>0
 OR settlement.overrun_disk_bytes<>0) LIMIT 1`, goalRef.String()).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, mapDatabaseError(err)
	}
	return true, nil
}

func requireBudgetEnvelopes(
	ctx context.Context,
	source queryer,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	policyHash string,
	policyRevision uint64,
) ([]governance.BudgetEnvelope, error) {
	refs, err := readSingleColumn(ctx, source, `
SELECT ref FROM budget_envelopes
WHERE policy_hash = ? AND revision = ? AND (
    scope = 'deployment'
    OR (scope = 'project' AND subject_ref = ?)
    OR (scope = 'goal' AND subject_ref = ?)
)
ORDER BY scope, subject_ref, ref`, policyHash, int64(policyRevision), projectRef.String(), goalRef.String())
	if err != nil {
		return nil, err
	}
	if len(refs) != 3 {
		return nil, conflict(errors.New("sqlite.budget_envelopes_missing"))
	}
	result, err := readLedger(refs, func(ref string) (governance.BudgetEnvelope, error) {
		return readBudgetEnvelope(ctx, source, ref)
	})
	if err != nil {
		return nil, err
	}
	scopes := make(map[governance.BudgetScope]struct{}, 3)
	for _, envelope := range result {
		if _, duplicate := scopes[envelope.Scope]; duplicate {
			return nil, invalid(errors.New("sqlite.budget_envelope_scope_duplicate"))
		}
		scopes[envelope.Scope] = struct{}{}
	}
	return result, nil
}

func insertBudgetEnvelopes(
	ctx context.Context,
	transaction *sql.Tx,
	envelopes []governance.BudgetEnvelope,
) error {
	persisted, err := sqliteTableHasColumn(ctx, transaction, "budget_envelopes", "ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	if !persisted {
		return nil
	}
	if len(envelopes) == 0 {
		return nil
	}
	if len(envelopes) != 3 {
		return invalid(errors.New("sqlite.budget_envelopes_invalid"))
	}
	scopes := make(map[governance.BudgetScope]struct{}, 3)
	policyHash := envelopes[0].PolicyHash
	for _, envelope := range envelopes {
		if err := governance.ValidateBudgetEnvelope(envelope); err != nil || envelope.PolicyHash != policyHash {
			return invalid(errors.New("sqlite.budget_envelope_invalid"))
		}
		if _, duplicate := scopes[envelope.Scope]; duplicate {
			return invalid(errors.New("sqlite.budget_envelope_scope_duplicate"))
		}
		scopes[envelope.Scope] = struct{}{}
		if err := insertOrVerifyBudgetEnvelope(ctx, transaction, envelope); err != nil {
			return err
		}
	}
	return nil
}

func insertOrVerifyBudgetEnvelope(
	ctx context.Context,
	transaction *sql.Tx,
	envelope governance.BudgetEnvelope,
) error {
	limit := envelope.Limit
	if _, err := transaction.ExecContext(ctx, `
INSERT OR IGNORE INTO budget_envelopes(
    ref, subject_ref, scope, tokens, money_micros, currency, active_time_ns,
    process_slots, disk_bytes, revision, policy_hash, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, envelope.Ref, envelope.SubjectRef,
		string(envelope.Scope), limit.Tokens, limit.MoneyMicros, string(limit.Currency),
		limit.ActiveTimeNS, limit.ProcessSlots, limit.DiskBytes, int64(envelope.Revision),
		envelope.PolicyHash, requiredTime(envelope.CreatedAt)); err != nil {
		return mapDatabaseError(err)
	}
	stored, err := readBudgetEnvelope(ctx, transaction, envelope.Ref)
	if err != nil {
		return err
	}
	// CreatedAt is evidence of first materialization. Shared envelopes reuse it.
	if stored.Ref != envelope.Ref || stored.SubjectRef != envelope.SubjectRef ||
		stored.Scope != envelope.Scope || stored.Limit != envelope.Limit ||
		stored.Revision != envelope.Revision || stored.PolicyHash != envelope.PolicyHash {
		return conflict(errors.New("sqlite.budget_envelope_conflict"))
	}
	return nil
}

func readBudgetEnvelope(
	ctx context.Context, source queryer, ref string,
) (governance.BudgetEnvelope, error) {
	var envelope governance.BudgetEnvelope
	var scope, currency string
	var revision, createdAt int64
	err := source.QueryRowContext(ctx, `
SELECT ref, subject_ref, scope, tokens, money_micros, currency, active_time_ns,
       process_slots, disk_bytes, revision, policy_hash, created_at
FROM budget_envelopes WHERE ref = ?`, ref).Scan(&envelope.Ref, &envelope.SubjectRef, &scope,
		&envelope.Limit.Tokens, &envelope.Limit.MoneyMicros, &currency,
		&envelope.Limit.ActiveTimeNS, &envelope.Limit.ProcessSlots, &envelope.Limit.DiskBytes,
		&revision, &envelope.PolicyHash, &createdAt)
	if err != nil {
		return governance.BudgetEnvelope{}, mapDatabaseError(err)
	}
	if revision <= 0 {
		return governance.BudgetEnvelope{}, invalid(errors.New("sqlite.budget_envelope_revision_invalid"))
	}
	envelope.Scope = governance.BudgetScope(scope)
	envelope.Limit.Currency = governance.Currency(currency)
	envelope.Revision = uint64(revision)
	envelope.CreatedAt = time.Unix(0, createdAt).UTC()
	if err := governance.ValidateBudgetEnvelope(envelope); err != nil {
		return governance.BudgetEnvelope{}, invalid(err)
	}
	return envelope, nil
}
