package db

import (
	"database/sql"
	"os"
	"testing"

	"orquesta/storage"
)

func TestSQLitePrepareOmiteBootstrapConSchemaActualSinRevision(t *testing.T) {
	prepararDBTemporal(t)
	dbPath := os.Getenv("ORQUESTA_DB")
	if dbPath == "" {
		t.Fatalf("ORQUESTA_DB vacio")
	}
	if _, err := DB.Exec(`DELETE FROM config WHERE clave = ?`, sqliteBootstrapRevisionKey); err != nil {
		t.Fatalf("borrar revision sqlite: %v", err)
	}
	Close()
	DB = nil
	if err := os.Unsetenv("ORQUESTA_DB_SKIP_POST_MIGRATIONS"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_SKIP_POST_MIGRATIONS: %v", err)
	}

	locker, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=100")
	if err != nil {
		t.Fatalf("abrir locker sqlite: %v", err)
	}
	defer locker.Close()
	if _, err := locker.Exec(`BEGIN IMMEDIATE`); err != nil {
		t.Fatalf("BEGIN IMMEDIATE: %v", err)
	}
	defer func() {
		_, _ = locker.Exec(`ROLLBACK`)
	}()

	cfg := storage.Config{
		Driver:          "sqlite",
		DSN:             dbPath + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=100",
		Path:            dbPath,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	}
	raw, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("abrir sqlite cruda: %v", err)
	}
	defer raw.Close()

	if err := (sqliteBackend{}).Prepare(raw, cfg); err != nil {
		t.Fatalf("prepare sqlite deberia omitir bootstrap con schema actual: %v", err)
	}
}

func TestSQLiteBootstrapRequiredEnDBVacia(t *testing.T) {
	path := t.TempDir() + "/orquesta-empty.db"
	cfg := storage.Config{
		Driver:          "sqlite",
		DSN:             storage.SQLiteDSN(path),
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	}
	raw, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("abrir sqlite vacia: %v", err)
	}
	defer raw.Close()

	required, err := sqliteBootstrapRequired(raw)
	if err != nil {
		t.Fatalf("sqliteBootstrapRequired: %v", err)
	}
	if !required {
		t.Fatalf("una DB vacia debe requerir bootstrap")
	}
}
