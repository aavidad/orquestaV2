package cmd

import (
	"fmt"
	"testing"
	"time"

	"orquesta/db"
)

func TestBuildWorkspaceControlReportAgregaStatusYCockpits(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
	}()

	statusService = stubStatusService{response: apiStatusResponse{
		TareasPorEstado: map[string]int{"en_progreso": 2, "asignada": 1},
		DeudaDispatch:   deudaDispatchResumen{Total: 3, Pendientes: 1, Notificadas: 1, WorkConfirmed: 1},
		Autonomia: autonomiaResumen{
			Supervisando: 1,
			Count:        2,
			ByKind:       map[string]int{"task_reassigned": 1, "handoff_completed": 1},
		},
		WorkersConectados:   2,
		WorkersTrabajando:   2,
		SupervisoresActivos: 1,
	}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "orquestador"},
			{"slug": "infra"},
		}, nil
	}
	now := time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		if slug == "infra" {
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: slug, Nombre: slug},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 2},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		}
		cockpit := &apiProyectoCockpit{
			Proyecto:        &db.Proyecto{Slug: slug, Nombre: slug},
			TareasPorEstado: map[string]int{"en_progreso": 1},
			AutonomyByKind:  map[string]int{"task_reassigned": 1},
			AutonomyEvents:  1,
			AutonomyLastAt:  &now,
			Autonomy: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   now,
				TargetAgent: "Codex1",
			}},
		}
		return cockpit, nil
	}

	report, err := buildWorkspaceControlReport()
	if err != nil {
		t.Fatalf("buildWorkspaceControlReport: %v", err)
	}
	if report.ActiveProjects != 2 || len(report.Projects) != 2 {
		t.Fatalf("projects inesperados: %+v", report)
	}
	if report.TaskCounts["en_progreso"] != 2 || report.DeudaDispatch.Total != 3 {
		t.Fatalf("resumen global inesperado: %+v", report)
	}
	if report.WorkersConectados != 2 || report.WorkersTrabajando != 2 || report.SupervisoresActivos != 1 {
		t.Fatalf("workers/supervisor inesperados: %+v", report)
	}
	if report.AutonomySurface == nil || report.AutonomySurface.Events != 1 || len(report.AutonomySurface.Projects) != 1 {
		t.Fatalf("autonomy surface inesperada: %+v", report.AutonomySurface)
	}
	if len(report.AutonomyHighlights) == 0 {
		t.Fatalf("faltan autonomy highlights: %+v", report)
	}
	if !containsStringWorkspace(report.AutonomyHighlights, "frentes_bloqueantes=1") {
		t.Fatalf("faltan highlights bloqueantes globales: %+v", report.AutonomyHighlights)
	}
	if !containsStringWorkspace(report.AutonomyHighlights, "integracion_bloqueada=15") {
		t.Fatalf("falta highlight global de integracion: %+v", report.AutonomyHighlights)
	}
	if !containsStringWorkspace(report.AutonomyHighlights, "riesgo_top=infra(15)") {
		t.Fatalf("falta highlight global de top risk: %+v", report.AutonomyHighlights)
	}
	if len(report.AutonomyRecent) != 1 || report.AutonomyRecent[0].Project != "orquestador" {
		t.Fatalf("timeline autonomy inesperada: %+v", report.AutonomyRecent)
	}
	if len(report.AutonomyProjects) != 2 || report.AutonomyProjects[0].Project != "infra" {
		t.Fatalf("autonomy projects inesperados: %+v", report.AutonomyProjects)
	}
	if report.AutonomyProjects[0].Blocking != 15 {
		t.Fatalf("bloqueo por proyecto inesperado: %+v", report.AutonomyProjects[0])
	}
	for _, token := range []string{"riesgo=critico", "integracion_bloqueada=15", "bloqueadas=2", "review_gates=1", "runtime_orders=1", "mailbox_rt=1", "propuestas_abiertas=1"} {
		if !containsStringWorkspace(report.AutonomyProjects[0].Highlights, token) {
			t.Fatalf("falta %q en highlights de proyecto: %+v", token, report.AutonomyProjects[0].Highlights)
		}
	}
	if report.Projects[0] == nil || report.Projects[0].Proyecto == nil || report.Projects[0].Proyecto.Slug != "infra" {
		t.Fatalf("orden de proyectos inesperado: %+v", report.Projects)
	}
}

func TestBuildWorkspaceControlReportCuentaProyectosActivosAunqueFalleCockpit(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "orquestador"},
			{"slug": "infra"},
			{"slug": "   "},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		if slug == "infra" {
			return nil, fmt.Errorf("cockpit no disponible")
		}
		return &apiProyectoCockpit{
			Proyecto:        &db.Proyecto{Slug: slug, Nombre: slug},
			TareasPorEstado: map[string]int{"en_progreso": 1},
		}, nil
	}

	report, err := buildWorkspaceControlReport()
	if err != nil {
		t.Fatalf("buildWorkspaceControlReport: %v", err)
	}
	if report.ActiveProjects != 2 {
		t.Fatalf("active projects deberia contar slugs activos aunque falte cockpit, got=%d", report.ActiveProjects)
	}
	if len(report.Projects) != 1 || report.Projects[0].Proyecto == nil || report.Projects[0].Proyecto.Slug != "orquestador" {
		t.Fatalf("cockpits cargados inesperados: %+v", report.Projects)
	}
}

func containsStringWorkspace(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
