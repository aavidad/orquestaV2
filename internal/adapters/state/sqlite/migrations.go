package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version  int
	name     string
	checksum string
	sql      string
}

func applyMigrations(ctx context.Context, database *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return invalid(err)
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return mapDatabaseError(err)
	}
	defer func() { _ = transaction.Rollback() }()

	var current int
	if err := transaction.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return mapDatabaseError(err)
	}
	if current > migrations[len(migrations)-1].version {
		return invalid(fmt.Errorf("sqlite.schema_newer_than_binary"))
	}
	for _, migration := range migrations {
		if migration.version <= current {
			continue
		}
		if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
			return mapDatabaseError(err)
		}
		if _, err := transaction.ExecContext(
			ctx,
			"INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)",
			migration.version,
			migration.name,
			migration.checksum,
		); err != nil {
			return mapDatabaseError(err)
		}
		if _, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version)); err != nil {
			return mapDatabaseError(err)
		}
		current = migration.version
	}
	if err := verifyAppliedMigrations(ctx, transaction, migrations, current); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	result := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		separator := strings.IndexByte(entry.Name(), '_')
		if separator <= 0 {
			return nil, fmt.Errorf("sqlite.migration_name_invalid")
		}
		version, err := strconv.Atoi(entry.Name()[:separator])
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("sqlite.migration_version_invalid")
		}
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		result = append(result, migration{
			version: version, name: entry.Name(),
			checksum: "sha256:" + hex.EncodeToString(digest[:]), sql: string(content),
		})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].version < result[right].version })
	if len(result) == 0 {
		return nil, fmt.Errorf("sqlite.migrations_missing")
	}
	for index, migration := range result {
		if migration.version != index+1 {
			return nil, fmt.Errorf("sqlite.migration_sequence_invalid")
		}
	}
	return result, nil
}

func verifyAppliedMigrations(
	ctx context.Context,
	transaction *sql.Tx,
	migrations []migration,
	current int,
) error {
	if current == 0 {
		return nil
	}
	rows, err := transaction.QueryContext(ctx, "SELECT version, name, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return mapDatabaseError(err)
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		var version int
		var name string
		var checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return mapDatabaseError(err)
		}
		if index >= len(migrations) || migrations[index].version != version || migrations[index].name != name ||
			migrations[index].checksum != checksum {
			return invalid(fmt.Errorf("sqlite.migration_history_invalid"))
		}
		index++
	}
	if err := rows.Err(); err != nil {
		return mapDatabaseError(err)
	}
	if index != current {
		return invalid(fmt.Errorf("sqlite.migration_history_incomplete"))
	}
	return nil
}
