package db

import "testing"

func TestUpsertSupervisorPipelineStateYListar(t *testing.T) {
	prepararDBTemporal(t)
	item, err := UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:    "OpenClaw",
		ProyectoSlug:  "orquestador",
		PipelineName:  "autopilot",
		CurrentPhase:  "review",
		Status:        "active",
		ArtifactsJSON: `{"gates":2}`,
		MetadataJSON:  `{"mode":"review"}`,
	})
	if err != nil {
		t.Fatalf("upsert pipeline state: %v", err)
	}
	if item == nil || item.CurrentPhase != "review" {
		t.Fatalf("pipeline state inesperado: %+v", item)
	}
	item, err = UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		PipelineName: "autopilot",
		CurrentPhase: "merge",
		Status:       "blocked",
	})
	if err != nil {
		t.Fatalf("upsert pipeline state 2: %v", err)
	}
	if item == nil || item.CurrentPhase != "merge" || item.Status != "blocked" {
		t.Fatalf("pipeline state 2 inesperado: %+v", item)
	}
	items, err := ListarSupervisorPipelineStates(FiltroSupervisorPipelineStates{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
	})
	if err != nil {
		t.Fatalf("listar pipeline states: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].PipelineName != "autopilot" {
		t.Fatalf("pipeline states inesperados: %+v", items)
	}
}

func TestSupervisorPipelineStateCanonizaRolePhaseYPipelineName(t *testing.T) {
	prepararDBTemporal(t)
	item, err := UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		PipelineName: " Supervisor Loop ",
		CurrentPhase: " Review Queue ",
		Status:       " BLOCKED ",
	})
	if err != nil {
		t.Fatalf("upsert pipeline canonico: %v", err)
	}
	if item == nil {
		t.Fatalf("esperaba item no nil")
	}
	if item.PipelineName != "supervisor-loop" {
		t.Fatalf("pipeline_name no canonico: %+v", item)
	}
	if item.Role != "supervisor" {
		t.Fatalf("role no inferido: %+v", item)
	}
	if item.CurrentPhase != "review_queue" {
		t.Fatalf("current_phase no canonica: %+v", item)
	}
	if item.Status != "blocked" {
		t.Fatalf("status no canonico: %+v", item)
	}
	got, err := GetSupervisorPipelineState("OpenClaw", "orquestador", " supervisor_loop ")
	if err != nil {
		t.Fatalf("get pipeline canonico: %v", err)
	}
	if got == nil || got.ID != item.ID {
		t.Fatalf("get canonico inesperado: %+v", got)
	}
}

func TestListarSupervisorPipelineStatesFiltraPorRoleStatusYPhase(t *testing.T) {
	prepararDBTemporal(t)
	casos := []UpsertSupervisorPipelineStateInput{
		{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			PipelineName: "supervisor-loop",
			CurrentPhase: "coordinar_workers",
			Status:       "active",
		},
		{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			PipelineName: "review-queue",
			CurrentPhase: "review",
			Status:       "blocked",
		},
		{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			PipelineName: "fix-queue",
			CurrentPhase: "repair",
			Status:       "active",
		},
	}
	for _, input := range casos {
		if _, err := UpsertSupervisorPipelineState(input); err != nil {
			t.Fatalf("upsert pipeline %+v: %v", input, err)
		}
	}
	items, err := ListarSupervisorPipelineStates(FiltroSupervisorPipelineStates{
		ProyectoSlug: "orquestador",
		Role:         "review",
		Status:       "blocked",
		CurrentPhase: "review",
	})
	if err != nil {
		t.Fatalf("listar filtrado: %v", err)
	}
	if len(items) != 1 || items[0] == nil {
		t.Fatalf("esperaba un item reviewer blocked, got=%+v", items)
	}
	if items[0].Role != "reviewer" || items[0].PipelineName != "review-queue" {
		t.Fatalf("item filtrado inesperado: %+v", items[0])
	}
	items, err = ListarSupervisorPipelineStates(FiltroSupervisorPipelineStates{
		ProyectoSlug: "orquestador",
	})
	if err != nil {
		t.Fatalf("listar sin filtros extra: %v", err)
	}
	if len(items) != len(casos) {
		t.Fatalf("esperaba listar sin sesgo por role/status, got=%d", len(items))
	}
}

func TestEnsureSupervisorPipelineStatesSchemaMigraRoleEnTablaLegacy(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := DB.Exec(`DROP TABLE IF EXISTS supervisor_pipeline_states`); err != nil {
		t.Fatalf("drop tabla pipeline actual: %v", err)
	}
	if _, err := DB.Exec(`
		CREATE TABLE supervisor_pipeline_states (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			supervisor      TEXT    NOT NULL,
			proyecto_slug   TEXT    NOT NULL DEFAULT '',
			pipeline_name   TEXT    NOT NULL DEFAULT '',
			current_phase   TEXT    NOT NULL DEFAULT '',
			status          TEXT    NOT NULL DEFAULT 'active',
			current_task_id INTEGER,
			current_gate_id INTEGER,
			current_merge_id INTEGER,
			artifacts_json  TEXT    NOT NULL DEFAULT '{}',
			metadata_json   TEXT    NOT NULL DEFAULT '{}',
			started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at    DATETIME,
			UNIQUE(supervisor, proyecto_slug, pipeline_name)
		)`); err != nil {
		t.Fatalf("crear tabla legacy: %v", err)
	}
	if _, err := UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		PipelineName: "review-queue",
		Status:       "active",
	}); err != nil {
		t.Fatalf("upsert tras migracion legacy: %v", err)
	}
	exists, err := ColumnExists("supervisor_pipeline_states", "role")
	if err != nil {
		t.Fatalf("ColumnExists(role): %v", err)
	}
	if !exists {
		t.Fatalf("esperaba columna role tras migracion")
	}
	item, err := GetSupervisorPipelineState("OpenClaw", "orquestador", "review-queue")
	if err != nil {
		t.Fatalf("get tras migracion legacy: %v", err)
	}
	if item == nil || item.Role != "reviewer" {
		t.Fatalf("role legacy inesperado: %+v", item)
	}
}
