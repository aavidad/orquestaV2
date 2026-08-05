package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

const effectApprovalSelect = `
SELECT ref, request_ref, request_fingerprint, intent_ref, intent_digest,
       project_ref, goal_ref, work_item_ref, execution_ref, plan_generation,
       app_spec_generation, spec_hash, actor_ref, proposed_by_ref, decided_by_ref,
       decision, source, security_criticality, reason, idempotency_key,
       policy_hash, policy_revision, target_digest, authorization_receipt_ref, decided_at, expires_at
FROM effect_approvals`

func readEffectIntent(ctx context.Context, source queryer, ref string) (application.EffectIntent, error) {
	var intent application.EffectIntent
	var actionKind, kind, projectRef, goalRef, workRef, executionRef, actorRef, proposedBy string
	var permission, currency, criticality, effort, authorityRef string
	var councilSubject string
	var councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest sql.NullString
	var planGeneration, appSpecGeneration, policyRevision, quotaRetryDelay, approvalTTL, createdAt int64
	var resources governance.ResourceVector
	councilPersisted, err := sqliteTableHasColumn(ctx, source, "effect_intents", "council_subject_digest")
	if err != nil {
		return application.EffectIntent{}, mapDatabaseError(err)
	}
	councilProjection := "'',NULL,NULL,NULL,NULL"
	if councilPersisted {
		councilProjection = "council_subject_digest,council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest"
	}
	err = source.QueryRowContext(ctx, fmt.Sprintf(`
SELECT ref, request_ref, request_fingerprint, action_ref, action_kind, kind,
       project_ref, goal_ref, work_item_ref, execution_ref, plan_generation,
       app_spec_generation, spec_hash, actor_ref, proposed_by_ref, permission,
       authority_receipt_ref, demand_ref, demand_tokens, demand_money_micros,
       demand_currency, demand_active_time_ns, demand_process_slots, demand_disk_bytes,
       security_criticality, reasoning_effort, policy_hash, policy_revision,
       quota_retry_delay_ns, approval_ttl_ns, target_digest, idempotency_key, created_at, digest,
       %s
FROM effect_intents WHERE ref = ?`, councilProjection), ref).Scan(
		&intent.Ref, &intent.RequestRef, &intent.RequestFingerprint, &intent.ActionRef,
		&actionKind, &kind, &projectRef, &goalRef, &workRef, &executionRef,
		&planGeneration, &appSpecGeneration, &intent.Subject.SpecHash, &actorRef, &proposedBy,
		&permission, &authorityRef, &intent.Demand.Ref, &resources.Tokens, &resources.MoneyMicros,
		&currency, &resources.ActiveTimeNS, &resources.ProcessSlots, &resources.DiskBytes,
		&criticality, &effort, &intent.PolicyHash, &policyRevision, &quotaRetryDelay, &approvalTTL,
		&intent.TargetDigest,
		&intent.IdempotencyKey, &createdAt, &intent.Digest, &councilSubject, &councilDecisionRef,
		&councilDecisionDigest, &councilSkipRef, &councilSkipDigest,
	)
	if err != nil {
		return application.EffectIntent{}, mapDatabaseError(err)
	}
	if err := restoreEffectSubject(&intent.Subject, projectRef, goalRef, workRef, executionRef,
		planGeneration, appSpecGeneration, actorRef); err != nil {
		return application.EffectIntent{}, err
	}
	principal, err := identity.NewPrincipalRef(proposedBy)
	if err != nil {
		return application.EffectIntent{}, invalid(err)
	}
	intent.ActionKind = application.ActionKind(actionKind)
	intent.Kind = application.EffectKind(kind)
	intent.ProposedBy = principal
	intent.Permission = identity.Permission(permission)
	intent.Demand.Resources = resources
	intent.Demand.Resources.Currency = governance.Currency(currency)
	intent.SecurityCriticality = governance.SecurityCriticality(criticality)
	intent.ReasoningEffort = governance.ReasoningEffort(effort)
	if policyRevision <= 0 || quotaRetryDelay <= 0 || approvalTTL <= 0 {
		return application.EffectIntent{}, invalid(errors.New("sqlite.effect_policy_revision_invalid"))
	}
	intent.PolicyRevision = uint64(policyRevision)
	intent.QuotaRetryDelay = time.Duration(quotaRetryDelay)
	intent.ApprovalTTL = time.Duration(approvalTTL)
	intent.CreatedAt = time.Unix(0, createdAt).UTC()
	intent.CouncilResolution, err = restoreCouncilResolution(
		councilSubject, councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest,
	)
	if err != nil {
		return application.EffectIntent{}, err
	}
	intent.Authority, err = readAuthorizationReceipt(ctx, source, authorityRef)
	if err != nil {
		return application.EffectIntent{}, err
	}
	if err := application.ValidateEffectIntent(intent); err != nil {
		return application.EffectIntent{}, invalid(err)
	}
	return intent, nil
}

func readEffectApprovalByRequest(
	ctx context.Context, source queryer, principalRef, requestRef string,
) (application.EffectApproval, bool, error) {
	return scanEffectApproval(ctx, source, effectApprovalSelect+`
WHERE decided_by_ref = ? AND request_ref = ? AND source = 'explicit_decision'`, principalRef, requestRef)
}

func readEffectApprovalByRef(
	ctx context.Context, source queryer, ref string,
) (application.EffectApproval, bool, error) {
	return scanEffectApproval(ctx, source, effectApprovalSelect+` WHERE ref = ?`, ref)
}

