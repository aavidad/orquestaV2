package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

const sqliteV28ReconciliationPendingCode = "agent.launch_reconciliation_pending"

func TestV28RecoveryRequeueRestartsWithSamePhysicalAuthority(t *testing.T) {
	system, _, attempt := seedV27AmbiguousEffectAttempt(t, "v28-recovery-requeue-restart")
	system.clock.Advance(time.Minute + time.Nanosecond)
	claim := claimSQLiteV28Recovery(t, system, "claim:v28-recovery-requeue:first", attempt.Ref)
	state := sqliteV28RecoveryRequeueState(t, system, claim)
	beforeCounts := v28RecoveryLedgerCounts(t, system.repository.db)

	if err := system.repository.RequeueAction(context.Background(), state); err != nil {
		t.Fatalf("requeue recovery: %s", sqliteTestErrorChain(err))
	}
	requeued, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(requeued.Executions, claim.Action.ExecutionRef)
	if !found || execution != state.Execution || len(requeued.EffectAttempts) != 1 ||
		requeued.EffectAttempts[0] != attempt || len(requeued.EffectReceipts) != 0 ||
		len(requeued.ConsumptionReceipts) != 0 || len(requeued.BudgetSettlements) != 0 ||
		v28RecoveryLedgerCounts(t, system.repository.db) != beforeCounts {
		t.Fatalf("execution=%+v attempts=%+v receipts=%+v consumptions=%+v settlements=%+v",
			execution, requeued.EffectAttempts, requeued.EffectReceipts,
			requeued.ConsumptionReceipts, requeued.BudgetSettlements)
	}
	var token, worker, recoveryRef sql.NullString
	var lease sql.NullInt64
	var availableAt int64
	var lastError string
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT claim_token,claimed_by,claimed_until,available_at,last_error_code,recovery_effect_attempt_ref
FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(
		&token, &worker, &lease, &availableAt, &lastError, &recoveryRef,
	))
	if token.Valid || worker.Valid || lease.Valid || availableAt != requiredTime(state.AvailableAt) ||
		lastError != sqliteV28ReconciliationPendingCode || !recoveryRef.Valid || recoveryRef.String != attempt.Ref {
		t.Fatalf("token=%+v worker=%+v lease=%+v available=%d error=%q recovery=%+v",
			token, worker, lease, availableAt, lastError, recoveryRef)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("unclaimed recovery ref is not recoverable: %s", sqliteTestErrorChain(err))
	}

	restartSQLiteV15System(t, system)
	system.clock.Advance(claim.Action.EffectIntent.QuotaRetryDelay)
	next, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v28-recovery-requeue:second", Token: "claim:v28-recovery-requeue:second",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	})
	if err != nil || !found {
		t.Fatalf("reclaim recovery found=%t err=%s", found, sqliteTestErrorChain(err))
	}
	nextAction := next.Action
	nextAction.AvailableAt = claim.Action.AvailableAt
	sameActionAuthority := reflect.DeepEqual(nextAction, claim.Action)
	if next.Disposition != application.ActionClaimDispositionRecoverEffect ||
		next.RecoveryEffectAttemptRef != attempt.Ref || next.Fence <= claim.Fence ||
		next.DeliveryAttempt != claim.DeliveryAttempt+1 ||
		!next.Action.AvailableAt.Equal(state.AvailableAt) || !sameActionAuthority ||
		next.EffectApproval != claim.EffectApproval ||
		next.BudgetReservationRef != claim.BudgetReservationRef || next.BudgetReservation != claim.BudgetReservation ||
		next.CapacityReservation != claim.CapacityReservation ||
		next.ReferenciaColocacion != claim.ReferenciaColocacion {
		t.Fatalf("disposition=%s recovery=%q/%q fence=%d>%d delivery=%d/%d available=%s/%s action_authority=%t approval=%t budget_ref=%q/%q budget=%t capacity=%t placement=%q/%q",
			next.Disposition, next.RecoveryEffectAttemptRef, attempt.Ref, next.Fence, claim.Fence,
			next.DeliveryAttempt, claim.DeliveryAttempt+1, next.Action.AvailableAt, state.AvailableAt,
			sameActionAuthority, next.EffectApproval == claim.EffectApproval,
			next.BudgetReservationRef, claim.BudgetReservationRef,
			next.BudgetReservation == claim.BudgetReservation,
			next.CapacityReservation == claim.CapacityReservation,
			next.ReferenciaColocacion, claim.ReferenciaColocacion)
	}
	restarted, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	if len(restarted.EffectAttempts) != 1 || restarted.EffectAttempts[0] != attempt ||
		len(restarted.EffectReceipts) != 0 || len(restarted.ConsumptionReceipts) != 0 ||
		len(restarted.BudgetReservations) != 1 || restarted.BudgetReservations[0] != claim.BudgetReservation ||
		len(restarted.EffectApprovals) != 1 || restarted.EffectApprovals[0] != claim.EffectApproval ||
		len(restarted.BudgetSettlements) != 0 || v28RecoveryLedgerCounts(t, system.repository.db) != beforeCounts {
		t.Fatalf("restart record=%+v counts=%+v want=%+v", restarted,
			v28RecoveryLedgerCounts(t, system.repository.db), beforeCounts)
	}
}

