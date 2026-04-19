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
	if estado != "trabajando" {
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
	if store.listRuntimesCalls != 1 {
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
