package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListarConectores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO conectores (slug, nombre, transporte, comando, args_json, env_json, metadata_json, activo)
		VALUES (?,?,?,?,?,?,?,?)`,
		"cli-local", "CLI local", "cli", "/bin/orquesta", `["serve"]`, `{"A":"B"}`, `{"x":1}`, true,
	); err != nil {
		t.Fatalf("insert conector: %v", err)
	}

	items, err := ListarConectores()
	if err != nil {
		t.Fatalf("ListarConectores: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 conector, tengo %d", len(items))
	}
	if items[0].Slug != "cli-local" || items[0].Nombre != "CLI local" || items[0].Transporte != "cli" {
		t.Fatalf("conector inesperado: %+v", items[0])
	}
}

func TestListarWorktrees(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	res, err := DB.Exec(`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1)`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo")
	if err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	proyectoID, _ := res.LastInsertId()
	if _, err := DB.Exec(`INSERT INTO tareas (titulo, descripcion, modulo, prioridad, creado_por) VALUES (?,?,?,?,?)`,
		"tarea-base", "", "orquestador", "media", "alberto"); err != nil {
		t.Fatalf("insert tarea: %v", err)
	}
	var tareaID int64 = 1
	if err := DB.QueryRow(`SELECT id FROM tareas WHERE titulo = ?`, "tarea-base").Scan(&tareaID); err != nil {
		t.Fatalf("query tarea: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, tarea_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		proyectoID, tareaID, "codex1", "wt-orquestador", "/tmp/wt-orquestador", "main", "origin/main", "activa", "principal",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}

	items, err := ListarWorktrees("activa", "codex1")
	if err != nil {
		t.Fatalf("ListarWorktrees: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 worktree, tengo %d", len(items))
	}
	if items[0].ProyectoSlug != "orquestador" || items[0].Agente != "codex1" || items[0].Nombre != "wt-orquestador" {
		t.Fatalf("worktree inesperado: %+v", items[0])
	}
}

func TestListarLocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	res, err := DB.Exec(`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1)`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo")
	if err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	proyectoID, _ := res.LastInsertId()
	if _, err := DB.Exec(`
		INSERT INTO locks (proyecto_id, agente, scope_type, scope_key, ruta_abs, branch, motivo, token_lease, estado, expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		proyectoID, "codex1", "project", "orquestador", "/tmp/orquestador", "main", "bloqueo", "lease-1", "activa",
	); err != nil {
		t.Fatalf("insert lock: %v", err)
	}

	items, err := ListarLocks("activa", "codex1")
	if err != nil {
		t.Fatalf("ListarLocks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 lock, tengo %d", len(items))
	}
	if items[0].ScopeKey != "orquestador" || items[0].Agente != "codex1" || items[0].Estado != "activa" {
		t.Fatalf("lock inesperado: %+v", items[0])
	}
}
