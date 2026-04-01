package cmd

import (
	"testing"

	"orquesta/db"
)

func TestBuildSupervisorPipelineSnapshotDerivaLoopReview(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		gateID, err := db.CrearReviewGate(&db.ReviewGate{
			ProyectoID:     &proyectoID,
			RequestedBy:    "OpenClaw",
			ReviewerAgente: "Codex4",
			Estado:         db.ReviewGateEnRevision,
			SeverityMax:    "high",
		})
		if err != nil {
			t.Fatalf("crear review gate: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{}}

		snapshot, err := buildSupervisorPipelineSnapshot("OpenClaw", "orquestador", 20)
		if err != nil {
			t.Fatalf("buildSupervisorPipelineSnapshot: %v", err)
		}
		pipelines, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
		loop := findSupervisorPipelineByName(pipelines, "supervisor-loop")
		if loop == nil {
			t.Fatalf("falta supervisor-loop: %#v", pipelines)
		}
		if loop.CurrentPhase != "review" || loop.Status != "active" {
			t.Fatalf("pipeline review inesperada: %+v", loop)
		}
		if loop.CurrentGateID == nil || *loop.CurrentGateID != gateID {
			t.Fatalf("gate derivado inesperado: %+v", loop)
		}
	})
}

func TestBuildSupervisorPipelineSnapshotDerivaBlockedByQuota(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			Agentes: []*db.Agente{
				{
					Nombre:                "Codex2",
					Rol:                   "programador",
					Activo:                false,
					EstadoCuota:           "enfriamiento",
					PresupuestoEstado:     "agotado_weekly",
					PresupuestoSemanalPct: intPtr(0),
				},
			},
			TareasActivas: []tareaLite{
				{ID: 412, Estado: db.TareaAsignada, Titulo: "Runtime operator", Agente: "Codex2"},
			},
		}}

		snapshot, err := buildSupervisorPipelineSnapshot("OpenClaw", "orquestador", 20)
		if err != nil {
			t.Fatalf("buildSupervisorPipelineSnapshot: %v", err)
		}
		pipelines, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
		loop := findSupervisorPipelineByName(pipelines, "supervisor-loop")
		if loop == nil {
			t.Fatalf("falta supervisor-loop: %#v", pipelines)
		}
		if loop.CurrentPhase != "blocked_by_quota" || loop.Status != "blocked" {
			t.Fatalf("pipeline quota inesperada: %+v", loop)
		}
		if loop.CurrentTaskID == nil || *loop.CurrentTaskID != 412 {
			t.Fatalf("tarea retenida inesperada: %+v", loop)
		}
	})
}

func TestBuildSupervisorPipelineSnapshotDerivaCoordinarWorkers(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Concurrency hardening", Agente: "Codex3"},
			},
		}}

		snapshot, err := buildSupervisorPipelineSnapshot("OpenClaw", "orquestador", 20)
		if err != nil {
			t.Fatalf("buildSupervisorPipelineSnapshot: %v", err)
		}
		pipelines, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
		loop := findSupervisorPipelineByName(pipelines, "supervisor-loop")
		if loop == nil {
			t.Fatalf("falta supervisor-loop: %#v", pipelines)
		}
		if loop.CurrentPhase != "coordinar_workers" || loop.Status != "active" {
			t.Fatalf("pipeline workers inesperada: %+v", loop)
		}
		if loop.CurrentTaskID == nil || *loop.CurrentTaskID != 410 {
			t.Fatalf("tarea activa inesperada: %+v", loop)
		}
	})
}

func findSupervisorPipelineByName(items []*db.SupervisorPipelineState, pipelineName string) *db.SupervisorPipelineState {
	for _, item := range items {
		if item != nil && item.PipelineName == pipelineName {
			return item
		}
	}
	return nil
}
