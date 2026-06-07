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
		"opsStatsProjection",
		"progress_source",
		"stats_fetch_status",
		"stats_reason_code",
		"stats_unavailable",
		"completed_snapshot",
		"usage_report_missing",
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
		"director autónomo",
		"directorDecisionSummary",
		"renderDirectorDecision",
		"Revisar y replanificar",
		"Esperar entregas acotadas",
		"Lanzar siguiente ola",
		"Estado del Director",
		"closure_status",
		"progress_source",
		"stats_reason_code",
		"flow-attention-card",
		"renderFlowSummary",
		"runNeedsAttention",
		"tareas cerradas",
		"stableSortByFirstSeen",
		"orquesta.ops.stable_order.v1",
		"mergeAgentDisplayCache",
		"taskByAgent",
		"data-stable-id",
		"queueStableID",
		"queueDetail",
		"renderTasks",
		"matchesStatus",
		"row-actions",
		"setQueueRowPriority",
		"controlQueueRow",
		"data-detail-key",
		"runtimeFileBlock",
		"Envelope",
		"redaccion",
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

func TestOpsDashboardWebEndpointV0CacheLiveNoQuedaDentroDeFetchRuntimeDetail(t *testing.T) {
	body := opsDashboardHTMLV0()
	fetchRuntime := strings.Index(body, "async function fetchRuntimeDetail")
	ensureRuntime := strings.Index(body, "async function ensureRuntimeDetail")
	refreshFunction := strings.Index(body, "async function refreshAll")
	injectionPoint := strings.Index(body, "ops-live-cache-policy injection point")
	runtimeDetail := strings.Index(body, "function runtimeDetailHTML")
	clamp := strings.Index(body, "function opsClampPercent")
	buildRuns := strings.Index(body, "function buildRuns")
	refreshCall := -1
	if injectionPoint >= 0 {
		if relative := strings.Index(body[injectionPoint:], "refreshAll();"); relative >= 0 {
			refreshCall = injectionPoint + relative
		}
	}
	if fetchRuntime < 0 || ensureRuntime < 0 || refreshFunction < 0 || injectionPoint < 0 || runtimeDetail < 0 || clamp < 0 || buildRuns < 0 || refreshCall < 0 {
		t.Fatalf("html incompleto: fetch=%d ensure=%d refreshFn=%d injection=%d runtime=%d clamp=%d buildRuns=%d refreshCall=%d", fetchRuntime, ensureRuntime, refreshFunction, injectionPoint, runtimeDetail, clamp, buildRuns, refreshCall)
	}
	if !(fetchRuntime < ensureRuntime && ensureRuntime < refreshFunction && refreshFunction < injectionPoint && injectionPoint < runtimeDetail && runtimeDetail < clamp && clamp < buildRuns && buildRuns < refreshCall) {
		t.Fatalf("orden invalido: fetch=%d ensure=%d refreshFn=%d injection=%d runtime=%d clamp=%d buildRuns=%d refreshCall=%d", fetchRuntime, ensureRuntime, refreshFunction, injectionPoint, runtimeDetail, clamp, buildRuns, refreshCall)
	}
	if strings.Contains(body[fetchRuntime:ensureRuntime], "function opsClampPercent") {
		t.Fatalf("ops cache policy no debe quedar dentro de fetchRuntimeDetail")
	}
	if strings.Contains(body[fetchRuntime:injectionPoint], "function runtimeDetailHTML") {
		t.Fatalf("runtime detail helpers no deben quedar dentro de chunks partidos antes del punto de inyeccion")
	}
}

func TestOpsDashboardWebEndpointV0SoloGET(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, WebOpsDashboardPageEndpointV0, nil)

	NewOpsDashboardWebEndpointV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != webPublicHTTPAllowHeaderV0(http.MethodGet) {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}
