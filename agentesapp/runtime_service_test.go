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

func strPtr(v string) *string { return &v }
func int64Ptr(v int64) *int64 { return &v }

type stubModelPolicyProvider struct {
	input      *db.ResolverPoliticaInput
	resolution *db.ResolucionModelo
	err        error
	calls      int
}

func (s *stubModelPolicyProvider) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	s.calls++
	copyInput := input
	s.input = &copyInput
	return s.resolution, s.err
}

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

func TestGetModelPolicyResolutionForPrepareUsaCacheCorta(t *testing.T) {
	provider := &stubModelPolicyProvider{
		resolution: &db.ResolucionModelo{
			PerfilTarea:     "implementacion",
			ModelSlug:       "gpt-5.4",
			ReasoningEffort: "high",
		},
	}
	svc := NewService(&fakeStore{}, provider)
	first, err := svc.getModelPolicyResolutionForPrepare("Codex2", "orquestador", "")
	if err != nil {
		t.Fatalf("primer getModelPolicyResolutionForPrepare: %v", err)
	}
	second, err := svc.getModelPolicyResolutionForPrepare("Codex2", "orquestador", "")
	if err != nil {
		t.Fatalf("segundo getModelPolicyResolutionForPrepare: %v", err)
	}
	if first == nil || second == nil || second.ModelSlug != "gpt-5.4" {
		t.Fatalf("resolucion inesperada: first=%+v second=%+v", first, second)
	}
	if provider.calls != 1 {
		t.Fatalf("deberia reutilizar cache de model policy, got=%d", provider.calls)
	}
}

func TestResolveLiveOperationalStateDevuelveFilaDelAgente(t *testing.T) {
	estado, detalle, err := resolveLiveOperationalState(func() ([]Row, error) {
		return []Row{
			{Agente: &db.Agente{Nombre: "Otro"}, EstadoOperativo: "disponible"},
			{Agente: &db.Agente{Nombre: "Codex1"}, EstadoOperativo: "bloqueado_por_cuota", DetalleOperativo: "worker bloqueado por cuota"},
		}, nil
	}, "Codex1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("resolve live state: %v", err)
	}
	if estado != "bloqueado_por_cuota" || detalle != "worker bloqueado por cuota" {
		t.Fatalf("estado/detalle inesperados: %q / %q", estado, detalle)
	}
}

func TestResolveLiveOperationalStateHaceTimeoutSeguro(t *testing.T) {
	estado, detalle, err := resolveLiveOperationalState(func() ([]Row, error) {
		time.Sleep(50 * time.Millisecond)
		return []Row{{Agente: &db.Agente{Nombre: "Codex1"}, EstadoOperativo: "bloqueado_por_cuota"}}, nil
	}, "Codex1", 5*time.Millisecond)
	if err != nil {
		t.Fatalf("resolve live state timeout no deberia fallar: %v", err)
	}
	if estado != "" || detalle != "" {
		t.Fatalf("con timeout deberia degradar a vacio, got estado=%q detalle=%q", estado, detalle)
	}
}

func TestOperationalStateForAgentEvitaPanelCompleto(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents:   []*db.Agente{{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true}},
		sessions: []*db.Sesion{{ID: 9, Agente: "Codex1", Estado: "activa", Inicio: now}},
		tasks:    []*db.Tarea{{ID: 41, Agente: strPtr("Codex1"), Estado: db.EstadoEnProgreso}},
	}

	estado, detalle, err := NewService(store, nil).OperationalStateForAgent("Codex1")
	if err != nil {
		t.Fatalf("OperationalStateForAgent: %v", err)
	}
	if estado != "arrancando" {
		t.Fatalf("estado=%q detalle=%q", estado, detalle)
	}
	if store.listAgentsCalls != 0 {
		t.Fatalf("no deberia listar todos los agentes para resolver uno solo: calls=%d", store.listAgentsCalls)
	}
}