func TestV28RecoveryRequeueThenRevocationParksUnclaimedExactAttempt(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	approver := testPrincipal(t, "principal:v28-requeue-approver", "actor:v28-requeue-approver", identity.PrincipalKindHuman)
	membership := grantTestMembership(t, system.repository, owner, approver, system.project,
		identity.RoleProjectAdmin, "membership:v28-requeue-approver", system.clock.Now())
	approverAccess, err := application.NewAccess(approver, system.project)
	sqliteTestNoError(t, err)
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v28-requeue-revocation", Statement: "sensitive recovery retry", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v28-requeue-revocation", Key: "phase:v28-requeue-revocation",
				TemplateRef: "phase-template:v28-requeue-revocation",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work", Objective: "sensitive recovery retry", Phase: "phase:v28-requeue-revocation",
				Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
				SecurityCriticality: governance.SecurityCriticalitySensitive,
				ReasoningEffort:     governance.ReasoningEffortMedium,
			}},
		},
	})
	sqliteTestNoError(t, err)
	intent := created.Record.EffectIntents[0]
	approved, err := system.orchestrator.DecideEffect(context.Background(), approverAccess, application.DecideEffectRequest{
		RequestRef: "approval:v28-requeue-revocation", GoalRef: created.Record.Goal.Ref(),
		IntentRef: intent.Ref, ExpectedIntentDigest: intent.Digest,
		Decision: application.EffectApproved, Reason: "bounded recovery authority",
	})
	if err != nil || !approved.Created {
		t.Fatalf("approval=%+v err=%v", approved, err)
	}
	first := claimSQLiteV15(t, system, "claim:v28-requeue-revocation:first")
	prepareSQLiteV15Launch(t, system, first)
	attempt := sqliteV15Attempt(first, system.clock.Now())
	attempt.ClaimLeaseUntil = first.LeaseUntil.UTC()
	_, createdAttempt, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: first, Attempt: attempt, OperationAt: system.clock.Now(),
		},
	)
	if err != nil || !createdAttempt {
		t.Fatalf("attempt created=%t err=%s", createdAttempt, sqliteTestErrorChain(err))
	}
	system.clock.Advance(time.Minute + time.Nanosecond)
	claim := claimSQLiteV28Recovery(t, system, "claim:v28-requeue-revocation:recovery", attempt.Ref)
	requeue := sqliteV28RecoveryRequeueState(t, system, claim)
	sqliteTestNoError(t, system.repository.RequeueAction(context.Background(), requeue))

	authorization := authorizeTest(t, system.repository, owner, system.project,
		identity.PermissionProjectMembershipManage, approver.Ref.String(),
		"authorization:v28-requeue-revoke-approver", system.clock.Now())
	revoke := testRevokeRequest(t, "membership:v28-requeue-revoke-approver", owner, approver.Ref,
		system.project, membership.Revision(), system.clock.Now())
	_, _, changed, err := system.repository.RevokeMembership(context.Background(), application.MembershipRevokeState{
		AuthorizationReceipt: authorization, Request: revoke,
	})
	if err != nil || !changed {
		t.Fatalf("revoke changed=%t err=%v", changed, err)
	}
	system.clock.Advance(claim.Action.EffectIntent.QuotaRetryDelay)
	next, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v28-requeue-revoked", Token: "claim:v28-requeue-revoked",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: system.policy, ExcludeLaunch: true,
	})
	if err != nil || found || next != (application.ActionClaim{}) {
		t.Fatalf("revoked retry claim=%+v found=%t err=%s", next, found, sqliteTestErrorChain(err))
	}
	var token, worker sql.NullString
	var recoveryRef, code string
	var lease sql.NullInt64
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT claim_token,claimed_by,claimed_until,recovery_effect_attempt_ref,last_error_code
FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&token, &worker, &lease, &recoveryRef, &code))
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, executionFound := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if token.Valid || worker.Valid || lease.Valid || recoveryRef != attempt.Ref ||
		code != "governance.effect_approval_required" || !executionFound ||
		execution.State != application.ExecutionDispatching ||
		execution.BudgetReservationRef != claim.BudgetReservationRef ||
		execution.EffectIntentRef != claim.Action.EffectIntentRef ||
		len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != attempt ||
		len(record.EffectReceipts) != 0 || len(record.ConsumptionReceipts) != 0 ||
		len(record.BudgetSettlements) != 0 {
		t.Fatalf("token=%+v worker=%+v lease=%+v recovery=%q code=%q execution=%+v attempts=%+v receipts=%+v consumptions=%+v settlements=%+v",
			token, worker, lease, recoveryRef, code, execution, record.EffectAttempts,
			record.EffectReceipts, record.ConsumptionReceipts, record.BudgetSettlements)
	}
}

