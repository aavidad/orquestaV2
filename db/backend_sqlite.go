package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type sqliteBackend struct{}

func init() {
	RegisterBackend(sqliteBackend{})
}

func (sqliteBackend) Name() string { return "sqlite" }

func (sqliteBackend) Open(target string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", target+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("abriendo DB sqlite en %s: %w", target, err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func (sqliteBackend) Prepare(db *sql.DB) error {
	if err := aplicarSchema(db); err != nil {
		return fmt.Errorf("aplicando schema sqlite: %w", err)
	}
	if err := postMigraciones(db); err != nil {
		return fmt.Errorf("post-migraciones sqlite: %w", err)
	}
	return nil
}

func (sqliteBackend) Backup(db *sql.DB, path string) error {
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO %s", literalSQLite(path))); err != nil {
		return fmt.Errorf("crear respaldo sqlite: %w", err)
	}
	return nil
}

func (sqliteBackend) IsBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

func literalSQLite(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}
