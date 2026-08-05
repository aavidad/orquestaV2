package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestClaimValidatesRuntimePolicyWithoutMutatingHistoricalAdmission(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	created := system.submit(t, "request:v15-runtime-policy")
	invalidPolicies := []application.BudgetPolicy{{}, system.policy}
	invalidPolicies[1].EffectApprovalTTL = 0
	for index, policy := range invalidPolicies {
		_, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v15-invalid-policy", Token: "claim:v15-invalid-policy:" + string(rune('a'+index)),
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: policy,
		})
		if found || !application.IsStateError(err, application.StateInvalid) {
			t.Fatalf("invalid policy %d found=%v err=%v", index, found, err)
		}
	}
	var reservations, attempts, deferred, claimed, fairness int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM budget_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*),SUM(CASE WHEN last_error_code<>'' THEN 1 ELSE 0 END),
 SUM(CASE WHEN claim_token IS NOT NULL THEN 1 ELSE 0 END) FROM outbox`).Scan(
		new(int), &deferred, &claimed,
	); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM fairness_cursors`).Scan(&fairness); err != nil {
		t.Fatal(err)
	}
	if reservations != 0 || attempts != 0 || deferred != 0 || claimed != 0 || fairness != 0 {
		t.Fatalf("invalid policy mutated state reservations=%d attempts=%d deferred=%d claimed=%d fairness=%d",
			reservations, attempts, deferred, claimed, fairness)
	}

	rotated := rotatedSQLiteV15Policy(system.policy, system.clock.Now())
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-rotated-policy", Token: "claim:v15-rotated-policy", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: rotated, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found {
		t.Fatalf("valid rotated runtime policy claim=%+v found=%v err=%v", claim, found, err)
	}
	intent := created.Record.EffectIntents[0]
	if claim.Action.EffectIntent.PolicyHash != intent.PolicyHash ||
		claim.Action.EffectIntent.PolicyRevision != intent.PolicyRevision ||
		claim.Action.EffectIntent.ApprovalTTL != intent.ApprovalTTL ||
		claim.Action.EffectIntent.QuotaRetryDelay != intent.QuotaRetryDelay ||
		claim.BudgetReservation.PolicyHash != intent.PolicyHash || claim.BudgetReservation.Resources != intent.Demand.Resources {
		t.Fatalf("runtime policy rewrote historical admission claim=%+v intent=%+v", claim, intent)
	}
}

func TestV15RejectsFutureSettlementWithoutPartialMutation(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	created := system.submit(t, "request:v15-future-settlement")
	claim := claimSQLiteV15(t, system, "claim:v15-future-settlement")
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("claimed execution missing")
	}
	zero := governance.ResourceVector{Currency: claim.BudgetReservation.Resources.Currency}
	settlement, err := governance.Reconcile(claim.BudgetReservation, governance.ResourceUsage{
		Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	settlement.SettledAt = system.clock.Now().Add(time.Second)
	execution.BudgetReservationRef, execution.EffectIntentRef, execution.LaunchReceiptRef = "", "", ""
	err = system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: settlement.SettledAt.Add(time.Second),
		OperationAt: settlement.SettledAt, BudgetSettlement: &settlement, ClearEffectBinding: true,
	})
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("future settlement accepted: %v", err)
	}
	after, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(after.BudgetSettlements) != 0 || after.Executions[0].BudgetReservationRef != claim.BudgetReservationRef {
		t.Fatalf("future settlement partially mutated record=%+v err=%v", after, err)
	}
	rewriteRecoveryTrigger(t, system.repository.db, "budget_reservations_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `UPDATE budget_reservations SET reserved_at=? WHERE ref=?`,
			requiredTime(claim.LeaseUntil), claim.BudgetReservationRef)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v15_budget_reservation_invalid") {
		t.Fatalf("post-lease reservation recovery accepted: %v", err)
	}
}

func TestV15EffectAttemptRequiresPersistedDispatchFrontier(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v15-attempt-before-dispatch")
	claim := claimSQLiteV15(t, system, "claim:v15-attempt-before-dispatch")
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("pre-dispatch attempt accepted: %v cause=%v", err, errors.Unwrap(err))
	}
	var attempts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatalf("pre-dispatch attempt partially persisted count=%d err=%v", attempts, err)
	}
	system.clock.Advance(time.Second)
	prepareSQLiteV15Launch(t, system, claim)
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: attempt.StartedAt,
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("attempt before dispatch event accepted: %v cause=%v", err, errors.Unwrap(err))
	}
	attempt = sqliteV15Attempt(claim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil || !created {
		t.Fatalf("prepared attempt created=%v err=%v", created, err)
	}
}

