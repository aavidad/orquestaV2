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
	"orquesta/internal/ports"
)

func (repository *Repository) AuthorizeTerminalAgentLaunchReconciliation(
	ctx context.Context,
	state application.AuthorizeTerminalAgentLaunchReconciliationState,
) (application.TerminalAgentLaunchReconciliationAuthority, bool, error) {
	authority := state.Authority
	if err := validateTerminalReconciliationAuthority(authority, state.AvailableAt); err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, invalid(err)
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if replay, found, readErr := readTerminalReconciliationAuthorityByRequest(ctx, tx, authority.RequestRef); readErr != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, readErr
	} else if found {
		if replay != authority {
			return application.TerminalAgentLaunchReconciliationAuthority{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_replay_conflict"))
		}
		if err := commit(tx); err != nil {
			return application.TerminalAgentLaunchReconciliationAuthority{}, false, err
		}
		return replay, false, nil
	}
	if err := requireTerminalReconciliationSource(ctx, tx, authority); err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_reconciliation_authorities(
ref,job_ref,request_ref,request_fingerprint,authorization_receipt_ref,principal_ref,project_ref,
goal_ref,work_item_ref,execution_ref,action_ref,effect_intent_ref,effect_intent_digest,effect_attempt_ref,
plan_generation,work_item_generation,action_fence,original_receipt_fingerprint,authorized_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		authority.Ref, authority.JobRef, authority.RequestRef, authority.RequestFingerprint,
		authority.AuthorizationReceipt.Ref(), authority.PrincipalRef.String(), authority.ProjectRef.String(),
		authority.GoalRef.String(), authority.WorkItemRef.String(), authority.ExecutionRef.String(),
		authority.ActionRef, authority.EffectIntentRef, authority.EffectIntentDigest, authority.EffectAttemptRef,
		int64(authority.PlanGeneration), int64(authority.WorkItemGeneration), int64(authority.ActionFence),
		authority.OriginalReceiptFingerprint, requiredTime(authority.AuthorizedAt))
	if err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, mapDatabaseError(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_reconciliation_jobs(
ref,authority_ref,available_at,delivery_attempt,fence,state,last_error_code,updated_at)
VALUES(?,?,?,0,?,'pending','',?)`, authority.JobRef, authority.Ref, requiredTime(state.AvailableAt),
		int64(authority.ActionFence), requiredTime(authority.AuthorizedAt))
	if err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, mapDatabaseError(err)
	}
	if err := commit(tx); err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, err
	}
	return authority, true, nil
}

func validateTerminalReconciliationAuthority(
	authority application.TerminalAgentLaunchReconciliationAuthority,
	availableAt time.Time,
) error {
	decision := authority.AuthorizationReceipt.Decision()
	request := decision.Request()
	if !validText(authority.Ref) || !validText(authority.JobRef) || !validText(authority.RequestRef) ||
		!validDigest(authority.RequestFingerprint) || !validDigest(authority.EffectIntentDigest) ||
		!validDigest(authority.OriginalReceiptFingerprint) || !validText(authority.ActionRef) ||
		!validText(authority.EffectIntentRef) || !validText(authority.EffectAttemptRef) ||
		authority.PrincipalRef.String() == "" || authority.ProjectRef.String() == "" ||
		authority.GoalRef.String() == "" || authority.WorkItemRef.String() == "" ||
		authority.ExecutionRef.String() == "" || authority.PlanGeneration == 0 ||
		authority.WorkItemGeneration == 0 || authority.ActionFence == 0 ||
		authority.AuthorizedAt.IsZero() || availableAt.IsZero() || availableAt.Before(authority.AuthorizedAt) ||
		decision.Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(decision.Role(), identity.PermissionEffectsApprove) ||
		request.Principal().Ref != authority.PrincipalRef || request.ProjectRef() != authority.ProjectRef ||
		request.Permission() != identity.PermissionEffectsApprove ||
		request.ResourceRef() != "agent-launch-reconciliation:"+authority.ActionRef+":"+authority.EffectAttemptRef ||
		authority.AuthorizationReceipt.RecordedAt().After(authority.AuthorizedAt) {
		return errors.New("sqlite.agent_launch_reconciliation_authority_invalid")
	}
	return nil
}

func requireTerminalReconciliationSource(
	ctx context.Context,
	tx *sql.Tx,
	authority application.TerminalAgentLaunchReconciliationAuthority,
) error {
	persistedAuthorization, err := readAuthorizationReceipt(ctx, tx, authority.AuthorizationReceipt.Ref())
	if err != nil {
		return err
	}
	if persistedAuthorization != authority.AuthorizationReceipt {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_authorization_conflict"))
	}
	receipts, err := readConsumptionReceipts(ctx, tx, authority.GoalRef.String())
	if err != nil {
		return err
	}
	var original application.ActionConsumptionReceipt
	matches := 0
	for _, receipt := range receipts {
		if receipt.ActionRef == authority.ActionRef {
			original = receipt
			matches++
		}
	}
	if matches != 1 || original.Kind != application.ActionLaunchAgent ||
		original.GoalRef != authority.GoalRef || original.WorkItemRef != authority.WorkItemRef ||
		original.ExecutionRef != authority.ExecutionRef || original.PlanGeneration != authority.PlanGeneration ||
		original.WorkItemGeneration != authority.WorkItemGeneration || original.Fence != authority.ActionFence ||
		original.Outcome != application.ActionConsumedQuarantined ||
		original.ErrorCode != "application.effect_unknown_applied" || original.EffectReceiptRef != "" ||
		application.TerminalAgentLaunchReceiptFingerprint(original) != authority.OriginalReceiptFingerprint {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_quarantine_conflict"))
	}
	var valid int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*)
FROM outbox action
JOIN executions execution ON execution.ref=action.execution_ref AND execution.goal_ref=action.goal_ref
 AND execution.work_item_ref=action.work_item_ref
JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
JOIN effect_attempts attempt ON attempt.ref=? AND attempt.action_ref=action.ref
 AND attempt.intent_ref=intent.ref AND attempt.action_fence=?
JOIN agent_capacity_reservations capacity ON capacity.action_ref=action.ref
LEFT JOIN effect_receipts effect_receipt ON effect_receipt.attempt_ref=attempt.ref OR effect_receipt.action_ref=action.ref
LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=execution.budget_reservation_ref
WHERE action.ref=? AND action.kind='launch_agent' AND action.goal_ref=? AND action.work_item_ref=?
 AND action.execution_ref=? AND action.plan_generation=? AND action.work_item_generation=?
 AND action.fence=? AND action.completed_at IS NOT NULL AND action.quarantined_at=action.completed_at
 AND action.last_error_code='application.effect_unknown_applied' AND action.effect_intent_ref=?
 AND intent.digest=? AND intent.kind='agent_launch' AND execution.state='dispatching'
 AND execution.effect_intent_ref=intent.ref AND execution.launch_receipt_ref IS NULL
 AND capacity.state='quarantined' AND capacity.fence=action.fence
 AND effect_receipt.ref IS NULL AND settlement.ref IS NULL`,
		authority.EffectAttemptRef, int64(authority.ActionFence), authority.ActionRef,
		authority.GoalRef.String(), authority.WorkItemRef.String(), authority.ExecutionRef.String(),
		int64(authority.PlanGeneration), int64(authority.WorkItemGeneration), int64(authority.ActionFence),
		authority.EffectIntentRef, authority.EffectIntentDigest).Scan(&valid)
	if err != nil {
		return mapDatabaseError(err)
	}
	if valid != 1 {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_source_conflict"))
	}
	return nil
}

