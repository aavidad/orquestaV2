package cmd

import (
	"strings"
	"testing"
	"time"

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

func TestDeriveSupervisorPipelineStateOmiteSupervisorReservadoEnConteosWorker(t *testing.T) {
	prevConfigGet := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfigGet
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "Codex2", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	state := deriveSupervisorPipelineState("Codex2", "orquestador", apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex2", Activo: true},
			{Nombre: "Codex4", Activo: true},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex2", Activo: true},
		},
	}, nil, nil)

	if state.MetadataJSON == "" || state.ArtifactsJSON == "" {
		t.Fatalf("estado pipeline sin metadata/artifacts: %+v", state)
	}
	if state.CurrentPhase != "idle" || state.Status != "active" {
		t.Fatalf("no deberia contar al supervisor como worker trabajando: %+v", state)
	}
	if !containsJSONFragment(state.MetadataJSON, `"connected_workers":1`) || !containsJSONFragment(state.MetadataJSON, `"working_workers":0`) {
		t.Fatalf("metadata worker inesperada: %s", state.MetadataJSON)
	}
	if !containsJSONFragment(state.ArtifactsJSON, `"connected_agents":["Codex4"]`) || !containsJSONFragment(state.ArtifactsJSON, `"working_agents":[]`) {
		t.Fatalf("artifacts worker inesperados: %s", state.ArtifactsJSON)
	}
}

func containsJSONFragment(raw, want string) bool {
	return strings.Contains(raw, want)
}

