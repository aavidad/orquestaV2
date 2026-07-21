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
	"orquesta/internal/ports"
)

type claimCandidate struct {
	action            application.ActionRecord
	projectRef        goal.ProjectRef
	governanceVersion int64
	roleKey           string
	providerRef       string
	modelRef          string
	agentRef          string
	executionState    application.ExecutionState
	deliveryAttempt   int64
}

const (
	legacyV4ModelUnattributed = "legacy:v4:model-unattributed"
	legacyV4AgentUnattributed = "legacy:v4:agent-unattributed"
)

func (repository *Repository) ClaimNextAction(
	ctx context.Context,
	request application.ClaimRequest,
) (application.ActionClaim, bool, error) {
	if err := validateClaimRequest(request); err != nil {
		return application.ActionClaim{}, false, err
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

	if err := requireFreshClaimToken(ctx, transaction, request.Token); err != nil {
		return application.ActionClaim{}, false, err
	}
	candidates, err := readClaimCandidates(ctx, transaction, now)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	selected, found, err := selectClaimCandidate(ctx, transaction, candidates, request.Capabilities, now)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	if !found {
		if err := commit(transaction); err != nil {
			return application.ActionClaim{}, false, err
		}
		return application.ActionClaim{}, false, nil
	}
	claim, err := claimSelectedCandidate(ctx, transaction, request, selected, now, leaseUntil)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.ActionClaim{}, false, err
	}
	return claim, true, nil
}

func validateClaimRequest(request application.ClaimRequest) error {
	if !validText(request.WorkerRef) || !validText(request.Token) {
		return invalid(errors.New("sqlite.claim_identity_invalid"))
	}
	if err := ports.ValidateAgentCapabilities(request.Capabilities); err != nil {
		return invalid(err)
	}
	if request.LeaseDuration <= 0 {
		return invalid(errors.New("sqlite.claim_time_invalid"))
	}
	if err := application.ValidateBudgetPolicy(request.BudgetPolicy); err != nil {
		return invalid(err)
	}
	return nil
}

