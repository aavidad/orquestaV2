package db

import (
	"database/sql"
	"os"
	"strings"
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

func TestSQLiteBackendOpenFuerzaJournalModeWALAunqueElDSNFalte(t *testing.T) {
	path := t.TempDir() + "/orquesta-open.db"
	cfg := storage.Config{
		Driver:          "sqlite",
		DSN:             path,
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	}
	raw, err := (sqliteBackend{}).Open(cfg)
	if err != nil {
		t.Fatalf("sqlite backend open: %v", err)
	}
	defer raw.Close()

	var journalMode string
	if err := raw.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("leer journal_mode: %v", err)
	}
	if strings.TrimSpace(strings.ToLower(journalMode)) != "wal" {
		t.Fatalf("journal_mode inesperado: %q", journalMode)
	}
}

func TestSQLiteBackendOpenReadOnlyNoFuerzaPragmasMutables(t *testing.T) {
	path := t.TempDir() + "/orquesta-open-readonly.db"
	rw, err := (sqliteBackend{}).Open(storage.Config{
		Driver:          "sqlite",
		DSN:             storage.SQLiteDSN(path),
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	})
	if err != nil {
		t.Fatalf("sqlite backend open rw: %v", err)
	}
	if _, err := rw.Exec(`CREATE TABLE IF NOT EXISTS prueba (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("crear tabla prueba: %v", err)
	}
	_ = rw.Close()

	ro, err := (sqliteBackend{}).Open(storage.Config{
		Driver:          "sqlite",
		DSN:             storage.SQLiteDSNWithMode(path, "ro"),
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: false,
	})
	if err != nil {
		t.Fatalf("sqlite backend open ro: %v", err)
	}
	defer ro.Close()

	var value int
	if err := ro.QueryRow(`SELECT 1`).Scan(&value); err != nil {
		t.Fatalf("consulta readonly: %v", err)
	}
	if value != 1 {
		t.Fatalf("valor inesperado en readonly: %d", value)
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

func TestSQLitePrepareAplicaPostMigracionesAunqueLaRevisionYaEsteMarcada(t *testing.T) {
	path := t.TempDir() + "/orquesta-runtime-orders-revision.db"
	cfg := storage.Config{
		Driver:          "sqlite",
		DSN:             storage.SQLiteDSN(path),
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	}
	raw, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("storage.Open sqlite: %v", err)
	}
	defer raw.Close()
	if err := (sqliteBackend{}).Prepare(raw, cfg); err != nil {
		t.Fatalf("Prepare sqlite inicial: %v", err)
	}

	if err := rebuildSQLiteTable(
		raw,
		"runtime_orders",
		[]string{"idx_runtime_orders_estado_created"},
		`CREATE TABLE runtime_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			handle_id INTEGER REFERENCES runtime_handles(id) ON DELETE SET NULL,
			tipo TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			resultado_json TEXT NOT NULL DEFAULT '{}',
			error_text TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','tomada','ejecutando','completada','fallida','expirada','cancelada')),
			available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		[]string{
			`CREATE INDEX idx_runtime_orders_estado_created ON runtime_orders(estado, available_at, id)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			return `INSERT INTO runtime_orders (
				id, agente, proyecto_id, runtime_id, handle_id, tipo, payload_json,
				resultado_json, error_text, estado, available_at, created_at,
				started_at, finished_at, updated_at
			)
			SELECT id, agente, proyecto_id, runtime_id, handle_id, tipo, payload_json,
			       resultado_json, error_text, estado, available_at, created_at,
			       started_at, finished_at, updated_at
			FROM ` + legacy, nil
		},
	); err != nil {
		t.Fatalf("rebuild runtime_orders legacy: %v", err)
	}
	if err := (sqliteBackend{}).Prepare(raw, cfg); err != nil {
		t.Fatalf("Prepare sqlite: %v", err)
	}

	for _, col := range []string{"claimed_by", "lease_token", "attempt_count", "lease_expires_at"} {
		ok, err := tablaTieneColumna(raw, "runtime_orders", col)
		if err != nil {
			t.Fatalf("tablaTieneColumna runtime_orders.%s: %v", col, err)
		}
		if !ok {
			t.Fatalf("faltaba columna post-migrada runtime_orders.%s", col)
		}
	}

	var updatedAt string
	if err := raw.QueryRow(`SELECT valor FROM config WHERE clave = ?`, sqliteBootstrapRevisionKey).Scan(&updatedAt); err != nil {
		t.Fatalf("leer revision sqlite: %v", err)
	}
	if updatedAt != sqliteBootstrapRevision {
		t.Fatalf("revision sqlite inesperada: %q", updatedAt)
	}
}

func TestSQLitePrepareRebuildRuntimeMailboxPermiteEmisorLogico(t *testing.T) {
	path := t.TempDir() + "/orquesta-runtime-mailbox-legacy.db"
	cfg := storage.Config{
		Driver:          "sqlite",
		DSN:             storage.SQLiteDSN(path),
		Path:            path,
		MaxOpenConns:    1,
		BootstrapSchema: true,
	}
	raw, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("storage.Open sqlite: %v", err)
	}
	defer raw.Close()
	if err := (sqliteBackend{}).Prepare(raw, cfg); err != nil {
		t.Fatalf("Prepare sqlite inicial: %v", err)
	}

	if err := rebuildSQLiteTable(
		raw,
		"runtime_mailbox",
		[]string{"idx_runtime_mailbox_destino_estado"},
		`CREATE TABLE runtime_mailbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_agente TEXT NOT NULL REFERENCES agentes(nombre),
			to_agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_order_id INTEGER REFERENCES runtime_orders(id) ON DELETE SET NULL,
			kind TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			estado TEXT NOT NULL DEFAULT 'pendiente'
				CHECK (estado IN ('pendiente','entregado','consumido','expirado','cancelado')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME,
			consumed_at DATETIME
		)`,
		[]string{
			`CREATE INDEX idx_runtime_mailbox_destino_estado ON runtime_mailbox(to_agente, estado, id DESC)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			return `INSERT INTO runtime_mailbox (
				id, from_agente, to_agente, proyecto_id, runtime_order_id,
				kind, payload_json, estado, created_at, delivered_at, consumed_at
			)
			SELECT id, from_agente, to_agente, proyecto_id, runtime_order_id,
			       kind, payload_json, estado, created_at, delivered_at, consumed_at
			FROM ` + legacy, nil
		},
	); err != nil {
		t.Fatalf("rebuild runtime_mailbox legacy: %v", err)
	}
	if err := (sqliteBackend{}).Prepare(raw, cfg); err != nil {
		t.Fatalf("Prepare sqlite tras runtime_mailbox legacy: %v", err)
	}

	sqlText, err := tablaSQL(raw, "runtime_mailbox")
	if err != nil {
		t.Fatalf("tablaSQL runtime_mailbox: %v", err)
	}
	if strings.Contains(strings.ToLower(sqlText), "from_agente text not null references agentes(nombre)") {
		t.Fatalf("runtime_mailbox sigue anclando from_agente a agentes: %s", sqlText)
	}
	if _, err := raw.Exec(`INSERT OR IGNORE INTO agentes (nombre, rol) VALUES ('Codex1', 'programador')`); err != nil {
		t.Fatalf("insert agente: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO runtime_mailbox (from_agente, to_agente, kind, payload_json, estado) VALUES ('server','Codex1','instruction','{}','pendiente')`); err != nil {
		t.Fatalf("insert mailbox emisor logico: %v", err)
	}
}
