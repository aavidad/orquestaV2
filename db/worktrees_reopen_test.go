package db

import (
	"database/sql"
	"os"
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

func TestCoordinationProjectGetByRefPrepareLiteForAgentPriorizaSesionDelAgente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoReal := filepath.Clean(filepath.Join(wd, ".."))
	if _, err := os.Stat(filepath.Join(repoReal, "go.mod")); err != nil {
		t.Fatalf("repo real sin go.mod en %s: %v", repoReal, err)
	}

	rutaStale := filepath.Join(tmp, "tmp-stale", "orquestador")
	if err := os.MkdirAll(rutaStale, 0o755); err != nil {
		t.Fatalf("mkdir stale: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaStale, "go.mod"), []byte("module stale\n"), 0o644); err != nil {
		t.Fatalf("write go.mod stale: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaStale,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:     "QwenCoder1",
		ProyectoID: &proyectoID,
		CWD:        repoReal,
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	repo := CoordinationProjectSQLRepository{}
	project, err := repo.GetByRefPrepareLiteForAgent("orquestador", "QwenCoder1")
	if err != nil {
		t.Fatalf("GetByRefPrepareLiteForAgent: %v", err)
	}
	if project == nil {
		t.Fatalf("GetByRefPrepareLiteForAgent devolvio nil")
	}
	if project.RootPath != repoReal {
		t.Fatalf("root inesperado: got %q want %q", project.RootPath, repoReal)
	}
}

func TestCoordinationWorktreeCreateCanonicalizaAliasCodexYListRawLoEncuentra(t *testing.T) {
	tmp := prepararDBTemporal(t)

	for _, nombre := range []string{"Codex31", "codex31"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "worktree-canon",
		Nombre:  "Worktree Canon",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	repo := CoordinationWorktreeSQLRepository{}
	creada, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "codex31",
		Name:      "wt-canon-codex31",
		Path:      filepath.Join(tmp, ".orquesta-worktrees", "wt-canon-codex31"),
		Branch:    "orq/worktree/canon",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "canon test",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if creada.Agent != "Codex31" {
		t.Fatalf("agente inesperado al crear worktree: %+v", creada)
	}

	agente := "codex31"
	items, err := repo.ListRaw(coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		Agent:     &agente,
		State:     ptrWorktreeState(coordinacion.WorktreeActive),
	})
	if err != nil {
		t.Fatalf("ListRaw: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].Agent != "Codex31" {
		t.Fatalf("worktrees inesperadas: %+v", items)
	}
}
