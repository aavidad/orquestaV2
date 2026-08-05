package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestRecoveryV26RemainsVerifiableAndRestorable(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state-v26.sqlite")
	database := openRawV10TestDatabase(t, path)
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	prefix, err := recoveryMigrationPrefix(migrations, recoverySchemaV38EnvironmentGate)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(ctx, database, prefix))
	wantSchemaRef := migrationSchemaRef(prefix)
	gotSchemaRef, _, err := validateRecoveryDatabase(ctx, database)
	if err != nil || gotSchemaRef != wantSchemaRef {
		t.Fatalf("validate V26 schema=%q want=%q err=%v", gotSchemaRef, wantSchemaRef, err)
	}

	repository := &Repository{db: database, path: path, now: time.Now}
	recovery, _, _ := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	backup, err := recovery.CreateBackup(ctx)
	sqliteTestNoError(t, err)
	if backup.SchemaRef != wantSchemaRef {
		t.Fatalf("backup V26 schema=%q want=%q", backup.SchemaRef, wantSchemaRef)
	}
	target, err := application.NewRecoveryTargetRef("recovery-target:v26-attempt-lease")
	sqliteTestNoError(t, err)
	_, err = recovery.RestoreBackup(ctx, backup.Ref, target)
	sqliteTestNoError(t, err)
	targetPath, err := recovery.TargetPath(target)
	sqliteTestNoError(t, err)
	restored := openRawV10TestDatabase(t, targetPath)
	defer restored.Close()
	assertRecoverySchemaVersion(t, restored, recoverySchemaV38EnvironmentGate)
	restoredSchemaRef, _, err := validateRecoveryDatabase(ctx, restored)
	if err != nil || restoredSchemaRef != wantSchemaRef {
		t.Fatalf("restored V26 schema=%q want=%q err=%v", restoredSchemaRef, wantSchemaRef, err)
	}
}

func TestRecoveryV27RoundTripPreservesExactAttemptClaimLease(t *testing.T) {
	ctx := context.Background()
	system, attempt := seedV27CompletedLaunch(t, "roundtrip")
	beforeSchemaRef, _, err := validateRecoveryDatabase(ctx, system.repository.db)
	sqliteTestNoError(t, err)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	wantPrefix, err := recoveryMigrationPrefix(migrations, recoverySchemaV38AttemptLease)
	sqliteTestNoError(t, err)
	if beforeSchemaRef != migrationSchemaRef(wantPrefix) {
		t.Fatalf("V27 schema ref=%q", beforeSchemaRef)
	}

	recovery, _, _ := newV09TestRecovery(t, system.repository, system.clock.Now().Add(time.Minute), nil)
	backup, err := recovery.CreateBackup(ctx)
	sqliteTestNoError(t, err)
	target, err := application.NewRecoveryTargetRef("recovery-target:v27-attempt-lease")
	sqliteTestNoError(t, err)
	_, err = recovery.RestoreBackup(ctx, backup.Ref, target)
	sqliteTestNoError(t, err)
	targetPath, err := recovery.TargetPath(target)
	sqliteTestNoError(t, err)
	restored := openRawV10TestDatabase(t, targetPath)
	defer restored.Close()
	assertRecoverySchemaVersion(t, restored, recoverySchemaV38AttemptLease)
	var leaseNanos int64
	if err := restored.QueryRow(`SELECT claim_lease_until FROM effect_attempts WHERE ref=?`, attempt.Ref).Scan(&leaseNanos); err != nil {
		t.Fatal(err)
	}
	if got := time.Unix(0, leaseNanos).UTC(); !got.Equal(attempt.ClaimLeaseUntil) {
		t.Fatalf("restored lease=%s want=%s", got, attempt.ClaimLeaseUntil)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restored); err != nil {
		t.Fatalf("restored V27 invalid: %v", err)
	}
}

