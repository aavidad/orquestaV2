package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListarAsignacionesYSesionesActivas(t *testing.T) {
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

	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`,
		"codex1", proyectoID, "activa", "principal"); err != nil {
		t.Fatalf("insert asignacion: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa, proyecto_id, estado, herramienta, host, branch) VALUES (?,?,?,?,?,?,?)`,
		"codex1", 1, proyectoID, "activa", "codex", "localhost", "main"); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}

	asignaciones, err := ListarAsignacionesOpsView("activa", "codex1")
	if err != nil {
		t.Fatalf("ListarAsignaciones: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].ProyectoSlug != "orquestador" {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}

	sesiones, err := ListarSesionesActivasOpsView()
	if err != nil {
		t.Fatalf("ListarSesionesActivas: %v", err)
	}
	if len(sesiones) != 1 || sesiones[0].ProyectoSlug != "orquestador" {
		t.Fatalf("sesiones inesperadas: %+v", sesiones)
	}
}

func TestListarSesionesActivasOpsViewOcultaSesionZombi(t *testing.T) {
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

	old := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := DB.Exec(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)`,
		"codex-zombie", 1, "activa", "codex", "localhost", old,
	); err != nil {
		t.Fatalf("insert sesion zombi: %v", err)
	}

	sesiones, err := ListarSesionesActivasOpsView()
	if err != nil {
		t.Fatalf("ListarSesionesActivasOpsView: %v", err)
	}
	if len(sesiones) != 0 {
		t.Fatalf("no deberia devolver sesiones zombi: %+v", sesiones)
	}
}
