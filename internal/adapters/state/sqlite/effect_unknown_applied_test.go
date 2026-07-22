package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestSQLiteEffectAttemptRejectsDifferentRefAtSameActionFence(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:sqlite-attempt-collision")
	claim := claimSQLiteV15(t, system, "claim:sqlite-attempt-collision")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil || !created {
		t.Fatalf("first created=%v err=%v", created, err)
	}
	collision := attempt
	collision.Ref += ":different"
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: collision, OperationAt: system.clock.Now(),
	}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("same action/fence different ref accepted: %v", err)
	}
	var attempts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts WHERE action_ref=? AND action_fence=?`,
		claim.Action.Ref, int64(claim.Fence)).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("attempt count=%d err=%v", attempts, err)
	}
}

func TestSQLiteRejectsMalformedFirstEffectAttemptLikeMemory(t *testing.T) {
	mutations := map[string]func(*application.RecordEffectAttemptState){
		"ref": func(state *application.RecordEffectAttemptState) {
			state.Attempt.Ref += ":different"
		},
		"intent ref": func(state *application.RecordEffectAttemptState) {
			state.Attempt.IntentRef += ":different"
		},
		"intent digest": func(state *application.RecordEffectAttemptState) {
			state.Attempt.IntentDigest += ":different"
		},
		"approval ref": func(state *application.RecordEffectAttemptState) {
			state.Attempt.ApprovalRef += ":different"
		},
		"subject": func(state *application.RecordEffectAttemptState) {
			state.Attempt.Subject.PlanGeneration++
		},
		"action ref": func(state *application.RecordEffectAttemptState) {
			state.Attempt.ActionRef += ":different"
		},
		"action fence": func(state *application.RecordEffectAttemptState) {
			state.Attempt.ActionFence++
		},
		"worker ref": func(state *application.RecordEffectAttemptState) {
			state.Attempt.WorkerRef += ":different"
		},
		"idempotency key": func(state *application.RecordEffectAttemptState) {
			state.Attempt.IdempotencyKey += ":different"
		},
		"started at": func(state *application.RecordEffectAttemptState) {
			state.Attempt.StartedAt = state.Attempt.StartedAt.Add(time.Second)
		},
		"operation at": func(state *application.RecordEffectAttemptState) {
			state.OperationAt = state.OperationAt.Add(time.Second)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			assertSQLiteRejectsMalformedFirstEffectAttempt(t, mutate)
		})
	}
}

func assertSQLiteRejectsMalformedFirstEffectAttempt(
	t *testing.T,
	mutate func(*application.RecordEffectAttemptState),
) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:sqlite-malformed-first-attempt")
	claim := claimSQLiteV15(t, system, "claim:sqlite-malformed-first-attempt")
	prepareSQLiteV15Launch(t, system, claim)
	want := sqliteV15Attempt(claim, system.clock.Now())
	state := application.RecordEffectAttemptState{Claim: claim, Attempt: want, OperationAt: system.clock.Now()}
	mutate(&state)
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), state); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("malformed first attempt accepted: %v", err)
	}
	var attempts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatalf("malformed first attempt persisted: count=%d err=%v", attempts, err)
	}
	got, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: want, OperationAt: system.clock.Now(),
	})
	if err != nil || !created || got != want {
		t.Fatalf("valid retry got=%+v created=%v err=%v", got, created, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid retry poisoned recovery: %v", err)
	}
}

func TestRecoveryUsesExplicitCausalSettlementWithoutWallTimeInference(t *testing.T) {
	t.Run("equal settlement time is valid", func(t *testing.T) {
		system := seedSQLiteDefinitelyUnappliedRetry(t)
		var attempts, settlements int
		if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM budget_settlements settlement
JOIN effect_attempts attempt ON attempt.ref=settlement.causal_attempt_ref
WHERE settlement.settled_at=attempt.started_at`).Scan(&settlements); err != nil || settlements != 1 {
			t.Fatalf("equal-time causal proof count=%d err=%v", settlements, err)
		}
		if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil || attempts != 2 {
			t.Fatalf("retry attempts=%d err=%v", attempts, err)
		}
	})
	t.Run("missing exact edge is rejected", func(t *testing.T) {
		system := seedSQLiteDefinitelyUnappliedRetry(t)
		rewriteRecoveryTrigger(t, system.repository.db, "budget_settlements_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `UPDATE budget_settlements SET causal_attempt_ref=NULL`)
		})
		assertUnknownAppliedRecoveryError(t, system.repository)
	})
}

func seedSQLiteDefinitelyUnappliedRetry(t *testing.T) *sqliteV15System {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:sqlite-causal-proof")
	system.external.launchErr = sqliteV15DefinitelyUnapplied{}
	if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:causal:first"); err != nil {
		t.Fatal(err)
	}
	system.external.mu.Lock()
	system.external.launchErr = nil
	system.external.mu.Unlock()
	system.clock.Advance(2 * time.Second)
	if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:causal:retry"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid causal retry: %v", err)
	}
	return system
}

func assertUnknownAppliedRecoveryError(t *testing.T, repository *Repository) {
	t.Helper()
	_, _, err := validateRecoveryDatabase(context.Background(), repository.db)
	if err == nil || !recoveryErrorContains(err, "sqlite.recovery_v17_unknown_applied_repeated") {
		t.Fatalf("unsafe causal proof accepted: %v", err)
	}
}