func requireFreshClaimToken(ctx context.Context, tx *sql.Tx, token string) error {
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM (
SELECT claim_token FROM outbox WHERE claim_token=? UNION ALL
SELECT claim_token FROM action_consumption_receipts WHERE claim_token=?) LIMIT 1`, token, token).Scan(&exists)
	if err == nil {
		return stateError(application.StateAlreadyClaimed, errors.New("sqlite.claim_token_reused"))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return mapDatabaseError(err)
}

type claimSelection struct {
	candidate         claimCandidate
	approval          application.EffectApproval
	reservation       governance.BudgetReservation
	reservationExists bool
}

func selectClaimCandidate(
	ctx context.Context, tx *sql.Tx, candidates []claimCandidate,
	capabilities ports.AgentCapabilities, now time.Time,
) (claimSelection, bool, error) {
	for index := range candidates {
		candidate := &candidates[index]
		matches, err := claimCandidateMatches(ctx, tx, *candidate, capabilities)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !matches {
			continue
		}
		approval, admitted, err := admitClaimEffect(ctx, tx, candidate, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !admitted {
			continue
		}
		reservation, exists, capacity, err := admitClaimBudget(ctx, tx, *candidate, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !capacity {
			continue
		}
		return claimSelection{*candidate, approval, reservation, exists}, true, nil
	}
	return claimSelection{}, false, nil
}

func claimCandidateMatches(
	ctx context.Context, tx *sql.Tx, candidate claimCandidate, capabilities ports.AgentCapabilities,
) (bool, error) {
	if candidate.action.Kind == application.ActionPrepareWorkspace || candidate.action.Kind == application.ActionCommitChange || candidate.action.Kind == application.ActionIntegrateChange {
		return true, nil
	}
	// Terminal stops settle locally; no provider identity is needed.
	if terminalStopSettlement(candidate) {
		return true, nil
	}
	requirements, err := readAgentRequirements(ctx, tx, candidate)
	if err != nil {
		return false, err
	}
	if !ports.MatchAgentCapabilities(capabilities, requirements) {
		return false, nil
	}
	if candidate.action.Kind == application.ActionObserveAgent || candidate.action.Kind == application.ActionStopAgent {
		return observeIdentityMatches(candidate, capabilities), nil
	}
	return true, nil
}

func admitClaimEffect(
	ctx context.Context, tx *sql.Tx, candidate *claimCandidate, now time.Time,
) (application.EffectApproval, bool, error) {
	if candidate.governanceVersion != 1 {
		return application.EffectApproval{}, true, nil
	}
	if terminalStopSettlement(*candidate) {
		intent, err := requireTerminalStopIntent(ctx, tx, *candidate)
		candidate.action.EffectIntent = intent
		return application.EffectApproval{}, err == nil, err
	}
	if candidate.action.Kind != application.ActionLaunchAgent && candidate.action.Kind != application.ActionStopAgent &&
		candidate.action.Kind != application.ActionPrepareWorkspace && candidate.action.Kind != application.ActionCommitChange && candidate.action.Kind != application.ActionIntegrateChange {
		return application.EffectApproval{}, true, nil
	}
	intent, approval, admitted, err := requireEffectAdmission(ctx, tx, *candidate, now)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	if !admitted {
		if candidate.action.Kind == application.ActionLaunchAgent && intent.Ref != "" {
			err = parkLaunchWithStaleAdmission(ctx, tx, *candidate, intent, now)
		}
		return application.EffectApproval{}, false, err
	}
	candidate.action.EffectIntent = intent
	return approval, true, nil
}

func admitClaimBudget(
	ctx context.Context, tx *sql.Tx, candidate claimCandidate, now time.Time,
) (governance.BudgetReservation, bool, bool, error) {
	if candidate.governanceVersion != 1 || candidate.action.Kind != application.ActionLaunchAgent {
		return governance.BudgetReservation{}, false, true, nil
	}
	reservation, exists, capacity, err := prepareBudgetAdmission(ctx, tx, candidate)
	if err != nil || capacity {
		return reservation, exists, capacity, err
	}
	err = deferQuotaLimitedAction(ctx, tx, candidate.action.Ref, now, candidate.action.EffectIntent.QuotaRetryDelay)
	return governance.BudgetReservation{}, false, false, err
}

func claimSelectedCandidate(
	ctx context.Context, tx *sql.Tx, request application.ClaimRequest,
	selected claimSelection, now, leaseUntil time.Time,
) (application.ActionClaim, error) {
	candidate := selected.candidate
	if candidate.deliveryAttempt < 0 || uint64(candidate.deliveryAttempt) >= maxSQLiteInteger {
		return application.ActionClaim{}, invalid(fmt.Errorf("sqlite.action_attempt_invalid"))
	}
	deliveryAttempt, fence := candidate.deliveryAttempt+1, int64(0)
	err := tx.QueryRowContext(ctx, `INSERT INTO work_item_fences(goal_ref,work_item_ref,fence) VALUES(?,?,1)
ON CONFLICT(goal_ref,work_item_ref) DO UPDATE SET fence=work_item_fences.fence+1 RETURNING fence`,
		candidate.action.GoalRef.String(), candidate.action.WorkItemRef.String()).Scan(&fence)
	if err != nil {
		return application.ActionClaim{}, mapDatabaseError(err)
	}
	if fence <= 0 {
		return application.ActionClaim{}, invalid(errors.New("sqlite.claim_fence_invalid"))
	}
	reservation := selected.reservation
	if candidate.governanceVersion == 1 && candidate.action.Kind == application.ActionLaunchAgent {
		if !selected.reservationExists {
			reservation, err = insertBudgetReservation(ctx, tx, candidate, uint64(fence), now)
		}
		if err == nil {
			err = bindClaimedLaunchExecution(ctx, tx, candidate, reservation)
		}
		if err != nil {
			return application.ActionClaim{}, err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE outbox SET claim_token=?,claimed_by=?,claimed_until=?,delivery_attempt=?,fence=?
WHERE ref=? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
AND available_at<=? AND (claim_token IS NULL OR claimed_until<=?)`, request.Token, request.WorkerRef,
		requiredTime(leaseUntil), deliveryAttempt, fence, candidate.action.Ref, requiredTime(now), requiredTime(now))
	if err != nil {
		return application.ActionClaim{}, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.ActionClaim{}, err
	}
	claim := application.ActionClaim{Action: candidate.action, Token: request.Token, WorkerRef: request.WorkerRef,
		DeliveryAttempt: uint64(deliveryAttempt), Fence: uint64(fence), BudgetReservationRef: reservation.Ref,
		BudgetReservation: reservation, EffectApproval: selected.approval, LeaseUntil: leaseUntil}
	if candidate.governanceVersion == 1 && candidate.action.Kind == application.ActionLaunchAgent {
		if err := advanceFairness(ctx, tx, candidate.projectRef, candidate.action.GoalRef, now); err != nil {
			return application.ActionClaim{}, err
		}
	}
	return claim, nil
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

