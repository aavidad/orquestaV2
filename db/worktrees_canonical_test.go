package db

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/coordinacion"
)

func TestListarWorktreesCoordCanonizaRelativasYOcultaActivasInexistentes(t *testing.T) {
	prepararDBTemporal(t)
	rutaBase := filepath.Join(t.TempDir(), "orquesta")
	rutaCodex11 := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codex11")
	if err := os.MkdirAll(rutaCodex11, 0o755); err != nil {
		t.Fatalf("mkdir worktree codex11: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := RegistrarAgente("Codex11", "programador"); err != nil {
		t.Fatalf("registrar Codex11: %v", err)
	}
	if err := RegistrarAgente("Codex12", "programador"); err != nil {
		t.Fatalf("registrar Codex12: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex11", "orquestador-codex11", ".orquesta-worktrees/orquestador-codex11", "orq-orquestador-codex11", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree codex11: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex12", "orquestador-codex12", ".orquesta-worktrees/orquestador-codex12", "orq-orquestador-codex12", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree codex12: %v", err)
	}

	estado := coordinacion.WorktreeActive
	worktrees, err := ListarWorktreesCoord(coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		State:     &estado,
	})
	if err != nil {
		t.Fatalf("listar worktrees: %v", err)
	}
	if len(worktrees) != 1 {
		t.Fatalf("deberia ocultar la activa inexistente: %+v", worktrees)
	}
	if got := worktrees[0].Path; got != rutaCodex11 {
		t.Fatalf("ruta canonica inesperada: got=%s want=%s", got, rutaCodex11)
	}
}

func TestCoordinationWorktreeGetActiveByPathResuelveLegacyRelativa(t *testing.T) {
	prepararDBTemporal(t)
	rutaBase := filepath.Join(t.TempDir(), "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codex11")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := RegistrarAgente("Codex11", "programador"); err != nil {
		t.Fatalf("registrar Codex11: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex11", "orquestador-codex11", ".orquesta-worktrees/orquestador-codex11", "orq-orquestador-codex11", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}

	worktree, err := CoordinationWorktreeRepository().GetActiveByPath(rutaWorktree)
	if err != nil {
		t.Fatalf("GetActiveByPath: %v", err)
	}
	if worktree == nil {
		t.Fatalf("deberia resolver la worktree legacy por ruta canonica")
	}
	if worktree.Path != rutaWorktree {
		t.Fatalf("ruta canonica inesperada: got=%s want=%s", worktree.Path, rutaWorktree)
	}
}

func TestUltimoRuntimeCheckpointCanonizaCWDLegacyALaWorktreeActiva(t *testing.T) {
	prepararDBTemporal(t)
	rutaBase := filepath.Join(t.TempDir(), "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codex11")
	rutaLegacy := filepath.Join(t.TempDir(), "legacy", "orquesta", ".orquesta-worktrees", "orquestador-codex11", "cmd")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := RegistrarAgente("Codex11", "programador"); err != nil {
		t.Fatalf("registrar Codex11: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex11", "orquestador-codex11", ".orquesta-worktrees/orquestador-codex11", "orq-orquestador-codex11", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_checkpoints (agente, proyecto_id, checkpoint_kind, resumen, branch, cwd, payload_json, resume_strategy, source)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		"Codex11", proyectoID, "pause", "legacy checkpoint", "orq-orquestador-codex11", rutaLegacy, `{}`, "resumen_y_payload", "test:legacy-worktree",
	); err != nil {
		t.Fatalf("insert checkpoint: %v", err)
	}

	checkpoint, err := UltimoRuntimeCheckpoint("Codex11", &proyectoID)
	if err != nil {
		t.Fatalf("UltimoRuntimeCheckpoint: %v", err)
	}
	if checkpoint == nil {
		t.Fatalf("checkpoint nil")
	}
	want := filepath.Join(rutaWorktree, "cmd")
	if checkpoint.CWD != want {
		t.Fatalf("cwd canonico inesperado: got=%s want=%s", checkpoint.CWD, want)
	}
}

func TestUltimoRuntimeCheckpointVuelveARaizSiLaWorktreeLegacyYaNoExiste(t *testing.T) {
	prepararDBTemporal(t)
	rutaBase := filepath.Join(t.TempDir(), "orquesta")
	rutaLegacy := filepath.Join(t.TempDir(), "legacy", "orquesta", ".orquesta-worktrees", "orquestador-codex11")
	if err := os.MkdirAll(rutaBase, 0o755); err != nil {
		t.Fatalf("mkdir ruta base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := RegistrarAgente("Codex11", "programador"); err != nil {
		t.Fatalf("registrar Codex11: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex11", "orquestador-codex11", ".orquesta-worktrees/orquestador-codex11", "orq-orquestador-codex11", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_checkpoints (agente, proyecto_id, checkpoint_kind, resumen, branch, cwd, payload_json, resume_strategy, source)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		"Codex11", proyectoID, "pause", "legacy checkpoint", "orq-orquestador-codex11", rutaLegacy, `{}`, "resumen_y_payload", "test:legacy-root",
	); err != nil {
		t.Fatalf("insert checkpoint: %v", err)
	}

	checkpoint, err := UltimoRuntimeCheckpoint("Codex11", &proyectoID)
	if err != nil {
		t.Fatalf("UltimoRuntimeCheckpoint: %v", err)
	}
	if checkpoint == nil {
		t.Fatalf("checkpoint nil")
	}
	if checkpoint.CWD != rutaBase {
		t.Fatalf("sin worktree usable deberia volver a la raiz del proyecto: got=%s want=%s", checkpoint.CWD, rutaBase)
	}
}