func readTerminalReconciliationAuthorityByRequest(
	ctx context.Context,
	source queryer,
	requestRef string,
) (application.TerminalAgentLaunchReconciliationAuthority, bool, error) {
	var value application.TerminalAgentLaunchReconciliationAuthority
	var principal, project, goalRef, workRef, executionRef, authorizationRef string
	var plan, workGeneration, fence, authorizedAt int64
	err := source.QueryRowContext(ctx, `SELECT ref,job_ref,request_ref,request_fingerprint,
authorization_receipt_ref,principal_ref,project_ref,goal_ref,work_item_ref,execution_ref,action_ref,
effect_intent_ref,effect_intent_digest,effect_attempt_ref,plan_generation,work_item_generation,
action_fence,original_receipt_fingerprint,authorized_at
FROM agent_launch_reconciliation_authorities WHERE request_ref=?`, requestRef).Scan(
		&value.Ref, &value.JobRef, &value.RequestRef, &value.RequestFingerprint, &authorizationRef,
		&principal, &project, &goalRef, &workRef, &executionRef, &value.ActionRef,
		&value.EffectIntentRef, &value.EffectIntentDigest, &value.EffectAttemptRef, &plan,
		&workGeneration, &fence, &value.OriginalReceiptFingerprint, &authorizedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, nil
	}
	if err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, mapDatabaseError(err)
	}
	var errs []error
	value.PrincipalRef, err = identity.NewPrincipalRef(principal)
	errs = append(errs, err)
	value.ProjectRef, err = goal.NewProjectRef(project)
	errs = append(errs, err)
	value.GoalRef, err = goal.NewGoalRef(goalRef)
	errs = append(errs, err)
	value.WorkItemRef, err = goal.NewWorkItemRef(workRef)
	errs = append(errs, err)
	value.ExecutionRef, err = goal.NewExecutionRef(executionRef)
	errs = append(errs, err)
	value.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, authorizationRef)
	errs = append(errs, err)
	if plan <= 0 || workGeneration <= 0 || fence <= 0 {
		errs = append(errs, errors.New("sqlite.agent_launch_reconciliation_generation_invalid"))
	}
	value.PlanGeneration, value.WorkItemGeneration, value.ActionFence =
		goal.PlanGeneration(plan), goal.Revision(workGeneration), uint64(fence)
	value.AuthorizedAt = time.Unix(0, authorizedAt).UTC()
	if err := errors.Join(errs...); err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, invalid(err)
	}
	return value, true, nil
}