func TestResolveActiveAssignmentFromAssignmentsPrefiereActivaDelProyecto(t *testing.T) {
	asignaciones := []*db.Asignacion{
		{Agente: "Codex1", ProyectoID: 9, ProyectoSlug: "otro", Estado: db.AsignacionActiva},
		{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionPausada},
	}

	asignado, slug := resolveActiveAssignmentFromAssignments(asignaciones, 7)
	if !asignado || slug != "orquestador" {
		t.Fatalf("resolucion inesperada: asignado=%t slug=%q", asignado, slug)
	}

	asignado, slug = resolveActiveAssignmentFromAssignments(asignaciones, 3)
	if asignado || slug != "otro" {
		t.Fatalf("fallback inesperado fuera de proyecto: asignado=%t slug=%q", asignado, slug)
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

func TestBuildPrepareErrorSiProyectoNoExiste(t *testing.T) {
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "CodexPg2", Rol: "programador", Activo: true, Habilitado: true},
		},
	}

	_, err := NewService(store, nil).BuildPrepare(PrepareInput{
		Agente:   "CodexPg2",
		Proyecto: "orquesta",
	})
	if err == nil || !strings.Contains(err.Error(), `proyecto "orquesta" no encontrado`) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestProcessTickErrorSiProyectoNoExiste(t *testing.T) {
	svc := NewService(&fakeStore{}, nil)

	_, err := svc.ProcessTick(TickInput{Agente: "CodexPg2", Proyecto: "orquesta"})
	if err == nil || !strings.Contains(err.Error(), `proyecto "orquesta" no encontrado`) {
		t.Fatalf("error inesperado: %v", err)
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

func TestBuildTickOutputNoCuentaBloqueadasComoTareasActivas(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Claude1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado Claude",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Claude1", "Dependencia externa pendiente de credenciales humanas"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='worker bloqueado por cuota', reanimar_at=datetime('now','+1 hour') WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("actualizar cuota: %v", err)
	}

	svc := NewService(Repository{}, nil)
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	out, err := svc.buildTickOutput("Claude1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if len(out.TareasActivas) != 0 {
		t.Fatalf("las tareas bloqueadas no deberian contarse como activas: %+v", out.TareasActivas)
	}
	if out.AccionRecomendada != "pausar_por_cuota" || !out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
}

func TestBuildTickOutputNoContinuaSiRuntimeBloqueado(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente acotado Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "cerrado", ProcessState: "stopped", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "fallido", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}
	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 0)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "esperar_recuperacion_runtime" || out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if strings.TrimSpace(out.Motivo) == "" {
		t.Fatalf("motivo inesperado: %q", out.Motivo)
	}
}

func TestBuildTickOutputConRuntimeBloqueadoYPendienteDurableSessionResumeContinuaTrabajo(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-10 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	workQueuePath := filepath.Join(tmp, "work-queue.json")
	for path, raw := range map[string]map[string]any{
		manifestPath: {
			"driver":                "tmux_cli_session",
			"transport":             "tmux",
			"tmux_session":          "orq-codex1-runtime-bloqueado",
			"tmux_pane_id":          "%9",
			"mailbox_delivery_mode": "session_resume",
			"external_session_id":   "sess-bloqueada",
		},
		statusPath: {
			"state":               "ready",
			"updated_at":          now.Format(time.RFC3339Nano),
			"alive":               true,
			"ready_at":            readyAt.Format(time.RFC3339Nano),
			"last_progress_at":    now.Add(-45 * time.Second).Format(time.RFC3339Nano),
			"external_session_id": "sess-bloqueada",
		},
		heartbeatPath: {
			"alive":               true,
			"heartbeat_at":        now.Format(time.RFC3339Nano),
			"ready_at":            readyAt.Format(time.RFC3339Nano),
			"external_session_id": "sess-bloqueada",
		},
		workQueuePath: {
			"version":    1,
			"updated_at": now.Format(time.RFC3339Nano),
			"current": map[string]any{
				"mailbox_id":  91,
				"kind":        "autonomia",
				"action":      "continuar_trabajo",
				"task_id":     42,
				"state":       "pending",
				"title":       "continuar frente",
				"recorded_at": now.Format(time.RFC3339Nano),
			},
		},
	} {
		data, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":             tmp,
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"mailbox_delivery_mode": "session_resume",
		"external_session_id":   "sess-bloqueada",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "cerrado", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 0)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("deberia priorizar continuidad durable session_resume: %+v", out)
	}
}

func TestBuildTickOutputConMailboxEnResumePayloadContinuaTrabajo(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "cerrado", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: `{"driver":"tmux_cli_session","mailbox_delivery_mode":"session_resume","external_session_id":"sess-codex1"}`},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}
	sesionActiva := &db.Sesion{
		ID:                11,
		Agente:            "Codex1",
		ProyectoID:        int64Ptr(7),
		Estado:            "activa",
		ExternalSessionID: "sess-codex1",
		ResumePayloadJSON: `{"mailbox":[{"id":91,"kind":"autonomia","payload":{"accion":"continuar_trabajo"}}]}`,
	}

	out, err := svc.buildTickOutput("Codex1", proyecto, sesionActiva, 0)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("deberia reutilizar mailbox del resume payload como continuidad: %+v", out)
	}
}

