/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestProyectoControlJSONAlineadoConAPI(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/proyectos/orquestador/control" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("desde"); got == "" {
			t.Fatalf("query desde vacia")
		}
		_ = json.NewEncoder(w).Encode(apiProyectoControlResponse{
			Control: &projectControlReport{
				Project: &db.Proyecto{
					ID:     7,
					Slug:   "orquestador",
					Nombre: "Orquestador",
				},
				Status:         apiStatusResponse{},
				Since:          time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC),
				Generated:      time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC),
				CompletionPct:  42,
				AutonomyEvents: 3,
				AutonomyByKind: map[string]int{
					"task_reassigned":           2,
					"worker_recovery_requested": 1,
				},
				AutonomyLastAt:       ptrTimeProyectoControlTest(time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC)),
				AutonomyHighlights:   []string{"task_reassigned=2", "worker_recovery_requested=1"},
				IntegrationRisk:      "alto",
				IntegrationRiskScore: 6,
				Autonomy: []autonomyEventSummary{{
					Kind:      "task_reassigned",
					CreatedAt: time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC),
					Agent:     "Codex1",
					Artifacts: []string{"runtime_checkpoint:41"},
				}},
				Git: projectControlGitAggregate{
					PendingAddedLines:   3,
					PendingDeletedLines: 1,
				},
			},
		})
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandTreeFlags(proyectoControlCmd)
	_ = proyectoControlCmd.Flags().Set("json", "true")

	out := capturarStdout(t, func() {
		if err := proyectoControlCmd.RunE(proyectoControlCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("proyecto control --json via api: %v", err)
		}
	})

	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		t.Fatalf("decode json cli: %v\n%s", err, out)
	}
	if _, ok := root["control"]; !ok {
		t.Fatalf("salida sin raiz control: %s", out)
	}
	if _, ok := root["project"]; ok {
		t.Fatalf("salida expone shape antiguo en raiz: %s", out)
	}

	var resp apiProyectoControlResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("decode wrapper control: %v\n%s", err, out)
	}
	if resp.Control == nil || resp.Control.Project == nil {
		t.Fatalf("control vacio: %+v", resp.Control)
	}
	if resp.Control.Project.Slug != "orquestador" || resp.Control.CompletionPct != 42 {
		t.Fatalf("payload control inesperado: %+v", resp.Control)
	}
	if strings.TrimSpace(resp.Control.Project.Nombre) != "Orquestador" {
		t.Fatalf("nombre de proyecto inesperado: %+v", resp.Control.Project)
	}
	if resp.Control.AutonomyEvents != 3 || resp.Control.AutonomyByKind["task_reassigned"] != 2 {
		t.Fatalf("resumen autonomy inesperado: %+v", resp.Control)
	}
	if len(resp.Control.Autonomy) != 1 || resp.Control.Autonomy[0].Kind != "task_reassigned" {
		t.Fatalf("autonomy inesperada en control: %+v", resp.Control.Autonomy)
	}
	if len(resp.Control.AutonomyHighlights) == 0 || resp.Control.AutonomyHighlights[0] != "task_reassigned=2" {
		t.Fatalf("autonomy highlights inesperados en control: %+v", resp.Control.AutonomyHighlights)
	}
	if resp.Control.IntegrationRisk != "alto" || resp.Control.IntegrationRiskScore != 6 {
		t.Fatalf("riesgo de integracion inesperado en control: %+v", resp.Control)
	}
	if len(resp.Control.Autonomy[0].Artifacts) != 1 || resp.Control.Autonomy[0].Artifacts[0] != "runtime_checkpoint:41" {
		t.Fatalf("artifacts autonomy inesperados en control: %+v", resp.Control.Autonomy)
	}
}

func TestProjectAutonomyActivityMomentUsaEventoMasReciente(t *testing.T) {
	older := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	got := projectAutonomyActivityMoment([]autonomyEventSummary{
		{Kind: "task_reassigned", CreatedAt: older},
		{Kind: "runtime_restart_requested", CreatedAt: newer},
	})
	if got == nil || !got.Equal(newer) {
		t.Fatalf("momento autonomia inesperado: %v", got)
	}
}

func TestProjectControlIntegrationRiskDesdeCockpit(t *testing.T) {
	score, label, highlights := projectControlIntegrationRisk(&apiProyectoCockpit{
		TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
		ReviewGatesAbiertas:     1,
		RuntimeOrdersAbiertas:   1,
		RuntimeMailboxPendiente: 1,
	}, []string{"task_reassigned=1"})
	if score != 14 || label != "critico" {
		t.Fatalf("riesgo inesperado: score=%d label=%q", score, label)
	}
	for _, token := range []string{"task_reassigned=1", "riesgo=critico", "integracion_bloqueada=14", "bloqueadas=1", "review_gates=1", "runtime_orders=1", "mailbox_rt=1"} {
		if !containsProyectoControlString(highlights, token) {
			t.Fatalf("falta %q en highlights: %+v", token, highlights)
		}
	}
}

func containsProyectoControlString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestCompactProjectControlIntegrationHighlightsOmiteDuplicadosEscalares(t *testing.T) {
	got := compactProjectControlIntegrationHighlights([]string{
		"riesgo=critico",
		"integracion_bloqueada=15",
		"review_gates=1",
		"mailbox_rt=1",
	})
	if len(got) != 2 || got[0] != "review_gates=1" || got[1] != "mailbox_rt=1" {
		t.Fatalf("compactado inesperado: %+v", got)
	}
}

