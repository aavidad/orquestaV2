package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestRecoveryV32MicroVMHostLaunchAuthorityPreparedAndBoundSurviveReopen(t *testing.T) {
	t.Run("prepared", func(t *testing.T) {
		system, _, _, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "prepared")
		requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, system.repository.db)
		if err := system.repository.Close(); err != nil {
			t.Fatal(err)
		}
		reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
		requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, reopened.db)
		resolved, err := reopened.Resolve(context.Background(), authority.Key)
		if err != nil || resolved.ExternalRef != "" {
			t.Fatalf("prepared resolve=%+v err=%v", resolved, err)
		}
	})

	t.Run("bound with terminal facts", func(t *testing.T) {
		system, claim, attempt, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "bound")
		const externalRef = "ejecucion:physical_bound"
		if _, err := system.repository.BindExternal(context.Background(), authority.Key, externalRef); err != nil {
			t.Fatal(err)
		}
		recordRecoveryV32AcceptedLaunch(t, system, claim, attempt, externalRef)
		requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, system.repository.db)
		if err := system.repository.Close(); err != nil {
			t.Fatal(err)
		}
		reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
		requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, reopened.db)
		resolved, err := reopened.Resolve(context.Background(), authority.Key)
		if err != nil || resolved.ExternalRef != externalRef {
			t.Fatalf("bound resolve=%+v err=%v", resolved, err)
		}
	})

	t.Run("bound before terminal facts", func(t *testing.T) {
		system, _, _, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "bound-preterminal")
		if _, err := system.repository.BindExternal(
			context.Background(), authority.Key, "ejecucion:physical_preterminal",
		); err != nil {
			t.Fatal(err)
		}
		requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, system.repository.db)
	})
}

func TestRecoveryV32MicroVMHostLaunchAuthorityRejectsPreparedWithTerminalFacts(t *testing.T) {
	system, claim, attempt, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "prepared-terminal")
	recordRecoveryV32AcceptedLaunch(t, system, claim, attempt, "ejecucion:physical_unbound")
	requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
	resolved, err := system.repository.Resolve(context.Background(), authority.Key)
	if err != nil || resolved.ExternalRef != "" {
		t.Fatalf("prepared fixture unexpectedly bound: authority=%+v err=%v", resolved, err)
	}
}

func TestRecoveryV32MicroVMHostLaunchAuthorityRejectsStaticContractCorruption(t *testing.T) {
	system, _, _, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "static-corruption")
	rewriteRecoveryTrigger(t, system.repository.db, "microvm_host_launch_authorities_bind_once", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE microvm_host_launch_authorities SET request_ref=?
WHERE execution_ref=? AND action_fence=?`,
			"request:microvm-host-launch-one-shot:sha256:"+strings.Repeat("f", 64),
			authority.Key.RunRef.String(), authority.Key.ActionFence,
		)
	})
	requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
}

func TestRecoveryV32MicroVMHostLaunchAuthorityRejectsCrossedAttemptAndIntent(t *testing.T) {
	t.Run("attempt", func(t *testing.T) {
		system := newSQLiteV15System(t, 2)
		_, first := seedV32EffectAttempt(t, system, "recovery-attempt-first")
		_, second := seedV32EffectAttempt(t, system, "recovery-attempt-second")
		authority := sqliteMicroVMHostLaunchAuthority(t, first, false)
		if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
			t.Fatal(err)
		}
		requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
			authority.Key, second.Ref, authority.SessionRef,
		)
		if err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "microvm_host_launch_authorities_bind_once", func() {
			connection, err := system.repository.db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
				t.Fatal(err)
			}
			if _, err := connection.ExecContext(context.Background(), `
UPDATE microvm_host_launch_authorities
SET effect_attempt_ref=?,request_ref=?
WHERE execution_ref=? AND action_fence=?`, second.Ref, requestRef,
				authority.Key.RunRef.String(), authority.Key.ActionFence); err != nil {
				t.Fatal(err)
			}
			if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`); err != nil {
				t.Fatal(err)
			}
		})
		requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
	})

	t.Run("intent", func(t *testing.T) {
		system := newSQLiteV15System(t, 2)
		_, first := seedV32EffectAttempt(t, system, "recovery-intent-first")
		_, second := seedV32EffectAttempt(t, system, "recovery-intent-second")
		authority := sqliteMicroVMHostLaunchAuthority(t, first, true)
		if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
			t.Fatal(err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "effect_attempts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `UPDATE effect_attempts SET intent_ref=? WHERE ref=?`,
				second.IntentRef, first.Ref)
		})
		requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
	})
}

