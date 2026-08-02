package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestSQLitePersistsUnknownPartialAndExactUsageReconciliation(t *testing.T) {
	tests := []struct {
		name        string
		usage       governance.ResourceUsage
		wantTokens  int64
		wantMoney   int64
		wantKnown   governance.ResourceDimensions
		wantQuality governance.UsageQuality
	}{
		{
			name: "unknown", usage: governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
			wantTokens: 100, wantMoney: 100,
			wantKnown:   governance.ResourceActiveTime | governance.ResourceProcessSlots | governance.ResourceDisk,
			wantQuality: governance.UsageQualityMeasured,
		},
		{
			name: "partial", usage: governance.ResourceUsage{
				Resources: governance.ResourceVector{Tokens: 7}, Known: governance.ResourceTokens,
				Quality: governance.UsageQualityEstimated,
			},
			wantTokens: 7, wantMoney: 100,
			wantKnown: governance.ResourceTokens | governance.ResourceActiveTime |
				governance.ResourceProcessSlots | governance.ResourceDisk,
			wantQuality: governance.UsageQualityEstimated,
		},
		{
			name: "exact", usage: governance.ResourceUsage{
				Resources: governance.ResourceVector{Tokens: 7, MoneyMicros: 8, Currency: "USD"},
				Known:     governance.ResourceTokens | governance.ResourceMoney, Quality: governance.UsageQualityExact,
			},
			wantTokens: 7, wantMoney: 8, wantKnown: governance.AllResourceDimensions,
			wantQuality: governance.UsageQualityMeasured,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newSQLiteV15System(t, 2)
			system.external.observationUsage = test.usage
			created := system.submit(t, "request:v15-usage:"+test.name)
			if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-usage-launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
				t.Fatalf("launch result=%+v err=%v", result, err)
			}
			system.clock.Advance(time.Second)
			if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-usage-observe"); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
				t.Fatalf("observe result=%+v err=%v", result, err)
			}
			record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
			if err != nil || len(record.BudgetSettlements) != 1 {
				t.Fatalf("settlement record=%+v err=%v", record, err)
			}
			settlement := record.BudgetSettlements[0]
			if settlement.Charged.Tokens != test.wantTokens || settlement.Charged.MoneyMicros != test.wantMoney ||
				settlement.Observed.Known != test.wantKnown || settlement.Observed.Quality != test.wantQuality ||
				governance.ValidateBudgetSettlement(settlement) != nil {
				t.Fatalf("usage settlement=%+v", settlement)
			}
			restartSQLiteV15System(t, system)
			restarted, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
			if err != nil || len(restarted.BudgetSettlements) != 1 || restarted.BudgetSettlements[0] != settlement {
				t.Fatalf("restart settlement=%+v err=%v", restarted.BudgetSettlements, err)
			}
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("usage recovery: %v cause=%v", err, errors.Unwrap(err))
			}
		})
	}
}

