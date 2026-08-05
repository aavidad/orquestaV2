package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func TestV29LaunchPreparationPersistsEnvironmentPreservationAcrossRestart(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	claim := prepareV29Launch(t, system, "request:v29-preserve", true)
	assertV29ExecutionPreservation(t, system.repository, claim, true)

	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertV29ExecutionPreservation(t, reopened, claim, true)
}

func TestV29LaunchPreparationKeepsFalseEnvironmentPreservationAcrossRestart(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	claim := prepareV29Launch(t, system, "request:v29-no-preserve", false)
	assertV29ExecutionPreservation(t, system.repository, claim, false)

	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertV29ExecutionPreservation(t, reopened, claim, false)
}

func TestV29EnvironmentPreservationRatchetRejectsReversionAndCrossStateMutation(t *testing.T) {
	t.Run("reversion after queued to dispatching", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		claim := prepareV29Launch(t, system, "request:v29-reversion", true)
		if _, err := system.repository.db.Exec(`
UPDATE executions SET environment_preservation_required=0 WHERE ref=?`, claim.Action.ExecutionRef.String()); err == nil {
			t.Fatal("environment preservation reversion was accepted")
		}
		assertV29ExecutionPreservation(t, system.repository, claim, true)
	})

	t.Run("dispatching without transition", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		claim := prepareV29Launch(t, system, "request:v29-dispatching-mutation", false)
		if _, err := system.repository.db.Exec(`
UPDATE executions SET environment_preservation_required=1 WHERE ref=?`, claim.Action.ExecutionRef.String()); err == nil {
			t.Fatal("dispatching preservation mutation without running transition was accepted")
		}
		assertV29ExecutionPreservation(t, system.repository, claim, false)
	})

	t.Run("queued cross state", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		created := system.submit(t, "request:v29-cross-state")
		execution := created.Record.Executions[0]
		for _, statement := range []string{
			`UPDATE executions SET environment_preservation_required=1 WHERE ref=?`,
			`UPDATE executions SET state='running',environment_preservation_required=1 WHERE ref=?`,
		} {
			if _, err := system.repository.db.Exec(statement, execution.Ref.String()); err == nil {
				t.Fatalf("cross-state preservation mutation was accepted: %s", statement)
			}
		}
		var state string
		var required int
		sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT state,environment_preservation_required FROM executions WHERE ref=?`, execution.Ref.String()).Scan(&state, &required))
		if state != string(application.ExecutionQueued) || required != 0 {
			t.Fatalf("rejected cross-state mutation escaped: state=%s required=%d", state, required)
		}
	})

	t.Run("historical dispatching to running compatibility", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		claim := prepareV29Launch(t, system, "request:v29-historical", false)
		_, err := system.repository.db.Exec(`
UPDATE executions SET state='running',environment_preservation_required=1 WHERE ref=?`, claim.Action.ExecutionRef.String())
		sqliteTestNoError(t, err)
		if _, err = system.repository.db.Exec(`
UPDATE executions SET environment_preservation_required=0 WHERE ref=?`, claim.Action.ExecutionRef.String()); err == nil {
			t.Fatal("historical ratchet accepted reversion after running")
		}
	})
}

func TestV29MigrationChangesOnlyTheEnvironmentPreservationRatchet(t *testing.T) {
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if len(migrations) != recoverySchemaLatest ||
		migrations[recoverySchemaV38PreservationRatchet-1].name != "029_execution_environment_preservation_ratchet.sql" {
		t.Fatalf("migration chain count=%d latest=%q", len(migrations), migrations[len(migrations)-1].name)
	}

	triggerSQL := func(version int) string {
		t.Helper()
		database, openErr := sql.Open(driverName, ":memory:")
		sqliteTestNoError(t, openErr)
		t.Cleanup(func() { _ = database.Close() })
		prefix, prefixErr := recoveryMigrationPrefix(migrations, version)
		sqliteTestNoError(t, prefixErr)
		sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, prefix))
		var statement string
		sqliteTestNoError(t, database.QueryRow(`
SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='executions_environment_preservation_write_once'`).Scan(&statement))
		return statement
	}

	v28 := triggerSQL(recoverySchemaV38RecoveryClaim)
	if strings.Contains(v28, "OLD.state = 'queued'") || !strings.Contains(v28, "OLD.state='dispatching'") {
		t.Fatalf("V28 preservation trigger changed unexpectedly: %s", v28)
	}
	v29 := triggerSQL(recoverySchemaV38PreservationRatchet)
	if !strings.Contains(v29, "OLD.state = 'queued'") || !strings.Contains(v29, "OLD.state = 'dispatching'") {
		t.Fatalf("V29 preservation trigger lacks exact compatible transitions: %s", v29)
	}
}

func prepareV29Launch(t *testing.T, system *sqliteV15System, requestRef string, required bool) application.ActionClaim {
	t.Helper()
	system.submit(t, requestRef)
	claim := claimSQLiteV15(t, system, "claim:"+requestRef)
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		t.Fatal("claimed WorkItem missing")
	}
	started, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), claim.Action.ExecutionRef, system.clock.Now(),
	)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("claimed execution missing")
	}
	execution.State = application.ExecutionDispatching
	execution.BudgetReservationRef = claim.BudgetReservationRef
	execution.EffectIntentRef = claim.Action.EffectIntentRef
	execution.RequierePreservacionEntorno = required
	state := application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: started, Execution: execution,
		OperationAt: system.clock.Now(), Event: application.EventRecord{
			Ref: "event:dispatching:" + claim.Token, Kind: "execution.dispatching", GoalRef: claim.Action.GoalRef,
			WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef, OccurredAt: system.clock.Now(),
		},
	}
	sqliteTestNoError(t, validateLaunchPrepared(state))
	sqliteTestNoError(t, system.repository.RecordLaunchPrepared(context.Background(), state))
	return claim
}

func assertV29ExecutionPreservation(t *testing.T, repository *Repository, claim application.ActionClaim, want bool) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionDispatching || execution.RequierePreservacionEntorno != want {
		t.Fatalf("execution preservation found=%t state=%s required=%t want=%t", found, execution.State, execution.RequierePreservacionEntorno, want)
	}
}