func TestRecoveryV32MicroVMHostLaunchAuthorityRejectsCrossedExecutionSession(t *testing.T) {
	system, _, _, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "crossed-session")
	crossed := recoveryV32ExecutionSessionRef(t, "recovery-crossed")
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
		authority.Key, authority.EffectAttemptRef, crossed,
	)
	if err != nil {
		t.Fatal(err)
	}
	rewriteRecoveryTrigger(t, system.repository.db, "microvm_host_launch_authorities_bind_once", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE microvm_host_launch_authorities SET session_ref=?,request_ref=?
WHERE execution_ref=? AND action_fence=?`, crossed.String(), requestRef,
			authority.Key.RunRef.String(), authority.Key.ActionFence)
	})
	rewritten, err := system.repository.Resolve(context.Background(), authority.Key)
	if err != nil || ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(rewritten) != nil {
		t.Fatalf("coherently crossed fixture invalid before recovery: authority=%+v err=%v", rewritten, err)
	}
	requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
}

func TestRecoveryV32MicroVMHostLaunchAuthorityRejectsExternalFactCrossing(t *testing.T) {
	for _, fact := range []string{"receipt", "execution"} {
		t.Run(fact, func(t *testing.T) {
			system, claim, attempt, authority := seedRecoveryV32MicroVMHostLaunchAuthority(t, "external-"+fact)
			const externalRef = "ejecucion:physical_exact"
			if _, err := system.repository.BindExternal(context.Background(), authority.Key, externalRef); err != nil {
				t.Fatal(err)
			}
			recordRecoveryV32AcceptedLaunch(t, system, claim, attempt, externalRef)
			requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, system.repository.db)
			switch fact {
			case "receipt":
				rewriteRecoveryTrigger(t, system.repository.db, "effect_receipts_immutable_update", func() {
					mustV10Exec(t, system.repository.db, `UPDATE effect_receipts SET external_ref=? WHERE attempt_ref=?`,
						"ejecucion:physical_crossed", attempt.Ref)
				})
			case "execution":
				rewriteRecoveryTrigger(t, system.repository.db, "executions_provider_identity_write_once", func() {
					mustV10Exec(t, system.repository.db, `UPDATE executions SET external_ref=? WHERE ref=?`,
						"ejecucion:physical_crossed", attempt.Subject.ExecutionRef.String())
				})
			}
			requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t, system.repository.db)
		})
	}
}

func TestRecoveryV31PrefixDoesNotRequireMicroVMHostLaunchAuthorityTable(t *testing.T) {
	database := agentCapacityDatabase(
		t, filepath.Join(t.TempDir(), "recovery-v31.sqlite"), recoverySchemaV38EgressAuthority,
	)
	defer database.Close()
	var tables int
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name='microvm_host_launch_authorities'`).Scan(&tables))
	if tables != 0 {
		t.Fatal("V31 unexpectedly contains V32 authority table")
	}
	transaction, err := database.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if err := validateRecoveryVersion(context.Background(), transaction, recoverySchemaV38EgressAuthority); err != nil {
		t.Fatalf("V31 recovery queried V32 authority: %v", err)
	}
}

func seedRecoveryV32MicroVMHostLaunchAuthority(
	t *testing.T,
	suffix string,
) (*sqliteV15System, application.ActionClaim, application.EffectAttempt, ports.MicroVMHostLaunchAuthorityV1) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	claim, attempt := seedV32EffectAttempt(t, system, suffix)
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	authority.SessionRef = recoveryV32ExecutionSessionRef(t, attempt.Ref)
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
		authority.Key, authority.EffectAttemptRef, authority.SessionRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	authority.OneShotClaim.RequestRef = requestRef
	mustV10Exec(t, system.repository.db, `
UPDATE executions SET execution_session_ref=? WHERE ref=?`,
		authority.SessionRef.String(), authority.Key.RunRef.String())
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}
	return system, claim, attempt, authority
}

func recoveryV32ExecutionSessionRef(t *testing.T, seed string) ports.ExecutionSessionRef {
	t.Helper()
	digest := sha256.Sum256([]byte(seed))
	ref, err := ports.NewExecutionSessionRef("execution-session:sha256:" + hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func recordRecoveryV32AcceptedLaunch(
	t *testing.T,
	system *sqliteV15System,
	claim application.ActionClaim,
	attempt application.EffectAttempt,
	externalRef string,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("claimed execution missing")
	}
	at := system.clock.Now()
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, at)
	receipt.ExternalRef = externalRef
	execution.State, execution.ProviderRef, execution.ModelRef =
		application.ExecutionRunning, "provider:microvm", "model:microvm"
	execution.AgentRef, execution.ExternalRef, execution.LaunchReceiptRef =
		"agent:microvm", externalRef, receipt.Ref
	execution.StartedAt, execution.DeadlineAt, execution.ProviderAcceptedAt = at, claim.LeaseUntil, at
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	err = system.repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, EffectReceipt: receipt, OperationAt: at,
		NextAction: application.ActionRecord{
			Ref: "action:observe:recovery-v32:" + claim.Token, Kind: application.ActionObserveAgent,
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: item.Revision(), AvailableAt: at,
		},
		Event: application.EventRecord{
			Ref: "event:accepted:recovery-v32:" + claim.Token, Kind: "execution.accepted",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	})
	sqliteTestNoError(t, err)
}

func requireRecoveryV32MicroVMHostLaunchAuthorityValid(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(context.Background(), database); err != nil {
		t.Fatalf("valid V32 authority recovery failed: %s", sqliteTestErrorChain(err))
	}
}

func requireRecoveryV32MicroVMHostLaunchAuthorityInvalid(t *testing.T, database *sql.DB) {
	t.Helper()
	transaction, err := database.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	err = validateRecoveryV38MicroVMHostLaunchAuthority(context.Background(), transaction)
	if !application.IsStateError(err, application.StateInvalid) ||
		!recoveryErrorContains(err, recoveryV38MicroVMHostLaunchAuthorityInvalid) {
		t.Fatalf("corrupt V32 authority accepted: %s", sqliteTestErrorChain(err))
	}
}