func TestBuildTickOutputReutilizaTareaActivaDesdeResumePayloadProjectContext(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: `{"driver":"tmux_cli_session","mailbox_delivery_mode":"session_resume","external_session_id":"sess-codex2"}`},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}
	sesionActiva := &db.Sesion{
		ID:                12,
		Agente:            "Codex2",
		ProyectoID:        int64Ptr(7),
		Estado:            "activa",
		ExternalSessionID: "sess-codex2",
		ResumePayloadJSON: `{"mailbox":[{"id":1163,"kind":"autonomia","payload":{"accion":"esperar_o_pedir_tarea"}}],"project_context":{"tareas_activas":[{"id":1,"estado":"asignada","titulo":"Definir briefing funcional de Orquestador"}]}}`,
	}

	out, err := svc.buildTickOutput("Codex2", proyecto, sesionActiva, 0)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("deberia reutilizar tarea activa embebida en resume payload: %+v", out)
	}
	if len(out.TareasActivas) != 1 || out.TareasActivas[0].ID != 1 {
		t.Fatalf("deberia exponer la tarea activa del resume payload: %+v", out.TareasActivas)
	}
}

func TestBuildTickOutputSupervisorAutobootstrapSupervisaProyectoSinTareaActiva(t *testing.T) {
	now := time.Now().UTC()
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}
	store := &fakeStore{
		project: proyecto,
		config: map[string]string{
			"server_autobootstrap_enabled":        "true",
			"server_autobootstrap_project_slug":   "orquestador",
			"server_autobootstrap_supervisor_agent": "Codex1",
		},
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: proyecto.ID, ProyectoSlug: proyecto.Slug, Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
		},
		sessions: []*db.Sesion{
			{ID: 49, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), Estado: "activa", Activa: true, Inicio: now, ResumePayloadJSON: `{"mailbox":[{"kind":"autonomia","payload":{"accion":"supervisar_proyecto"}}]}`},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}

	out, err := NewService(store, nil).buildTickOutput("Codex1", proyecto, store.sessions[0], 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out == nil || out.AccionRecomendada != "supervisar_proyecto" {
		t.Fatalf("deberia supervisar proyecto al ser supervisor operativo, got=%+v", out)
	}
}

func TestBuildTickOutputSupervisaProyectoSiLaContinuidadVivaLoMarca(t *testing.T) {
	now := time.Now().UTC()
	proyecto := &db.Proyecto{ID: 8, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}
	store := &fakeStore{
		project: proyecto,
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: proyecto.ID, ProyectoSlug: proyecto.Slug, Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
		},
		sessions: []*db.Sesion{
			{ID: 50, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), Estado: "activa", Activa: true, Inicio: now, ResumePayloadJSON: `{"mailbox":[{"kind":"autonomia","payload":{"accion":"supervisar_proyecto"}}]}`},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 22, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 32, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), RuntimeID: int64Ptr(22), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 32, Agente: "Codex1", ProyectoID: int64Ptr(proyecto.ID), RuntimeID: int64Ptr(22), Estado: "activo", UpdatedAt: now},
		},
	}

	out, err := NewService(store, nil).buildTickOutput("Codex1", proyecto, store.sessions[0], 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out == nil || out.AccionRecomendada != "supervisar_proyecto" {
		t.Fatalf("deberia supervisar proyecto al venir marcado por continuidad viva, got=%+v", out)
	}
}

func TestBuildTickOutputConRuntimeSanoNoReleeFallbackPorProyecto(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if store.listRuntimesCalls != 0 {
		t.Fatalf("no deberia releer runtimes fuera del contexto compacto: calls=%d", store.listRuntimesCalls)
	}
	if store.listPassiveHandlesCalls != 0 {
		t.Fatalf("no deberia releer handles pasivos con row sano: calls=%d", store.listPassiveHandlesCalls)
	}
}

