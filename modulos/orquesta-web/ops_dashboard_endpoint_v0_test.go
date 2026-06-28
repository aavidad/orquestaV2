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
		"/api/v0/observability/workspace-timeline",
		"sources: ['director_stats', 'run_queue', 'runtime_progress']",
		"include_sources: ['director_stats', 'run_queue', 'runtime_progress']",
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
		"ops_snapshot",
		"stats_fetch_status",
		"stats_reason_code",
		"stats_unavailable",
		"completed_snapshot",
		"usage_report_missing",
		"goal: payload.goal",
		"director_execution_mode",
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
		"Fuentes",
		"sources-detail",
		"renderWorkspaceSources",
		"watermark",
		"effective_config",
		"requestServerShutdownFromAdmin(false)",
		"requestServerShutdownFromAdmin(true)",
		"Variables pendientes para reinicio",
		"orquesta.ops.pending_env.v1",
		"director-primary-action",
		"/api/v0/runs/control",
		"/api/v0/runs/queue/priority",
		"/api/v0/runs/supervise",
		"controlSelectedRun(\\'pause\\')",
		"controlSelectedRun(\\'resume\\')",
		"controlSelectedRun(\\'stop\\')",
		"controlSelectedRun(\\'cancel\\')",
		"reactivateSelectedRun()",
		"superviseGlobalWave()",
		"advanceSelectedRun()",
		"advanceQueueRow",
		"runIsGoalFirst",
		"runNeedsGoalStateRepair",
		"repairGoalStateOps",
		"safeActions",
		"staleRunning",
		"allAutoprogrammingSafeActions",
		"autoprogrammingObserveGoalSafeAction",
		"executeAutoprogrammingSafeActionOps",
		"observeGoalOps",
		"/api/v0/apps/director/goal/observe",
		"/api/v0/autoprogramming/goal/observe",
		"Observar goal",
		"superviseSelectedRun()",
		"superviseQueueRow",
		"function superviseOps",
		"Acción segura",
		"Avanzar run",
		"Avanzar run desde fila",
		"max_runs_per_tick",
		"max_external_waits",
		"stop_reason",
		"last.status",
		"history",
		"Quitar de cola",
		"Memoria proceso",
		"Disco peor uso",
		"flujo operativo",
		"director autónomo",
		"directorDecisionSummary",
		"opsSnapshotDecision",
		"opsDecisionLabel",
		"opsDecisionReason",
		"snapshot operativo",
		"renderDirectorDecision",
		"agentNeedsAttention",
		"Revisar y replanificar",
		"Esperar entregas acotadas",
		"Lanzar siguiente ola",
		"Reparar estado goal-first",
		"repair_goal_state",
		"Estado del Director",
		"closure_status",
		"progress_source",
		"stats_reason_code",
		"loop_detected_agents",
		"stopped_agents",
		"checkpoint_agents_pending",
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

func TestOpsDashboardWebEndpointV0GoalFirstUsaObserveGoalEnAvance(t *testing.T) {
	body := opsDashboardHTMLV0()
	for _, want := range []string{
		"function runIsGoalFirst",
		"function selectedRunAdvanceActionLabel",
		"function queueAdvanceActionTitle",
		"case 'observe_goal'",
		"Run goal-first",
		"const goalRun = (lastSnapshot.runs || []).find(function(run) { return runIsGoalFirst(run); })",
		"function autoprogrammingObserveGoalSafeAction",
		"function autoprogrammingObserveGoalDecision",
		"function runOpsSnapshotDecision",
		"function runNeedsGoalStateRepair",
		"async function repairGoalStateOps",
		"opsSnapshot: (data.auto || {}).ops_snapshot || null",
		"staleRunning: staleRunning",
		"const decision = runOpsSnapshotDecision(ref)",
		"ops_snapshot || {}).decision",
		"autoprogrammingObserveGoalDecision((run || {}).run_ref)",
		"if (runNeedsGoalStateRepair(run.run_ref)) return repairGoalStateOps(run, 'seleccionado')",
		"if (runNeedsGoalStateRepair(run.run_ref || runRef)) return repairGoalStateOps(run, 'fila')",
		"async function advanceQueueRow",
		"async function advanceSelectedRun",
		"async function observeGoalOps",
		"async function executeAutoprogrammingSafeActionOps",
		"/api/v0/apps/director/goal/observe",
		"/api/v0/autoprogramming/goal/observe",
		"Boolean(autoprogrammingObserveGoalSafeAction((run || {}).run_ref))",
		"const safeAction = autoprogrammingObserveGoalSafeAction(run.run_ref)",
		"if (runIsGoalFirst(run)) return observeGoalOps(run, 'goal seleccionado')",
		"if (runIsGoalFirst(run)) return observeGoalOps(run, 'goal desde fila')",
		"Reparar GoalWorkStateV0 antes de supervision legacy",
		"return 'Reparar goal'",
		"Observar goal",
		"goalOpsResultText",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("html goal-first no contiene %q", want)
		}
	}
}

