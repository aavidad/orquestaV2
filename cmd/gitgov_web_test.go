package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebGitPanelRenderizaWorktreesYLocks(t *testing.T) {
	prepararDBTemporalCmd(t)

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

	req := httptest.NewRequest(http.MethodGet, "/git?estado=activa&agente=codex2&lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerGitGov(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("git panel status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Git Governance") || !strings.Contains(body, "Worktrees") || !strings.Contains(body, "wt-panel") {
		t.Fatalf("panel incompleto: %s", body)
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}
}

func TestWebGitDetalleRenderizaWorktreeYLock(t *testing.T) {
	prepararDBTemporalCmd(t)

	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO tareas (titulo, descripcion, modulo, prioridad, creado_por) VALUES (?,?,?,?,?)`,
		"git-detalle", "", "orquestador", "media", "alberto"); err != nil {
		t.Fatalf("insert tarea: %v", err)
	}
	var tareaID int64
	if err := db.DB.QueryRow(`SELECT id FROM tareas WHERE titulo = ?`, "git-detalle").Scan(&tareaID); err != nil {
		t.Fatalf("query tarea: %v", err)
	}
	res, err := db.DB.Exec(`
		INSERT INTO locks (proyecto_id, tarea_id, agente, scope_type, scope_key, ruta_abs, branch, motivo, token_lease, estado, expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		proyectoID, tareaID, "codex2", "worktree", "wt-detalle", "/tmp/wt-detalle", "feature/detalle", "proteccion", "lease-1", "activa",
	)
	if err != nil {
		t.Fatalf("insert lock: %v", err)
	}
	lockID, _ := res.LastInsertId()
	res, err = db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, tarea_id, lock_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		proyectoID, tareaID, lockID, "codex2", "wt-detalle", "/tmp/wt-detalle", "feature/detalle", "origin/main", "activa", "detalle",
	)
	if err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
	worktreeID, _ := res.LastInsertId()

	req := httptest.NewRequest(http.MethodGet, "/git/worktrees/"+itoa(worktreeID)+"?lang=en", nil)
	rec := httptest.NewRecorder()
	webRouterGitGov(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle worktree status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Worktree #") || !strings.Contains(body, "wt-detalle") || !strings.Contains(body, "feature/detalle") {
		t.Fatalf("detalle worktree incompleto: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/git/locks/"+itoa(lockID)+"?lang=en", nil)
	rec = httptest.NewRecorder()
	webRouterGitGov(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle lock status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Lock #") || !strings.Contains(body, "lease-1") || !strings.Contains(body, "wt-detalle") {
		t.Fatalf("detalle lock incompleto: %s", body)
	}
}