func TestSQLiteIrreversibleGoalChargeInterruptsImpossibleAutomaticRetry(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	for _, envelope := range []*governance.BudgetEnvelope{
		&system.policy.DeploymentEnvelope,
		&system.policy.ProjectEnvelopeTemplate,
		&system.policy.GoalEnvelopeTemplate,
	} {
		envelope.Limit.ActiveTimeNS = system.policy.DefaultWorkItemDemand.ActiveTimeNS
	}
	system.orchestrator = newSQLiteV15Orchestrator(
		t, system.repository, system.clock, system.external, system.policy, system.ids,
	)
	system.external.observationStatus = ports.AgentFailed
	system.external.observationError = "codex.process_failed"
	system.external.observationUsage = governance.ResourceUsage{Quality: governance.UsageQualityUnknown}
	created := system.submit(t, "request:v15-irreversible-retry-budget")

	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-irreversible-launch",
	); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch result=%+v err=%v", result, err)
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-irreversible-observe",
	); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("observe result=%+v err=%v", result, err)
	}

	assertSQLiteIrreversibleRetryFrontier(t, system, created.Record.Goal.Ref())
	for restart := 0; restart < 2; restart++ {
		restartSQLiteV15System(t, system)
		if result, err := system.orchestrator.ProcessNext(
			context.Background(), fmt.Sprintf("worker:v15-irreversible-replay:%d", restart),
		); err != nil || result.Processed {
			t.Fatalf("replay %d result=%+v err=%v", restart, result, err)
		}
		assertSQLiteIrreversibleRetryFrontier(t, system, created.Record.Goal.Ref())
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("irreversible retry recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLiteQueuedRetryBecomesIrreversibleAfterPeerSettlement(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	for _, envelope := range []*governance.BudgetEnvelope{
		&system.policy.DeploymentEnvelope,
		&system.policy.ProjectEnvelopeTemplate,
		&system.policy.GoalEnvelopeTemplate,
	} {
		envelope.Limit.Tokens = 200
		envelope.Limit.MoneyMicros = 200
	}
	system.orchestrator = newSQLiteV15Orchestrator(
		t, system.repository, system.clock, system.external, system.policy, system.ids,
	)
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v15-late-irreversible", Statement: "two concurrent retry frontiers", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v15-late-irreversible", Key: "phase:v15-late-irreversible",
				TemplateRef: "phase-template:parallel",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "a", Objective: "retry after peer settlement", Phase: "phase:v15-late-irreversible",
					Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "settle active peer", Phase: "phase:v15-late-irreversible",
					Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		result, processErr := system.orchestrator.ProcessNext(
			context.Background(), fmt.Sprintf("worker:v15-late-launch:%d", index),
		)
		if processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
			t.Fatalf("launch %d result=%+v err=%v", index, result, processErr)
		}
	}
	system.external.observationStatus = ports.AgentFailed
	system.external.observationError = "codex.process_failed"
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-late-first-failure",
	); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("first failure result=%+v err=%v", result, err)
	}
	intermediate, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	var replacementAtCreation application.ExecutionRecord
	for _, execution := range intermediate.Executions {
		if execution.AttemptNo == 2 {
			replacementAtCreation = execution
			break
		}
	}
	if err != nil || len(intermediate.Executions) != 3 ||
		replacementAtCreation.State != application.ExecutionQueued ||
		replacementAtCreation.ReplacesExecutionRef.String() == "" {
		t.Fatalf("replacement was not created while peer active: record=%+v err=%v", intermediate, err)
	}
	system.external.observationStatus = ports.AgentCompleted
	system.external.observationError = ""
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-late-peer-settlement",
	); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("peer settlement result=%+v err=%v", result, err)
	}
	settled, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(settled.BudgetSettlements) != 2 {
		t.Fatalf("peer settlement frontier=%+v err=%v", settled, err)
	}
	// Local exhaustion must remain claimable after launch authority becomes
	// stale: it performs no provider effect and only consumes the queued retry.
	revokeSQLiteV15Owner(t, system)
	system.clock.Advance(time.Second)
	firstClaim := claimSQLiteV15(t, system, "claim:v15-late-irreversible:first")
	if firstClaim.Disposition != application.ActionClaimDispositionRetryBudgetIrreversible ||
		firstClaim.BudgetReservationRef != "" || firstClaim.RetryBudgetExhaustion.FrontierDigest == "" {
		t.Fatalf("late irreversible claim=%+v", firstClaim)
	}
	for name, forge := range map[string]func(application.ActionClaim) application.ActionClaim{
		"disposition": func(value application.ActionClaim) application.ActionClaim {
			value.Disposition = application.ActionClaimDispositionNormal
			value.RetryBudgetExhaustion = application.RetryBudgetExhaustion{}
			return value
		},
		"marker": func(value application.ActionClaim) application.ActionClaim {
			value.RetryBudgetExhaustion.FrontierDigest =
				"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			return value
		},
	} {
		t.Run("forged_"+name, func(t *testing.T) {
			err := system.repository.mutate(context.Background(), forge(firstClaim), system.clock.Now(),
				func(*sql.Tx) error { return nil })
			if !application.IsStateError(err, application.StateConflict) &&
				!application.IsStateError(err, application.StateInvalid) {
				t.Fatalf("forged %s accepted: %v", name, err)
			}
		})
	}
	system.clock.Advance(2 * time.Minute)
	restartSQLiteV15System(t, system)
	reclaimed := claimSQLiteV15(t, system, "claim:v15-late-irreversible:restart")
	if reclaimed.Disposition != application.ActionClaimDispositionRetryBudgetIrreversible ||
		reclaimed.Fence <= firstClaim.Fence ||
		reclaimed.RetryBudgetExhaustion != firstClaim.RetryBudgetExhaustion {
		t.Fatalf("restart claim first=%+v reclaimed=%+v", firstClaim, reclaimed)
	}
	system.clock.Advance(2 * time.Minute)
	launchesBefore := system.external.launchCalls
	result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-late-irreversible:consume",
	)
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("consume result=%+v err=%v", result, err)
	}
	if system.external.launchCalls != launchesBefore {
		t.Fatalf("irreversible retry reached provider calls=%d before=%d",
			system.external.launchCalls, launchesBefore)
	}
	final, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	var completed, receipts, activeReservations, attempts int
	if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*),SUM(CASE WHEN receipt.action_ref IS NOT NULL THEN 1 ELSE 0 END)