func TestBuildSupervisorPipelineSnapshotRefinaConFollowupSidecarParalelo(t *testing.T) {
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

		_, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-pipeline-followup-visible",
			ThreadID:     "slice-visible-1",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-visible",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","task_id":530,"task_title":"Frente amplio del control plane","slice_index":1,"slice_total":2,"write_set_slice":["cmd/controlplane_support.go"],"pipeline_parent_followup_dispatched":true,"pipeline_parent_followup_dispatched_at":"2026-04-21T12:00:00Z","pipeline_parent_followup_phase":"revision","pipeline_parent_followup_action":"avanzar_fase","pipeline_parent_followup_git_merge_id":91}`,
		})
		if err != nil {
			t.Fatalf("crear subagente sidecar: %v", err)
		}

		snapshot, err := buildSupervisorPipelineSnapshot("OpenClaw", "orquestador", 20)
		if err != nil {
			t.Fatalf("buildSupervisorPipelineSnapshot: %v", err)
		}
		pipelines, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
		loop := findSupervisorPipelineByName(pipelines, "supervisor-loop")
		if loop == nil {
			t.Fatalf("falta supervisor-loop: %#v", pipelines)
		}
		if loop.CurrentPhase != "revision" || loop.Status != "active" {
			t.Fatalf("pipeline refinada inesperada: %+v", loop)
		}
		if loop.CurrentTaskID == nil || *loop.CurrentTaskID != 530 {
			t.Fatalf("task derivada inesperada: %+v", loop)
		}
		if loop.CurrentMergeID == nil || *loop.CurrentMergeID != 91 {
			t.Fatalf("merge derivado inesperado: %+v", loop)
		}
		followup, _ := snapshot["followup"].(map[string]any)
		if followup == nil || followup["phase"] != "revision" {
			t.Fatalf("followup compacta inesperada: %#v", snapshot["followup"])
		}
	})
}

func TestDeriveSupervisorPipelineStateIncluyeAutonomiaRecienteEnMetadataYArtifacts(t *testing.T) {
	ts := time.Date(2026, 4, 24, 12, 30, 0, 0, time.UTC)
	taskID := int64(530)
	state := deriveSupervisorPipelineState("OpenClaw", "orquestador", apiStatusResponse{
		Autonomia: autonomiaResumen{
			Count:  2,
			ByKind: map[string]int{"handoff_failed": 1, "repair_helper_opened": 1},
			LastAt: &ts,
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   ts,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
	}, nil, nil)

	if !containsJSONFragment(state.MetadataJSON, `"autonomy_event_count":2`) {
		t.Fatalf("metadata sin autonomy_event_count: %s", state.MetadataJSON)
	}
	if !containsJSONFragment(state.MetadataJSON, `"handoff_failed":1`) {
		t.Fatalf("metadata sin autonomy_by_kind: %s", state.MetadataJSON)
	}
	if !containsJSONFragment(state.ArtifactsJSON, `"recent_autonomy_events"`) || !containsJSONFragment(state.ArtifactsJSON, `"handoff_failed"`) {
		t.Fatalf("artifacts sin autonomia reciente: %s", state.ArtifactsJSON)
	}
}

func TestDeriveSupervisorPipelineStateIncluyeRiesgoCanonicoEnMetadataYArtifacts(t *testing.T) {
	ts := time.Date(2026, 4, 24, 12, 30, 0, 0, time.UTC)
	state := deriveSupervisorPipelineState("OpenClaw", "orquestador", apiStatusResponse{
		AutonomySurface: &autonomySurface{
			Events:     2,
			ByKind:     map[string]int{"handoff_failed": 1, "repair_helper_opened": 1},
			LastAt:     &ts,
			Highlights: []string{"handoff_failed=1"},
		},
		AutonomyHighlights: []string{
			"frentes_bloqueantes=1",
			"integracion_bloqueada=10",
			"riesgo_top=orquestador(10)",
		},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "orquestador",
			Events:     2,
			Blocking:   10,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=10", "review_gates=1"},
		},
	}, nil, nil)

	if !containsJSONFragment(state.MetadataJSON, `"autonomy_highlights"`) || !containsJSONFragment(state.MetadataJSON, `"integracion_bloqueada=10"`) {
		t.Fatalf("metadata sin autonomy_highlights canonicos: %s", state.MetadataJSON)
	}
	if !containsJSONFragment(state.MetadataJSON, `"critical_project_risk"`) || !containsJSONFragment(state.MetadataJSON, `"project":"orquestador"`) {
		t.Fatalf("metadata sin critical_project_risk canonico: %s", state.MetadataJSON)
	}
	if !containsJSONFragment(state.ArtifactsJSON, `"autonomy_surface"`) || !containsJSONFragment(state.ArtifactsJSON, `"critical_project_risk"`) {
		t.Fatalf("artifacts sin surface/risk canonicos: %s", state.ArtifactsJSON)
	}
}

func TestBuildSupervisorPipelineSnapshotExponeRiesgoCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		snapshot, err := buildSupervisorPipelineSnapshotFromInputs("OpenClaw", "orquestador", 20, apiStatusResponse{
			AutonomySurface: &autonomySurface{
				Events:     1,
				ByKind:     map[string]int{"task_reassigned": 1},
				Highlights: []string{"task_reassigned=1"},
			},
			AutonomyHighlights: []string{
				"frentes_bloqueantes=1",
				"integracion_bloqueada=12",
				"riesgo_top=infra(12)",
			},
			CriticalProjectRisk: &workspaceAutonomyProjectSummary{
				Project:    "infra",
				Events:     1,
				Blocking:   12,
				Highlights: []string{"riesgo=critico", "integracion_bloqueada=12", "runtime_orders=1"},
			},
		}, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("buildSupervisorPipelineSnapshotFromInputs: %v", err)
		}
		highlights, _ := snapshot["autonomy_highlights"].([]string)
		if len(highlights) == 0 || !strings.Contains(strings.Join(highlights, "|"), "integracion_bloqueada=12") {
			t.Fatalf("autonomy_highlights inesperados: %#v", snapshot["autonomy_highlights"])
		}
		risk, _ := snapshot["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "infra" || risk.Blocking != 12 {
			t.Fatalf("critical_project_risk inesperado: %#v", snapshot["critical_project_risk"])
		}
	})
}

func TestDeriveSupervisorPipelineStateMarcaBlockedPorRiesgoCanonicoAltoSinWorkers(t *testing.T) {
	state := deriveSupervisorPipelineState("OpenClaw", "orquestador", apiStatusResponse{
		AutonomyHighlights: []string{
			"frentes_bloqueantes=1",
			"integracion_bloqueada=10",
			"riesgo_top=orquestador(10)",
		},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "orquestador",
			Blocking:   10,
			Highlights: []string{"riesgo=critico", "integracion_bloqueada=10", "review_gates=1"},
		},
	}, nil, nil)

	if state.CurrentPhase != "idle" || state.Status != "blocked" {
		t.Fatalf("pipeline sin workers debería quedar blocked por riesgo alto: %+v", state)
	}
	for _, token := range []string{`"integration_risk":"critico"`, `"integration_risk_score":10`, `"risk_watch_mode":"blocked"`, `"integration_highlights":["review_gates=1"]`} {
		if !containsJSONFragment(state.MetadataJSON, token) {
			t.Fatalf("metadata sin token %s: %s", token, state.MetadataJSON)
		}
	}
}

func TestDeriveSupervisorPipelineStateCoordinaWorkersPorRiesgoCanonicoAlto(t *testing.T) {
	state := deriveSupervisorPipelineState("OpenClaw", "orquestador", apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"},
		},
		AutonomyHighlights: []string{
			"frentes_bloqueantes=1",
			"integracion_bloqueada=6",
			"riesgo_top=orquestador(6)",
		},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "orquestador",
			Blocking:   6,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=6", "runtime_orders=1"},
		},
	}, nil, nil)

	if state.CurrentPhase != "coordinar_workers" || state.Status != "active" {
		t.Fatalf("pipeline con workers debería coordinar por riesgo alto: %+v", state)
	}
	for _, token := range []string{`"integration_risk":"alto"`, `"integration_risk_score":6`, `"risk_watch_mode":"active"`} {
		if !containsJSONFragment(state.MetadataJSON, token) {
			t.Fatalf("metadata sin token %s: %s", token, state.MetadataJSON)
		}
	}
	if !containsJSONFragment(state.ArtifactsJSON, `"integration_highlights":["runtime_orders=1"]`) {
		t.Fatalf("artifacts sin integration_highlights canonicos: %s", state.ArtifactsJSON)
	}
}

func TestPhaseFromRecommendedActionMapeaAutonomyRecoveryYFollowup(t *testing.T) {
	tests := []struct {
		action supervisorRecommendedAction
		phase  string
		taskID int64
	}{
		{
			action: supervisorRecommendedAction{Kind: "autonomy_event", Action: "inspeccionar_handoff_fallido", Target: "tarea:10"},
			phase:  "autonomy_recovery",
			taskID: 10,
		},
		{
			action: supervisorRecommendedAction{Kind: "autonomy_event", Action: "seguir_reinicio_runtime", Target: "agente:Codex1"},
			phase:  "autonomy_recovery",
		},
		{
			action: supervisorRecommendedAction{Kind: "autonomy_event", Action: "seguir_reasignacion", Target: "tarea:11"},
			phase:  "autonomy_followup",
			taskID: 11,
		},
		{
			action: supervisorRecommendedAction{Kind: "autonomy_event", Action: "verificar_handoff_consolidado", Target: "tarea:12"},
			phase:  "autonomy_followup",
			taskID: 12,
		},
	}
	for _, tc := range tests {
		phase, status, taskID, _, _ := phaseFromRecommendedAction(tc.action)
		if phase != tc.phase || status != "active" {
			t.Fatalf("phase/status inesperados para %+v: phase=%s status=%s", tc.action, phase, status)
		}
		if tc.taskID > 0 {
			if taskID == nil || *taskID != tc.taskID {
				t.Fatalf("taskID inesperado para %+v: %v", tc.action, taskID)
			}
		}
	}
}

func findSupervisorPipelineByName(items []*db.SupervisorPipelineState, pipelineName string) *db.SupervisorPipelineState {
	for _, item := range items {
		if item != nil && item.PipelineName == pipelineName {
			return item
		}
	}
	return nil
}
