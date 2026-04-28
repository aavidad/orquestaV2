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

func TestProcesarHigieneRuntimesAutonomosBatchPurgaRutaWorktreeFallida(t *testing.T) {
	tmp := prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "failed-worktree-hygiene",
		Nombre:  "Failed Worktree Hygiene",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	ruta := filepath.Join(tmp, ".orquesta-worktrees", "wt-fallida")
	if err := os.MkdirAll(ruta, 0o755); err != nil {
		t.Fatalf("mkdir ruta: %v", err)
	}
	repo := CoordinationWorktreeSQLRepository{}
	if _, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "wt-fallida",
		Path:      ruta,
		Branch:    "orq/wt/fallida",
		BaseRef:   "main",
		State:     coordinacion.WorktreeFailed,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia purgar al menos una ruta fallida, got=%d", n)
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Fatalf("la ruta fallida deberia eliminarse: %v", err)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchNoPurgaRutaFueraDeRoot(t *testing.T) {
	tmp := prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "outside-root-worktree-hygiene",
		Nombre:  "Outside Root Worktree Hygiene",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	ruta := filepath.Join(t.TempDir(), "wt-fuera")
	if err := os.MkdirAll(ruta, 0o755); err != nil {
		t.Fatalf("mkdir ruta: %v", err)
	}
	repo := CoordinationWorktreeSQLRepository{}
	if _, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "wt-fuera",
		Path:      ruta,
		Branch:    "orq/wt/fuera",
		BaseRef:   "main",
		State:     coordinacion.WorktreeClosed,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia contar purga fuera de root, got=%d", n)
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("la ruta fuera de root no deberia tocarse: %v", err)
	}
}