func (repository *Repository) claimTerminalAgentLaunchReconciliation(
	ctx context.Context,
	tx *sql.Tx,
	request application.ClaimRequest,
	now time.Time,
) (application.ActionClaim, bool, error) {
	var authorityRef, requestFingerprint string
	err := tx.QueryRowContext(ctx, `SELECT authority.ref,authority.request_fingerprint
FROM agent_launch_reconciliation_jobs job
JOIN agent_launch_reconciliation_authorities authority ON authority.ref=job.authority_ref
WHERE job.state='pending' AND job.available_at<=? AND (job.claim_token IS NULL OR job.claimed_until<=?)
ORDER BY job.available_at,job.ref LIMIT 1`, requiredTime(now), requiredTime(now)).Scan(&authorityRef, &requestFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ActionClaim{}, false, nil
	}
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	authority, found, err := readTerminalReconciliationAuthorityByRef(ctx, tx, authorityRef)
	if err != nil || !found {
		return application.ActionClaim{}, false, err
	}
	record, err := readGoalRecord(ctx, tx, authority.GoalRef.String())
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	intent, found := effectIntentByRefSQLite(record.EffectIntents, authority.EffectIntentRef)
	if !found {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_intent_missing"))
	}
	attempt, found := effectAttemptByRefSQLite(record.EffectAttempts, authority.EffectAttemptRef)
	if !found {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_attempt_missing"))
	}
	approval, found := exactEffectApprovalSQLite(record.EffectApprovals, attempt.ApprovalRef)
	if !found {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_approval_missing"))
	}
	var availableAt, deliveryAttempt, fence int64
	err = tx.QueryRowContext(ctx, `SELECT available_at,delivery_attempt,fence FROM agent_launch_reconciliation_jobs
WHERE authority_ref=? AND state='pending'`, authority.Ref).Scan(&availableAt, &deliveryAttempt, &fence)
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	var originalAvailableAt int64
	err = tx.QueryRowContext(ctx, `SELECT available_at FROM outbox WHERE ref=?`, authority.ActionRef).Scan(&originalAvailableAt)
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	action := application.ActionRecord{
		Ref: authority.ActionRef, Kind: application.ActionLaunchAgent, GoalRef: authority.GoalRef,
		WorkItemRef: authority.WorkItemRef, ExecutionRef: authority.ExecutionRef,
		EffectIntentRef: authority.EffectIntentRef, EffectIntent: intent,
		PlanGeneration: authority.PlanGeneration, WorkItemGeneration: authority.WorkItemGeneration,
		AvailableAt: time.Unix(0, originalAvailableAt).UTC(),
	}
	item, ok := record.Goal.WorkItem(authority.WorkItemRef)
	if !ok || !ports.MatchAgentCapabilities(request.Capabilities, ports.AgentRequirements{RoleKey: item.Role().String()}) {
		return application.ActionClaim{}, false, nil
	}
	budget, active, err := readActiveBudgetReservation(ctx, tx, authority.ActionRef)
	if err != nil || !active {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_budget_missing"))
	}
	capacity, placement, active, err := leerReservaCapacidadAccion(ctx, tx, authority.ActionRef)
	if err != nil || !active || capacity.State != application.AgentCapacityQuarantined {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_capacity_missing"))
	}
	leaseUntil, err := safeLeaseUntil(now, request.LeaseDuration)
	if err != nil {
		return application.ActionClaim{}, false, invalid(err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE agent_launch_reconciliation_jobs
SET claim_token=?,claimed_by=?,claimed_until=?,delivery_attempt=delivery_attempt+1,fence=fence+1,
 last_error_code='',updated_at=?
WHERE authority_ref=? AND state='pending' AND available_at<=?
 AND (claim_token IS NULL OR claimed_until<=?)`, request.Token, request.WorkerRef,
		requiredTime(leaseUntil), requiredTime(now), authority.Ref, requiredTime(now), requiredTime(now))
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.ActionClaim{}, false, err
	}
	claimFence := uint64(fence + 1)
	if claimFence <= authority.ActionFence || deliveryAttempt < 0 {
		return application.ActionClaim{}, false, conflict(errors.New("sqlite.agent_launch_reconciliation_fence_invalid"))
	}
	return application.ActionClaim{
		Action: action, Token: request.Token, WorkerRef: request.WorkerRef,
		DeliveryAttempt: uint64(deliveryAttempt + 1), Fence: claimFence,
		Disposition:              application.ActionClaimDispositionReconcileTerminalLaunch,
		RecoveryEffectAttemptRef: authority.EffectAttemptRef,
		BudgetReservationRef:     budget.Ref, BudgetReservation: budget,
		CapacityReservation: capacity, ReferenciaColocacion: placement,
		EffectApproval: approval, LeaseUntil: leaseUntil, TerminalReconciliationRef: authority.Ref,
		TerminalReconciliationFingerprint: requestFingerprint,
	}, true, nil
}

func readTerminalReconciliationAuthorityByRef(ctx context.Context, source queryer, ref string) (application.TerminalAgentLaunchReconciliationAuthority, bool, error) {
	var requestRef string
	err := source.QueryRowContext(ctx, `SELECT request_ref FROM agent_launch_reconciliation_authorities WHERE ref=?`, ref).Scan(&requestRef)
	if errors.Is(err, sql.ErrNoRows) {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, nil
	}
	if err != nil {
		return application.TerminalAgentLaunchReconciliationAuthority{}, false, mapDatabaseError(err)
	}
	return readTerminalReconciliationAuthorityByRequest(ctx, source, requestRef)
}

func effectIntentByRefSQLite(values []application.EffectIntent, ref string) (application.EffectIntent, bool) {
	for _, value := range values {
		if value.Ref == ref {
			return value, true
		}
	}
	return application.EffectIntent{}, false
}

func effectAttemptByRefSQLite(values []application.EffectAttempt, ref string) (application.EffectAttempt, bool) {
	for _, value := range values {
		if value.Ref == ref {
			return value, true
		}
	}
	return application.EffectAttempt{}, false
}

func exactEffectApprovalSQLite(values []application.EffectApproval, ref string) (application.EffectApproval, bool) {
	var result application.EffectApproval
	matches := 0
	for _, value := range values {
		if value.Ref == ref {
			result = value
			matches++
		}
	}
	return result, matches == 1
}

func validDigest(value string) bool {
	return len(value) == 64 && value != "" && !containsNonHex(value)
}

func containsNonHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && (character < 'a' || character > 'f') {
			return true
		}
	}
	return false
}

func (repository *Repository) ValidateTerminalAgentLaunchReconciliationClaim(
	ctx context.Context,
	claim application.ActionClaim,
) error {
	tx, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireTerminalReconciliationClaim(ctx, tx, claim); err != nil {
		return err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return err
	}
	if !now.Before(claim.LeaseUntil) {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_lease_expired"))
	}
	return commit(tx)
}

func (repository *Repository) RecordTerminalAgentLaunchReconciliationAttempt(
	ctx context.Context,
	state application.RecordTerminalAgentLaunchReconciliationAttemptState,
) error {
	if err := validateTerminalReconciliationAttempt(state.Claim, state.Attempt); err != nil {
		return invalid(err)
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireTerminalReconciliationClaim(ctx, tx, state.Claim); err != nil {
		return err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return err
	}
	if state.Attempt.StartedAt.After(now) || !now.Before(state.Claim.LeaseUntil) {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_attempt_time_invalid"))
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_reconciliation_attempts(
ref,authority_ref,original_effect_attempt_ref,request_fingerprint,job_fence,delivery_attempt,
claim_token,worker_ref,started_at,claim_lease_until) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		state.Attempt.Ref, state.Attempt.AuthorityRef, state.Attempt.OriginalEffectAttemptRef,
		state.Attempt.RequestFingerprint, int64(state.Attempt.JobFence), int64(state.Attempt.DeliveryAttempt),
		state.Attempt.ClaimToken, state.Attempt.WorkerRef, requiredTime(state.Attempt.StartedAt),
		requiredTime(state.Attempt.ClaimLeaseUntil))
	if err != nil {
		return mapDatabaseError(err)
	}
	return commit(tx)
}

func validateTerminalReconciliationAttempt(
	claim application.ActionClaim,
	attempt application.TerminalAgentLaunchReconciliationAttempt,
) error {
	wantRef := "agent-launch-reconciliation-attempt:" + claim.TerminalReconciliationRef + ":" + claim.Token
	if claim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch ||
		!validText(claim.TerminalReconciliationRef) || !validText(attempt.Ref) || attempt.Ref != wantRef ||
		attempt.AuthorityRef != claim.TerminalReconciliationRef ||
		attempt.OriginalEffectAttemptRef != claim.RecoveryEffectAttemptRef ||
		attempt.RequestFingerprint != claim.TerminalReconciliationFingerprint ||
		!validDigest(attempt.RequestFingerprint) || attempt.JobFence != claim.Fence ||
		attempt.DeliveryAttempt != claim.DeliveryAttempt || attempt.ClaimToken != claim.Token ||
		attempt.WorkerRef != claim.WorkerRef || attempt.StartedAt.IsZero() ||
		attempt.ClaimLeaseUntil != claim.LeaseUntil || !attempt.ClaimLeaseUntil.After(attempt.StartedAt) {
		return errors.New("sqlite.agent_launch_reconciliation_attempt_invalid")
	}
	return nil
}

func (repository *Repository) RequeueTerminalAgentLaunchReconciliation(
	ctx context.Context,
	state application.TerminalAgentLaunchReconciliationRequeuedState,
) error {
	if !validText(state.ErrorCode) || state.AvailableAt.IsZero() || state.OperationAt.IsZero() ||
		state.AvailableAt.Before(state.OperationAt) {
		return invalid(errors.New("sqlite.agent_launch_reconciliation_requeue_invalid"))
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireTerminalReconciliationClaim(ctx, tx, state.Claim); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE agent_launch_reconciliation_jobs
SET available_at=?,claim_token=NULL,claimed_by=NULL,claimed_until=NULL,last_error_code=?,updated_at=?
WHERE authority_ref=? AND state='pending' AND claim_token=? AND claimed_by=? AND claimed_until=?
 AND delivery_attempt=? AND fence=?`, requiredTime(state.AvailableAt), state.ErrorCode,
		requiredTime(state.OperationAt), state.Claim.TerminalReconciliationRef, state.Claim.Token,
		state.Claim.WorkerRef, requiredTime(state.Claim.LeaseUntil), int64(state.Claim.DeliveryAttempt),
		int64(state.Claim.Fence))
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	return commit(tx)
}

func (repository *Repository) QuarantineTerminalAgentLaunchReconciliation(
	ctx context.Context,
	state application.TerminalAgentLaunchReconciliationQuarantinedState,
) error {
	if !validText(state.ErrorCode) || state.OperationAt.IsZero() {
		return invalid(errors.New("sqlite.agent_launch_reconciliation_quarantine_invalid"))
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireTerminalReconciliationClaim(ctx, tx, state.Claim); err != nil {
		return err
	}
	var attemptRef any
	if state.Attempt != nil {
		if err := validateTerminalReconciliationAttempt(state.Claim, *state.Attempt); err != nil {
			return invalid(err)
		}
		if err := requireStoredTerminalReconciliationAttempt(ctx, tx, *state.Attempt); err != nil {
			return err
		}
		attemptRef = state.Attempt.Ref
	}
	result, err := tx.ExecContext(ctx, `UPDATE agent_launch_reconciliation_jobs
SET state='quarantined',last_error_code=?,updated_at=?
WHERE authority_ref=? AND state='pending' AND claim_token=? AND claimed_by=? AND claimed_until=?
 AND delivery_attempt=? AND fence=?`, state.ErrorCode, requiredTime(state.OperationAt),
		state.Claim.TerminalReconciliationRef, state.Claim.Token, state.Claim.WorkerRef,
		requiredTime(state.Claim.LeaseUntil), int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence))
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_reconciliation_receipts(
ref,authority_ref,reconciliation_attempt_ref,outcome,error_code,effect_receipt_ref,completed_at)
VALUES(?,?,?,'quarantined',?,NULL,?)`,
		"agent-launch-reconciliation-receipt:"+state.Claim.TerminalReconciliationRef,
		state.Claim.TerminalReconciliationRef, attemptRef, state.ErrorCode, requiredTime(state.OperationAt))
	if err != nil {
		return mapDatabaseError(err)
	}
	return commit(tx)
}

func (repository *Repository) RecordTerminalAgentLaunchReconciled(
	ctx context.Context,
	state application.TerminalAgentLaunchReconciliationCompletedState,
) error {
	if state.OperationAt.IsZero() || state.Execution.State != application.ExecutionRunning ||
		state.EffectReceipt.Status != application.EffectStatusAccepted || state.NextAction.Kind == "" ||
		state.Event.Kind != "execution.accepted" {
		return invalid(errors.New("sqlite.agent_launch_reconciliation_completion_invalid"))
	}
	if err := validateTerminalReconciliationAttempt(state.Claim, state.Attempt); err != nil {
		return invalid(err)
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireTerminalReconciliationClaim(ctx, tx, state.Claim); err != nil {
		return err
	}
	if err := requireStoredTerminalReconciliationAttempt(ctx, tx, state.Attempt); err != nil {
		return err
	}
	now, err := repository.transactionTime()
	if err != nil {
		return err
	}
	if state.OperationAt.After(now) || !now.Before(state.Claim.LeaseUntil) ||
		state.EffectReceipt.ConfirmedAt.After(state.OperationAt) {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_completion_time_invalid"))
	}
	if err := updateExecutionCAS(ctx, tx, state.Execution, application.ExecutionDispatching); err != nil {
		return err
	}
	if err := insertTerminalReconciledEffectReceipt(ctx, tx, state); err != nil {
		return err
	}
	if err := persistirTransicionCapacidad(ctx, tx, state.Claim.CapacityReservation,
		application.AgentCapacityConsumed, application.AgentCapacityCauseReconciliation,
		state.Claim.TerminalReconciliationRef, state.EffectReceipt.AttemptRef,
		state.EffectReceipt.Ref, state.OperationAt); err != nil {
		return err
	}
	if err := insertAction(ctx, tx, state.NextAction); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, state.Event); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE agent_launch_reconciliation_jobs
SET state='completed',last_error_code='',updated_at=?
WHERE authority_ref=? AND state='pending' AND claim_token=? AND claimed_by=? AND claimed_until=?
 AND delivery_attempt=? AND fence=?`, requiredTime(state.OperationAt), state.Claim.TerminalReconciliationRef,
		state.Claim.Token, state.Claim.WorkerRef, requiredTime(state.Claim.LeaseUntil),
		int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence))
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_reconciliation_receipts(
ref,authority_ref,reconciliation_attempt_ref,outcome,error_code,effect_receipt_ref,completed_at)
VALUES(?,?,?,'completed','',?,?)`,
		"agent-launch-reconciliation-receipt:"+state.Claim.TerminalReconciliationRef,
		state.Claim.TerminalReconciliationRef, state.Attempt.Ref, state.EffectReceipt.Ref,
		requiredTime(state.OperationAt))
	if err != nil {
		return mapDatabaseError(err)
	}
	return commit(tx)
}

func requireTerminalReconciliationClaim(ctx context.Context, tx *sql.Tx, claim application.ActionClaim) error {
	if claim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch ||
		claim.Action.Kind != application.ActionLaunchAgent || claim.Token == "" || claim.WorkerRef == "" ||
		claim.DeliveryAttempt == 0 || claim.Fence == 0 || claim.LeaseUntil.IsZero() ||
		claim.TerminalReconciliationRef == "" || claim.RecoveryEffectAttemptRef == "" {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_claim_invalid"))
	}
	var token, worker, state string
	var lease, delivery, fence int64
	err := tx.QueryRowContext(ctx, `SELECT claim_token,claimed_by,claimed_until,delivery_attempt,fence,state
