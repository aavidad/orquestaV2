package sqlite

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
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