FROM outbox action LEFT JOIN action_consumption_receipts receipt ON receipt.action_ref=action.ref
WHERE action.execution_ref=? AND action.kind='launch_agent'
 AND action.last_error_code LIKE 'budget.retry_irreversible:%'`,
		replacementAtCreation.Ref.String()).Scan(&completed, &receipts); queryErr != nil {
		t.Fatal(queryErr)
	}
	if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM budget_reservations reservation
LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
WHERE reservation.execution_ref=? AND settlement.ref IS NULL`,
		replacementAtCreation.Ref.String()).Scan(&activeReservations); queryErr != nil {
		t.Fatal(queryErr)
	}
	if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM effect_attempts WHERE execution_ref=?`,
		replacementAtCreation.Ref.String()).Scan(&attempts); queryErr != nil {
		t.Fatal(queryErr)
	}
	var replacement application.ExecutionRecord
	for _, execution := range final.Executions {
		if execution.Ref == replacementAtCreation.Ref {
			replacement = execution
			break
		}
	}
	item, _ := final.Goal.WorkItem(replacement.WorkItemRef)
	if err != nil || replacement.State != application.ExecutionFailed ||
		replacement.FailureCode != "codex.process_failed" ||
		item.State() != goal.WorkItemStateInterrupted ||
		completed != 1 || receipts != 1 || activeReservations != 0 || attempts != 0 {
		t.Fatalf("late irreversible final=%+v replacement=%+v item=%+v completed=%d receipts=%d reservations=%d attempts=%d err=%v",
			final, replacement, item, completed, receipts, activeReservations, attempts, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("late irreversible recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLitePreparedRetryClosesLocallyAfterLateBudgetExhaustion(t *testing.T) {
	tests := []struct {
		name             string
		explicitApproval bool
		staleAdmission   func(*testing.T, *sqliteV15System)
		assertAdmission  func(*testing.T, *sqliteV15System, application.ActionClaim)
	}{
		{
			name: "revoked_authority",
			staleAdmission: func(t *testing.T, system *sqliteV15System) {
				revokeSQLiteV15Owner(t, system)
				system.clock.Advance(2 * time.Minute)
			},
		},
		{
			name: "expired_explicit_approval", explicitApproval: true,
			staleAdmission: func(_ *testing.T, system *sqliteV15System) {
				system.clock.Advance(system.policy.EffectApprovalTTL + time.Second)
			},
			assertAdmission: func(t *testing.T, system *sqliteV15System, claim application.ActionClaim) {
				t.Helper()
				var expiresAt int64
				if err := system.repository.db.QueryRow(`
