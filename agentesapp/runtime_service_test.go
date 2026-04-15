package agentesapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/runtimeagente"
)

func strPtr(v string) *string   { return &v }
func int64Ptr(v int64) *int64   { return &v }

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

func TestBuildTickOutputPausaPorCuotaSiWorkerTMUXBloqueado(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"agent":        "Claude1",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-claude1-1",
		"tmux_pane_id": "%9",
	})
	writeJSON(statusPath, map[string]any{
		"state":      "blocked_quota",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Claude1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Claude1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Claude1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente acotado Claude"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Claude1", ProyectoID: int64Ptr(7), LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Claude1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}
	out, err := svc.buildTickOutput("Claude1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "pausar_por_cuota" || !out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if !strings.Contains(strings.ToLower(out.Motivo), "cuota") {
		t.Fatalf("motivo inesperado: %q", out.Motivo)
	}
}

func TestBuildPreparePrefierePoolLocalCompartidoOllama(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:           "ollama-gemma4",
		Proveedor:      "Ollama",
		Runtime:        "ollama",
		Plan:           "local",
		CapacidadTotal: 1,
		MetadataJSON:   `{"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","slots_maximos":1}`,
		Activo:         true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("ollama-gemma4", &db.PoolModelo{
		ModelSlug:     "gemma4:26b",
		Activo:        true,
		Prioridad:     10,
		CosteRelativo: 1,
	}); err != nil {
		t.Fatalf("guardar pool modelo: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica: %v", err)
	}

	out, err := NewService(Repository{}, capacidadapp.Repository{}).BuildPrepare(PrepareInput{
		Agente:   "Gemma1",
		Proyecto: "orquestador",
		Perfil:   "implementacion",
	})
	if err != nil {
		t.Fatalf("BuildPrepare: %v", err)
	}
	if out.Conector.Slug != "ollama_pool_local" {
		t.Fatalf("conector inesperado: %+v", out.Conector)
	}
	if out.Plan == nil || out.Plan.Transporte != "api" {
		t.Fatalf("plan inesperado: %+v", out.Plan)
	}
	if out.Proyecto.ID != proyectoID {
		t.Fatalf("proyecto inesperado: %+v", out.Proyecto)
	}
}

func TestBuildPreparePremiumIgnoraModeloLocalIncompatible(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexPremium", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "codex-cli",
		Nombre:       "Codex CLI",
		Transporte:   "cli",
		Comando:      "codex",
		MetadataJSON: `{"familia":"openai","model_flag":"--model"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		ModelSlug:       "qwen2.5-coder:14b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica: %v", err)
	}

	out, err := NewService(Repository{}, nil).BuildPrepare(PrepareInput{
		Agente:   "CodexPremium",
		Proyecto: "orquestador",
		Conector: "codex-cli",
		Perfil:   "implementacion",
	})
	if err != nil {
		t.Fatalf("BuildPrepare: %v", err)
	}
	if out == nil || out.Plan == nil {
		t.Fatalf("prepare inesperado: %+v", out)
	}
	if out.Plan.Modelo == "qwen2.5-coder:14b" {
		t.Fatalf("el premium no deberia heredar modelo local incompatible: %+v", out.Plan)
	}
	if rendered := runtimeagente.RenderCommand(out.Plan); strings.Contains(rendered, "qwen2.5-coder:14b") {
		t.Fatalf("rendered command no deberia incluir modelo local incompatible: %s", rendered)
	}
	if out.Plan.Modelo != "gpt-5.4" {
		t.Fatalf("deberia caer al modelo premium compatible por defecto: %+v", out.Plan)
	}
}

func TestProcessTickNoPideIntervencionPorBloqueoAutorecuperableConSesionActiva(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         rutaProyecto,
		Herramienta: "gemini-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado autorecuperable",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: tmux pane finalizado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	out, err := NewService(Repository{}, nil).ProcessTick(TickInput{
		Agente:   "Gemini1",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	if out.AccionRecomendada == "pedir_intervencion" {
		t.Fatalf("no deberia pedir intervencion por bloqueo autorecuperable con sesion activa: %+v", out)
	}
}

func TestProcessTickNoPideIntervencionPorBloqueoSinRelevoSanoConAsignacionActiva(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar Gemini1: %v", err)
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado sin relevo sano",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "orquesta", "Bloqueada automáticamente por degradación operativa sin relevo sano"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	out, err := NewService(Repository{}, nil).ProcessTick(TickInput{
		Agente:   "Gemini1",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	if out.AccionRecomendada == "pedir_intervencion" {
		t.Fatalf("no deberia pedir intervencion por bloqueo premium recuperable sin relevo sano: %+v", out)
	}
}
