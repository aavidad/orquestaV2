package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"orquesta/storage"
)

func TestOpenMigraRuntimeTablesLegacy(t *testing.T) {
	if DB != nil {
		Close()
	}

	prev := os.Getenv("ORQUESTA_DB")
	ruta := filepath.Join(t.TempDir(), "legacy_runtime.db")
	if err := os.Setenv("ORQUESTA_DB", ruta); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		Close()
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
			return
		}
		_ = os.Setenv("ORQUESTA_DB", prev)
	})

	raw, err := sql.Open("sqlite", storage.SQLiteDSN(ruta))
	if err != nil {
		t.Fatalf("sql.Open legacy: %v", err)
	}
	defer raw.Close()

	stmts := []string{
		`CREATE TABLE agentes (
			nombre TEXT PRIMARY KEY,
			rol TEXT NOT NULL CHECK (rol IN ('programador','documentador','admin')),
			activo INTEGER NOT NULL DEFAULT 0,
			habilitado INTEGER NOT NULL DEFAULT 1,
			ultima_sesion DATETIME
		)`,
		`INSERT INTO agentes (nombre, rol) VALUES ('Codex1', 'programador')`,
		`CREATE TABLE runtime_handles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL,
			sesion_id INTEGER,
			transporte TEXT NOT NULL DEFAULT 'cli',
			handle_kind TEXT NOT NULL DEFAULT 'session',
			handle_ref TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','cerrado')),
			last_seen_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO runtime_handles (id, agente, sesion_id, transporte, handle_kind, handle_ref, estado)
		 VALUES (7, 'Codex1', NULL, 'cli', 'session', 'codex:1', 'activo')`,
		`CREATE TABLE runtime_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL,
			tipo TEXT NOT NULL CHECK (tipo IN ('sync_status')),
			payload_json TEXT NOT NULL DEFAULT '{}',
			estado TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','completada','fallida')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME
		)`,
		`INSERT INTO runtime_orders (id, agente, tipo, payload_json, estado)
		 VALUES (9, 'Codex1', 'sync_status', '{}', 'pendiente')`,
	}
	for _, stmt := range stmts {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("crear schema legacy: %v\nstmt=%s", err, stmt)
		}
	}

	if err := Open(); err != nil {
		t.Fatalf("Open tras schema legacy: %v", err)
	}

	for _, col := range []string{"runtime_id", "lease_token", "capabilities_json", "metadata_json", "updated_at"} {
		ok, err := tablaTieneColumna(DB.DB, "runtime_handles", col)
		if err != nil {
			t.Fatalf("tablaTieneColumna runtime_handles.%s: %v", col, err)
		}
		if !ok {
			t.Fatalf("falta columna migrada runtime_handles.%s", col)
		}
	}
	for _, col := range []string{"proyecto_id", "runtime_id", "handle_id", "resultado_json", "error_text", "available_at", "updated_at"} {
		ok, err := tablaTieneColumna(DB.DB, "runtime_orders", col)
		if err != nil {
			t.Fatalf("tablaTieneColumna runtime_orders.%s: %v", col, err)
		}
		if !ok {
			t.Fatalf("falta columna migrada runtime_orders.%s", col)
		}
	}

	var (
		handleCount int
		orderCount  int
	)
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_handles`).Scan(&handleCount); err != nil {
		t.Fatalf("count runtime_handles: %v", err)
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders`).Scan(&orderCount); err != nil {
		t.Fatalf("count runtime_orders: %v", err)
	}
	if handleCount != 1 || orderCount != 1 {
		t.Fatalf("filas no preservadas: handles=%d orders=%d", handleCount, orderCount)
	}

	handle, err := GetRuntimeHandleActivoAgente("Codex1")
	if err != nil {
		t.Fatalf("GetRuntimeHandleActivoAgente: %v", err)
	}
	if handle == nil || handle.ID != 7 {
		t.Fatalf("handle migrado inesperado: %+v", handle)
	}

	order, err := GetRuntimeOrder(9)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.ID != 9 {
		t.Fatalf("order migrada inesperada: %+v", order)
	}
	if order.AvailableAt.IsZero() {
		t.Fatalf("available_at no migrado: %+v", order)
	}
}
