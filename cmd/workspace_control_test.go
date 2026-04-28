package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestBuildWorkspaceControlReportAgregaStatusYCockpits(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
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
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		ts := time.Date(2026, 4, 24, 13, 30, 0, 0, time.UTC)
		if slug == "infra" {
			return &projectControlReport{
				Project:   &db.Proyecto{Slug: slug, Nombre: slug},
				Since:     since,
				Generated: ts,
				Progress: projectControlProgress{
					State:          "bloqueado",
					StateReason:    "bloqueos abiertos sin ejecución activa",
					AttentionScore: 6,
					AttentionLabel: "alto",
				},
				Agents: []projectControlAgentRow{{
					Name:             "Codex9",
					OperationalState: "bloqueado",
					OpenTasks:        2,
					BlockedTasks:     2,
					MailboxPending:   1,
				}},
				Autonomy: []autonomyEventSummary{{
					Kind:        "worker_recovery_requested",
					CreatedAt:   ts.Add(30 * time.Minute),
					TargetAgent: "Codex9",
				}},
				Git: projectControlGitAggregate{
					TouchedFiles:          []string{"cmd/api.go", "cmd/workspace_control.go"},
					PendingAddedLines:     4,
					PendingDeletedLines:   1,
					CommittedAddedLines:   3,
					CommittedDeletedLines: 2,
				},
			}, nil
		}
		return &projectControlReport{
			Project:   &db.Proyecto{Slug: slug, Nombre: slug},
			Since:     since,
			Generated: ts,
			Progress: projectControlProgress{
				State:          "activo",
				StateReason:    "hay trabajo en progreso",
				AttentionScore: 2,
				AttentionLabel: "bajo",
			},
			Agents: []projectControlAgentRow{{
				Name:             "Codex1",
				OperationalState: "trabajando",
				OpenTasks:        1,
			}},
			Autonomy: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   ts,
				TargetAgent: "Codex1",
			}},
			Git: projectControlGitAggregate{
				TouchedFiles:          []string{"cmd/workspace_control.go"},
				PendingAddedLines:     2,
				CommittedAddedLines:   1,
				CommittedDeletedLines: 1,
			},
		}, nil
	}

	report, err := buildWorkspaceControlReportSince(time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC))
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
	if !report.Since.Equal(time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("ventana inesperada: %s", report.Since)
	}
	if report.Operational.State != "bloqueado" || report.Operational.AttentionScore != 6 || report.Operational.BlockingProjects != 1 {
		t.Fatalf("estado operativo global inesperado: %+v", report.Operational)
	}
	if report.Git.FilesChanged != 2 || report.Git.Insertions != 10 || report.Git.Deletions != 4 || report.Git.LinesNet != 6 {
		t.Fatalf("git global inesperado: %+v", report.Git)
	}
	if len(report.Agents) != 2 || report.Agents[0].Project != "infra" || report.Agents[1].Project != "orquestador" {
		t.Fatalf("agentes globales inesperados: %+v", report.Agents)
	}
	if len(report.Timeline) != 2 || report.Timeline[0].Project != "infra" || report.Timeline[0].Kind != "worker_recovery_requested" {
		t.Fatalf("timeline global inesperada: %+v", report.Timeline)
	}
	if len(report.ProjectControls) != 2 {
		t.Fatalf("project controls inesperados: %+v", report.ProjectControls)
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
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
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
	workspaceControlProjectBuilder = nil

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

func TestBuildWorkspaceControlReportCuentaProjectControlsFallidos(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "orquestador"},
			{"slug": "infra"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		return &apiProyectoCockpit{Proyecto: &db.Proyecto{Slug: slug, Nombre: slug}}, nil
	}
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		if slug == "infra" {
			return nil, fmt.Errorf("control no disponible")
		}
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: slug},
			Since:   since,
		}, nil
	}

	report, err := buildWorkspaceControlReport()
	if err != nil {
		t.Fatalf("buildWorkspaceControlReport: %v", err)
	}
	if report.ProjectControlsFailed != 1 {
		t.Fatalf("project controls fallidos inesperados: %+v", report)
	}
	if len(report.ProjectControls) != 1 || report.ProjectControls[0].Project == nil || report.ProjectControls[0].Project.Slug != "orquestador" {
		t.Fatalf("project controls cargados inesperados: %+v", report.ProjectControls)
	}
}

