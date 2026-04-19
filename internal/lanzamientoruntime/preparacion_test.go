package lanzamientoruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/coordinacion"
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
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		return err
	}
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

func TestPrepararDesdeDatosRecuperaWorktreeActivaSiUltimaSesionTraeCWDObsoleto(t *testing.T) {
	prepararDBTemporalLanzamiento(t)
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

	agente, err := db.GetAgente("Codex4")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	proyecto, err := db.GetProyecto("orquesta")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}
	ultima, err := db.ObtenerUltimaSesion("Codex4", &proyectoID)
	if err != nil {
		t.Fatalf("obtener ultima sesion: %v", err)
	}

	prep, err := prepararDesdeDatosConWorkspace(agente, proyecto, conector, ultima, "", "", "", &fakeWorkspaceManager{})
	if err != nil {
		t.Fatalf("preparar desde datos: %v", err)
	}
	if prep == nil || prep.Plan == nil {
		t.Fatalf("preparacion inesperada: %+v", prep)
	}
	if prep.Plan.WorkingDir != rutaWorktree {
		t.Fatalf("working dir deberia recuperar la worktree activa: got=%s want=%s", prep.Plan.WorkingDir, rutaWorktree)
	}
	if prep.Plan.WorkingDir == rutaObsoleta {
		t.Fatalf("working dir no deberia conservar la ruta obsoleta: %s", prep.Plan.WorkingDir)
	}
}

func TestPrepararDesdeDatosCreaWorktreeDesdeRutaProyectoEfectivaAunqueLaUltimaSesionSeaStale(t *testing.T) {
	prepararDBTemporalLanzamiento(t)
	tmp := t.TempDir()
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	rutaStale := filepath.Join(tmp, "tmp-stale", "orquestador")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod actual: %v", err)
	}
	if err := os.MkdirAll(rutaStale, 0o755); err != nil {
		t.Fatalf("mkdir ruta stale: %v", err)
	}

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
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
		Titulo:      "Rehabilitar Codex4",
		Descripcion: "tarea en curso",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex4"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	ultimaID, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex4",
		ProyectoID:  &proyectoID,
		CWD:         rutaStale,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion stale: %v", err)
	}
	if err := db.FinSesion("Codex4"); err != nil {
		t.Fatalf("fin sesion Codex4: %v", err)
	}
	_ = ultimaID

	agente, err := db.GetAgente("Codex4")
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
	ultima, err := db.ObtenerUltimaSesion("Codex4", &proyectoID)
	if err != nil {
		t.Fatalf("obtener ultima sesion: %v", err)
	}

	workspace := &fakeWorkspaceManager{}
	prep, err := prepararDesdeDatosConWorkspace(agente, proyecto, conector, ultima, "", "", "", workspace)
	if err != nil {
		t.Fatalf("preparar desde datos: %v", err)
	}
	if prep == nil || prep.Plan == nil {
		t.Fatalf("preparacion inesperada: %+v", prep)
	}
	if len(workspace.calls) != 1 {
		t.Fatalf("deberia crear una worktree, calls=%d", len(workspace.calls))
	}
	if workspace.calls[0].repoPath != rutaActual {
		t.Fatalf("repoPath inesperado: got=%s want=%s", workspace.calls[0].repoPath, rutaActual)
	}
	if prep.Proyecto == nil || prep.Proyecto.RutaAbs != rutaActual {
		t.Fatalf("proyecto efectivo inesperado: %+v", prep.Proyecto)
	}
	if strings.HasPrefix(prep.Plan.WorkingDir, rutaStale) {
		t.Fatalf("working dir no deberia usar la ruta stale: %s", prep.Plan.WorkingDir)
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
	if prep.Plan.Razonamiento != "high" {
		t.Fatalf("reasoning inesperado: %+v", prep.Plan)
	}
	rendered := runtimeagente.RenderCommand(prep.Plan)
	if strings.Contains(rendered, "'--reasoning-effort'") {
		t.Fatalf("comando no deberia usar flag legacy de razonamiento: %s", rendered)
	}
	if !strings.Contains(rendered, "'-c'") || !strings.Contains(rendered, `'model_reasoning_effort="high"'`) {
		t.Fatalf("comando sin high: %s", rendered)
	}
}

func TestResolverConectorAgenteOllamaUsaDefaultOllama(t *testing.T) {
	prepararDBTemporalLanzamiento(t)

	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "ollama-cli",
		Nombre:     "Ollama CLI",
		Transporte: "cli",
		Comando:    "ollama",
		ArgsJSON:   `["run"]`,
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector ollama: %v", err)
	}

	conector, err := ResolverConector("Ollama1", "", nil)
	if err != nil {
		t.Fatalf("ResolverConector: %v", err)
	}
	if conector.Slug != "ollama-cli" {
		t.Fatalf("conector inesperado: %+v", conector)
	}
}