func rotatedSQLiteV15Policy(base application.BudgetPolicy, at time.Time) application.BudgetPolicy {
	const hash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	base.PolicyHash, base.QuotaRetryDelay, base.EffectApprovalTTL = hash, 9*time.Second, 2*time.Hour
	for _, envelope := range []*governance.BudgetEnvelope{
		&base.DeploymentEnvelope, &base.ProjectEnvelopeTemplate, &base.GoalEnvelopeTemplate,
	} {
		envelope.PolicyHash, envelope.Revision, envelope.CreatedAt = hash, 2, at.UTC()
		envelope.Ref += ":rotated"
	}
	return base
}

func TestV15RejectsCrossKindEffectReceiptsBeforeAtomicCommit(t *testing.T) {
	t.Run("launch cannot persist stop receipt", func(t *testing.T) {
		system := newSQLiteV15System(t, 2)
		created := system.submit(t, "request:v15-cross-kind-launch")
		claim := claimSQLiteV15(t, system, "claim:v15-cross-kind-launch")
		record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
		if err != nil {
			t.Fatal(err)
		}
		item := record.Goal.WorkItems()[0]
		started, err := record.Goal.StartWorkItem(
			record.Goal.Revision(), item.Revision(), item.Ref(), claim.Action.ExecutionRef, system.clock.Now(),
		)
		if err != nil {
			t.Fatal(err)
		}
		execution := record.Executions[0]
		execution.State = application.ExecutionDispatching
		execution.BudgetReservationRef = claim.BudgetReservationRef
		execution.EffectIntentRef = claim.Action.EffectIntentRef
		if err := system.repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
			Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: started, Execution: execution,
			OperationAt: system.clock.Now(), Event: application.EventRecord{
				Ref: "event:v15-cross-kind-dispatching", Kind: "execution.dispatching", GoalRef: started.Ref(),
				WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: system.clock.Now(),
			},
		}); err != nil {
			t.Fatal(err)
		}
		attempt := sqliteV15Attempt(claim, system.clock.Now())
		if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		execution.State = application.ExecutionRunning
		execution.ProviderRef, execution.ModelRef, execution.AgentRef = "provider:codex", "model:codex", "agent:codex"
		execution.ExternalRef, execution.LaunchReceiptRef = "external:"+execution.Ref.String(), ""
		execution.StartedAt, execution.ProviderAcceptedAt = system.clock.Now(), system.clock.Now()
		execution.DeadlineAt = system.clock.Now().Add(time.Hour)
		receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusStopped, system.clock.Now())
		execution.LaunchReceiptRef = receipt.Ref
		startedItem, _ := started.WorkItem(item.Ref())
		err = system.repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
			Claim: claim, Execution: execution, EffectReceipt: receipt, OperationAt: system.clock.Now(),
			NextAction: application.ActionRecord{
				Ref: "action:observe:" + execution.Ref.String(), Kind: application.ActionObserveAgent,
				GoalRef: started.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
				PlanGeneration: started.PlanGeneration(), WorkItemGeneration: startedItem.Revision(), AvailableAt: system.clock.Now(),
			},
			Event: application.EventRecord{
				Ref: "event:v15-cross-kind-accepted", Kind: "execution.accepted", GoalRef: started.Ref(),
				WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: system.clock.Now(),
			},
		})
		if !application.IsStateError(err, application.StateInvalid) {
			t.Fatalf("launch accepted stop receipt: %v", err)
		}
		after, readErr := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
		if readErr != nil || after.Executions[0].State != application.ExecutionDispatching ||
			len(after.EffectReceipts) != 0 || len(after.ConsumptionReceipts) != 0 {
			t.Fatalf("cross-kind launch partially committed record=%+v err=%v", after, readErr)
		}
	})

	t.Run("stop cannot persist launch receipt", func(t *testing.T) {
		system := newSQLiteV15System(t, 2)
		created := system.submit(t, "request:v15-cross-kind-stop")
		if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-cross-kind-stop-launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
			t.Fatalf("launch result=%+v err=%v", result, err)
		}
		running, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
		if err != nil {
			t.Fatal(err)
		}
		controlSQLiteV15Execution(t, system, running, "control:v15-cross-kind-stop", ports.AgentStopCooperative)
		claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v15-cross-kind-stop", Token: "claim:v15-cross-kind-stop",
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		})
		if err != nil || !found || claim.Action.Kind != application.ActionStopAgent {
			t.Fatalf("stop claim=%+v found=%v err=%v", claim, found, err)
		}
		attempt := sqliteV15Attempt(claim, system.clock.Now())
		if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, system.clock.Now())
		tx, err := beginTransaction(context.Background(), system.repository)
		if err != nil {
			t.Fatal(err)
		}
		err = insertEffectReceipt(context.Background(), tx, receipt)
		_ = tx.Rollback()
		if !application.IsStateError(err, application.StateInvalid) {
			t.Fatalf("stop accepted launch receipt: %v", err)
		}
		var count int
		if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_receipts WHERE intent_ref=?`,
			claim.Action.EffectIntentRef).Scan(&count); err != nil || count != 0 {
			t.Fatalf("cross-kind stop receipt count=%d err=%v", count, err)
		}
	})
}

func sqliteV15EffectReceipt(
	claim application.ActionClaim,
	attempt application.EffectAttempt,
	status application.EffectStatus,
	at time.Time,
) application.EffectReceipt {
	intent := claim.Action.EffectIntent
	return application.EffectReceipt{
		Ref: "effect-receipt:" + intent.Ref, IntentRef: intent.Ref, IntentDigest: intent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, AttemptRef: attempt.Ref, Subject: intent.Subject,
		ActionRef: claim.Action.Ref, ActionFence: claim.Fence, IdempotencyKey: intent.IdempotencyKey,
		ExternalRef: "provider-receipt:" + claim.Action.Ref, Status: status,
		Usage: governance.ResourceUsage{Quality: governance.UsageQualityUnknown}, ConfirmedAt: at.UTC(),
	}
}

func TestEffectReceiptLeaseBoundaryIsRejectedBeforeInsert(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	system.submit(t, "request:v27-receipt-lease-boundary")
	claim := claimSQLiteV15(t, system, "claim:v27-receipt-lease-boundary")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil {
		t.Fatal(sqliteTestErrorChain(err))
	}
	if _, err := system.repository.db.Exec(`