func terminalStopSettlement(candidate claimCandidate) bool {
	if candidate.action.Kind != application.ActionStopAgent {
		return false
	}
	switch candidate.executionState {
	case application.ExecutionSucceeded, application.ExecutionFailed,
		application.ExecutionCanceled, application.ExecutionStopped:
		return true
	default:
		return false
	}
}

func requireTerminalStopIntent(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
) (application.EffectIntent, error) {
	intent, err := readEffectIntent(ctx, transaction, candidate.action.EffectIntentRef)
	if err != nil {
		return application.EffectIntent{}, err
	}
	if intent.ActionRef != candidate.action.Ref || intent.ActionKind != application.ActionStopAgent ||
		intent.Kind != application.EffectKindAgentStop || intent.Subject.ProjectRef != candidate.projectRef ||
		intent.Subject.GoalRef != candidate.action.GoalRef ||
		intent.Subject.WorkItemRef != candidate.action.WorkItemRef ||
		intent.Subject.ExecutionRef != candidate.action.ExecutionRef ||
		intent.Subject.PlanGeneration != candidate.action.PlanGeneration {
		return application.EffectIntent{}, invalid(errors.New("sqlite.terminal_stop_effect_intent_causal_invalid"))
	}
	return intent, nil
}

const claimCandidatesQuery = `
SELECT o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.control_ref, %s, o.effect_intent_ref, o.governance_version,
       o.plan_generation, o.work_item_generation, o.available_at, g.project_ref,
       o.delivery_attempt, wi.role_key, e.state,
       e.provider_ref, e.model_ref, e.agent_ref
FROM outbox o
JOIN work_items wi ON wi.goal_ref = o.goal_ref AND wi.ref = o.work_item_ref
JOIN goals g ON g.ref = o.goal_ref
JOIN executions e
  ON e.goal_ref = o.goal_ref AND e.work_item_ref = o.work_item_ref AND e.ref = o.execution_ref
LEFT JOIN fairness_cursors project_cursor
  ON project_cursor.scope = 'project' AND project_cursor.subject_ref = g.project_ref
LEFT JOIN fairness_cursors goal_cursor
  ON goal_cursor.scope = 'goal' AND goal_cursor.subject_ref = g.ref
WHERE o.completed_at IS NULL
  AND o.retired_at IS NULL
  AND o.quarantined_at IS NULL
  AND o.kind IN ('launch_agent', 'observe_agent', 'stop_agent', 'prepare_workspace', 'commit_change', 'integrate_change')
  AND (o.governance_version = 1 OR o.kind = 'observe_agent'
       OR (o.kind = 'stop_agent' AND e.state IN ('succeeded', 'failed', 'canceled', 'stopped'))
       OR (o.governance_version = 0 AND o.last_error_code <> 'governance.legacy_reauthorization_required'))
  AND o.available_at <= ?
  AND (o.claim_token IS NULL OR o.claimed_until <= ?)
  AND NOT EXISTS (
      SELECT 1 FROM outbox leased
      WHERE leased.goal_ref = o.goal_ref AND leased.work_item_ref = o.work_item_ref
        AND leased.kind IN ('launch_agent', 'observe_agent', 'stop_agent', 'prepare_workspace', 'commit_change', 'integrate_change')
        AND leased.ref <> o.ref AND leased.completed_at IS NULL
        AND leased.retired_at IS NULL AND leased.quarantined_at IS NULL
        AND leased.claim_token IS NOT NULL AND leased.claimed_until > ?
  )
  AND (
      o.kind <> 'launch_agent' OR e.state = 'dispatching'
      OR (e.state = 'queued' AND g.paused = 0 AND g.cancel_requested = 0
          AND wi.paused = 0 AND wi.cancel_requested = 0)
  )
  AND (o.kind <> 'stop_agent' OR (
      (e.state = 'running' AND e.external_ref <> '')
      OR e.state IN ('succeeded', 'failed', 'canceled', 'stopped')
  ))
  AND (o.kind <> 'prepare_workspace' OR e.state = 'queued')
  AND (o.kind <> 'commit_change' OR e.state = 'awaiting_commit')
  AND (o.kind <> 'integrate_change' OR e.state = 'awaiting_integration')
  AND (o.kind <> 'observe_agent' OR NOT EXISTS (
      SELECT 1 FROM outbox stop
      WHERE stop.goal_ref = o.goal_ref AND stop.execution_ref = o.execution_ref
        AND stop.kind = 'stop_agent' AND stop.completed_at IS NULL
        AND stop.governance_version = 1
        AND stop.retired_at IS NULL AND stop.quarantined_at IS NULL
  ))
-- Stop is urgent. Governed launches use hierarchical round-robin before
-- their FIFO tie-break; observations run after no launch fits admission.
ORDER BY CASE o.kind WHEN 'stop_agent' THEN 0 WHEN 'prepare_workspace' THEN 1 WHEN 'launch_agent' THEN 2 WHEN 'commit_change' THEN 3 WHEN 'integrate_change' THEN 4 WHEN 'observe_agent' THEN 5 ELSE 6 END,
	     CASE WHEN o.kind = 'launch_agent' THEN COALESCE(project_cursor.ordinal, 0) ELSE 0 END,
	     CASE WHEN o.kind = 'launch_agent' THEN COALESCE(goal_cursor.ordinal, 0) ELSE 0 END,
	     o.available_at,
	     o.ref`