func TestBuildWorkspaceControlReportExponeCriticalProjectRiskEstructurado(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "infra"},
			{"slug": "orquestador"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		switch slug {
		case "infra":
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra", Nombre: "Infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
			}, nil
		default:
			return &apiProyectoCockpit{
				Proyecto: &db.Proyecto{Slug: slug, Nombre: slug},
			}, nil
		}
	}
	workspaceControlProjectBuilder = nil

	report, err := buildWorkspaceControlReport()
	if err != nil {
		t.Fatalf("buildWorkspaceControlReport: %v", err)
	}
	if report.CriticalProjectRisk == nil {
		t.Fatalf("falta critical project risk estructurado: %+v", report)
	}
	if report.CriticalProjectRisk.Project != "infra" || report.CriticalProjectRisk.Blocking != 14 {
		t.Fatalf("critical project risk inesperado: %+v", report.CriticalProjectRisk)
	}
	for _, token := range []string{"riesgo=critico", "integracion_bloqueada=14", "review_gates=1", "runtime_orders=1", "mailbox_rt=1"} {
		if !containsStringWorkspace(report.CriticalProjectRisk.Highlights, token) {
			t.Fatalf("falta %q en critical project risk: %+v", token, report.CriticalProjectRisk)
		}
	}
	raw, err := json.Marshal(apiWorkspaceControlResponse{Control: report})
	if err != nil {
		t.Fatalf("marshal workspace control: %v", err)
	}
	if !strings.Contains(string(raw), `"critical_project_risk":{"project":"infra"`) {
		t.Fatalf("json sin critical_project_risk estructurado: %s", raw)
	}
}

func TestBuildWorkspaceControlReportDerivaOperationalDesdeCockpitSiFaltaProjectControl(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "infra"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		return &apiProyectoCockpit{
			Proyecto:                &db.Proyecto{Slug: "infra", Nombre: "Infra"},
			TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
			ReviewGatesAbiertas:     1,
			RuntimeOrdersAbiertas:   1,
			RuntimeMailboxPendiente: 1,
		}, nil
	}
	workspaceControlProjectBuilder = nil

	report, err := buildWorkspaceControlReport()
	if err != nil {
		t.Fatalf("buildWorkspaceControlReport: %v", err)
	}
	if report.Operational.State != "bloqueado" {
		t.Fatalf("estado operativo fallback inesperado: %+v", report.Operational)
	}
	if report.Operational.AttentionScore != 14 || report.Operational.AttentionLabel != "critico" {
		t.Fatalf("attention fallback inesperada: %+v", report.Operational)
	}
	if report.Operational.BlockingProjects != 1 || report.Operational.ProjectsByState["bloqueado"] != 1 {
		t.Fatalf("blocking fallback inesperado: %+v", report.Operational)
	}
	for _, token := range []string{"proyecto=infra", "integración bloqueada", "bloqueadas=1", "review_gates=1", "runtime_orders=1", "mailbox_rt=1"} {
		if !strings.Contains(report.Operational.StateReason, token) {
			t.Fatalf("state reason fallback sin %q: %+v", token, report.Operational)
		}
	}
}

func TestParseWorkspaceControlSinceUsa24hPorDefecto(t *testing.T) {
	before := time.Now().UTC()
	since, err := parseWorkspaceControlSince("")
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("parseWorkspaceControlSince: %v", err)
	}
	minWant := before.Add(-workspaceControlDefaultWindow).Add(-2 * time.Second)
	maxWant := after.Add(-workspaceControlDefaultWindow).Add(2 * time.Second)
	if since.Before(minWant) || since.After(maxWant) {
		t.Fatalf("ventana por defecto inesperada: got=%s want_between=[%s,%s]", since, minWant, maxWant)
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