func TestBuildTickOutputConBloqueadaNoReleeAgenteNiAsignaciones(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado Claude",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Claude1", "Agente Claude1 en estado bloqueado_por_cuota: worker bloqueado por cuota"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Claude1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Claude1", ProyectoID: proyectoID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: tareaID, Agente: strPtr("Claude1"), ProyectoID: int64Ptr(proyectoID), Estado: db.EstadoBloqueada, Titulo: "Frente bloqueado Claude"},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: proyectoID, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}

	out, err := svc.buildTickOutput("Claude1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	if store.listAssignmentsCalls != 1 {
		t.Fatalf("no deberia releer asignaciones para bloqueos: calls=%d", store.listAssignmentsCalls)
	}
	if store.getAgentCalls != 1 {
		t.Fatalf("no deberia releer agente para bloqueos: calls=%d", store.getAgentCalls)
	}
}

func TestBuildTickOutputConTrabajoActivoNoConsultaPropuestas(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now},
		},
		proposals: []*db.Propuesta{
			{ID: 99, Codigo: "OP-99", Titulo: "No deberia consultarse", Estado: db.PropuestaAbierta},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if store.pendingVotesCalls != 0 || store.openProposalsCalls != 0 {
		t.Fatalf("el hot path con trabajo activo no deberia consultar propuestas: pending=%d open=%d", store.pendingVotesCalls, store.openProposalsCalls)
	}
}

func TestBuildTickOutputConWorkerReadySessionResumeSanoContinuaTrabajo(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-25 * time.Minute)
	lastProgressAt := now.Add(-1 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                     "tmux_cli_session",
		"transport":                  "tmux",
		"tmux_session":               "orq-codex1-120000",
		"tmux_pane_id":               "%3",
		"mailbox_delivery_mode":      "session_resume",
		"external_session_id":        "sess-123",
		"worker_external_session_id": "sess-123",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":               "ready",
		"updated_at":          now.Format(time.RFC3339Nano),
		"alive":               true,
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"last_progress_at":    lastProgressAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-123",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":               true,
		"heartbeat_at":        now.Format(time.RFC3339Nano),
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-123",
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":       manifestPath,
		"driver":                     "tmux_cli_session",
		"worker_status_path":         statusPath,
		"worker_heartbeat_path":      heartbeatPath,
		"mailbox_delivery_mode":      "session_resume",
		"external_session_id":        "sess-123",
		"worker_external_session_id": "sess-123",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "esperando_io", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
}

func TestBuildTickOutputConWorkerReadySessionResumeSinProgresoRecienteEsperaRecuperacion(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-25 * time.Minute)
	lastOutputAt := now.Add(-5 * time.Minute)
	lastProgressAt := now.Add(-25 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                     "tmux_cli_session",
		"transport":                  "tmux",
		"tmux_session":               "orq-codex1-120001",
		"tmux_pane_id":               "%4",
		"mailbox_delivery_mode":      "session_resume",
		"external_session_id":        "sess-456",
		"worker_external_session_id": "sess-456",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":               "ready",
		"updated_at":          now.Format(time.RFC3339Nano),
		"alive":               true,
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"last_output_at":      lastOutputAt.Format(time.RFC3339Nano),
		"last_progress_at":    lastProgressAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-456",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":               true,
		"heartbeat_at":        now.Format(time.RFC3339Nano),
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-456",
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":       manifestPath,
		"driver":                     "tmux_cli_session",
		"worker_status_path":         statusPath,
		"worker_heartbeat_path":      heartbeatPath,
		"mailbox_delivery_mode":      "session_resume",
		"external_session_id":        "sess-456",
		"worker_external_session_id": "sess-456",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "esperando_io", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "esperar_recuperacion_runtime" || out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if !strings.Contains(strings.ToLower(out.Motivo), "sin progreso") {
		t.Fatalf("motivo inesperado: %+v", out)
	}
}