func readClaimCandidates(ctx context.Context, transaction *sql.Tx, now time.Time) ([]claimCandidate, error) {
	workspaceColumns, err := sqliteTableHasColumn(ctx, transaction, "outbox", "change_ref")
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	changeProjection := "'' AS change_ref, '' AS expected_target_oid"
	if workspaceColumns {
		changeProjection = "o.change_ref, o.expected_target_oid"
	}
	rows, err := transaction.QueryContext(ctx, fmt.Sprintf(claimCandidatesQuery, changeProjection),
		requiredTime(now), requiredTime(now), requiredTime(now))
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []claimCandidate
	for rows.Next() {
		candidate, err := scanClaimCandidate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func scanClaimCandidate(rows *sql.Rows) (claimCandidate, error) {
	var candidate claimCandidate
	var kind, goalValue, itemValue, executionValue, projectValue string
	var controlRef, effectIntentRef sql.NullString
	var changeRef, expectedTarget string
	var planGeneration, itemGeneration, availableAt int64
	err := rows.Scan(&candidate.action.Ref, &kind, &goalValue, &itemValue, &executionValue,
		&controlRef, &changeRef, &expectedTarget, &effectIntentRef, &candidate.governanceVersion, &planGeneration,
		&itemGeneration, &availableAt, &projectValue, &candidate.deliveryAttempt,
		&candidate.roleKey, &candidate.executionState, &candidate.providerRef,
		&candidate.modelRef, &candidate.agentRef)
	if err != nil {
		return candidate, mapDatabaseError(err)
	}
	if planGeneration <= 0 || itemGeneration <= 0 || candidate.deliveryAttempt < 0 ||
		(candidate.governanceVersion != 0 && candidate.governanceVersion != 1) {
		return candidate, invalid(errors.New("sqlite.action_generation_invalid"))
	}
	candidate.action.Kind = application.ActionKind(kind)
	if controlRef.Valid {
		candidate.action.ControlRef = controlRef.String
	}
	if changeRef != "" {
		candidate.action.ChangeRef, _ = ports.NewChangeSetRef(changeRef)
	}
	candidate.action.ExpectedTargetOID = expectedTarget
	if effectIntentRef.Valid {
		candidate.action.EffectIntentRef = effectIntentRef.String
	}
	var refErr error
	if candidate.projectRef, refErr = goal.NewProjectRef(projectValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	candidate.action.PlanGeneration = goal.PlanGeneration(planGeneration)
	candidate.action.WorkItemGeneration = goal.Revision(itemGeneration)
	candidate.action.AvailableAt = time.Unix(0, availableAt).UTC()
	if err := validateAction(candidate.action); err != nil {
		return candidate, invalid(err)
	}
	return candidate, nil
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
	var controlRef sql.NullString
	var changeRef, expectedTarget string
	var availableAt, planGeneration, itemGeneration int64
	var token, worker sql.NullString
	var leaseUntil, completedAt, quarantinedAt sql.NullInt64
	var deliveryAttempt, fence, currentFence int64
	workspaceColumns, err := sqliteTableHasColumn(ctx, transaction, "outbox", "change_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	changeProjection := "'' AS change_ref, '' AS expected_target_oid"
	if workspaceColumns {
		changeProjection = "o.change_ref, o.expected_target_oid"
	}
	err = transaction.QueryRowContext(ctx, fmt.Sprintf(`
SELECT o.kind, o.goal_ref, o.work_item_ref, o.execution_ref, o.control_ref, %s,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
       o.completed_at, o.quarantined_at, wf.fence
FROM outbox o
JOIN work_item_fences wf ON wf.goal_ref = o.goal_ref AND wf.work_item_ref = o.work_item_ref
WHERE o.ref = ?`, changeProjection), claim.Action.Ref).Scan(
		&kind, &goalValue, &itemValue, &executionValue, &controlRef, &changeRef, &expectedTarget,
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
		controlRef.String != claim.Action.ControlRef ||
		changeRef != claim.Action.ChangeRef.String() || expectedTarget != claim.Action.ExpectedTargetOID ||
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
	return completeClaimWithEffect(ctx, transaction, claim, at, errorCode, quarantined, nil)
}

func completeClaimWithEffect(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
	at time.Time,
	errorCode string,
	quarantined bool,
	effect *application.EffectReceipt,
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
	if effect != nil {
		if err := insertEffectReceipt(ctx, transaction, *effect); err != nil {
			return err
		}
	}
	version := int64(0)
	if claim.Action.EffectIntentRef != "" {
		version = 1
	}
	receipt := application.ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind, GoalRef: claim.Action.GoalRef,
		WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
		PlanGeneration: claim.Action.PlanGeneration, WorkItemGeneration: claim.Action.WorkItemGeneration,
		Fence: claim.Fence, DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: outcome, ErrorCode: errorCode, ConsumedAt: at,
		ChangeRef: claim.Action.ChangeRef,
	}
	if effect != nil {
		receipt.EffectReceiptRef = effect.Ref
	}
	return insertActionConsumptionReceipt(ctx, transaction, receipt, version, effect)
}

func insertActionConsumptionReceipt(
	ctx context.Context, tx *sql.Tx, receipt application.ActionConsumptionReceipt,
	governanceVersion int64, effect *application.EffectReceipt,
) error {
	persisted, err := sqliteTableHasColumn(ctx, tx, "action_consumption_receipts", "governance_version")
	if err != nil {
		return mapDatabaseError(err)
	}
	if persisted {
		workspaceColumns, columnErr := sqliteTableHasColumn(ctx, tx, "action_consumption_receipts", "change_ref")
		if columnErr != nil {
			return mapDatabaseError(columnErr)
		}
		if !workspaceColumns {
			_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at, effect_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				receipt.ActionRef, governanceVersion, string(receipt.Kind), receipt.GoalRef.String(),
				receipt.WorkItemRef.String(), receipt.ExecutionRef.String(), int64(receipt.PlanGeneration),
				int64(receipt.WorkItemGeneration), int64(receipt.Fence), int64(receipt.DeliveryAttempt),
				receipt.ClaimToken, receipt.WorkerRef, string(receipt.Outcome), receipt.ErrorCode,
				requiredTime(receipt.ConsumedAt), nullableString(receipt.EffectReceiptRef),
			)
			return mapDatabaseError(err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref,
    change_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at, effect_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			receipt.ActionRef, governanceVersion, string(receipt.Kind), receipt.GoalRef.String(),
			receipt.WorkItemRef.String(), receipt.ExecutionRef.String(), receipt.ChangeRef.String(), int64(receipt.PlanGeneration),
			int64(receipt.WorkItemGeneration), int64(receipt.Fence), int64(receipt.DeliveryAttempt),
			receipt.ClaimToken, receipt.WorkerRef, string(receipt.Outcome), receipt.ErrorCode,
			requiredTime(receipt.ConsumedAt), nullableString(receipt.EffectReceiptRef),
		)
		return mapDatabaseError(err)
	}
	var effectStatus, effectAt any
	if effect != nil {
		effectStatus, effectAt = effect.Status, requiredTime(effect.ConfirmedAt)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at,
    effect_receipt_ref, effect_status, effect_confirmed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.ActionRef, string(receipt.Kind), receipt.GoalRef.String(), receipt.WorkItemRef.String(),
		receipt.ExecutionRef.String(), int64(receipt.PlanGeneration), int64(receipt.WorkItemGeneration),
		int64(receipt.Fence), int64(receipt.DeliveryAttempt), receipt.ClaimToken, receipt.WorkerRef,
		string(receipt.Outcome), receipt.ErrorCode, requiredTime(receipt.ConsumedAt),
		nullableString(receipt.EffectReceiptRef), effectStatus, effectAt,
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
