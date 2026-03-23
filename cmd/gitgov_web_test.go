package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestAPIGitWorktreesYLocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO tareas (titulo, descripcion, modulo, prioridad, creado_por) VALUES (?,?,?,?,?)`,
		"git-base", "", "orquestador", "media", "alberto"); err != nil {
		t.Fatalf("insert tarea: %v", err)
	}
	var tareaID int64
	if err := db.DB.QueryRow(`SELECT id FROM tareas WHERE titulo = ?`, "git-base").Scan(&tareaID); err != nil {
		t.Fatalf("query tarea: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, tarea_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		proyectoID, tareaID, "codex2", "wt-git", "/tmp/wt-git", "feature/panel", "origin/main", "activa", "panel",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO locks (proyecto_id, agente, scope_type, scope_key, ruta_abs, branch, motivo, token_lease, estado, expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		proyectoID, "codex2", "worktree", "wt-git", "/tmp/wt-git", "feature/panel", "proteccion", "lease-1", "activa",
	); err != nil {
		t.Fatalf("insert lock: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/git/worktrees?estado=activa&agente=codex2", nil)
	rec := httptest.NewRecorder()
	webHandlerAPIWorktrees(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api/worktrees status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var worktrees struct {
		Items []db.Worktree `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &worktrees); err != nil {
		t.Fatalf("json worktrees: %v", err)
	}
	if len(worktrees.Items) != 1 || worktrees.Items[0].Nombre != "wt-git" {
		t.Fatalf("worktrees inesperadas: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/git/locks?estado=activa&agente=codex2", nil)
	rec = httptest.NewRecorder()
	webHandlerAPILocks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api/locks status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var locks struct {
		Items []db.Lock `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &locks); err != nil {
		t.Fatalf("json locks: %v", err)
	}
	if len(locks.Items) != 1 || locks.Items[0].ScopeKey != "wt-git" {
		t.Fatalf("locks inesperados: %s", rec.Body.String())
	}
}

func TestWebGitPanelRenderizaWorktreesYLocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO tareas (titulo, descripcion, modulo, prioridad, creado_por) VALUES (?,?,?,?,?)`,
		"git-panel", "", "orquestador", "media", "alberto"); err != nil {
		t.Fatalf("insert tarea: %v", err)
	}
	var tareaID int64
	if err := db.DB.QueryRow(`SELECT id FROM tareas WHERE titulo = ?`, "git-panel").Scan(&tareaID); err != nil {
		t.Fatalf("query tarea: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, tarea_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		proyectoID, tareaID, "codex2", "wt-panel", "/tmp/wt-panel", "feature/git-panel", "origin/main", "activa", "panel",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/git?estado=activa&agente=codex2", nil)
	rec := httptest.NewRecorder()
	webHandlerGitGov(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("git panel status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Gobierno Git") || !strings.Contains(body, "wt-panel") {
		t.Fatalf("panel incompleto: %s", body)
	}
}

func TestAPIMergesCrearYListar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1)`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo"); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	body := bytes.NewBufferString(`{"proyecto_slug":"orquestador","source_branch":"feat-a","target_branch":"main","requested_by":"codex2","estado":"pendiente","notas":"merge de prueba"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/git/merges", body)
	rec := httptest.NewRecorder()
	webHandlerAPIMerges(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/git/merges?proyecto=orquestador&estado=pendiente", nil)
	rec = httptest.NewRecorder()
	webHandlerAPIMerges(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado listando: %d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []db.GitMerge `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].SourceBranch != "feat-a" {
		t.Fatalf("merges inesperados: %s", rec.Body.String())
	}
}