func scanEffectApproval(
	ctx context.Context, source queryer, query string, arguments ...any,
) (application.EffectApproval, bool, error) {
	var approval application.EffectApproval
	var projectRef, goalRef, workRef, executionRef, actorRef, proposedBy, decidedBy string
	var decision, sourceValue, criticality, authorizationRef string
	var planGeneration, appSpecGeneration, policyRevision, decidedAt int64
	var expiresAt sql.NullInt64
	err := source.QueryRowContext(ctx, query, arguments...).Scan(
		&approval.Ref, &approval.RequestRef, &approval.RequestFingerprint,
		&approval.IntentRef, &approval.IntentDigest, &projectRef, &goalRef, &workRef,
		&executionRef, &planGeneration, &appSpecGeneration, &approval.Subject.SpecHash,
		&actorRef, &proposedBy, &decidedBy, &decision, &sourceValue, &criticality,
		&approval.Reason, &approval.IdempotencyKey, &approval.PolicyHash, &policyRevision,
		&approval.TargetDigest,
		&authorizationRef, &decidedAt, &expiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.EffectApproval{}, false, nil
	}
	if err != nil {
		return application.EffectApproval{}, false, mapDatabaseError(err)
	}
	if err := restoreEffectSubject(&approval.Subject, projectRef, goalRef, workRef, executionRef,
		planGeneration, appSpecGeneration, actorRef); err != nil {
		return application.EffectApproval{}, false, err
	}
	approval.ProposedBy, err = identity.NewPrincipalRef(proposedBy)
	if err != nil {
		return application.EffectApproval{}, false, invalid(err)
	}
	approval.DecidedBy, err = identity.NewPrincipalRef(decidedBy)
	if err != nil {
		return application.EffectApproval{}, false, invalid(err)
	}
	approval.Decision = application.EffectDecision(decision)
	approval.Source = application.EffectApprovalSource(sourceValue)
	approval.SecurityCriticality = governance.SecurityCriticality(criticality)
	if policyRevision <= 0 {
		return application.EffectApproval{}, false, invalid(errors.New("sqlite.effect_approval_policy_revision_invalid"))
	}
	approval.PolicyRevision = uint64(policyRevision)
	approval.DecidedAt = time.Unix(0, decidedAt).UTC()
	approval.ExpiresAt = restoredTime(expiresAt)
	approval.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, authorizationRef)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	return approval, true, nil
}

func restoreEffectSubject(
	subject *application.EffectSubject,
	projectValue, goalValue, workValue, executionValue string,
	planGeneration, appSpecGeneration int64,
	actorValue string,
) error {
	if planGeneration <= 0 || appSpecGeneration <= 0 {
		return invalid(errors.New("sqlite.effect_subject_generation_invalid"))
	}
	var err error
	if subject.ProjectRef, err = goal.NewProjectRef(projectValue); err != nil {
		return invalid(err)
	}
	if subject.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
		return invalid(err)
	}
	if subject.WorkItemRef, err = goal.NewWorkItemRef(workValue); err != nil {
		return invalid(err)
	}
	if subject.ExecutionRef, err = goal.NewExecutionRef(executionValue); err != nil {
		return invalid(err)
	}
	if subject.ActorRef, err = goal.NewActorRef(actorValue); err != nil {
		return invalid(err)
	}
	subject.PlanGeneration = goal.PlanGeneration(planGeneration)
	subject.AppSpecGeneration = goal.AppSpecGeneration(appSpecGeneration)
	return nil
}

func readEffectAttemptByFence(
	ctx context.Context, source queryer, actionRef string, fence uint64,
) (application.EffectAttempt, bool, error) {
	var attempt application.EffectAttempt
	var projectRef, goalRef, workRef, executionRef, actorRef string
	var planGeneration, appSpecGeneration, actionFence, startedAt int64
	var claimLeaseUntil sql.NullInt64
	claimLeaseProjection := "NULL"
	claimLeasePersisted, err := sqliteTableHasColumn(ctx, source, "effect_attempts", "claim_lease_until")
	if err != nil {
		return application.EffectAttempt{}, false, mapDatabaseError(err)
	}
	if claimLeasePersisted {
		claimLeaseProjection = "claim_lease_until"
	}
	err = source.QueryRowContext(ctx, fmt.Sprintf(`
SELECT ref, intent_ref, intent_digest, approval_ref, project_ref, goal_ref,
       work_item_ref, execution_ref, plan_generation, app_spec_generation,
       spec_hash, actor_ref, action_ref, action_fence, worker_ref,
       idempotency_key, started_at, %s
FROM effect_attempts WHERE action_ref = ? AND action_fence = ?`, claimLeaseProjection), actionRef, int64(fence)).Scan(
		&attempt.Ref, &attempt.IntentRef, &attempt.IntentDigest, &attempt.ApprovalRef,
		&projectRef, &goalRef, &workRef, &executionRef, &planGeneration, &appSpecGeneration,
		&attempt.Subject.SpecHash, &actorRef, &attempt.ActionRef, &actionFence, &attempt.WorkerRef,
		&attempt.IdempotencyKey, &startedAt, &claimLeaseUntil,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.EffectAttempt{}, false, nil
	}
	if err != nil {
		return application.EffectAttempt{}, false, mapDatabaseError(err)
	}
	if err := restoreEffectSubject(&attempt.Subject, projectRef, goalRef, workRef, executionRef,
		planGeneration, appSpecGeneration, actorRef); err != nil {
		return application.EffectAttempt{}, false, err
	}
	if actionFence <= 0 {
		return application.EffectAttempt{}, false, invalid(fmt.Errorf("sqlite.effect_attempt_fence_invalid"))
	}
	attempt.ActionFence = uint64(actionFence)
	attempt.StartedAt = time.Unix(0, startedAt).UTC()
	if claimLeaseUntil.Valid {
		attempt.ClaimLeaseUntil = time.Unix(0, claimLeaseUntil.Int64).UTC()
	}
	return attempt, true, nil
}
