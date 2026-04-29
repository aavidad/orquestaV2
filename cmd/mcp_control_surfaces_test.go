package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestMCPToolsIncluyeControlSurfaces(t *testing.T) {
	tools := listMCPTools()
	for _, name := range []string{
		"orquesta.server.operational",
		"orquesta.server.self_heal",
		"orquesta.server.rearm",
		"orquesta.workspace.control",
	} {
		if !mcpToolListed(tools, name) {
			t.Fatalf("tool MCP no registrada: %s", name)
		}
	}
}

func TestMCPToolServerOperationalUsaFallbackCanonico(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevNow := statusNowFunc
	prevDegraded := apiServerOperationalDegradedBuilder
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		statusNowFunc = prevNow
		apiServerOperationalDegradedBuilder = prevDegraded
	}()

	now := time.Date(2026, 4, 28, 22, 10, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{}, now, time.Second)
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{
			Generado:          "2026-04-28T22:10:00Z",
			AgentesActivos:    []*db.Agente{{Nombre: "Codex9", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex9", Activo: true}},
			TareasPorEstado:   map[string]int{string(db.TareaEnProgreso): 1},
			Autonomia:         autonomiaResumen{WorkConfirmed: 1, ByKind: map[string]int{}},
		}, nil
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiServerOperationalDegradedBuilder = func() serverOperationalInfo {
		return serverOperationalInfo{State: "degraded", Operational: false, Reason: "unexpected"}
	}

	result, err := callMCPTool("orquesta.server.operational", nil)
	if err != nil {
		t.Fatalf("server operational MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server operational marcado como error: %#v", result)
	}
	info, _ := result["structuredContent"].(serverOperationalInfo)
	if info.State != "ready" || !info.Operational || info.TasksInProgress != 1 || info.ActiveAgents != 1 {
		t.Fatalf("server operational inesperado: %+v", info)
	}
}

func TestMCPToolServerRearmAplicaSiguienteAccionSegura(t *testing.T) {
	prevApply := serverOperationalApplyNextActionFn
	defer func() {
		serverOperationalApplyNextActionFn = prevApply
	}()

	serverOperationalApplyNextActionFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"queue_kind": "safe",
			"next_safe_action": supervisorRecommendedAction{
				Target:   "tarea:24",
				Action:   "inspeccionar_handoff_fallido",
				Priority: "alta",
			},
		}, nil
	}

	result, err := callMCPTool("orquesta.server.rearm", map[string]any{"supervisor": "OpenClaw"})
	if err != nil {
		t.Fatalf("server rearm MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server rearm marcado como error: %#v", result)
	}
	payload, _ := result["structuredContent"].(map[string]any)
	if payload["queue_kind"] != "safe" || payload["supervisor"] != "OpenClaw" {
		t.Fatalf("payload server rearm inesperado: %#v", payload)
	}
}

func TestMCPToolServerSelfHealEjecutaCarrilesCanonicosYDevuelveOperational(t *testing.T) {
	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	runtimeProcessHygieneBatch = func() (int, error) { return 6, nil }
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		return runtimeProcessDegradadosSummary{
			Count:                     4,
			GhostAssignmentsCompacted: 1,
			ReactivatedWithoutRuntime: 2,
			IdleAutoassigned:          1,
		}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) { return 3, nil }
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:                true,
			Count:             2,
			Reactivated:       2,
			CooldownSustained: 1,
		}
	}
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:           "ready",
			Operational:     true,
			Reason:          "control_plane_responsive",
			ActiveAgents:    4,
			WorkingAgents:   2,
			TasksInProgress: 3,
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", nil)
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal marcado como error: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if !payload.OK || payload.Hygiene.Count != 6 || payload.Degradados.Count != 4 || payload.Autonomia.Count != 3 {
		t.Fatalf("payload self_heal inesperado: %+v", payload)
	}
	if !payload.Wake.Orders || !payload.Wake.Mailbox {
		t.Fatalf("wake self_heal inesperado: %+v", payload.Wake)
	}
	if payload.Operational.State != "ready" || !payload.Operational.Operational || payload.Operational.TasksInProgress != 3 {
		t.Fatalf("operational final inesperado: %+v", payload.Operational)
	}
}

func TestMCPToolServerSelfHealMarcaErroresParciales(t *testing.T) {
	prevHygiene := runtimeProcessHygieneBatch
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		runtimeProcessHygieneBatch = prevHygiene
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	runtimeProcessHygieneBatch = func() (int, error) { return 0, fmt.Errorf("boom") }
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{State: "degraded", Operational: false, Reason: "status_temporarily_degraded"}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"orders":        false,
		"mailbox":       false,
		"warm":          false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != true {
		t.Fatalf("server self_heal deberia marcar error parcial: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.OK || len(payload.Errors) == 0 || !strings.Contains(payload.Errors[0], "hygiene:") {
		t.Fatalf("errores self_heal inesperados: %+v", payload)
	}
}

func TestMCPToolWorkspaceControlAceptaDesde(t *testing.T) {
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
		TareasPorEstado:     map[string]int{string(db.TareaEnProgreso): 1},
		WorkersConectados:   2,
		WorkersTrabajando:   1,
		SupervisoresActivos: 1,
		Autonomia:           autonomiaResumen{ByKind: map[string]int{}},
	}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "orquestador"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		return &apiProyectoCockpit{Proyecto: &db.Proyecto{Slug: slug, Nombre: "Orquestador"}}, nil
	}
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
		}, nil
	}

	result, err := callMCPTool("orquesta.workspace.control", map[string]any{"desde": "2026-04-24T13:00:00Z"})
	if err != nil {
		t.Fatalf("workspace control MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("workspace control marcado como error: %#v", result)
	}
	resp, _ := result["structuredContent"].(apiWorkspaceControlResponse)
	if resp.Control == nil {
		t.Fatalf("workspace control sin payload: %#v", result["structuredContent"])
	}
	if got := resp.Control.Since.Format(time.RFC3339); got != "2026-04-24T13:00:00Z" {
		t.Fatalf("since inesperado: %s", got)
	}
	if resp.Control.ActiveProjects != 1 || resp.Control.WorkersConectados != 2 {
		t.Fatalf("workspace control inesperado: %+v", resp.Control)
	}
}

func TestMCPToolWorkspaceControlRechazaDesdeInvalido(t *testing.T) {
	result, err := callMCPTool("orquesta.workspace.control", map[string]any{"desde": "xxx"})
	if err != nil {
		t.Fatalf("workspace control MCP debería devolver toolResult de error, err=%v", err)
	}
	if result["isError"] != true {
		t.Fatalf("workspace control debería marcar error: %#v", result)
	}
	content, _ := result["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatalf("workspace control sin contenido de error: %#v", result)
	}
	text, _ := content[0]["text"].(string)
	if !strings.Contains(text, "valor --desde inválido") {
		t.Fatalf("texto de error inesperado: %s", text)
	}
}
