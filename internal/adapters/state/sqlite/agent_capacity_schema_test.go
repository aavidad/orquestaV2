package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"orquesta/internal/application"
)

func TestAgentCapacityMigrationRunsOnceAndRollsBackAsAUnit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capacity-v21.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV23)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("migrar capacidad: %s", sqliteTestErrorChain(err))
	}
	var version, receipts int
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38Physical,
	).Scan(&receipts))
	if version != recoverySchemaV38Capacity || receipts != 1 {
		t.Fatalf("migración inicial version=%d recibos=%d", version, receipts)
	}
	sqliteTestNoError(t, repository.Close())

	rollbackPath := filepath.Join(t.TempDir(), "capacity-rollback.db")
	database = agentCapacityDatabase(t, rollbackPath, recoverySchemaV23)
	_, err = database.Exec(`CREATE TABLE agent_capacity_reservations(sentinel INTEGER)`)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(rollbackPath, 0o600))
	if _, openErr := Open(context.Background(), Options{
		Path: rollbackPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	}); openErr == nil {
		t.Fatalf("migración parcialmente aplicable aceptada: %v", openErr)
	}
	database = openFastV18MigrationFixture(t, rollbackPath)
	defer database.Close()
	var observations int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE type='table' AND name='agent_capacity_observations'`).Scan(&observations))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38Physical,
	).Scan(&receipts))
	if version != recoverySchemaV23 || observations != 0 || receipts != 0 {
		t.Fatalf("rollback version=%d observaciones=%d recibos=%d", version, observations, receipts)
	}
}

func TestAgentCapacityFactsFenceScopeCASReopenAndBackupRestore(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteV15System(t, 2)
	system.submit(t, "request:a03-capacity")
	claim := claimSQLiteV15(t, system, "claim:a03-capacity")

	insertAgentCapacityObservation(t, system.repository.db,
		"capacity-observation:one", "capacity-window:one", 0, 1, "observe:one")
	if _, err := system.repository.db.Exec(agentCapacityObservationInsert,
		"capacity-observation:gap", "capacity-window:one", 3, 2, 4, "observe:gap"); err == nil {
		t.Fatal("la observación omitió la revisión anterior")
	}
	insertAgentCapacityObservation(t, system.repository.db,
		"capacity-observation:two", "capacity-window:one", 1, 2, "observe:two")

	if err := insertAgentCapacityReservation(system.repository.db, claim,
		"project:otro", "capacity-observation:two", 2,
		"capacity-reservation:cross", "reserve:cross", claim.Fence, "reserved", 1); err == nil {
		t.Fatal("la reserva cruzó de proyecto")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), "capacity-observation:one", 1,
		"capacity-reservation:stale", "reserve:stale", claim.Fence, "reserved", 1); err == nil {
		t.Fatal("la reserva usó una observación obsoleta")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), "capacity-observation:two", 2,
		"capacity-reservation:fence", "reserve:fence", claim.Fence+1, "reserved", 1); err == nil {
		t.Fatal("la reserva aceptó un fence ajeno")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), "capacity-observation:two", 2,
		"capacity-reservation:initial", "reserve:initial", claim.Fence, "consumed", 2); err == nil {
		t.Fatal("la reserva saltó el snapshot inicial")
	}
	sqliteTestNoError(t, insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), "capacity-observation:two", 2,
		"capacity-reservation:one", "reserve:one", claim.Fence, "reserved", 1))
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), "capacity-observation:two", 2,
		"capacity-reservation:duplicate", "reserve:one", claim.Fence, "reserved", 1); err == nil {
		t.Fatal("la reserva perdió idempotencia")
	}

	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	_, created, err := system.repository.RecordEffectAttempt(ctx, application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	})
	if err != nil || !created {
		t.Fatalf("intento created=%v err=%v", created, err)
	}
	applyAgentCapacityTransition(t, system.repository.db,
		"capacity-transition:quarantined", 1, "quarantined", "unknown_applied",
		attempt.Ref, attempt.Ref, "", "transition:quarantined", claim.Fence)
	receiptRef := acceptAgentCapacityLaunch(t, system, claim, attempt)
	applyAgentCapacityTransition(t, system.repository.db,
		"capacity-transition:consumed", 2, "consumed", "reconciliation",
		receiptRef, attempt.Ref, receiptRef, "transition:consumed", claim.Fence)
	if err := insertAgentCapacityTransition(system.repository.db,
		"capacity-transition:impossible", 4, "released", "execution_terminal",
		claim.Action.ExecutionRef.String(), "", "", "transition:impossible", claim.Fence); err == nil {
		t.Fatal("la transición omitió la revisión esperada")
	}
	applyAgentCapacityTransition(t, system.repository.db,
		"capacity-transition:released", 3, "released", "execution_terminal",
		claim.Action.ExecutionRef.String(), "", "", "transition:released", claim.Fence)
	assertAgentCapacityCounts(t, system.repository.db, 2, 1, 3)

	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertAgentCapacityCounts(t, reopened.db, 2, 1, 3)

	recovery, _, _ := newV09TestRecovery(t, reopened, system.clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	sqliteTestNoError(t, err)
	_, err = recovery.VerifyBackup(ctx, backup.Ref)
	sqliteTestNoError(t, err)
	target, err := application.NewRecoveryTargetRef("recovery-target:a03-capacity")
	sqliteTestNoError(t, err)
	_, err = recovery.RestoreBackup(ctx, backup.Ref, target)
	sqliteTestNoError(t, err)
	targetPath, err := recovery.TargetPath(target)
	sqliteTestNoError(t, err)
	restored := openSQLiteV15Repository(t, targetPath, system.clock.Now)
	assertAgentCapacityCounts(t, restored.db, 2, 1, 3)
}

func acceptAgentCapacityLaunch(t *testing.T, system *sqliteV15System, claim application.ActionClaim, attempt application.EffectAttempt) string {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found {
		t.Fatal("claimed execution missing")
	}
	at := system.clock.Now()
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, at)
	execution.State, execution.ProviderRef, execution.ModelRef = application.ExecutionRunning, "provider:codex", "model:codex"
	execution.AgentRef, execution.ExternalRef, execution.LaunchReceiptRef = "agent:codex", "external:"+execution.Ref.String(), receipt.Ref
	execution.StartedAt, execution.DeadlineAt, execution.ProviderAcceptedAt = at, claim.LeaseUntil, at
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	sqliteTestNoError(t, system.repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, EffectReceipt: receipt, OperationAt: at,
		NextAction: application.ActionRecord{Ref: "action:observe:a03", Kind: application.ActionObserveAgent,
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: item.Revision(), AvailableAt: at},
		Event: application.EventRecord{Ref: "event:accepted:a03", Kind: "execution.accepted",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
	}))
	return receipt.Ref
}

const agentCapacityObservationInsert = `
INSERT INTO agent_capacity_observations(
 ref,source_ref,pool_ref,window_ref,revision,expected_revision,status,quality,observed_at,expires_at,
 slots_applicability,slots_limit,slots_remaining,seconds_applicability,messages_applicability,tokens_applicability,credits_applicability,idempotency_key
) VALUES (?, 'capacity-source:test', 'capacity-pool:test', ?, ?, ?,
 'available','exact',1,100,'applicable',5,?,'not_applicable','not_applicable','not_applicable','not_applicable',?)`

func insertAgentCapacityObservation(t *testing.T, db *sql.DB, ref, window string, expected, revision int, key string) {
	t.Helper()
	_, err := db.Exec(agentCapacityObservationInsert, ref, window, revision, expected, 4, key)
	sqliteTestNoError(t, err)
}

func insertAgentCapacityReservation(db *sql.DB, claim application.ActionClaim, project, observation string,
	observationRevision int, ref, key string, fence uint64, state string, revision int) error {
	_, err := db.Exec(`
