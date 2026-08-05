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
		recoverySchemaV38Claim,
	).Scan(&receipts))
	if version != recoverySchemaLatest || receipts != 1 {
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
	if claim.CapacityReservation.Ref == "" {
		t.Fatal("el claim no creó la reserva física")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		"project:otro", claim.CapacityReservation.ObservationRef, int(claim.CapacityReservation.ObservationRevision),
		"capacity-reservation:cross", "reserve:cross", claim.Fence, "reserved", 1); err == nil {
		t.Fatal("la reserva cruzó de proyecto")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), claim.CapacityReservation.ObservationRef, int(claim.CapacityReservation.ObservationRevision),
		"capacity-reservation:fence", "reserve:fence", claim.Fence+1, "reserved", 1); err == nil {
		t.Fatal("la reserva aceptó un fence ajeno")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), claim.CapacityReservation.ObservationRef, int(claim.CapacityReservation.ObservationRevision),
		"capacity-reservation:initial", "reserve:initial", claim.Fence, "consumed", 2); err == nil {
		t.Fatal("la reserva saltó el snapshot inicial")
	}
	if err := insertAgentCapacityReservation(system.repository.db, claim,
		system.project.String(), claim.CapacityReservation.ObservationRef, int(claim.CapacityReservation.ObservationRevision),
		"capacity-reservation:duplicate", "reserve:duplicate", claim.Fence, "reserved", 1); err == nil {
		t.Fatal("la acción obtuvo una segunda reserva")
	}

	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	_, created, err := system.repository.RecordEffectAttempt(ctx, application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	})
	if err != nil || !created {
		t.Fatalf("intento created=%v err=%v", created, err)
	}
	acceptAgentCapacityLaunch(t, system, claim, attempt)
	var estado string
	var revision int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT state,revision FROM agent_capacity_reservations WHERE ref=?`, claim.CapacityReservation.Ref).Scan(&estado, &revision))
	if estado != "consumed" || revision != 2 {
		t.Fatalf("reserva aceptada estado=%s revisión=%d", estado, revision)
	}
	transaction, err := system.repository.db.Begin()
	sqliteTestNoError(t, err)
	execution := application.ExecutionRecord{Ref: claim.Action.ExecutionRef, State: application.ExecutionFailed, FinishedAt: system.clock.Now()}
	sqliteTestNoError(t, liberarCapacidadEjecucionTerminal(ctx, transaction, execution))
	sqliteTestNoError(t, transaction.Commit())
	assertAgentCapacityCounts(t, system.repository.db, claim.CapacityReservation.Ref, 1, 1, 2, 3)

	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertAgentCapacityCounts(t, reopened.db, claim.CapacityReservation.Ref, 1, 1, 2, 3)

	recovery, _, _ := newV09TestRecovery(t, reopened, system.clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("crear backup: %s", sqliteTestErrorChain(err))
	}
	_, err = recovery.VerifyBackup(ctx, backup.Ref)
	sqliteTestNoError(t, err)
	target, err := application.NewRecoveryTargetRef("recovery-target:a03-capacity")
	sqliteTestNoError(t, err)
	_, err = recovery.RestoreBackup(ctx, backup.Ref, target)
	sqliteTestNoError(t, err)
	targetPath, err := recovery.TargetPath(target)
	sqliteTestNoError(t, err)
	restored := openSQLiteV15Repository(t, targetPath, system.clock.Now)
	assertAgentCapacityCounts(t, restored.db, claim.CapacityReservation.Ref, 1, 1, 2, 3)
}

func TestAgentCapacityRecoveryRejectsPhysicalSemanticTampering(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	system.submit(t, "request:q4-recuperacion-fisica")
	claim := claimSQLiteV15(t, system, "claim:q4-recuperacion-fisica")
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("candidato físico válido rechazado: %s", sqliteTestErrorChain(err))
	}
	rewriteRecoveryTrigger(t, system.repository.db, "agent_capacity_observations_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `UPDATE agent_capacity_observations SET window_ref='capacity-window:alterada' WHERE ref=?`, claim.CapacityReservation.ObservationRef)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v38_agent_placement_invalid") {
		t.Fatalf("la recuperación aceptó observación física alterada: %v", err)
	}
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

func assertAgentCapacityCounts(t *testing.T, db *sql.DB, reservationRef string, observations, reservations, transitions, expectedRevision int) {
	t.Helper()
	var gotObservations, gotReservations, gotTransitions int
	var state string
	var revision int
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_observations`).Scan(&gotObservations))
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_reservations`).Scan(&gotReservations))
	sqliteTestNoError(t, db.QueryRow(`SELECT COUNT(*) FROM agent_capacity_transitions`).Scan(&gotTransitions))
	sqliteTestNoError(t, db.QueryRow(`SELECT state,revision FROM agent_capacity_reservations
WHERE ref=?`, reservationRef).Scan(&state, &revision))
	if gotObservations != observations || gotReservations != reservations || gotTransitions != transitions {
		t.Fatalf("hechos observaciones=%d reservas=%d transiciones=%d",
			gotObservations, gotReservations, gotTransitions)
	}
	if state != "released" || revision != expectedRevision {
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