func TestResolverConectorIgnoraSesionContaminadaIncompatible(t *testing.T) {
	prepararDBTemporalLanzamiento(t)

	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "claude-code",
		Nombre:       "Claude Code",
		Transporte:   "cli",
		Comando:      "claude-code",
		MetadataJSON: `{"familia":"anthropic"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector claude: %v", err)
	}

	conector, err := ResolverConector("Claude1", "", &db.Sesion{
		ConectorSlug: "ollama_pool_local",
		Herramienta:  "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("ResolverConector: %v", err)
	}
	if conector == nil || conector.Slug != "claude-code" {
		t.Fatalf("conector inesperado: %+v", conector)
	}
}

func TestPrepararDesdeDatosOllamaNoPisaPoliticaResueltaConDefaultsDelConector(t *testing.T) {
	prepararDBTemporalLanzamiento(t)
	tmp := t.TempDir()

	if err := db.RegistrarAgente("Ollama1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "ollama-cli",
		Nombre:       "Ollama CLI",
		Transporte:   "cli",
		Comando:      "ollama",
		ArgsJSON:     `["run"]`,
		MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","model_positional":true,"launch_prompt_transport":"post_start"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector ollama: %v", err)
	}

	agente, err := db.GetAgente("Ollama1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	conector, err := db.GetConector("ollama-cli")
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
	if prep.Plan.Modelo != "gpt-5.4" {
		t.Fatalf("modelo inesperado: %+v", prep.Plan)
	}
	if prep.Plan.Razonamiento != "high" {
		t.Fatalf("reasoning inesperado: %+v", prep.Plan)
	}
	rendered := runtimeagente.RenderCommand(prep.Plan)
	if !strings.Contains(rendered, "'run'") || !strings.Contains(rendered, "'gpt-5.4'") {
		t.Fatalf("comando ollama inesperado: %s", rendered)
	}
}

func TestPrepararDesdeDatosOllamaIgnoraResumeNoReanudable(t *testing.T) {
	prepararDBTemporalLanzamiento(t)
	tmp := t.TempDir()
	rutaProyecto := filepath.Join(tmp, "orquestador")
	rutaSesion := filepath.Join(rutaProyecto, ".orquesta-worktrees", "orq-ollama1")
	if err := os.MkdirAll(rutaSesion, 0o755); err != nil {
		t.Fatalf("mkdir ruta sesion: %v", err)
	}

	if err := db.RegistrarAgente("Ollama1", "programador"); err != nil {
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
	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "ollama-cli",
		Nombre:       "Ollama CLI",
		Transporte:   "cli",
		Comando:      "ollama",
		ArgsJSON:     `["run"]`,
		MetadataJSON: `{"reanudable":false,"default_model":"qwen2.5-coder:7b","model_positional":true,"launch_prompt_transport":"post_start"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector ollama: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Ollama1",
		ProyectoID:         &proyectoID,
		CWD:                rutaSesion,
		Herramienta:        "ollama-cli",
		ExternalSessionID:  "sess-ollama-previa",
		ResumenContinuidad: "continuidad vieja que no debe reusarse",
		ResumePayloadJSON:  `{"checkpoint":{"id":77,"kind":"stop"},"mailbox":[{"id":2}]}`,
		Branch:             "orq-orquestador-ollama1",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	agente, err := db.GetAgente("Ollama1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	conector, err := db.GetConector("ollama-cli")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}
	ultima, err := db.ObtenerUltimaSesion("Ollama1", &proyecto.ID)
	if err != nil {
		t.Fatalf("get ultima sesion: %v", err)
	}

	prep, err := prepararDesdeDatosConWorkspace(agente, proyecto, conector, ultima, "", "", "", &fakeWorkspaceManager{})
	if err != nil {
		t.Fatalf("preparar desde datos: %v", err)
	}
	if prep == nil || prep.Plan == nil {
		t.Fatalf("plan nil: %+v", prep)
	}
	if prep.Plan.Modo != "launch" {
		t.Fatalf("ollama no deberia reanudar en preparacion app: %+v", prep.Plan)
	}
	if strings.TrimSpace(prep.Plan.ContinuityPrompt) != "" {
		t.Fatalf("ollama no deberia heredar continuity prompt: %q", prep.Plan.ContinuityPrompt)
	}
	if prep.Plan.WorkingDir != rutaSesion {
		t.Fatalf("working dir inesperado: %s", prep.Plan.WorkingDir)
	}
}