func TestV28RecoveryRequeueRejectsPersistedAuthorityTransitionAtomically(t *testing.T) {
	tests := map[string]func(*testing.T, *sqliteV15System, application.ActionClaim){
		"capacity transitioned": func(t *testing.T, system *sqliteV15System, claim application.ActionClaim) {
			transaction, err := beginTransaction(context.Background(), system.repository)
			sqliteTestNoError(t, err)
			sqliteTestNoError(t, persistirTransicionCapacidad(
				context.Background(), transaction, claim.CapacityReservation,
				application.AgentCapacityQuarantined, application.AgentCapacityCauseUnknownApplied,
				"cause:v28-requeue-capacity-transition", "", "", system.clock.Now(),
			))
			sqliteTestNoError(t, commit(transaction))
		},
		"budget settled": func(t *testing.T, system *sqliteV15System, claim application.ActionClaim) {
			zero := governance.ResourceVector{Currency: claim.BudgetReservation.Resources.Currency}
			settlement, err := governance.Reconcile(claim.BudgetReservation, governance.ResourceUsage{
				Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
			})
			sqliteTestNoError(t, err)
			settlement.SettledAt = system.clock.Now()
			transaction, err := beginTransaction(context.Background(), system.repository)
			sqliteTestNoError(t, err)
			sqliteTestNoError(t, insertBudgetSettlement(context.Background(), transaction, settlement))
			sqliteTestNoError(t, commit(transaction))
		},
	}
	for name, transition := range tests {
		t.Run(name, func(t *testing.T) {
			system, _, attempt := seedV27AmbiguousEffectAttempt(t,
				"v28-recovery-requeue-persisted-"+testRefSuffix(name))
			system.clock.Advance(time.Minute + time.Nanosecond)
			claim := claimSQLiteV28Recovery(t, system,
				"claim:v28-recovery-requeue-persisted:"+testRefSuffix(name), attempt.Ref)
			state := sqliteV28RecoveryRequeueState(t, system, claim)
			transition(t, system, claim)
			beforeRecord, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
			sqliteTestNoError(t, err)
			beforeCapacity, beforePlacement, found, err := leerReservaCapacidadAccion(
				context.Background(), system.repository.db, claim.Action.Ref,
			)
			sqliteTestNoError(t, err)
			if !found {
				t.Fatal("capacity reservation missing")
			}
			beforeOutbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, claim.Action.Ref)

			err = system.repository.RequeueAction(context.Background(), state)
			if !application.IsStateError(err, application.StateConflict) {
				t.Fatalf("persisted transition error=%s", sqliteTestErrorChain(err))
			}
			afterRecord, getErr := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
			sqliteTestNoError(t, getErr)
			afterCapacity, afterPlacement, found, getErr := leerReservaCapacidadAccion(
				context.Background(), system.repository.db, claim.Action.Ref,
			)
			sqliteTestNoError(t, getErr)
			if !found || !reflect.DeepEqual(afterRecord, beforeRecord) || afterCapacity != beforeCapacity ||
				afterPlacement != beforePlacement ||
				sqliteV28RecoveryRequeueOutbox(t, system.repository.db, claim.Action.Ref) != beforeOutbox {
				t.Fatalf("record_changed=%t capacity before=%+v/%s after=%+v/%s found=%t",
					!reflect.DeepEqual(afterRecord, beforeRecord), beforeCapacity, beforePlacement,
					afterCapacity, afterPlacement, found)
			}
		})
	}
}

