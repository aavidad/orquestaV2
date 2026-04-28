package db

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/coordinacion"
)

func TestProcesarHigieneRuntimesAutonomosBatchPurgaRutaWorktreeCerrada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "closed-worktree-hygiene",
		Nombre:  "Closed Worktree Hygiene",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	ruta := filepath.Join(tmp, ".orquesta-worktrees", "wt-cerrada")
	if err := os.MkdirAll(ruta, 0o755); err != nil {
		t.Fatalf("mkdir ruta: %v", err)
	}
	repo := CoordinationWorktreeSQLRepository{}
	created, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "wt-cerrada",
		Path:      ruta,
		Branch:    "orq/wt/cerrada",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	})
	if err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	if _, err := repo.Close(created.ID, created.CreatedAt, "cerrada_test"); err != nil {
		t.Fatalf("cerrar worktree: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia purgar al menos una ruta cerrada, got=%d", n)
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Fatalf("la ruta cerrada deberia eliminarse: %v", err)
	}
}