func TestRecoveryV27RejectsAmbiguousOrInvalidAttemptClaimLease(t *testing.T) {
	t.Run("lease no supera inicio", func(t *testing.T) {
		system, attempt := seedV27AmbiguousLaunch(t, "invalid-order")
		mutateV27AttemptIgnoringChecks(t, system, `claim_lease_until=started_at`, attempt.Ref)
		requireV27ValidatorError(t, system, "sqlite.recovery_v27_effect_attempt_claim_lease_invalid")
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
			t.Fatal("full recovery accepted violated lease CHECK")
		}
	})

	t.Run("NULL ambiguo", func(t *testing.T) {
		system, attempt := seedV27AmbiguousLaunch(t, "missing-proof")
		mutateV27AttemptIgnoringChecks(t, system, `claim_lease_until=NULL`, attempt.Ref)
		requireV27RecoveryError(t, system, "sqlite.recovery_v27_effect_attempt_claim_lease_missing_proof")
	})

	t.Run("receipt fuera del lease historico", func(t *testing.T) {
		system, attempt := seedV27CompletedLaunch(t, "late-receipt")
		rewriteRecoveryTrigger(t, system.repository.db, "effect_attempts_immutable_update", func() {
			rewriteRecoveryTrigger(t, system.repository.db, "effect_receipts_immutable_update", func() {
				mustV10Exec(t, system.repository.db, `UPDATE effect_attempts
SET claim_lease_until=started_at+1 WHERE ref=?`, attempt.Ref)
				mustV10Exec(t, system.repository.db, `UPDATE effect_receipts
SET confirmed_at=(SELECT started_at+1 FROM effect_attempts WHERE ref=?)
WHERE attempt_ref=?`, attempt.Ref, attempt.Ref)
			})
		})
		requireV27RecoveryError(t, system, "sqlite.recovery_v27_effect_receipt_outside_claim_lease")
	})
}

func TestRecoveryV27AllowsLegacyNullOnlyWithExactTerminalProof(t *testing.T) {
	t.Run("receipt causal", func(t *testing.T) {
		system, attempt := seedV27CompletedLaunch(t, "receipt-proof")
		mutateV27AttemptIgnoringChecks(t, system, `claim_lease_until=NULL`, attempt.Ref)
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
			t.Fatalf("exact receipt did not permit legacy NULL: %v", err)
		}
	})

	t.Run("liberacion cero causal unica", func(t *testing.T) {
		system := seedSQLiteDefinitelyUnappliedRetry(t)
		var attemptRef string
		sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT causal_attempt_ref
FROM budget_settlements WHERE causal_attempt_ref IS NOT NULL`).Scan(&attemptRef))
		mutateV27AttemptIgnoringChecks(t, system, `claim_lease_until=NULL`, attemptRef)
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
			t.Fatalf("exact zero release did not permit legacy NULL: %v", err)
		}
	})
}

func seedV27AmbiguousLaunch(t *testing.T, suffix string) (*sqliteV15System, application.EffectAttempt) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v27-attempt-"+suffix)
	claim := claimSQLiteV15(t, system, "claim:v27-attempt-"+suffix)
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	stored, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	})
	if err != nil || !created || stored != attempt {
		t.Fatalf("seed attempt stored=%+v created=%t err=%v", stored, created, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid ambiguous V27 seed: %v", err)
	}
	return system, attempt
}

func seedV27CompletedLaunch(t *testing.T, suffix string) (*sqliteV15System, application.EffectAttempt) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v27-completed-"+suffix)
	if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:v27:"+suffix); err != nil {
		t.Fatalf("complete launch: %v", err)
	}
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 || record.EffectAttempts[0].ClaimLeaseUntil.IsZero() {
		t.Fatalf("completed seed attempts=%d receipts=%d", len(record.EffectAttempts), len(record.EffectReceipts))
	}
	return system, record.EffectAttempts[0]
}

func mutateV27AttemptIgnoringChecks(t *testing.T, system *sqliteV15System, assignment, attemptRef string) {
	t.Helper()
	rewriteRecoveryTrigger(t, system.repository.db, "effect_attempts_immutable_update", func() {
		connection, err := system.repository.db.Conn(context.Background())
		sqliteTestNoError(t, err)
		defer connection.Close()
		_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=ON`)
		sqliteTestNoError(t, err)
		_, err = connection.ExecContext(context.Background(), `UPDATE effect_attempts SET `+assignment+` WHERE ref=?`, attemptRef)
		sqliteTestNoError(t, err)
		_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=OFF`)
		sqliteTestNoError(t, err)
	})
}

func requireV27RecoveryError(t *testing.T, system *sqliteV15System, code string) {
	t.Helper()
	_, _, err := validateRecoveryDatabase(context.Background(), system.repository.db)
	if err == nil || !recoveryErrorContains(err, code) {
		t.Fatalf("V27 corruption accepted or wrong error: %s", sqliteTestErrorChain(err))
	}
}

func requireV27ValidatorError(t *testing.T, system *sqliteV15System, code string) {
	t.Helper()
	transaction, err := system.repository.db.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	err = validateRecoveryV27EffectAttemptClaimLease(context.Background(), transaction)
	if err == nil || !recoveryErrorContains(err, code) {
		t.Fatalf("V27 validator accepted corruption or returned wrong error: %s", sqliteTestErrorChain(err))
	}
}