FROM agent_launch_reconciliation_jobs WHERE authority_ref=?`, claim.TerminalReconciliationRef).Scan(
		&token, &worker, &lease, &delivery, &fence, &state)
	if err != nil {
		return mapDatabaseError(err)
	}
	if state != "pending" || token != claim.Token || worker != claim.WorkerRef ||
		lease != requiredTime(claim.LeaseUntil) || delivery != int64(claim.DeliveryAttempt) ||
		fence != int64(claim.Fence) {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_claim_conflict"))
	}
	authority, found, err := readTerminalReconciliationAuthorityByRef(ctx, tx, claim.TerminalReconciliationRef)
	if err != nil || !found {
		return err
	}
	if authority.ActionRef != claim.Action.Ref || authority.GoalRef != claim.Action.GoalRef ||
		authority.WorkItemRef != claim.Action.WorkItemRef || authority.ExecutionRef != claim.Action.ExecutionRef ||
		authority.EffectIntentRef != claim.Action.EffectIntentRef ||
		authority.EffectIntentDigest != claim.Action.EffectIntent.Digest ||
		authority.EffectAttemptRef != claim.RecoveryEffectAttemptRef ||
		authority.RequestFingerprint != claim.TerminalReconciliationFingerprint ||
		authority.PlanGeneration != claim.Action.PlanGeneration ||
		authority.WorkItemGeneration != claim.Action.WorkItemGeneration ||
		claim.Fence <= authority.ActionFence {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_authority_conflict"))
	}
	return requireTerminalReconciliationSource(ctx, tx, authority)
}

func requireStoredTerminalReconciliationAttempt(
	ctx context.Context,
	tx *sql.Tx,
	attempt application.TerminalAgentLaunchReconciliationAttempt,
) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_launch_reconciliation_attempts
WHERE ref=? AND authority_ref=? AND original_effect_attempt_ref=? AND request_fingerprint=?
 AND job_fence=? AND delivery_attempt=? AND claim_token=? AND worker_ref=?
 AND started_at=? AND claim_lease_until=?`, attempt.Ref, attempt.AuthorityRef,
		attempt.OriginalEffectAttemptRef, attempt.RequestFingerprint, int64(attempt.JobFence),
		int64(attempt.DeliveryAttempt), attempt.ClaimToken, attempt.WorkerRef,
		requiredTime(attempt.StartedAt), requiredTime(attempt.ClaimLeaseUntil)).Scan(&count)
	if err != nil {
		return mapDatabaseError(err)
	}
	if count != 1 {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_attempt_conflict"))
	}
	return nil
}

