package sqlite

import (
	"context"
	"testing"

	"orquesta/internal/application"
)

func TestV21RecoveryRejectsImpossibleExecutionSessionRevocations(t *testing.T) {
	tests := []struct{ name, trigger, query string }{
		{"ref", "", `UPDATE outbox SET ref='action:revoke-execution-session:other' WHERE kind='revoke_execution_session'`},
		{"plan", "outbox_identity_immutable", `UPDATE outbox SET plan_generation=plan_generation+1 WHERE kind='revoke_execution_session'`},
		{"item", "outbox_identity_immutable", `UPDATE outbox SET work_item_generation=work_item_generation+1 WHERE kind='revoke_execution_session'`},
		{"stale_item_without_receipt", "outbox_identity_immutable", `UPDATE outbox SET work_item_generation=work_item_generation-1 WHERE kind='revoke_execution_session'`},
		{"fields", "", `UPDATE outbox SET last_error_code='application.other_failure' WHERE kind='revoke_execution_session'`},
		{"fence", "", `UPDATE outbox SET claim_token='token:foreign',claimed_by='worker:foreign',claimed_until=available_at+1000000000,delivery_attempt=1,fence=(SELECT fence+1 FROM work_item_fences LIMIT 1) WHERE kind='revoke_execution_session'`},
		{"terminal", "", `UPDATE executions SET state='running',finished_at=NULL WHERE ref=(SELECT execution_ref FROM outbox WHERE kind='revoke_execution_session')`},
		{"session", "executions_session_ref_once", `UPDATE executions SET execution_session_ref='' WHERE ref=(SELECT execution_ref FROM outbox WHERE kind='revoke_execution_session')`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := seedV21PendingExecutionRevocation(t, test.name)
			mutate := func() { mustV10Exec(t, system.repository.db, test.query) }
			if test.trigger == "" {
				mutate()
			} else {
				rewriteRecoveryTrigger(t, system.repository.db, test.trigger, mutate)
			}
			requireV21RevocationError(t, system, "sqlite.recovery_v21_execution_session_revocation_invalid")
		})
	}
}

func TestV21RecoveryRequiresCompleteExactSessionRevocations(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		system := seedV21PendingExecutionRevocation(t, "missing")
		mustV10Exec(t, system.repository.db, `DELETE FROM outbox WHERE kind='revoke_execution_session'`)
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_v21_execution_session_revocation_missing") {
			t.Fatalf("terminal session without revocation passed recovery: %v", err)
		}
	})
	t.Run("receipt", func(t *testing.T) {
		system := seedV21PendingExecutionRevocation(t, "receipt")
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v21-revoke-receipt")
		if err != nil || !result.Processed || result.Action != application.ActionRevokeSession {
			t.Fatalf("consume revocation result=%+v err=%v", result, err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
			t.Fatalf("exact consumed revocation recovery: %v", err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "action_consumption_receipts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `UPDATE action_consumption_receipts SET governance_version=1 WHERE kind='revoke_execution_session'`)
		})
		requireV21RevocationError(t, system, "sqlite.recovery_v21_execution_session_revocation_invalid")
	})
	t.Run("stale_generation_does_not_borrow_receipt", func(t *testing.T) {
		system := seedV21PendingExecutionRevocation(t, "stale-generation-receipt")
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v21-revoke-stale-generation")
		if err != nil || !result.Processed || result.Action != application.ActionRevokeSession {
			t.Fatalf("consume revocation result=%+v err=%v", result, err)
		}
		rewriteRecoveryTrigger(t, system.repository.db, "outbox_identity_immutable", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE outbox SET work_item_generation=work_item_generation-1
WHERE kind='revoke_execution_session'`)
		})
		requireV21RevocationError(t, system, "sqlite.recovery_v21_execution_session_revocation_invalid")
	})
}

func requireV21RevocationError(t *testing.T, system *sqliteV15System, code string) {
	t.Helper()
	tx, err := system.repository.db.BeginTx(context.Background(), nil)
	if err == nil {
		err = validateRecoveryV21ExecutionSessionRevocations(context.Background(), tx)
		_ = tx.Rollback()
	}
	if err == nil || !recoveryErrorContains(err, code) {
		t.Fatalf("invalid revocation passed recovery: %s", sqliteTestErrorChain(err))
	}
}

func requireV21RevocationThenMissing(t *testing.T, repository *Repository, executionRef string) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("required revocation recovery: %s", sqliteTestErrorChain(err))
	}
	mustV10Exec(t, repository.db,
		`DELETE FROM outbox WHERE execution_ref=? AND kind='revoke_execution_session'`, executionRef)
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v21_execution_session_revocation_missing") {
		t.Fatalf("missing required revocation passed recovery: %v", err)
	}
}

func seedV21PendingExecutionRevocation(t *testing.T, suffix string) *sqliteV15System {
	t.Helper()
	system := newSQLiteV15System(t, 2)
	system.orchestrator = sqliteV22Orchestrator(t, system, &sqliteExecutionSessionBroker{at: system.clock.Now()})
	system.submit(t, "request:v21-revocation-"+suffix)
	for _, want := range []application.ActionKind{application.ActionLaunchAgent, application.ActionObserveAgent} {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v21-revocation-"+suffix)
		if err != nil || !result.Processed || result.Action != want {
			t.Fatalf("seed revocation result=%+v want=%s err=%v", result, want, err)
		}
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid pending revocation recovery: %v", err)
	}
	return system
}
