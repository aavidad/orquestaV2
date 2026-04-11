package agentesapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/runtimeagente"
)

func prepararDBTemporalRuntimeService(t *testing.T) {
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

func TestBuildBootstrapPromptIncluyeGobernanzaYTarea(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex2", Rol: "programador"}
	proyecto := &db.Proyecto{ID: 7, Slug: "demo-app", RutaAbs: "/tmp/demo-app"}
	plan := &runtimeagente.LaunchPlan{
		WorkingDir:   "/tmp/demo-app",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		PerfilTarea:  "implementacion",
	}
	catalogo := &db.GovernanceCatalog{
		TipoAgente: "programador",
		Reglas: []*db.Regla{
			{Categoria: "calidad", Titulo: "Tests coherentes", Descripcion: "Mantén tests rápidos y mantenibles."},
			{Categoria: "sesion", Titulo: "Fuente de verdad", Descripcion: "Usa API/daemon de Orquesta."},
		},
		Skills: []*db.Skill{
			{Nombre: "rg", CuandoUsar: "Buscar rápido en el repo"},
		},
		Workflows: []*db.Workflow{
			{Nombre: "inicio-sesion", Descripcion: "Entrar con contexto correcto", Pasos: `["1. orquesta sesion inicio","2. revisar tarea"]`},
		},
	}
	memoria := []*db.EntidadMemoria{
		{Nombre: "Core_API", Tipo: "api", ValorJSON: `{"version":"v2"}`},
	}
	tareas := []*db.Tarea{
		{ID: 42, Titulo: "Montar app de prueba", Estado: db.TareaAsignada},
	}
	propuestas := []*db.Propuesta{
		{Codigo: "OP-321"},
	}

	prompt := buildBootstrapPrompt(agente, proyecto, plan, catalogo, memoria, tareas, propuestas)
	for _, token := range []string{
		"Bootstrap de Orquesta para Codex2",
		"Proyecto: demo-app",
		"docs/BIBLIA_APP_ORQUESTA.md",
		"no ejecutes sesion inicio de nuevo",
		"Tareas activas: #42 [asignada] Montar app de prueba.",
		"Reglas efectivas:",
		"[calidad] Tests coherentes: Mantén tests rápidos y mantenibles.",
		"Skills relevantes:",
		"rg: Buscar rápido en el repo",
		"Workflows aplicables:",
		"inicio-sesion: Entrar con contexto correcto",
		"Memoria compartida:",
		"Core_API [api]",
		"Propuestas pendientes de voto: OP-321.",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("bootstrap prompt sin %q:\n%s", token, prompt)
		}
	}
}

func TestBuildPrepareRecuperaWorktreeActivaSiUltimaSesionTraeCWDObsoleto(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquesta-codex4")
	rutaObsoleta := "/home/berserk/Trabajo/orquesta/.orquesta-worktrees/orquesta-codex4"
	if err := os.MkdirAll(filepath.Join(rutaWorktree, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaWorktree, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod worktree: %v", err)
	}

	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: rutaBase,
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
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex4", "orquesta-codex4", rutaWorktree, "orq-orquesta-codex4", "HEAD", coordinacion.WorktreeActive, "test",
	); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex4",
		ProyectoID:         &proyectoID,
		CWD:                rutaObsoleta,
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex4",
		ResumenContinuidad: "continuidad valida del mismo proyecto",
		ResumePayloadJSON:  `{"persist":"ok"}`,
		Branch:             "orq-orquesta-codex4",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.FinSesion("Codex4"); err != nil {
		t.Fatalf("fin sesion: %v", err)
	}

	out, err := NewService(Repository{}, nil).BuildPrepare(PrepareInput{
		Agente:   "Codex4",
		Proyecto: "orquesta",
	})
	if err != nil {
		t.Fatalf("BuildPrepare: %v", err)
	}
	if out == nil || out.Plan == nil {
		t.Fatalf("prepare inesperado: %+v", out)
	}
	if out.Plan.WorkingDir != rutaWorktree {
		t.Fatalf("working dir deberia recuperar la worktree activa: got=%s want=%s", out.Plan.WorkingDir, rutaWorktree)
	}
	if out.Plan.WorkingDir == rutaObsoleta {
		t.Fatalf("working dir no deberia conservar la ruta obsoleta: %s", out.Plan.WorkingDir)
	}
}
