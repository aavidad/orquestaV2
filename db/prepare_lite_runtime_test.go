package db

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/coordinacion"
)

func TestPrepareLiteConnectorAndSessionUseReadOnlySQLite(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	conectorID, err := UpsertConector(&Conector{
		Slug:       "ollama-cli",
		Nombre:     "Ollama CLI",
		Transporte: "cli",
		Comando:    "ollama",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("UpsertConector: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "QwenCoder1",
		ConectorID:  &conectorID,
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if sesion == nil || sesion.ID <= 0 {
		t.Fatalf("sesion invalida: %+v", sesion)
	}
	Close()

	conector, err := GetConectorPrepareLite("ollama-cli")
	if err != nil {
		t.Fatalf("GetConectorPrepareLite: %v", err)
	}
	if conector == nil || conector.Slug != "ollama-cli" {
		t.Fatalf("conector inesperado: %+v", conector)
	}

	ultima, err := ObtenerUltimaSesionPrepareLite("QwenCoder1", &proyectoID)
	if err != nil {
		t.Fatalf("ObtenerUltimaSesionPrepareLite: %v", err)
	}
	if ultima == nil || ultima.Agente != "QwenCoder1" {
		t.Fatalf("ultima sesion inesperada: %+v", ultima)
	}
}

func TestPrepareLiteWorktreeAndTaskUseReadOnlySQLite(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoRuta := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(proyectoRuta, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: proyectoRuta,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	agente := "QwenCoder1"
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Afinar prepare lite",
		Descripcion: "Recortar lecturas del pool principal",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, agente); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, agente); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     agente,
		Name:      "wt-qwen",
		Path:      filepath.Join(proyectoRuta, ".worktrees", "qwen"),
		Branch:    "agent/qwen",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("Create worktree: %v", err)
	}
	Close()

	worktrees, err := ListarWorktreesCoordPrepareLite(coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		Agent:     &agente,
		State:     ptrWorktreeState(coordinacion.WorktreeActive),
	})
	if err != nil {
		t.Fatalf("ListarWorktreesCoordPrepareLite: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0] == nil || worktrees[0].Agent != agente {
		t.Fatalf("worktrees inesperadas: %+v", worktrees)
	}

	tarea, err := GetTareaActivaPrepareLite(agente, proyectoID)
	if err != nil {
		t.Fatalf("GetTareaActivaPrepareLite: %v", err)
	}
	if tarea == nil || tarea.ID <= 0 || tarea.Estado != TareaEnProgreso {
		t.Fatalf("tarea inesperada: %+v", tarea)
	}
}

func TestPrepareLiteMailboxAndTaskContextCanonicalizanAliasCodex(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, nombre := range []string{"Codex82", "codex82"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoRuta := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(proyectoRuta, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "preparelite-canon",
		Nombre:  "PrepareLite Canon",
		RutaAbs: proyectoRuta,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex82",
		ProyectoID:  &proyectoID,
		CWD:         proyectoRuta,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Contexto prepare lite canon",
		Descripcion: "debe salir por alias",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex82"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex82"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}
	if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex82",
		ToAgente:    "Codex82",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: "{}",
		Estado:      "pendiente",
	}); err != nil {
		t.Fatalf("EnviarRuntimeMailbox: %v", err)
	}
	agente := "codex82"
	tareas, err := ListarTareasContextPrepareLite(agente, proyectoID, 5)
	if err != nil {
		t.Fatalf("ListarTareasContextPrepareLite: %v", err)
	}
	if len(tareas) != 1 || tareas[0] == nil || tareas[0].Agente == nil || *tareas[0].Agente != "Codex82" {
		t.Fatalf("tareas inesperadas: %+v", tareas)
	}

	mailbox, err := ListarRuntimeMailboxPrepareLite(FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("ListarRuntimeMailboxPrepareLite: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0] == nil || mailbox[0].ToAgente != "Codex82" || mailbox[0].FromAgente != "Codex82" {
		t.Fatalf("mailbox inesperado: %+v", mailbox)
	}
}

func TestCoordinationWorktreeCreateSQLiteFastPersistsWithoutReadback(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	proyectoRuta := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(proyectoRuta, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: proyectoRuta,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	created, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "GemmaSmoke5",
		Name:      "orquestador-gemmasmoke5-t559",
		Path:      filepath.Join(proyectoRuta, ".orquesta-worktrees", "orquestador-gemmasmoke5-t559"),
		Branch:    "orq-orquestador-gemmasmoke5-t559",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	})
	if err != nil {
		t.Fatalf("Create worktree: %v", err)
	}
	if created == nil || created.ID <= 0 {
		t.Fatalf("worktree creada invalida: %+v", created)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("timestamps no inicializados: %+v", created)
	}
	loaded, err := (CoordinationWorktreeSQLRepository{}).GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID created worktree: %v", err)
	}
	if loaded == nil || loaded.Path != created.Path || loaded.Branch != created.Branch {
		t.Fatalf("worktree persistida inesperada: %+v", loaded)
	}
}

func ptrWorktreeState(state coordinacion.WorktreeState) *coordinacion.WorktreeState {
	return &state
}