SELECT expires_at FROM effect_approvals WHERE ref=? AND source='explicit_decision'`,
					claim.EffectApproval.Ref).Scan(&expiresAt); err != nil {
					t.Fatal(err)
				}
				if expiresAt > system.clock.Now().UTC().UnixNano() {
					t.Fatalf("explicit approval remains current: expires_at=%d now=%d",
						expiresAt, system.clock.Now().UTC().UnixNano())
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newSQLiteV15System(t, 2)
			for _, envelope := range []*governance.BudgetEnvelope{
				&system.policy.DeploymentEnvelope,
				&system.policy.ProjectEnvelopeTemplate,
				&system.policy.GoalEnvelopeTemplate,
			} {
				envelope.Limit.Tokens = 300
				envelope.Limit.MoneyMicros = 300
			}
			system.orchestrator = newSQLiteV15Orchestrator(
				t, system.repository, system.clock, system.external, system.policy, system.ids,
			)
			criticality := governance.SecurityCriticalityNormal
			if test.explicitApproval {
				criticality = governance.SecurityCriticalitySensitive
			}
			created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
				RequestRef: "request:v15-prepared-late-" + test.name,
				Statement:  "prepared retry must close without provider after late exhaustion",
				Confirm:    true,
				Plan: &application.PlanSpec{
					Phases: []application.PhaseSpec{{
						Ref: "phase-instance:v15-prepared-late-" + test.name,
						Key: "phase:v15-prepared-late-" + test.name, TemplateRef: "phase-template:parallel",
					}},
					WorkItems: []application.WorkItemSpec{
						{Key: "a", Objective: "crash after launch preparation",
							Phase: "phase:v15-prepared-late-" + test.name,
							Role:  "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
							SecurityCriticality: criticality, ReasoningEffort: governance.ReasoningEffortMedium},
						{Key: "b", Objective: "settle peer with overrun",
							Phase: "phase:v15-prepared-late-" + test.name,
							Role:  "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
							SecurityCriticality: criticality, ReasoningEffort: governance.ReasoningEffortMedium},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if test.explicitApproval {
				for index, intent := range created.Record.EffectIntents {
					approveSQLiteV15Intent(t, system, created.Record.Goal.Ref(), intent,
						fmt.Sprintf("initial:%d", index))
				}
			}
			for index := 0; index < 2; index++ {
				if result, processErr := system.orchestrator.ProcessNext(
					context.Background(), fmt.Sprintf("worker:v15-prepared-initial:%s:%d", test.name, index),
				); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
					t.Fatalf("initial launch %d result=%+v err=%v", index, result, processErr)
				}
			}
			system.external.observationStatus = ports.AgentFailed
			system.external.observationError = "codex.process_failed"
			system.clock.Advance(time.Second)
			if result, processErr := system.orchestrator.ProcessNext(
				context.Background(), "worker:v15-prepared-first-failure:"+test.name,
			); processErr != nil || !result.Processed || result.Action != application.ActionObserveAgent {
				t.Fatalf("first failure result=%+v err=%v", result, processErr)
			}

			system.clock.Advance(system.policy.QuotaRetryDelay)
			if test.explicitApproval {
				record, recordErr := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
				if recordErr != nil {
					t.Fatal(recordErr)
				}
				var retryIntent application.EffectIntent
				for _, execution := range record.Executions {
					if execution.AttemptNo != 2 {
						continue
					}
					for _, intent := range record.EffectIntents {
						if intent.ActionKind == application.ActionLaunchAgent &&
							intent.Subject.ExecutionRef == execution.Ref {
							retryIntent = intent
						}
					}
				}
				if retryIntent.Ref == "" {
					t.Fatal("retry effect intent missing")
				}
				approveSQLiteV15Intent(t, system, created.Record.Goal.Ref(), retryIntent, "retry")
			}
			retryClaim := claimSQLiteV15(t, system, "claim:v15-prepared-retry:"+test.name)
			if retryClaim.Disposition != application.ActionClaimDispositionNormal ||
				retryClaim.Action.Kind != application.ActionLaunchAgent ||
				retryClaim.BudgetReservationRef == "" {
				t.Fatalf("prepared retry claim=%+v", retryClaim)
			}
			retry, sessionRef := prepareSQLiteV15RetryLaunchWithSession(t, system, retryClaim)

			system.external.observationStatus = ports.AgentCompleted
			system.external.observationError = ""
			system.external.observationUsage = governance.ResourceUsage{
				Resources: governance.ResourceVector{
					Tokens: 200, MoneyMicros: 200, Currency: governance.Currency("USD"),
				},
				Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
			}
			if result, processErr := system.orchestrator.ProcessNext(
				context.Background(), "worker:v15-prepared-peer:"+test.name,
			); processErr != nil || !result.Processed || result.Action != application.ActionObserveAgent {
				t.Fatalf("peer settlement result=%+v err=%v", result, processErr)
			}
			test.staleAdmission(t, system)
			if test.assertAdmission != nil {
				test.assertAdmission(t, system, retryClaim)
			}

			launchesBefore := system.external.launchCalls
			if result, processErr := system.orchestrator.ProcessNext(
				context.Background(), "worker:v15-prepared-park:"+test.name,
			); processErr != nil || result.Processed {
				t.Fatalf("stale prepared admission result=%+v err=%v", result, processErr)
			}
			parked, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
			parkedRetry, found := sqliteExecutionByRef(parked.Executions, retry.Ref)
			if err != nil || !found || parkedRetry.State != application.ExecutionDispatching ||
				parkedRetry.BudgetReservationRef != "" || parkedRetry.EffectIntentRef != "" ||
				parkedRetry.ExecutionSessionRef != sessionRef {
				t.Fatalf("parked retry=%+v found=%v err=%v", parkedRetry, found, err)
			}

			system.clock.Advance(system.policy.QuotaRetryDelay)
			if result, processErr := system.orchestrator.ProcessNext(
				context.Background(), "worker:v15-prepared-local:"+test.name,
			); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
				t.Fatalf("local closure result=%+v err=%v", result, processErr)
			}
			if system.external.launchCalls != launchesBefore {
				t.Fatalf("prepared retry reached provider calls=%d before=%d",
					system.external.launchCalls, launchesBefore)
			}
			final, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
			failed, found := sqliteExecutionByRef(final.Executions, retry.Ref)
			item, itemFound := final.Goal.WorkItem(retry.WorkItemRef)
			var revokeActions, attempts, activeReservations int
			if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE kind='revoke_execution_session' AND execution_ref=? AND completed_at IS NULL`,
				retry.Ref.String()).Scan(&revokeActions); queryErr != nil {
				t.Fatal(queryErr)
			}
			if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM effect_attempts WHERE execution_ref=?`,
				retry.Ref.String()).Scan(&attempts); queryErr != nil {
				t.Fatal(queryErr)
			}
			if queryErr := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM budget_reservations reservation
LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
WHERE reservation.execution_ref=? AND settlement.ref IS NULL`,
				retry.Ref.String()).Scan(&activeReservations); queryErr != nil {
				t.Fatal(queryErr)
			}
			if err != nil || !found || !itemFound ||
				failed.State != application.ExecutionFailed ||
				failed.FailureCode != "codex.process_failed" ||
				failed.ExecutionSessionRef != sessionRef ||
				item.State() != goal.WorkItemStateInterrupted ||
				revokeActions != 1 || attempts != 0 || activeReservations != 0 {
				t.Fatalf("prepared retry final=%+v item=%+v found=%v/%v revoke=%d attempts=%d active=%d err=%v",
					failed, item, found, itemFound, revokeActions, attempts, activeReservations, err)
			}
			restartSQLiteV15System(t, system)
			if _, _, recoveryErr := validateRecoveryDatabase(
				context.Background(), system.repository.db,
			); recoveryErr != nil {
				t.Fatalf("prepared retry recovery: %v cause=%v", recoveryErr, errors.Unwrap(recoveryErr))
			}
		})
	}
}

