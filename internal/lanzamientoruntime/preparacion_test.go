package lanzamientoruntime

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/runtimeagente"
)

type fakeWorkspaceManager struct {
	calls []struct {
		repoPath     string
		worktreePath string
		branch       string
		baseRef      string
	}
}

func (f *fakeWorkspaceManager) CreateWorktree(repoPath, worktreePath, branch, baseRef string) error {
	f.calls = append(f.calls, struct {
		repoPath     string
		worktreePath string
		branch       string
		baseRef      string
	}{
		repoPath:     repoPath,
		worktreePath: worktreePath,
		branch:       branch,
		baseRef:      baseRef,
	})
	return nil
}

func (f *fakeWorkspaceManager) RemoveWorktree(repoPath, worktreePath string) error {
	return nil
}

func prepararDBTemporalLanzamiento(t *testing.T) {
	t.Helper()
	db.Close()
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", filepath.Join(t.TempDir(), "orquesta.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("db open: %v", err)
	}
	t.Cleanup(db.Close)
}

func TestPrepararDesdeDatosAseguraWorktreeActiva(t *testing.T) {
	prepararDBTemporalLanzamiento(t)
	tmp := t.TempDir()

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Implementar worktree automatica",
		Descripcion: "tarea en curso",
		ProyectoID:  &proyectoID,
		Modulo:      "coordinacion",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}

	workspace := &fakeWorkspaceManager{}
	prep, err := prepararDesdeDatosConWorkspace(agente, proyecto, conector, nil, "", "", "", workspace)
	if err != nil {
		t.Fatalf("preparar desde datos: %v", err)
	}
	if len(workspace.calls) != 1 {
		t.Fatalf("deberia crear una worktree, calls=%d", len(workspace.calls))
	}
	if prep == nil || prep.Plan == nil {
		t.Fatalf("preparacion inesperada: %+v", prep)
	}
	if prep.Plan.WorkingDir == "" || prep.Plan.WorkingDir == proyecto.RutaAbs {
		t.Fatalf("working dir deberia usar la worktree activa: %s", prep.Plan.WorkingDir)
	}
	if got := filepath.Base(prep.Plan.WorkingDir); got == "" || got == filepath.Base(proyecto.RutaAbs) {
		t.Fatalf("working dir sin nombre de worktree: %s", prep.Plan.WorkingDir)
	}
	if prep.Plan.WorkingDir != workspace.calls[0].worktreePath {
		t.Fatalf("working dir distinto a la worktree creada: got=%s want=%s", prep.Plan.WorkingDir, workspace.calls[0].worktreePath)
	}
}

func TestPrepararDesdeDatosResuelveXHighPorDefectoParaImplementacion(t *testing.T) {
	prepararDBTemporalLanzamiento(t)

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "codex-cli",
		Nombre:       "Codex CLI",
		Transporte:   "cli",
		Comando:      "codex",
		MetadataJSON: `{"model_flag":"--model","reasoning_flag":"--reasoning-effort"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	_ = proyectoID
	agente, err := db.GetAgente("Codex2")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}

	prep, err := prepararDesdeDatosConWorkspace(agente, proyecto, conector, nil, "", "", "", &fakeWorkspaceManager{})
	if err != nil {
		t.Fatalf("preparar desde datos: %v", err)
	}
	if prep == nil || prep.Plan == nil {
		t.Fatalf("plan nil: %+v", prep)
	}
	if prep.Plan.PerfilTarea != "implementacion" {
		t.Fatalf("perfil inesperado: %+v", prep.Plan)
	}
	if prep.Plan.Modelo != "gpt-5.4" {
		t.Fatalf("modelo inesperado: %+v", prep.Plan)
	}
	if prep.Plan.Razonamiento != "xhigh" {
		t.Fatalf("reasoning inesperado: %+v", prep.Plan)
	}
	rendered := runtimeagente.RenderCommand(prep.Plan)
	if !strings.Contains(rendered, "'--reasoning-effort'") || !strings.Contains(rendered, "'xhigh'") {
		t.Fatalf("comando sin xhigh: %s", rendered)
	}
}