func TestOpsDashboardWebEndpointV0DecisionPorRunTienePrioridadSobreSnapshotGlobal(t *testing.T) {
	body := opsDashboardHTMLV0()
	fn := htmlFunctionSliceForTest(t, body, "function runOpsSnapshotDecision", "function runNeedsGoalStateRepair")
	runDecision := strings.Index(fn, "(run || {}).ops_snapshot")
	topDecision := strings.Index(fn, "const topDecision")

	if runDecision < 0 || topDecision < 0 || !(runDecision < topDecision) {
		t.Fatalf("runOpsSnapshotDecision debe evaluar ops_snapshot por run antes del snapshot global:\n%s", fn)
	}
	for _, want := range []string{
		"if (ref && String((run || {}).run_ref || '') !== ref) return",
		"return decisions.find(function(decision)",
		"String(decision.run_ref || '') === ref",
	} {
		if !strings.Contains(fn, want) {
			t.Fatalf("runOpsSnapshotDecision no contiene %q:\n%s", want, fn)
		}
	}
}

func TestOpsDashboardWebEndpointV0SinSafeActionNoLlamaSupervisorLegacyGlobal(t *testing.T) {
	body := opsDashboardHTMLV0()
	fn := htmlFunctionSliceForTest(t, body, "async function superviseGlobalWave", "async function advanceSelectedRun")

	for _, want := range []string{
		"const superviseAction = firstAutoprogrammingSafeAction('supervise', 'queue')",
		"if (!superviseAction)",
		"Sin accion segura de cola",
		"executeAutoprogrammingSafeActionOps(superviseAction, 'ola global')",
	} {
		if !strings.Contains(fn, want) {
			t.Fatalf("superviseGlobalWave no contiene %q:\n%s", want, fn)
		}
	}
	if strings.Contains(fn, "superviseOps({queue_ref: 'global'}") {
		t.Fatalf("superviseGlobalWave conserva fallback legacy global:\n%s", fn)
	}
}

func TestOpsDashboardWebEndpointV0StalledNoDisparaAtencion(t *testing.T) {
	body := opsDashboardHTMLV0()
	agentFn := htmlFunctionSliceForTest(t, body, "function agentNeedsAttention", "function runNeedsAttention")
	runFn := htmlFunctionSliceForTest(t, body, "function runNeedsAttention", "function buildAgents")

	for _, forbidden := range []string{"no_progress_ticks", "stalled_agents", "stall"} {
		if strings.Contains(agentFn, forbidden) {
			t.Fatalf("agentNeedsAttention no debe usar %q como atencion dura:\n%s", forbidden, agentFn)
		}
		if strings.Contains(runFn, forbidden) {
			t.Fatalf("runNeedsAttention no debe usar %q como atencion dura:\n%s", forbidden, runFn)
		}
	}
	for _, want := range []string{"needs_attention", "loop_detected", "stopped", "failed", "error"} {
		if !strings.Contains(agentFn, want) {
			t.Fatalf("agentNeedsAttention no contiene %q:\n%s", want, agentFn)
		}
	}
	for _, want := range []string{"blocked", "agents_failed", "agents_need_attention", "loop_detected_agents", "stopped_agents", "checkpoint_agents_pending"} {
		if !strings.Contains(runFn, want) {
			t.Fatalf("runNeedsAttention no contiene %q:\n%s", want, runFn)
		}
	}
}

func TestOpsDashboardWebEndpointV0SupervisorPayloadAcotado(t *testing.T) {
	body := opsDashboardHTMLV0()
	fn := htmlFunctionSliceForTest(t, body, "async function superviseOps", "function supervisorResultText")

	for _, want := range []string{
		"queue_ref: (scope || {}).queue_ref || ''",
		"run_ref: (scope || {}).run_ref || ''",
		"continue_message",
		"max_ticks: 1",
		"max_runs_per_tick: 1",
		"max_executions: 1",
		"max_bursts: 1",
		"max_steps_per_burst: 1",
		"max_dispatches_per_wait: 1",
		"max_commands: 1",
		"max_outbox_per_cycle: 1",
		"max_decision_cycles: 1",
		"max_external_waits: 1",
		"idempotency_key",
		"/api/v0/runs/supervise",
	} {
		if !strings.Contains(fn, want) {
			t.Fatalf("superviseOps no contiene %q:\n%s", want, fn)
		}
	}
}

func htmlFunctionSliceForTest(t *testing.T, body, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(body, startMarker)
	if start < 0 {
		t.Fatalf("html no contiene %q", startMarker)
	}
	end := strings.Index(body[start:], endMarker)
	if end < 0 {
		t.Fatalf("html no contiene fin %q tras %q", endMarker, startMarker)
	}
	return body[start : start+end]
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