func TestBuildTickOutputConWorkQueueDurableMantieneContinuidad(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-25 * time.Minute)
	lastProgressAt := now.Add(-25 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	workQueuePath := filepath.Join(tmp, "work-queue.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-120000",
		"tmux_pane_id":          "%3",
		"mailbox_delivery_mode": "session_resume",
		"external_session_id":   "sess-123",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":               "ready",
		"updated_at":          now.Format(time.RFC3339Nano),
		"alive":               true,
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"last_progress_at":    lastProgressAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-123",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":               true,
		"heartbeat_at":        now.Format(time.RFC3339Nano),
		"ready_at":            readyAt.Format(time.RFC3339Nano),
		"external_session_id": "sess-123",
	})
	workQueueRaw, _ := json.Marshal(map[string]any{
		"version":    1,
		"updated_at": now.Format(time.RFC3339Nano),
		"current": map[string]any{
			"mailbox_id":  77,
			"kind":        "autonomia",
			"action":      "continuar_trabajo",
			"task_id":     42,
			"state":       "pending",
			"title":       "seguir frente",
			"recorded_at": now.Format(time.RFC3339Nano),
		},
	})
	for path, raw := range map[string][]byte{
		manifestPath:  manifestRaw,
		statusPath:    statusRaw,
		heartbeatPath: heartbeatRaw,
		workQueuePath: workQueueRaw,
	} {
		if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":             tmp,
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"mailbox_delivery_mode": "session_resume",
		"external_session_id":   "sess-123",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex1"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex1", ProyectoID: int64Ptr(7), LogicalState: "esperando_io", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex1", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex1", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if store.getTaskForTickCalls != 1 {
		t.Fatalf("deberia leer la tarea viva desde work-queue durable: calls=%d", store.getTaskForTickCalls)
	}
	if store.tickTasksCalls != 0 {
		t.Fatalf("no deberia barrer todas las tareas si la work-queue ya identifica la tarea activa: tickTasksCalls=%d", store.tickTasksCalls)
	}
}