func approveSQLiteV15Intent(
	t *testing.T,
	system *sqliteV15System,
	goalRef goal.GoalRef,
	intent application.EffectIntent,
	suffix string,
) {
	t.Helper()
	result, err := system.orchestrator.DecideEffect(context.Background(), system.access,
		application.DecideEffectRequest{
			RequestRef: "approval:v15-prepared:" + suffix, GoalRef: goalRef, IntentRef: intent.Ref,
			ExpectedIntentDigest: intent.Digest, Decision: application.EffectApproved,
			Reason: "bounded explicit approval for prepared retry regression",
		})
	if err != nil || !result.Created {
		t.Fatalf("approve intent %s created=%v err=%v", intent.Ref, result.Created, err)
	}
}

func prepareSQLiteV15RetryLaunchWithSession(
	t *testing.T,
	system *sqliteV15System,
	claim application.ActionClaim,
) (application.ExecutionRecord, ports.ExecutionSessionRef) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	if err != nil {
		t.Fatal(err)
	}
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionQueued || execution.AttemptNo <= 1 {
		t.Fatalf("retry execution before prepare=%+v found=%v", execution, found)
	}
	sessionRef, err := ports.NewExecutionSessionRef(
		"execution-session:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	if err != nil {
		t.Fatal(err)
	}
	execution.State = application.ExecutionDispatching
	execution.BudgetReservationRef = claim.BudgetReservationRef
	execution.EffectIntentRef = claim.Action.EffectIntentRef
	execution.ExecutionSessionRef = sessionRef
	at := system.clock.Now().UTC()
	state := application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(),
		Goal: record.Goal, Execution: execution, OperationAt: at,
		Event: application.EventRecord{
			Ref:  "event:execution-dispatching:" + execution.Ref.String(),
			Kind: "execution.dispatching", GoalRef: execution.GoalRef,
			WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}
	if err := system.repository.RecordLaunchPrepared(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	return execution, sessionRef
}

