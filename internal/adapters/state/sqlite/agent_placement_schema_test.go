package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentPlacementMigrationIsProgressiveAtomicAndExact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "placement-v22.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38Physical)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	sqliteTestNoError(t, err)
	var version, receipts, tables int
	var columns string
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT user_version,(SELECT COUNT(*) FROM schema_migrations WHERE version=23),(SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name IN ('agent_quota_observations','agent_placement_bindings')),(SELECT group_concat(name,',') FROM pragma_table_info('agent_placement_bindings')) FROM pragma_user_version`).Scan(&version, &receipts, &tables, &columns))
	sqliteTestNoError(t, repository.Close())
	if version != recoverySchemaV38Environment || receipts != 1 || tables != 2 ||
		columns != "reservation_ref,placement_ref,quota_observation_ref,quota_observation_revision" {
		t.Fatalf("migración version=%d recibos=%d tablas=%d columnas=%q", version, receipts, tables, columns)
	}

	rollbackPath := filepath.Join(t.TempDir(), "placement-rollback.db")
	database = agentCapacityDatabase(t, rollbackPath, recoverySchemaV38Physical)
	_, err = database.Exec(`CREATE TABLE agent_placement_bindings(sentinel INTEGER)`)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(rollbackPath, 0o600))
	if failed, openErr := Open(context.Background(), Options{Path: rollbackPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4}); openErr == nil {
		_ = failed.Close()
		t.Fatal("migración parcialmente aplicable aceptada")
	}
	database = openFastV18MigrationFixture(t, rollbackPath)
	defer database.Close()
	sqliteTestNoError(t, database.QueryRow(`SELECT user_version,(SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name='agent_quota_observations'),(SELECT COUNT(*) FROM schema_migrations WHERE version=23) FROM pragma_user_version`).Scan(&version, &tables, &receipts))
	if version != recoverySchemaV38Physical || tables != 0 || receipts != 0 {
		t.Fatalf("rollback version=%d tablas=%d recibos=%d", version, tables, receipts)
	}
}

func TestAgentPlacementSchemaFencesQuotaAndImmutableBinding(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	system.submit(t, "request:q2-placement")
	claim := claimSQLiteV15(t, system, "claim:q2-placement")
	database := system.repository.db
	const quotaSQL = `INSERT INTO agent_quota_observations(ref,placement_ref,window_ref,status,quality,observed_at,expires_at,reset_at,retry_at,idempotency_key,expected_revision,revision) VALUES(?,?,'quota-window:q2',?,?,?,?,?,?,?,?,?)`
	quota := func(ref, placement, status, quality, key string, observed, expires int, reset, retry any, expected, revision int) error {
		_, err := database.Exec(quotaSQL, ref, placement, status, quality, observed, expires, reset, retry, key, expected, revision)
		return err
	}
	reject := func(err error, invariant string) {
		t.Helper()
		if err == nil {
			t.Fatal(invariant)
		}
	}
	sqliteTestNoError(t, quota("quota:q2-one", "placement:q2-one", "available", "measured", "quota-key:shared", 1, 10, nil, nil, 0, 1))
	reject(quota("quota:q2-gap", "placement:q2-one", "available", "measured", "quota-key:gap", 1, 10, nil, nil, 2, 3), "la cuota aceptó un salto CAS")
	reject(quota("quota:q2-stale", "placement:q2-one", "available", "measured", "quota-key:stale", 1, 10, nil, nil, 0, 1), "la cuota aceptó una revisión obsoleta")
	sqliteTestNoError(t, quota("quota:q2-two", "placement:q2-one", "exhausted", "exact", "quota-key:two", 1, 10, nil, nil, 1, 2))
	reject(quota("quota:q2-duplicate", "placement:q2-one", "unknown", "unknown", "quota-key:shared", 1, 10, nil, nil, 2, 3), "la cuota perdió idempotencia por colocación")
	sqliteTestNoError(t, quota("quota:q2-other", "placement:q2-other", "available", "estimated", "quota-key:shared", 1, 10, nil, nil, 0, 1))
	for _, invalid := range []struct {
		name, ref, status, quality string
		observed, expires          int
		reset, retry               any
	}{{"ref", " ", "available", "measured", 1, 10, nil, nil}, {"status", "quota:q2-bad-status", "bad", "measured", 1, 10, nil, nil}, {"quality", "quota:q2-bad-quality", "available", "bad", 1, 10, nil, nil}, {"expiry", "quota:q2-bad-expiry", "available", "measured", 10, 10, nil, nil}, {"reset", "quota:q2-bad-reset", "available", "measured", 1, 10, 1, nil}, {"retry", "quota:q2-bad-retry", "available", "measured", 1, 10, nil, 1}} {
		reject(quota(invalid.ref, "placement:q2-invalid", invalid.status, invalid.quality, "quota-key:invalid", invalid.observed, invalid.expires, invalid.reset, invalid.retry, 0, 1), "la cuota aceptó "+invalid.name+" inválido")
	}
	const bindingSQL = `INSERT INTO agent_placement_bindings(reservation_ref,placement_ref,quota_observation_ref,quota_observation_revision) VALUES(?,?,?,?)`
	bind := func(reservation, placement, observation string, revision int) error {
		_, err := database.Exec(bindingSQL, reservation, placement, observation, revision)
		return err
	}
	reject(bind(claim.CapacityReservation.Ref, "placement:q2-one", "quota:q2-one", 1), "una reserva aceptó dos bindings")
	var colocacion, cuotaRef string
	var cuotaRevision int
	sqliteTestNoError(t, database.QueryRow(`SELECT placement_ref,quota_observation_ref,quota_observation_revision FROM agent_placement_bindings WHERE reservation_ref=?`, claim.CapacityReservation.Ref).Scan(&colocacion, &cuotaRef, &cuotaRevision))
	if colocacion != claim.ReferenciaColocacion.String() || cuotaRef != system.capacidad[0].Quota.ObservationRef || cuotaRevision != int(system.capacidad[0].Quota.ObservationRevision) {
		t.Fatalf("binding del claim alterado: %s %s/%d", colocacion, cuotaRef, cuotaRevision)
	}
	for _, mutation := range []string{`UPDATE agent_placement_bindings SET placement_ref=placement_ref`, `DELETE FROM agent_placement_bindings`, `UPDATE agent_quota_observations SET status=status`, `DELETE FROM agent_quota_observations`} {
		_, err := database.Exec(mutation)
		reject(err, "un hecho inmutable aceptó UPDATE/DELETE")
	}
	transaction, err := database.Begin()
	sqliteTestNoError(t, err)
	_, err = transaction.Exec(bindingSQL, "capacity-reservation:missing", "placement:q2-other", "quota:q2-other", 1)
	sqliteTestNoError(t, err)
	reject(transaction.Commit(), "la FK diferida aceptó commit sin reserva")
}

func TestAgentPlacementRecoveryReopensAndRejectsSemanticTampering(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteV15System(t, 2)
	system.submit(t, "request:q2-recovery")
	claimSQLiteV15(t, system, "claim:q2-recovery")
	_, err := system.repository.db.Exec(`INSERT INTO agent_quota_observations(ref,placement_ref,window_ref,status,quality,observed_at,expires_at,evidence_ref,idempotency_key,expected_revision,revision) VALUES('quota:q2-recovery-one','placement:q2-recovery','quota-window:q2-recovery','available','measured',1,10,'artifact:q2-recovery','quota-key:q2-recovery-one',0,1)`)
	sqliteTestNoError(t, err)
	_, digest, err := validateRecoveryDatabase(ctx, system.repository.db)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	_, reopenedDigest, err := validateRecoveryDatabase(ctx, reopened.db)
	sqliteTestNoError(t, err)
	var quotaColumns, bindingColumns int
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT (SELECT COUNT(*) FROM pragma_table_info('agent_quota_observations')),(SELECT COUNT(*) FROM pragma_table_info('agent_placement_bindings'))`).Scan(&quotaColumns, &bindingColumns))
	if reopenedDigest != digest || quotaColumns != 13 || bindingColumns != 4 {
		t.Fatalf("reapertura cambió digest/columnas: %t %d+%d", reopenedDigest == digest, quotaColumns, bindingColumns)
	}
	_, err = reopened.db.Exec(`INSERT INTO agent_quota_observations(ref,placement_ref,window_ref,status,quality,observed_at,expires_at,idempotency_key,expected_revision,revision) VALUES('quota:q2-recovery-two','placement:q2-recovery','quota-window:q2-recovery','unknown','unknown',2,11,'quota-key:q2-recovery-two',1,2)`)
	sqliteTestNoError(t, err)
	_, changedDigest, err := validateRecoveryDatabase(ctx, reopened.db)
	sqliteTestNoError(t, err)
	if changedDigest == digest {
		t.Fatal("alterar un hecho no cambió el inventario lógico")
	}
	reopened.db.SetMaxOpenConns(1)
	rewriteRecoveryTrigger(t, reopened.db, "agent_quota_observations_immutable_update", func() {
		mustV10Exec(t, reopened.db, `PRAGMA ignore_check_constraints=ON`)
		mustV10Exec(t, reopened.db, `UPDATE agent_quota_observations SET status='invalid' WHERE ref='quota:q2-recovery-two'`)
		mustV10Exec(t, reopened.db, `PRAGMA ignore_check_constraints=OFF`)
	})
	transaction, err := reopened.db.BeginTx(ctx, nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	if err := validateRecoveryV38AgentPlacement(ctx, transaction); err == nil || !recoveryErrorContains(err, "sqlite.recovery_v38_agent_placement_invalid") {
		t.Fatalf("la recuperación aceptó cuota alterada: %v", err)
	}
	v22 := agentCapacityDatabase(t, filepath.Join(t.TempDir(), "placement-v22-recovery.db"), recoverySchemaV38Physical)
	defer v22.Close()
	if _, _, err := validateRecoveryDatabase(ctx, v22); err != nil {
		t.Fatalf("el esquema 22 dejó de ser recuperable: %v", err)
	}
}