func TestBuildTickOutputConWorkerReadyBootstrapOnlySanoContinuaTrabajo(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-25 * time.Minute)
	lastProgressAt := now.Add(-1 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex3-120000",
		"tmux_pane_id":          "%4",
		"mailbox_delivery_mode": "bootstrap_only",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"updated_at":            now.Format(time.RFC3339Nano),
		"alive":                 true,
		"ready_at":              readyAt.Format(time.RFC3339Nano),
		"last_progress_at":      lastProgressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":                 true,
		"heartbeat_at":          now.Format(time.RFC3339Nano),
		"ready_at":              readyAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"mailbox_delivery_mode": "bootstrap_only",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex3", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex3"), ProyectoID: int64Ptr(7), Estado: db.EstadoAsignada, Titulo: "Frente activo Codex"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "Codex3", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "Codex3", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(52), Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}
	svc := NewService(store, nil)
	proyecto := &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()}

	out, err := svc.buildTickOutput("Codex3", proyecto, nil, 100)
	if err != nil {
		t.Fatalf("buildTickOutput: %v", err)
	}
	if out.AccionRecomendada != "continuar_trabajo" || out.DebePausar {
		t.Fatalf("salida inesperada: %+v", out)
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

func TestBuildPreparePasaAgenteANivelDePoliticaModelo(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaBase := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaBase, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquestador\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: rutaBase,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	provider := &stubModelPolicyProvider{
		resolution: &db.ResolucionModelo{
			PerfilTarea:     "implementacion",
			ModelSlug:       "gemini-2.5-flash-lite",
			ReasoningEffort: "medium",
		},
	}
	out, err := NewService(Repository{}, provider).BuildPrepare(PrepareInput{
		Agente:   "Gemini1",
		Proyecto: "orquestador",
		Perfil:   "implementacion",
	})
	if err != nil {
		t.Fatalf("BuildPrepare: %v", err)
	}
	if provider.input == nil || provider.input.AgenteNombre == nil || *provider.input.AgenteNombre != "Gemini1" {
		t.Fatalf("el resolver de politica deberia recibir el agente: %+v", provider.input)
	}
	if out.Plan == nil || out.Plan.Modelo != "gemini-2.5-flash-lite" {
		t.Fatalf("modelo resuelto inesperado: %+v", out.Plan)
	}
}

func TestBuildPrepareResumeNativoOmiteBootstrapPromptPesado(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base: %v", err)
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-live-resume",
		Branch:            "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	out, err := NewService(Repository{}, nil).BuildPrepare(PrepareInput{
		Agente:   "Codex1",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("BuildPrepare: %v", err)
	}
	if out == nil || out.Plan == nil {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if strings.TrimSpace(out.Plan.ContinuityPrompt) == "" {
		t.Fatalf("deberia preparar continuity prompt de resume: %+v", out.Plan)
	}
	if strings.TrimSpace(out.BootstrapPrompt) != "" {
		t.Fatalf("no deberia construir bootstrap prompt pesado en native resume: %q", out.BootstrapPrompt)
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

func TestProcessTickNoPideIntervencionPorBloqueoAtascadoAutorecuperableConSesionActiva(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Codex3", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         rutaProyecto,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado por atasco autorecuperable",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex3", "Agente Codex3 atascado: sin progreso reciente tras reinicios"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	out, err := NewService(Repository{}, nil).ProcessTick(TickInput{
		Agente:   "Codex3",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	if out.AccionRecomendada == "pedir_intervencion" {
		t.Fatalf("no deberia pedir intervencion por bloqueo autorecuperable de atasco con sesion activa: %+v", out)
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

func TestProcessTickAckBootstrapRuntimeLeaseByEvidenceCompletaHandoff(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	for _, agente := range []string{"Codex0", "Codex3"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
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
	for _, agente := range []string{"Codex0", "Codex3"} {
		if err := db.ActivarAsignacion(agente, proyectoID, "handoff"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex0",
		ProyectoID:  &proyectoID,
		CWD:         rutaProyecto,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	sesionDestino, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         rutaProyecto,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion destino: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesionDestino.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle destino: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesionDestino.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime destino: %+v err=%v", runtimeInst, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Continuidad handoff por tick",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex0"); err != nil {
		t.Fatalf("tomar tarea origen: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex0"); err != nil {
		t.Fatalf("iniciar tarea origen: %v", err)
	}
	handoffID, err := db.CrearHandoffAgenteStale("Codex0", "Codex3", &tareaID, "traspaso", "handoff listo", "")
	if err != nil {
		t.Fatalf("crear handoff stale: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:     "orquesta",
		ToAgente:       "Codex3",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &handoffID,
		Kind:           "handoff",
		PayloadJSON:    `{"texto":"continua el handoff"}`,
	})
	if err != nil {
		t.Fatalf("mailbox handoff: %v", err)
	}
	resultadoHandoff, _ := json.Marshal(map[string]any{
		"ok":          true,
		"bootstrap":   true,
		"lease_state": "waiting_for_evidence",
		"sesion_id":   sesionDestino.ID,
		"mailbox_ids": []int64{msgID},
	})
	if err := db.MarcarRuntimeOrderEstado(handoffID, "ejecutando", string(resultadoHandoff), ""); err != nil {
		t.Fatalf("preparar handoff ejecutando: %v", err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex3-handoff-tick")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	ackAt := time.Now().UTC().Add(2 * time.Second)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex3","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex3-handoff","started_at":"`+ackAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"ready_at":"`+ackAt.Format(time.RFC3339Nano)+`","updated_at":"`+ackAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+ackAt.Format(time.RFC3339Nano)+`","last_output_at":"`+ackAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+ackAt.Format(time.RFC3339Nano)+`","ready_at":"`+ackAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+ackAt.Format(time.RFC3339Nano)+`","last_output_at":"`+ackAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	out, err := NewService(Repository{}, nil).ProcessTick(TickInput{
		Agente:   "Codex3",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	order, err := db.GetRuntimeOrder(handoffID)
	if err != nil || order == nil {
		t.Fatalf("get handoff: %+v err=%v", order, err)
	}
	if order.Estado != "completada" {
		t.Fatalf("handoff deberia quedar completada tras tick con evidencia: %+v", order)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia quedar en progreso tras tick con evidencia: %+v", tarea)
	}
	if !strings.Contains(tarea.Notas, "handoff completado por Codex3") {
		t.Fatalf("notas sin evidencia de handoff completado: %s", tarea.Notas)
	}
}

func TestProcessTickPriorizaContinuidadAccionableSobreBloqueoResidual(t *testing.T) {
	prepararDBTemporalRuntimeService(t)
	tmp := t.TempDir()
	now := time.Now().UTC()

	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	workQueuePath := filepath.Join(tmp, "work-queue.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeJSON(manifestPath, runtimeagente.WorkerManifest{
		Version:             1,
		Agent:               "Codex3",
		Project:             "orquestador",
		Driver:              "tmux_cli_session",
		Transport:           "tmux",
		MailboxDeliveryMode: "session_resume",
		StatusPath:          statusPath,
		HeartbeatPath:       heartbeatPath,
		CreatedAt:           now.Add(-2 * time.Minute).Format(time.RFC3339),
		StartedAt:           now.Add(-2 * time.Minute).Format(time.RFC3339),
		WorkingDir:          tmp,
	})
	writeJSON(statusPath, runtimeagente.WorkerStatus{
		State:               "ready",
		UpdatedAt:           now.Format(time.RFC3339),
		Alive:               true,
		Agent:               "Codex3",
		Project:             "orquestador",
		WorkingDir:          tmp,
		MailboxDeliveryMode: "session_resume",
		ReadyAt:             now.Add(-30 * time.Second).Format(time.RFC3339),
		LastProgressAt:      now.Add(-5 * time.Second).Format(time.RFC3339),
	})
	writeJSON(heartbeatPath, runtimeagente.WorkerHeartbeat{
		Alive:          true,
		HeartbeatAt:    now.Format(time.RFC3339),
		StartedAt:      now.Add(-2 * time.Minute).Format(time.RFC3339),
		Agent:          "Codex3",
		Project:        "orquestador",
		ReadyAt:        now.Add(-30 * time.Second).Format(time.RFC3339),
		LastProgressAt: now.Add(-5 * time.Second).Format(time.RFC3339),
	})

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex3", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime handle: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get runtime handle: handle=%+v err=%v", handle, err)
	}

	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloqueo residual",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}
	if err := db.BloquearTarea(tareaBloqueadaID, "orquesta", "Esperando decision humana externa"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	tareaActivaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente activo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea activa: %v", err)
	}
	if err := db.TomarTarea(tareaActivaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea activa: %v", err)
	}
	if err := db.IniciarTarea(tareaActivaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea activa: %v", err)
	}

	writeJSON(workQueuePath, runtimeagente.WorkerWorkQueue{
		Version:   1,
		UpdatedAt: now.Format(time.RFC3339),
		Current: &runtimeagente.WorkerWorkQueueEntry{
			MailboxID:  1129,
			Kind:       "pipeline_local",
			Action:     "especificar",
			TaskID:     tareaActivaID,
			State:      "pending",
			Title:      "Frente activo",
			RecordedAt: now.Format(time.RFC3339),
		},
	})

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                 "tmux_cli_session",
		"worker_manifest_path":   manifestPath,
		"worker_status_path":     statusPath,
		"worker_heartbeat_path":  heartbeatPath,
		"mailbox_delivery_mode":  "session_resume",
	})
	if err := db.ActualizarMetadataRuntimeHandle(handle.ID, string(metaJSON)); err != nil {
		t.Fatalf("actualizar metadata handle: %v", err)
	}

	out, err := NewService(Repository{}, nil).ProcessTick(TickInput{
		Agente:   "Codex3",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil {
		t.Fatal("tick output nil")
	}
	if out.AccionRecomendada != "continuar_trabajo" {
		t.Fatalf("deberia priorizar continuidad accionable sobre bloqueo residual: %+v", out)
	}
}

func TestProcessTickReutilizaSesionActivaEnBuilderCompacto(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	out, err := NewService(store, nil).ProcessTick(TickInput{
		Agente:   "Codex2",
		Proyecto: "orquestador",
	})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil || out.AccionRecomendada != "continuar_trabajo" {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if store.getActiveCalls != 1 {
		t.Fatalf("deberia reutilizar la sesion activa ya resuelta: calls=%d", store.getActiveCalls)
	}
}

func TestProcessTickCalientaCacheLigeraDeAgenteYProyecto(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)
	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	beforeAgents := store.getAgentCalls
	beforeProjects := store.getProjectCalls
	if _, err := svc.getAgentForPrepare("Codex2"); err != nil {
		t.Fatalf("getAgentForPrepare: %v", err)
	}
	if _, err := svc.getProjectForPrepare("orquestador"); err != nil {
		t.Fatalf("getProjectForPrepare: %v", err)
	}
	if store.getAgentCalls != beforeAgents {
		t.Fatalf("no deberia releer agente tras tick: before=%d after=%d", beforeAgents, store.getAgentCalls)
	}
	if store.getProjectCalls != beforeProjects {
		t.Fatalf("no deberia releer proyecto tras tick: before=%d after=%d", beforeProjects, store.getProjectCalls)
	}
}

func TestProcessTickNoEsperaProjectFlightDePrepare(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)
	svc.prepareProjectFlight["orquestador"] = &prepareProjectFlight{done: make(chan struct{})}

	done := make(chan error, 1)
	go func() {
		_, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ProcessTick: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("ProcessTick no deberia esperar al project flight de prepare")
	}

	if store.getProjectCalls != 1 {
		t.Fatalf("deberia resolver proyecto directamente una vez, calls=%d", store.getProjectCalls)
	}
}

func TestProcessTickCacheaSesionActivaLigera(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now, HeartbeatAt: timePtr(now)},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)

	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick primera: %v", err)
	}
	before := store.getActiveCalls
	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick segunda: %v", err)
	}
	if store.getActiveCalls != before {
		t.Fatalf("no deberia releer sesion activa en segundo tick: before=%d after=%d", before, store.getActiveCalls)
	}
}

func TestProcessTickThrottleaWriteHeartbeatSesionActivaReciente(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now, HeartbeatAt: timePtr(now)},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)

	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if store.saveActiveCalls != 0 {
		t.Fatalf("no deberia persistir heartbeat reciente, calls=%d", store.saveActiveCalls)
	}
}

func TestProcessTickPersisteWriteHeartbeatSesionActivaVieja(t *testing.T) {
	now := time.Now().UTC()
	oldHeartbeat := now.Add(-45 * time.Second)
	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now.Add(-time.Minute), HeartbeatAt: timePtr(oldHeartbeat)},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)

	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if store.saveActiveCalls != 1 {
		t.Fatalf("deberia persistir heartbeat viejo, calls=%d", store.saveActiveCalls)
	}
}

func TestProcessTickOmiteWriteHeartbeatViejaSiBurstDesactivaPersistenciaDB(t *testing.T) {
	now := time.Now().UTC()
	oldHeartbeat := now.Add(-45 * time.Second)
	prev := ShouldPersistTickHeartbeatDB
	ShouldPersistTickHeartbeatDB = func() bool { return false }
	t.Cleanup(func() {
		ShouldPersistTickHeartbeatDB = prev
	})

	store := &fakeStore{
		project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
		},
		sessions: []*db.Sesion{
			{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now.Add(-time.Minute), HeartbeatAt: timePtr(oldHeartbeat)},
		},
		tasks: []*db.Tarea{
			{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
		},
	}
	svc := NewService(store, nil)

	if _, err := svc.ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"}); err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if store.saveActiveCalls != 0 {
		t.Fatalf("no deberia persistir heartbeat viejo durante burst, calls=%d", store.saveActiveCalls)
	}
}

func TestProcessTickUsaFuentesRapidasParaRuntimeYTareas(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStoreWithLiteAgent{
		fakeStore: &fakeStore{
			project: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir()},
			assignments: []*db.Asignacion{
				{Agente: "Codex2", ProyectoID: 7, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
			},
			sessions: []*db.Sesion{
				{ID: 11, Agente: "Codex2", ProyectoID: int64Ptr(7), Estado: "activa", Inicio: now},
			},
			tasks: []*db.Tarea{
				{ID: 42, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoEnProgreso, Titulo: "Frente activo"},
				{ID: 43, Agente: strPtr("Codex2"), ProyectoID: int64Ptr(7), Estado: db.EstadoCompletada, Titulo: "Hecha"},
			},
			runtimes: []*db.RuntimeInstance{
				{ID: 21, Agente: "Codex2", ProyectoID: int64Ptr(7), LogicalState: "activo", ProcessState: "running", UpdatedAt: now},
			},
			canonicalHandles: []*db.RuntimeHandle{
				{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
			},
			handles: []*db.RuntimeHandle{
				{ID: 31, Agente: "Codex2", ProyectoID: int64Ptr(7), RuntimeID: int64Ptr(21), Estado: "activo", UpdatedAt: now},
			},
		},
		liteAgent: &db.Agente{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
	}

	out, err := NewService(store, nil).ProcessTick(TickInput{Agente: "Codex2", Proyecto: "orquestador"})
	if err != nil {
		t.Fatalf("ProcessTick: %v", err)
	}
	if out == nil || out.AccionRecomendada != "continuar_trabajo" {
		t.Fatalf("salida inesperada: %+v", out)
	}
	if store.listRuntimesCalls != 0 {
		t.Fatalf("no deberia listar runtimes completos: calls=%d", store.listRuntimesCalls)
	}
	if store.listTasksCalls != 0 {
		t.Fatalf("no deberia listar todas las tareas: calls=%d", store.listTasksCalls)
	}
	if store.tickRuntimeCalls == 0 || store.tickTasksCalls == 0 {
		t.Fatalf("deberia usar fuentes rapidas: runtime=%d tareas=%d", store.tickRuntimeCalls, store.tickTasksCalls)
	}
	if store.getAgentCalls != 0 || store.liteCalls == 0 {
		t.Fatalf("deberia usar agente ligero: getAgent=%d lite=%d", store.getAgentCalls, store.liteCalls)
	}
}
