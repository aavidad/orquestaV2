package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"orquesta/coordinacion"
)

func TestCoordinationWorktreeCreateReabreRutaCerrada(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	ruta := filepath.Join(tmp, ".orquesta-worktrees", "orquestador-codex1")
	repo := CoordinationWorktreeSQLRepository{}
	creada, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "orquestador-codex1",
		Path:      ruta,
		Branch:    "orq/orquestador/codex1",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "primera",
	})
	if err != nil {
		t.Fatalf("crear worktree inicial: %v", err)
	}
	cerrada, err := repo.Close(creada.ID, creada.CreatedAt, "cerrada")
	if err != nil {
		t.Fatalf("cerrar worktree inicial: %v", err)
	}
	if cerrada.State != coordinacion.WorktreeClosed {
		t.Fatalf("estado cerrado inesperado: %+v", cerrada)
	}

	reabierta, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "orquestador-codex1",
		Path:      ruta,
		Branch:    "orq/orquestador/codex1",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "reabierta",
	})
	if err != nil {
		t.Fatalf("reabrir worktree: %v", err)
	}
	if reabierta.ID != creada.ID {
		t.Fatalf("deberia reutilizar la misma fila: got=%d want=%d", reabierta.ID, creada.ID)
	}
	if reabierta.State != coordinacion.WorktreeActive || reabierta.BaseRef != "main" || reabierta.Reason != "reabierta" {
		t.Fatalf("worktree reabierta inesperada: %+v", reabierta)
	}
}

func TestCoordinationProjectGetByRefDevuelveNoRowsSiNoExiste(t *testing.T) {
	prepararDBTemporal(t)

	repo := CoordinationProjectSQLRepository{}
	project, err := repo.GetByRef("inexistente")
	if err != sql.ErrNoRows {
		t.Fatalf("se esperaba sql.ErrNoRows, got project=%+v err=%v", project, err)
	}
	if project != nil {
		t.Fatalf("no deberia devolver proyecto: %+v", project)
	}
}
