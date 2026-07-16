package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type claimCandidate struct {
	action          application.ActionRecord
	roleKey         string
	providerRef     string
	modelRef        string
	agentRef        string
	deliveryAttempt int64
}

const (
	legacyV4ModelUnattributed = "legacy:v4:model-unattributed"
	legacyV4AgentUnattributed = "legacy:v4:agent-unattributed"
)

func (repository *Repository) ClaimNextAction(
	ctx context.Context,
	request application.ClaimRequest,
) (application.ActionClaim, bool, error) {
	if !validText(request.WorkerRef) || !validText(request.Token) {
		return application.ActionClaim{}, false, invalid(errors.New("sqlite.claim_identity_invalid"))
	}
	if err := ports.ValidateAgentCapabilities(request.Capabilities); err != nil {
		return application.ActionClaim{}, false, invalid(err)
	}
	if request.LeaseDuration <= 0 {
		return application.ActionClaim{}, false, invalid(errors.New("sqlite.claim_time_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	// The lease starts only after BEGIN IMMEDIATE has acquired the writer. A
	// caller cannot age, extend, or steal a lease by supplying its own clock.
	now, err := repository.transactionTime()
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	leaseUntil, err := safeLeaseUntil(now, request.LeaseDuration)
	if err != nil {
		return application.ActionClaim{}, false, invalid(err)
	}

	var tokenExists int
	err = transaction.QueryRowContext(ctx, `
SELECT 1
FROM (
    SELECT claim_token AS token FROM outbox WHERE claim_token = ?
    UNION ALL
    SELECT claim_token AS token FROM action_consumption_receipts WHERE claim_token = ?
)
LIMIT 1`, request.Token, request.Token).Scan(&tokenExists)
	if err == nil {
		return application.ActionClaim{}, false, stateError(application.StateAlreadyClaimed, errors.New("sqlite.claim_token_reused"))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}

	candidates, err := readClaimCandidates(ctx, transaction, now)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	var selected *claimCandidate
	for index := range candidates {
		requirements, readErr := readAgentRequirements(ctx, transaction, candidates[index])
		if readErr != nil {
			return application.ActionClaim{}, false, readErr
		}
		if !ports.MatchAgentCapabilities(request.Capabilities, requirements) {
			continue
		}
		if candidates[index].action.Kind == application.ActionObserveAgent &&
			!observeIdentityMatches(candidates[index], request.Capabilities) {
			continue
		}
		selected = &candidates[index]
		break
	}
	if selected == nil {
		if err := commit(transaction); err != nil {
			return application.ActionClaim{}, false, err
		}
		return application.ActionClaim{}, false, nil
	}
	if selected.deliveryAttempt < 0 || uint64(selected.deliveryAttempt) >= maxSQLiteInteger {
		return application.ActionClaim{}, false, invalid(fmt.Errorf("sqlite.action_attempt_invalid"))
	}
	deliveryAttempt := selected.deliveryAttempt + 1

	var fence int64
	err = transaction.QueryRowContext(ctx, `
INSERT INTO work_item_fences(goal_ref, work_item_ref, fence)
VALUES (?, ?, 1)
ON CONFLICT(goal_ref, work_item_ref)
DO UPDATE SET fence = work_item_fences.fence + 1
RETURNING fence`,
		selected.action.GoalRef.String(), selected.action.WorkItemRef.String(),
	).Scan(&fence)
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	if fence <= 0 {
		return application.ActionClaim{}, false, invalid(errors.New("sqlite.claim_fence_invalid"))
	}

	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET claim_token = ?, claimed_by = ?, claimed_until = ?, delivery_attempt = ?, fence = ?
WHERE ref = ?
  AND completed_at IS NULL
  AND retired_at IS NULL
  AND quarantined_at IS NULL
  AND available_at <= ?
  AND (claim_token IS NULL OR claimed_until <= ?)`,
		request.Token,
		request.WorkerRef,
		requiredTime(leaseUntil),
		deliveryAttempt,
		fence,
		selected.action.Ref,
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
		Action:          selected.action,
		Token:           request.Token,
		WorkerRef:       request.WorkerRef,
		DeliveryAttempt: uint64(deliveryAttempt),
		Fence:           uint64(fence),
		LeaseUntil:      leaseUntil,
	}
	if err := commit(transaction); err != nil {
		return application.ActionClaim{}, false, err
	}
	return claim, true, nil
}

func observeIdentityMatches(candidate claimCandidate, capabilities ports.AgentCapabilities) bool {
	if candidate.providerRef != capabilities.ProviderRef {
		return false
	}
	if candidate.modelRef == legacyV4ModelUnattributed && candidate.agentRef == legacyV4AgentUnattributed {
		return true
	}
	return candidate.modelRef == capabilities.ModelRef && candidate.agentRef == capabilities.AgentRef
}

func readClaimCandidates(ctx context.Context, transaction *sql.Tx, now time.Time) ([]claimCandidate, error) {
	rows, err := transaction.QueryContext(ctx, `
SELECT o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.delivery_attempt, wi.role_key,
       e.provider_ref, e.model_ref, e.agent_ref
FROM outbox o
JOIN work_items wi ON wi.goal_ref = o.goal_ref AND wi.ref = o.work_item_ref
JOIN executions e
  ON e.goal_ref = o.goal_ref AND e.work_item_ref = o.work_item_ref AND e.ref = o.execution_ref
WHERE o.completed_at IS NULL
  AND o.retired_at IS NULL
  AND o.quarantined_at IS NULL
  AND o.kind IN ('launch_agent', 'observe_agent')
  AND o.available_at <= ?
  AND (o.claim_token IS NULL OR o.claimed_until <= ?)
ORDER BY o.available_at, o.ref`, requiredTime(now), requiredTime(now))
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []claimCandidate
	for rows.Next() {
		var candidate claimCandidate
		var kind, goalValue, itemValue, executionValue string
		var planGeneration, itemGeneration, availableAt int64
		if err := rows.Scan(
			&candidate.action.Ref, &kind, &goalValue, &itemValue, &executionValue,
			&planGeneration, &itemGeneration, &availableAt, &candidate.deliveryAttempt,
			&candidate.roleKey, &candidate.providerRef, &candidate.modelRef, &candidate.agentRef,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		if planGeneration <= 0 || itemGeneration <= 0 || candidate.deliveryAttempt < 0 {
			return nil, invalid(errors.New("sqlite.action_generation_invalid"))
		}
		var refErr error
		candidate.action.Kind = application.ActionKind(kind)
		if candidate.action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if candidate.action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if candidate.action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
			return nil, invalid(refErr)
		}
		candidate.action.PlanGeneration = goal.PlanGeneration(planGeneration)
		candidate.action.WorkItemGeneration = goal.Revision(itemGeneration)
		candidate.action.AvailableAt = time.Unix(0, availableAt).UTC()
		if err := validateAction(candidate.action); err != nil {
			return nil, invalid(err)
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readAgentRequirements(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
) (ports.AgentRequirements, error) {
	requirements := ports.AgentRequirements{RoleKey: candidate.roleKey}
	rows, err := transaction.QueryContext(ctx, `
SELECT kind, value
FROM work_item_requirement_refs
WHERE goal_ref = ? AND work_item_ref = ?
ORDER BY kind, position`, candidate.action.GoalRef.String(), candidate.action.WorkItemRef.String())
	if err != nil {
		return ports.AgentRequirements{}, mapDatabaseError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind, value string
		if err := rows.Scan(&kind, &value); err != nil {
			return ports.AgentRequirements{}, mapDatabaseError(err)
		}
		switch kind {
		case "skill":
			requirements.SkillRefs = append(requirements.SkillRefs, value)
		case "tool":
			requirements.ToolRefs = append(requirements.ToolRefs, value)
		case "capability":
			requirements.CapabilityRefs = append(requirements.CapabilityRefs, value)
		default:
			return ports.AgentRequirements{}, invalid(errors.New("sqlite.requirement_kind_invalid"))
		}
	}
	if err := rows.Err(); err != nil {
		return ports.AgentRequirements{}, mapDatabaseError(err)
	}
	return requirements, nil
}

func requireClaim(ctx context.Context, transaction *sql.Tx, claim application.ActionClaim) error {
	if err := validateClaim(claim); err != nil {
		return invalid(err)
	}
	var kind, goalValue, itemValue, executionValue string
	var availableAt, planGeneration, itemGeneration int64
	var token, worker sql.NullString
	var leaseUntil, completedAt, quarantinedAt sql.NullInt64
	var deliveryAttempt, fence, currentFence int64
	err := transaction.QueryRowContext(ctx, `
SELECT o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
       o.completed_at, o.quarantined_at, wf.fence
FROM outbox o
JOIN work_item_fences wf ON wf.goal_ref = o.goal_ref AND wf.work_item_ref = o.work_item_ref
WHERE o.ref = ?`, claim.Action.Ref).Scan(
		&kind, &goalValue, &itemValue, &executionValue,
		&planGeneration, &itemGeneration, &availableAt,
		&token, &worker, &leaseUntil, &deliveryAttempt, &fence,
		&completedAt, &quarantinedAt, &currentFence,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict(err)
	}
	if err != nil {
		return mapDatabaseError(err)
	}
	if completedAt.Valid || quarantinedAt.Valid || !token.Valid || !worker.Valid || !leaseUntil.Valid ||
		token.String != claim.Token || worker.String != claim.WorkerRef ||
		deliveryAttempt != int64(claim.DeliveryAttempt) || fence != int64(claim.Fence) || currentFence != int64(claim.Fence) ||
		leaseUntil.Int64 != requiredTime(claim.LeaseUntil) || kind != string(claim.Action.Kind) ||
		goalValue != claim.Action.GoalRef.String() || itemValue != claim.Action.WorkItemRef.String() ||
		executionValue != claim.Action.ExecutionRef.String() ||
		planGeneration != int64(claim.Action.PlanGeneration) ||
		itemGeneration != int64(claim.Action.WorkItemGeneration) ||
		availableAt != requiredTime(claim.Action.AvailableAt) {
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
	outcome := application.ActionConsumedCompleted
	if quarantined {
		quarantineAt = requiredTime(at)
		outcome = application.ActionConsumedQuarantined
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET completed_at = ?, quarantined_at = ?, last_error_code = ?
WHERE ref = ? AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(at), quarantineAt, errorCode,
		claim.Action.Ref, claim.Token, claim.WorkerRef, requiredTime(claim.LeaseUntil),
		int64(claim.DeliveryAttempt), int64(claim.Fence),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		claim.Action.Ref, string(claim.Action.Kind), claim.Action.GoalRef.String(),
		claim.Action.WorkItemRef.String(), claim.Action.ExecutionRef.String(),
		int64(claim.Action.PlanGeneration), int64(claim.Action.WorkItemGeneration),
		int64(claim.Fence), int64(claim.DeliveryAttempt), claim.Token, claim.WorkerRef,
		string(outcome), errorCode, requiredTime(at),
	)
	return mapDatabaseError(err)
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
  AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(state.AvailableAt), state.ErrorCode,
		state.Claim.Action.Ref, state.Claim.Token, state.Claim.WorkerRef,
		requiredTime(state.Claim.LeaseUntil), int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func (repository *Repository) transactionTime() (time.Time, error) {
	if repository == nil || repository.now == nil {
		return time.Time{}, invalid(errors.New("sqlite.clock_unavailable"))
	}
	now := repository.now().Round(0).UTC()
	if now.IsZero() {
		return time.Time{}, invalid(errors.New("sqlite.clock_invalid"))
	}
	return now, nil
}
