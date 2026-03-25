package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"

	_ "github.com/go-sql-driver/mysql"
)

type mysqlBackend struct{}

func init() {
	RegisterBackend(mysqlBackend{})
}

func (mysqlBackend) Name() string { return "mysql" }

func (mysqlBackend) Open(cfg storage.Config) (*sql.DB, error) {
	db, err := storage.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("abriendo DB mysql en %s: %w", storage.DisplayTarget(cfg), err)
	}
	return db, nil
}

func (mysqlBackend) Prepare(db *sql.DB, cfg storage.Config) error {
	if cfg.BootstrapSchema {
		if err := aplicarSchemaPorDriver(db, "mysql"); err != nil {
			return fmt.Errorf("aplicando schema mysql: %w", err)
		}
	}
	if err := postMigracionesPorDriver(db, "mysql"); err != nil {
		return fmt.Errorf("post-migraciones mysql: %w", err)
	}
	return nil
}

func (mysqlBackend) Backup(_ *sql.DB, _ string) error {
	return fmt.Errorf("respaldo mysql no soportado todavia desde Orquesta; usa mysqldump o un adaptador de backup dedicado")
}

func (mysqlBackend) IsBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "error 1205") ||
		strings.Contains(msg, "lock wait timeout exceeded") ||
		strings.Contains(msg, "error 3572") ||
		strings.Contains(msg, "statement aborted because lock(s) could not be acquired immediately")
}