func TestV28RecoveryRequeueRejectsCrossedBindingsAndSettlementAtomically(t *testing.T) {
	tests := map[string]func(*application.ActionRequeuedState){
		"execution not dispatching": func(state *application.ActionRequeuedState) {
			state.Execution.State = application.ExecutionQueued
		},
		"recovery attempt crossed": func(state *application.ActionRequeuedState) {
			state.Claim.RecoveryEffectAttemptRef += ":crossed"
		},
		"budget binding crossed": func(state *application.ActionRequeuedState) {
			state.Execution.BudgetReservationRef += ":crossed"
		},
		"intent binding crossed": func(state *application.ActionRequeuedState) {
			state.Execution.EffectIntentRef += ":crossed"
		},
		"settlement forbidden": func(state *application.ActionRequeuedState) {
			zero := governance.ResourceVector{Currency: state.Claim.BudgetReservation.Resources.Currency}
			settlement, err := governance.Reconcile(state.Claim.BudgetReservation, governance.ResourceUsage{
				Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
			})
			if err != nil {
				panic(err)
			}
			settlement.SettledAt = state.OperationAt
			state.BudgetSettlement = &settlement
		},
		"effect clear forbidden": func(state *application.ActionRequeuedState) {
			state.ClearEffectBinding = true
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			system, _, attempt := seedV27AmbiguousEffectAttempt(t, "v28-recovery-requeue-negative-"+testRefSuffix(name))
			system.clock.Advance(time.Minute + time.Nanosecond)
			claim := claimSQLiteV28Recovery(t, system,
				"claim:v28-recovery-requeue-negative:"+testRefSuffix(name), attempt.Ref)
			state := sqliteV28RecoveryRequeueState(t, system, claim)
			beforeRecord, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
			sqliteTestNoError(t, err)
			beforeOutbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, claim.Action.Ref)
			beforeCounts := v28RecoveryLedgerCounts(t, system.repository.db)
			mutate(&state)

			if err := system.repository.RequeueAction(context.Background(), state); err == nil {
				t.Fatalf("invalid recovery requeue accepted: %+v", state)
			}
			afterRecord, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
			sqliteTestNoError(t, err)
			afterOutbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, claim.Action.Ref)
			if !reflect.DeepEqual(afterRecord, beforeRecord) || afterOutbox != beforeOutbox ||
				v28RecoveryLedgerCounts(t, system.repository.db) != beforeCounts {
				t.Fatalf("record_changed=%t outbox before=%+v after=%+v counts before=%+v after=%+v",
					!reflect.DeepEqual(afterRecord, beforeRecord), beforeOutbox, afterOutbox,
					beforeCounts, v28RecoveryLedgerCounts(t, system.repository.db))
			}
		})
	}
}

func sqliteV28RecoveryRequeueState(
	t *testing.T, system *sqliteV15System, claim application.ActionClaim,
) application.ActionRequeuedState {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionDispatching {
		t.Fatalf("recovery execution=%+v found=%t", execution, found)
	}
	now := system.clock.Now().UTC()
	return application.ActionRequeuedState{
		Claim: claim, Execution: execution, ErrorCode: sqliteV28ReconciliationPendingCode,
		AvailableAt: now.Add(claim.Action.EffectIntent.QuotaRetryDelay), OperationAt: now,
	}
}

type sqliteV28RecoveryRequeueOutboxState struct {
	token, worker, recoveryRef sql.NullString
	lease                      sql.NullInt64
	availableAt                int64
	deliveryAttempt, fence     int64
	lastError                  string
}

func sqliteV28RecoveryRequeueOutbox(
	t *testing.T, database *sql.DB, actionRef string,
) sqliteV28RecoveryRequeueOutboxState {
	t.Helper()
	var state sqliteV28RecoveryRequeueOutboxState
	sqliteTestNoError(t, database.QueryRow(`
SELECT claim_token,claimed_by,claimed_until,available_at,delivery_attempt,fence,
       last_error_code,recovery_effect_attempt_ref
FROM outbox WHERE ref=?`, actionRef).Scan(
		&state.token, &state.worker, &state.lease, &state.availableAt,
		&state.deliveryAttempt, &state.fence, &state.lastError, &state.recoveryRef,
	))
	return state
}