func TestSQLiteQueuedRetryWaitsForReleasablePeerCapacity(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	for _, envelope := range []*governance.BudgetEnvelope{
		&system.policy.DeploymentEnvelope,
		&system.policy.ProjectEnvelopeTemplate,
		&system.policy.GoalEnvelopeTemplate,
	} {
		envelope.Limit.Tokens = 200
		envelope.Limit.MoneyMicros = 200
	}
	system.orchestrator = newSQLiteV15Orchestrator(
		t, system.repository, system.clock, system.external, system.policy, system.ids,
	)
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v15-late-temporary", Statement: "wait for releasable peer capacity", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v15-late-temporary", Key: "phase:v15-late-temporary",
				TemplateRef: "phase-template:parallel",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "a", Objective: "retry after peer release", Phase: "phase:v15-late-temporary",
					Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "release active peer", Phase: "phase:v15-late-temporary",
					Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		if result, processErr := system.orchestrator.ProcessNext(
			context.Background(), fmt.Sprintf("worker:v15-temporary-launch:%d", index),
		); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
			t.Fatalf("launch %d result=%+v err=%v", index, result, processErr)
		}
	}
	system.external.observationStatus = ports.AgentFailed
	system.external.observationError = "codex.process_failed"
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-temporary-first-failure",
	); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("first failure result=%+v err=%v", result, err)
	}
	system.external.observationStatus = ports.AgentCompleted
	system.external.observationError = ""
	system.external.observationUsage = governance.ResourceUsage{
		Resources: governance.ResourceVector{Currency: "USD"},
		Known:     governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-temporary-peer-release",
	); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("peer release result=%+v err=%v", result, err)
	}
	var deferred int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox action JOIN executions execution ON execution.ref=action.execution_ref
