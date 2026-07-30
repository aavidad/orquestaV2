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
	if version != recoverySchemaV38Capacity || receipts != 1 || tables != 2 ||
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
