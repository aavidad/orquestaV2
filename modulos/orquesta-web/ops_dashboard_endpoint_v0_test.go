package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, WebOpsDashboardPageEndpointV0, nil)

	NewOpsDashboardWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Orquesta Ops",
		"/api/v0/server/status",
		"/api/v0/server/resources",
		"/api/v0/autoprogramming/status",
		"/api/v0/director/stats",
		"/api/v0/ops/agent-runtime-detail",
		"refreshMs = 1000",
		"Proyectos / runs",
		"Filtrar por tarea, run o proyecto",
		"Fases comparadas",
		"esquema de fases",
		"phase-map",
		"phase-node",
		"phase-body",
		"compact-table",
		"buildPhaseSummary",
		"renderPhaseMatrix",
		"Uso comparado",
		"usage-body",
		"renderUsageMatrix",
		"tableCell('Tokens'",
		"tableCell('Atención'",
		"Tareas completas",
		"filter-task-status",
		"Agentes",
		"Filtrar por agente, run o tarea",
		"Capacidad / uso",
		"formatTokens",
		"agentUsageLabel",
		"Uso observado",
		"Cola",
		"Completadas observadas",
		"Detalle y control",
		"Admin",
		"effective_config",
		"requestServerShutdownFromAdmin(false)",
		"requestServerShutdownFromAdmin(true)",
		"Variables pendientes para reinicio",
		"orquesta.ops.pending_env.v1",
		"/api/v0/runs/control",
		"/api/v0/runs/queue/priority",
		"controlSelectedRun(\\'pause\\')",
		"controlSelectedRun(\\'resume\\')",
		"controlSelectedRun(\\'stop\\')",
		"controlSelectedRun(\\'cancel\\')",
		"reactivateSelectedRun()",
		"Quitar de cola",
		"Memoria proceso",
		"Disco peor uso",
		"flujo operativo",
		"flow-attention-card",
		"renderFlowSummary",
		"runNeedsAttention",
		"tareas cerradas",
		"stableSortByFirstSeen",
		"orquesta.ops.stable_order.v1",
		"mergeAgentDisplayCache",
		"data-stable-id",
		"queueStableID",
		"queueDetail",
		"renderTasks",
		"matchesStatus",
		"row-actions",
		"setQueueRowPriority",
		"controlQueueRow",
		"data-detail-key",
		"Prompt exacto",
		"agent_packet.json",
		"agent_ack.json",
		"Qué se consigue",
		"run-flow",
		"runFlowHTML",
		"function tableCell(label, content, attrs)",
		"tableCell('Fase'",
		"tableCell('Tareas cerradas'",
		"data-label=\"",
		"content: attr(data-label)",
		".table-wrap thead",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("html no contiene %q", want)
		}
	}
}

func TestOpsDashboardWebEndpointV0SoloGET(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, WebOpsDashboardPageEndpointV0, nil)

	NewOpsDashboardWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}