CREATE TRIGGER test_effect_receipt_lease_boundary_insert
BEFORE INSERT ON effect_receipts
BEGIN SELECT RAISE(ABORT, 'test.effect_receipt_insert_reached'); END`); err != nil {
		t.Fatal(err)
	}
	receipt := sqliteV15EffectReceipt(
		claim, attempt, application.EffectStatusAccepted, attempt.ClaimLeaseUntil,
	)
	transaction, err := beginTransaction(context.Background(), system.repository)
	if err != nil {
		t.Fatal(err)
	}
	err = insertEffectReceipt(context.Background(), transaction, receipt)
	_ = transaction.Rollback()
	if !application.IsStateError(err, application.StateConflict) ||
		strings.Contains(sqliteTestErrorChain(err), "test.effect_receipt_insert_reached") {
		t.Fatalf("exclusive lease boundary reached INSERT: %s", sqliteTestErrorChain(err))
	}
	var receiptCount, consumptionCount int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?`, claim.Action.Ref).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?`, claim.Action.Ref).Scan(&consumptionCount); err != nil {
		t.Fatal(err)
	}
	if receiptCount != 0 || consumptionCount != 0 {
		t.Fatalf("lease boundary mutated receipts: effect=%d consumption=%d", receiptCount, consumptionCount)
	}
}

func TestPhysicalReceiptGoGuardsRejectExclusiveLeaseBoundary(t *testing.T) {
	boundary := time.Date(2026, 8, 5, 20, 30, 0, 0, time.UTC)
	subject := application.EffectSubject{
		GoalRef:      mustRef(t, "goal:receipt-boundary", goal.NewGoalRef),
		WorkItemRef:  mustRef(t, "work-item:receipt-boundary", goal.NewWorkItemRef),
		ExecutionRef: mustRef(t, "execution:receipt-boundary", goal.NewExecutionRef),
	}
	intent := application.EffectIntent{
		Ref: "effect-intent:receipt-boundary", Digest: "digest:receipt-boundary",
		Subject: subject, IdempotencyKey: "idempotency:receipt-boundary",
	}
	approval := application.EffectApproval{Ref: "effect-approval:receipt-boundary"}
	receipt := application.EffectReceipt{
		Ref: "effect-receipt:receipt-boundary", IntentRef: intent.Ref, IntentDigest: intent.Digest,
		ApprovalRef: approval.Ref, AttemptRef: "effect-attempt:receipt-boundary", Subject: subject,
		ActionFence: 1, IdempotencyKey: intent.IdempotencyKey,
		ExternalRef: "provider-receipt:receipt-boundary", Usage: governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ConfirmedAt: boundary,
	}

	launchIntent := intent
	launchIntent.Kind = application.EffectKindAgentLaunch
	launchClaim := application.ActionClaim{
		Action: application.ActionRecord{
			Ref: "action:launch:receipt-boundary", Kind: application.ActionLaunchAgent,
			EffectIntentRef: launchIntent.Ref, EffectIntent: launchIntent,
		},
		EffectApproval: approval, Fence: 1, LeaseUntil: boundary,
	}
	launchReceipt := receipt
	launchReceipt.ActionRef, launchReceipt.Status = launchClaim.Action.Ref, application.EffectStatusAccepted
	launch := application.LaunchAcceptedState{
		Claim: launchClaim, EffectReceipt: launchReceipt, OperationAt: boundary,
		Event: application.EventRecord{OccurredAt: boundary},
	}
	if err := validateLaunchEffectReceipt(launch); err == nil {
		t.Fatal("launch guard accepted receipt at exclusive lease boundary")
	}
	launch.OperationAt, launch.Event.OccurredAt = boundary.Add(-time.Nanosecond), boundary.Add(-time.Nanosecond)
	launch.EffectReceipt.ConfirmedAt = boundary.Add(-time.Nanosecond)
	if err := validateLaunchEffectReceipt(launch); err != nil {
		t.Fatalf("launch guard rejected final live instant: %v", err)
	}

	stopClaim := application.ActionClaim{
		Action: application.ActionRecord{Ref: "action:stop:receipt-boundary", Kind: application.ActionStopAgent},
		Fence:  1, LeaseUntil: boundary,
	}
	stopReceipt := receipt
	stopReceipt.ActionRef, stopReceipt.Status = stopClaim.Action.Ref, application.EffectStatusStopped
	stop := application.ApplyControlState{
		Claim: stopClaim, EffectReceipt: &stopReceipt, OperationAt: boundary,
	}
	current := application.GoalRecord{Executions: []application.ExecutionRecord{{
		Ref: subject.ExecutionRef, GoalRef: subject.GoalRef, WorkItemRef: subject.WorkItemRef,
	}}}
	if err := validateControlStopReceipt(stop, current); err == nil {
		t.Fatal("stop guard accepted receipt at exclusive lease boundary")
	}
	stop.OperationAt = boundary.Add(-time.Nanosecond)
	stop.EffectReceipt.ConfirmedAt = boundary.Add(-time.Nanosecond)
	if err := validateControlStopReceipt(stop, current); err != nil {
		t.Fatalf("stop guard rejected final live instant: %v", err)
	}
}

func controlSQLiteV15Execution(
	t *testing.T,
	system *sqliteV15System,
	record application.GoalRecord,
	requestRef string,
	mode ports.AgentStopMode,
) application.ControlResult {
	t.Helper()
	item, execution := record.Goal.WorkItems()[0], record.Executions[0]
	result, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: requestRef, Operation: application.ControlStop, Target: application.ControlTargetExecution,
		GoalRef: record.Goal.Ref(), ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedPlanGeneration:    record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: mode, Reason: "governed stop edge test",
	})
	if err != nil || result.Control.Status != application.ControlRequested {
		t.Fatalf("request stop result=%+v err=%v", result, err)
	}
	return result
}

func TestV15RecoveryRejectsCrossKindReceiptTampering(t *testing.T) {
	for _, test := range []struct {
		name       string
		seed       func(*testing.T) *sqliteV15System
		from, to   application.EffectStatus
		intentKind application.EffectKind
	}{
		{
			name: "launch_as_stopped", from: application.EffectStatusAccepted, to: application.EffectStatusStopped,
			intentKind: application.EffectKindAgentLaunch,
			seed: func(t *testing.T) *sqliteV15System {
				system := newSQLiteV15System(t, 2)
				system.submit(t, "request:v15-recovery-cross-launch")
				if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-recovery-cross-launch"); err != nil {
					t.Fatal(err)
				}
				return system
			},
		},
		{
			name: "stop_as_accepted", from: application.EffectStatusStopped, to: application.EffectStatusAccepted,
			intentKind: application.EffectKindAgentStop,
			seed: func(t *testing.T) *sqliteV15System {
				system := newSQLiteV15System(t, 2)
				created := system.submit(t, "request:v15-recovery-cross-stop")
				if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-recovery-cross-stop-launch"); err != nil {
					t.Fatal(err)
				}
				running, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
				if err != nil {
					t.Fatal(err)
				}
				controlSQLiteV15Execution(t, system, running, "control:v15-recovery-cross-stop", ports.AgentStopCooperative)
				if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-recovery-cross-stop"); err != nil || !result.Processed || result.Action != application.ActionStopAgent {
					t.Fatalf("stop result=%+v err=%v", result, err)
				}
				return system
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			system := test.seed(t)
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("seed recovery: %v cause=%v", err, errors.Unwrap(err))
			}
			rewriteRecoveryTrigger(t, system.repository.db, "effect_receipts_immutable_update", func() {
				mustV10Exec(t, system.repository.db, `
UPDATE effect_receipts SET status=? WHERE status=? AND intent_ref IN
 (SELECT ref FROM effect_intents WHERE kind=?)`, string(test.to), string(test.from), string(test.intentKind))
			})
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
				!recoveryErrorContains(err, "sqlite.recovery_v15_effect_receipt_invalid") {
				t.Fatalf("cross-kind recovery accepted: %v", err)
			}
		})
	}
}