INSERT INTO agent_capacity_reservations(
 ref,observation_ref,observation_revision,effect_intent_ref,project_ref,goal_ref,work_item_ref,execution_ref,
 action_ref,plan_generation,work_item_generation,fence,state,revision,slots,seconds,messages,tokens,credits,idempotency_key,reserved_at,updated_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,0,0,0,0,?,2,2)`,
		ref, observation, observationRevision, claim.Action.EffectIntentRef, project,
		claim.Action.GoalRef.String(), claim.Action.WorkItemRef.String(),
		claim.Action.ExecutionRef.String(), claim.Action.Ref,
		int64(claim.Action.PlanGeneration), int64(claim.Action.WorkItemGeneration),
		fence, state, revision, key)
	return err
}

func insertAgentCapacityTransition(db interface {
	Exec(string, ...any) (sql.Result, error)
},
	ref string, expected int, outcome, causeKind, causeRef, attemptRef, receiptRef, key string, fence uint64) error {
	_, err := db.Exec(`
INSERT INTO agent_capacity_transitions(
 ref,reservation_ref,project_ref,fence,expected_revision,revision,outcome,cause_kind,cause_ref,effect_attempt_ref,effect_receipt_ref,idempotency_key,recorded_at
) VALUES (?,'capacity-reservation:one','project:v15',?,?,?, ?,?,?,?,?,?,3)`,
		ref, fence, expected, expected+1, outcome, causeKind, causeRef,
		nullableString(attemptRef), nullableString(receiptRef), key)
	return err
}

func applyAgentCapacityTransition(t *testing.T, db *sql.DB, ref string, expected int, outcome, causeKind,
	causeRef, attemptRef, receiptRef, key string, fence uint64) {
	t.Helper()
	transaction, err := db.Begin()
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	sqliteTestNoError(t, insertAgentCapacityTransition(
		transaction, ref, expected, outcome, causeKind, causeRef,
		attemptRef, receiptRef, key, fence,
	))
	var settled any
	if outcome == "released" {
		settled = int64(3)
	}
	_, err = transaction.Exec(`
UPDATE agent_capacity_reservations
SET state=?,revision=?,last_transition_ref=?,last_cause_ref=?,updated_at=3,settled_at=?
WHERE ref='capacity-reservation:one' AND revision=?`,
		outcome, expected+1, ref, causeRef, settled, expected)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, transaction.Commit())
}

func assertAgentCapacityCounts(t *testing.T, db *sql.DB, observations, reservations, transitions int) {
	t.Helper()
	var gotObservations, gotReservations, gotTransitions int
	var state string
	var revision int
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_observations`).Scan(&gotObservations))
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_reservations`).Scan(&gotReservations))
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_transitions`).Scan(&gotTransitions))
	sqliteTestNoError(t, db.QueryRow(`SELECT state,revision FROM agent_capacity_reservations
WHERE ref='capacity-reservation:one'`).Scan(&state, &revision))
	if gotObservations != observations || gotReservations != reservations || gotTransitions != transitions {
		t.Fatalf("hechos observaciones=%d reservas=%d transiciones=%d",
			gotObservations, gotReservations, gotTransitions)
	}
	if state != "released" || revision != 4 {
		t.Fatalf("snapshot autoritativo state=%s revision=%d", state, revision)
	}
}

func agentCapacityDatabase(t *testing.T, path string, version int) *sql.DB {
	t.Helper()
	sqliteTestNoError(t, os.Chmod(filepath.Dir(path), 0o700))
	database := openFastV18MigrationFixture(t, path)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	prefix, err := recoveryMigrationPrefix(migrations, version)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, prefix))
	return database
}