func TestImprimirProyectoControlNoDuplicaEscalaresDeIntegracionEnHighlights(t *testing.T) {
	report := &projectControlReport{
		Project: &db.Proyecto{
			Slug:   "orquestador",
			Nombre: "Orquestador",
		},
		Status: apiStatusResponse{},
		TaskCounts: map[string]int{
			string(db.TareaBloqueada): 1,
		},
		Progress: projectControlProgress{
			State:          "bloqueado",
			StateReason:    "bloqueos abiertos sin ejecución activa",
			OpenTasks:      1,
			BlockedTasks:   1,
			AttentionScore: 6,
			AttentionLabel: "alto",
		},
		IntegrationRisk:      "alto",
		IntegrationRiskScore: 6,
		AutonomyHighlights: []string{
			"task_reassigned=1",
			"riesgo=alto",
			"integracion_bloqueada=6",
			"review_gates=1",
			"mailbox_rt=1",
		},
	}

	out := capturarStdout(t, func() {
		if err := imprimirProyectoControl(report); err != nil {
			t.Fatalf("imprimirProyectoControl: %v", err)
		}
	})
	if !strings.Contains(out, "Integración: riesgo=alto · integracion_bloqueada=6") {
		t.Fatalf("sin bloque de integracion esperado: %s", out)
	}
	if !strings.Contains(out, "Control:    estado=bloqueado · bloqueos abiertos sin ejecución activa · atención=alto(6) · abiertas=1 activas=0 bloqueadas=1 libres=0") {
		t.Fatalf("sin bloque control esperado: %s", out)
	}
	if !strings.Contains(out, "Highlights: task_reassigned=1 | review_gates=1 | mailbox_rt=1") {
		t.Fatalf("highlights sin compactar correctamente: %s", out)
	}
	if strings.Contains(out, "Highlights: task_reassigned=1 | riesgo=alto") || strings.Contains(out, "Highlights: task_reassigned=1 | integracion_bloqueada=6") {
		t.Fatalf("los highlights siguen duplicando escalares de integracion: %s", out)
	}
}

func TestBuildProjectControlProgressDetectaAtascoPorLagYBloqueo(t *testing.T) {
	generated := time.Date(2026, 4, 24, 18, 0, 0, 0, time.UTC)
	lastActivity := generated.Add(-26 * time.Hour)
	report := &projectControlReport{
		Generated:      generated,
		LastActivityAt: &lastActivity,
		Tasks: []*db.Tarea{
			{Estado: db.TareaBloqueada},
			{Estado: db.TareaLibre},
			{Estado: db.TareaCompletada},
		},
		Cockpit: &apiProyectoCockpit{
			ReviewGatesAbiertas:     1,
			RuntimeMailboxPendiente: 1,
			RuntimeOrdersAbiertas:   1,
		},
	}

	got := buildProjectControlProgress(report)
	if got.State != "atascado" || got.StateReason != "bloqueos sin ejecución ni actividad reciente" {
		t.Fatalf("estado inesperado: %+v", got)
	}
	if got.TotalTasks != 3 || got.OpenTasks != 2 || got.CompletedTasks != 1 || got.BlockedTasks != 1 || got.UnassignedTasks != 1 {
		t.Fatalf("agregados inesperados: %+v", got)
	}
	if got.ActivityLagMinutes != 26*60 {
		t.Fatalf("lag inesperado: %+v", got)
	}
	if got.AttentionScore != 12 || got.AttentionLabel != "critico" {
		t.Fatalf("atencion inesperada: %+v", got)
	}
}

func TestBuildProjectControlProgressMarcaActivoConTrabajoEnCurso(t *testing.T) {
	generated := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	lastActivity := generated.Add(-35 * time.Minute)
	report := &projectControlReport{
		Generated:      generated,
		LastActivityAt: &lastActivity,
		Tasks: []*db.Tarea{
			{Estado: db.TareaEnProgreso},
			{Estado: db.TareaAsignada},
		},
	}

	got := buildProjectControlProgress(report)
	if got.State != "activo" || got.StateReason != "hay trabajo en progreso" {
		t.Fatalf("estado activo inesperado: %+v", got)
	}
	if got.ActiveTasks != 1 || got.ReservedTasks != 1 || got.OpenTasks != 2 {
		t.Fatalf("conteos activos inesperados: %+v", got)
	}
	if got.ActivityLagMinutes != 35 {
		t.Fatalf("lag activo inesperado: %+v", got)
	}
	if got.AttentionScore != 0 || got.AttentionLabel != "estable" {
		t.Fatalf("atencion activa inesperada: %+v", got)
	}
}

func TestFormatProjectControlLag(t *testing.T) {
	cases := map[int]string{
		0:   "0m",
		25:  "25m",
		60:  "1h",
		135: "2h15m",
	}
	for minutes, want := range cases {
		if got := formatProjectControlLag(minutes); got != want {
			t.Fatalf("lag %d inesperado: got=%q want=%q", minutes, got, want)
		}
	}
}

func ptrTimeProyectoControlTest(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}