func insertTerminalReconciledEffectReceipt(
	ctx context.Context,
	tx *sql.Tx,
	state application.TerminalAgentLaunchReconciliationCompletedState,
) error {
	receipt := state.EffectReceipt
	intent := state.Claim.Action.EffectIntent
	if receipt.IntentRef != intent.Ref || receipt.IntentDigest != intent.Digest ||
		receipt.ApprovalRef != state.Claim.EffectApproval.Ref ||
		receipt.AttemptRef != state.Claim.RecoveryEffectAttemptRef || receipt.Subject != intent.Subject ||
		receipt.ActionRef != state.Claim.Action.Ref || receipt.ActionFence >= state.Claim.Fence ||
		receipt.IdempotencyKey != intent.IdempotencyKey || receipt.ExternalRef == "" ||
		receipt.ConfirmedAt.Before(intent.CreatedAt) ||
		governance.ValidateResourceUsage(receipt.Usage) != nil {
		return conflict(errors.New("sqlite.agent_launch_reconciliation_effect_receipt_invalid"))
	}
	usage := receipt.Usage.Resources
	_, err := tx.ExecContext(ctx, `INSERT INTO effect_receipts(
ref,intent_ref,intent_digest,approval_ref,attempt_ref,project_ref,goal_ref,work_item_ref,execution_ref,
plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,idempotency_key,
external_ref,status,usage_tokens,usage_money_micros,usage_currency,usage_active_time_ns,
usage_process_slots,usage_disk_bytes,usage_known,usage_quality,confirmed_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		receipt.Ref, receipt.IntentRef, receipt.IntentDigest, receipt.ApprovalRef, receipt.AttemptRef,
		receipt.Subject.ProjectRef.String(), receipt.Subject.GoalRef.String(), receipt.Subject.WorkItemRef.String(),
		receipt.Subject.ExecutionRef.String(), int64(receipt.Subject.PlanGeneration),
		int64(receipt.Subject.AppSpecGeneration), receipt.Subject.SpecHash, receipt.Subject.ActorRef.String(),
		receipt.ActionRef, int64(receipt.ActionFence), receipt.IdempotencyKey, receipt.ExternalRef,
		receipt.Status, usage.Tokens, usage.MoneyMicros, string(usage.Currency), usage.ActiveTimeNS,
		usage.ProcessSlots, usage.DiskBytes, int64(receipt.Usage.Known), string(receipt.Usage.Quality),
		requiredTime(receipt.ConfirmedAt))
	return mapDatabaseError(err)
}
