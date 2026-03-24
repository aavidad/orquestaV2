package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"

	_ "modernc.org/sqlite"
)

type sqliteBackend struct{}

func init() {
	RegisterBackend(sqliteBackend{})
}

func (sqliteBackend) Name() string { return "sqlite" }

func (sqliteBackend) Open(cfg storage.Config) (*sql.DB, error) {
	target := strings.TrimSpace(cfg.Path)
	if target == "" {
		target = strings.TrimSpace(cfg.DSN)
	}
	db, err := storage.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("abriendo DB sqlite en %s: %w", target, err)
	}
	return db, nil
}

func (sqliteBackend) Prepare(db *sql.DB, cfg storage.Config) error {
	if cfg.BootstrapSchema {
		if err := aplicarSchemaPorDriver(db, "sqlite"); err != nil {
			return fmt.Errorf("aplicando schema sqlite: %w", err)
		}
	}
	if err := postMigracionesPorDriver(db, "sqlite"); err != nil {
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