WHERE action.goal_ref=? AND action.kind='launch_agent' AND execution.attempt_no=2
 AND action.last_error_code='budget.temporarily_unavailable'
 AND action.delivery_attempt=0 AND action.completed_at IS NULL`,
		created.Record.Goal.Ref().String()).Scan(&deferred); err != nil || deferred != 1 {
		t.Fatalf("retry did not retain temporary contention deferred=%d err=%v", deferred, err)
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v15-temporary-retry-launch",
	); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("retry launch result=%+v err=%v", result, err)
	}
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	var retry application.ExecutionRecord
	for _, execution := range record.Executions {
		if execution.AttemptNo == 2 {
			retry = execution
			break
		}
	}
	if err != nil || retry.State != application.ExecutionRunning ||
		retry.BudgetReservationRef == "" || system.external.launchCalls != 3 {
		t.Fatalf("temporary retry record=%+v retry=%+v calls=%d err=%v",
			record, retry, system.external.launchCalls, err)
	}
}

func assertSQLiteIrreversibleRetryFrontier(
	t *testing.T,
	system *sqliteV15System,
	goalRef goal.GoalRef,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	items := record.Goal.WorkItems()
	if err != nil || record.Goal.State() != goal.GoalStateRunning || len(items) != 1 ||
		items[0].State() != goal.WorkItemStateInterrupted || len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionFailed ||
		record.Executions[0].FailureCode != "codex.process_failed" ||
		record.Executions[0].MaxExecutionAttempts <= 1 ||
		len(record.BudgetSettlements) != 1 {
		t.Fatalf("irreversible retry frontier record=%+v err=%v", record, err)
	}
	settlement := record.BudgetSettlements[0]
	if settlement.Observed.Quality != governance.UsageQualityMeasured ||
		settlement.Charged.Tokens != system.policy.DefaultWorkItemDemand.Tokens ||
		settlement.Charged.MoneyMicros != system.policy.DefaultWorkItemDemand.MoneyMicros ||
		settlement.Charged.ActiveTimeNS != int64(time.Second) {
		t.Fatalf("irreversible retry settlement=%+v", settlement)
	}
	var pendingLaunches, consumedObservations int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE goal_ref=? AND kind='launch_agent' AND completed_at IS NULL`,
		goalRef.String()).Scan(&pendingLaunches); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE goal_ref=? AND kind='observe_agent' AND outcome='completed'`,
		goalRef.String()).Scan(&consumedObservations); err != nil {
		t.Fatal(err)
	}
	if pendingLaunches != 0 || consumedObservations != 1 || system.external.launchCalls != 1 {
		t.Fatalf("retry replay launches=%d observations=%d calls=%d",
			pendingLaunches, consumedObservations, system.external.launchCalls)
	}
}

func TestSQLiteSmallOverrunParksNewExposureAcrossRestart(t *testing.T) {
	system := newSQLiteV15System(t, 8)
	system.external.observationUsage = governance.ResourceUsage{
		Resources: governance.ResourceVector{Tokens: 101}, Known: governance.ResourceTokens,
		Quality: governance.UsageQualityExact,
	}
	created := submitSQLiteV15DependentPair(t, system, "request:v15-small-overrun")
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-overrun-launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch result=%+v err=%v", result, err)
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-overrun-observe"); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("observe result=%+v err=%v", result, err)
	}
	before, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(before.BudgetSettlements) != 1 || before.BudgetSettlements[0].Overrun.Tokens != 1 ||
		len(before.Executions) != 2 || before.Executions[1].State != application.ExecutionQueued {
		t.Fatalf("small overrun frontier=%+v err=%v", before, err)
	}
	for restart := 0; restart < 2; restart++ {
		restartSQLiteV15System(t, system)
		calls := system.external.launchCalls
		result, processErr := system.orchestrator.ProcessNext(
			context.Background(), fmt.Sprintf("worker:v15-overrun-park:%d", restart),
		)
		after, readErr := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
		if processErr != nil || result.Processed || readErr != nil || after.Goal.IsTerminal() ||
			len(after.EffectAttempts) != 1 || system.external.launchCalls != calls {
			t.Fatalf("overrun restart=%d result=%+v err=%v record=%+v calls=%d/%d",
				restart, result, processErr, after, calls, system.external.launchCalls)
		}
		var code string
		var deliveries int
		if err := system.repository.db.QueryRow(`
