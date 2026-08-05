package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestV28MigrationSeparatesRecoveryClaimRefFromPhysicalReceiptFence(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	var version, recoveryColumn int
	sqliteTestNoError(t, system.repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('outbox') WHERE name='recovery_effect_attempt_ref'`).Scan(&recoveryColumn))
	if version != recoverySchemaLatest || recoveryColumn != 1 {
		t.Fatalf("schema=%d recovery_column=%d", version, recoveryColumn)
	}

	system.submit(t, "request:v28-normal-claim")
	claim := claimSQLiteV15(t, system, "claim:v28-normal-claim")
	var recoveryRef sql.NullString
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT recovery_effect_attempt_ref FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&recoveryRef))
	if recoveryRef.Valid || claim.RecoveryEffectAttemptRef != "" {
		t.Fatalf("normal claim acquired recovery ref db=%+v claim=%q", recoveryRef, claim.RecoveryEffectAttemptRef)
	}
}

func TestV28RecoveryKeepsV27ResolvedNullAttemptCompatibility(t *testing.T) {
	system, attempt := seedV27CompletedLaunch(t, "v28-null-receipt-proof")
	mutateV27AttemptIgnoringChecks(t, system, `claim_lease_until=NULL`, attempt.Ref)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("V28 rejected V27 terminal receipt proof: %s", sqliteTestErrorChain(err))
	}

	zeroRelease := seedSQLiteDefinitelyUnappliedRetry(t)
	var releasedAttempt string
	sqliteTestNoError(t, zeroRelease.repository.db.QueryRow(`
SELECT causal_attempt_ref FROM budget_settlements WHERE causal_attempt_ref IS NOT NULL`).Scan(&releasedAttempt))
	mutateV27AttemptIgnoringChecks(t, zeroRelease, `claim_lease_until=NULL`, releasedAttempt)
	if _, _, err := validateRecoveryDatabase(context.Background(), zeroRelease.repository.db); err != nil {
		t.Fatalf("V28 rejected V27 exact zero-release proof: %s", sqliteTestErrorChain(err))
	}
}

func TestV28RequireClaimUsesPersistedRecoveryAttemptRefAsCAS(t *testing.T) {
	system, first, attempt := seedV27AmbiguousEffectAttempt(t, "v28-recovery-cas")
	system.clock.Advance(time.Minute + time.Nanosecond)
	reclaimed := claimSQLiteV28Recovery(t, system, "claim:v28-recovery-cas:second", attempt.Ref)
	if reclaimed.Fence <= first.Fence {
		t.Fatalf("reclaimed fence=%d first=%d", reclaimed.Fence, first.Fence)
	}
	if _, err := system.repository.db.Exec(`
UPDATE outbox SET recovery_effect_attempt_ref=NULL WHERE ref=?`, reclaimed.Action.Ref); err == nil {
		t.Fatal("post-hoc recovery ref clear accepted outside claim CAS")
	}
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	if err := requireClaim(context.Background(), transaction, reclaimed); err != nil {
		_ = transaction.Rollback()
		t.Fatalf("exact persisted recovery claim rejected: %s", sqliteTestErrorChain(err))
	}
	_ = transaction.Rollback()

	wrong := reclaimed
	wrong.RecoveryEffectAttemptRef += ":crossed"
	transaction, err = beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	err = requireClaim(context.Background(), transaction, wrong)
	_ = transaction.Rollback()
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("crossed recovery ref accepted: %s", sqliteTestErrorChain(err))
	}

	normal := reclaimed
	normal.Disposition, normal.RecoveryEffectAttemptRef = application.ActionClaimDispositionNormal, ""
	transaction, err = beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	err = requireClaim(context.Background(), transaction, normal)
	_ = transaction.Rollback()
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("normal claim accepted durable recovery ref: %s", sqliteTestErrorChain(err))
	}

	second := sqliteV15Attempt(reclaimed, system.clock.Now())
	transaction, err = beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectAttempt(context.Background(), transaction, second))
	sqliteTestNoError(t, commit(transaction))
	transaction, err = beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	err = requireClaim(context.Background(), transaction, reclaimed)
	_ = transaction.Rollback()
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("recovery claim accepted second physical attempt: %s", sqliteTestErrorChain(err))
	}
}

func claimSQLiteV28Recovery(
	t *testing.T, system *sqliteV15System, token, attemptRef string,
) application.ActionClaim {
	t.Helper()
	ctx := context.Background()
	transaction, err := beginTransaction(ctx, system.repository)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	now := system.clock.Now().UTC()
	candidates, err := readClaimCandidateWindow(ctx, transaction, now, true, false, nil)
	sqliteTestNoError(t, err)
	requirements, err := readAgentRequirementsBatch(ctx, transaction, candidates)
	sqliteTestNoError(t, err)
	selected, found, err := selectClaimCandidate(
		ctx, transaction, candidates, requirements, sqliteTestCapabilities(), system.capacidad, false, now,
	)
	if err != nil || !found {
		t.Fatalf("select recovery candidate found=%t err=%v", found, err)
	}
	selected.disposition = application.ActionClaimDispositionRecoverEffect
	selected.recoveryEffectAttemptRef = attemptRef
	claim, err := claimSelectedCandidate(ctx, transaction, application.ClaimRequest{
		WorkerRef: "worker:v28-recovery", Token: token, LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		CapacityCandidates: system.capacidad,
	}, selected, now, now.Add(time.Minute))
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, commit(transaction))
	return claim
}