SELECT last_error_code,delivery_attempt FROM outbox
WHERE execution_ref=? AND completed_at IS NULL`, after.Executions[1].Ref.String()).Scan(&code, &deliveries); err != nil {
			t.Fatal(err)
		}
		if code != "budget.temporarily_unavailable" || deliveries != 0 {
			t.Fatalf("overrun parking code=%q deliveries=%d", code, deliveries)
		}
	}
}

func TestSQLiteOverrunDoesNotBlockCrashReplayOfActiveReservation(t *testing.T) {
	system := newSQLiteV15System(t, 4)
	system.external.observationUsage = governance.ResourceUsage{
		Resources: governance.ResourceVector{Tokens: 101}, Known: governance.ResourceTokens,
		Quality: governance.UsageQualityExact,
	}
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v15-overrun-active-replay", Statement: "two independent governed effects", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v15-overrun-active-replay", Key: "phase:v15-overrun-active-replay",
				TemplateRef: "phase-template:parallel",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "a", Objective: "crash before receipt", Phase: "phase:v15-overrun-active-replay", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "settle with small overrun", Phase: "phase:v15-overrun-active-replay", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	first := claimSQLiteV15(t, system, "claim:v15-overrun-active:first")
	prepareSQLiteV15Launch(t, system, first)
	attempt := sqliteV15Attempt(first, system.clock.Now())
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: first, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-overrun-active:b-launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("second launch result=%+v err=%v", result, err)
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-overrun-active:b-observe"); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("second observe result=%+v err=%v", result, err)
	}
	before, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(before.BudgetReservations) != 2 || len(before.BudgetSettlements) != 1 ||
		before.BudgetSettlements[0].Overrun.Tokens != 1 {
		t.Fatalf("overrun replay seed=%+v err=%v", before, err)
	}
	system.clock.Advance(2 * time.Minute)
	restartSQLiteV15System(t, system)
	replayed := claimSQLiteV15(t, system, "claim:v15-overrun-active:replay")
	if replayed.Action.Ref != first.Action.Ref || replayed.BudgetReservationRef != first.BudgetReservationRef ||
		replayed.BudgetReservation != first.BudgetReservation ||
		replayed.Action.EffectIntent.IdempotencyKey != first.Action.EffectIntent.IdempotencyKey {
		t.Fatalf("active reservation was not replayed first=%+v replay=%+v", first, replayed)
	}
	var reservations int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM budget_reservations`).Scan(&reservations); err != nil || reservations != 2 {
		t.Fatalf("replay reservations=%d err=%v", reservations, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("overrun active replay recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLiteDependentWorkKeepsHistoricalPolicyAfterRuntimeRotation(t *testing.T) {
	system := newSQLiteV15System(t, 4)
	created := submitSQLiteV15DependentPair(t, system, "request:v15-policy-rotation-dependent")
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-policy-old-launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("old launch result=%+v err=%v", result, err)
	}
	oldIntent := created.Record.EffectIntents[0]
	rotated := rotatedSQLiteV15Policy(system.policy, system.clock.Now())
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	repository := openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.repository, system.policy = repository, rotated
	system.orchestrator = newSQLiteV15Orchestrator(t, repository, system.clock, system.external, rotated, system.ids)
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-policy-rotated-observe"); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("rotated observe result=%+v err=%v", result, err)
	}
	record, err := repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(record.EffectIntents) != 2 {
		t.Fatalf("rotated dependent record=%+v err=%v", record, err)
	}
	newIntent := record.EffectIntents[1]
	if newIntent.PolicyHash != oldIntent.PolicyHash || newIntent.PolicyRevision != oldIntent.PolicyRevision ||
		newIntent.ApprovalTTL != oldIntent.ApprovalTTL || newIntent.QuotaRetryDelay != oldIntent.QuotaRetryDelay {
		t.Fatalf("dependent policy rotated old=%+v new=%+v", oldIntent, newIntent)
	}
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-policy-rotated-claim", Token: "claim:v15-policy-rotated-dependent",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: rotated,
		CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || claim.Action.EffectIntent.PolicyHash != oldIntent.PolicyHash ||
		claim.BudgetReservation.PolicyHash != oldIntent.PolicyHash {
		t.Fatalf("dependent claim=%+v found=%v err=%v", claim, found, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("rotated policy recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func submitSQLiteV15DependentPair(
	t *testing.T,
	system *sqliteV15System,
	requestRef string,
) application.SubmitResult {
	t.Helper()
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: requestRef, Statement: "governed dependent pair", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:" + requestRef, Key: "phase:" + requestRef,
				TemplateRef: "phase-template:dependent",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "first", Objective: "first governed work", Phase: "phase:" + requestRef, Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "second", Objective: "second governed work", Phase: "phase:" + requestRef, Role: "role:reviewer", Dependencies: []string{"first"}, OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
