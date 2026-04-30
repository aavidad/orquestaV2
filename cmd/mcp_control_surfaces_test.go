package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
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
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	prevRearmApply := serverOperationalApplyNextActionFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
		serverOperationalApplyNextActionFn = prevRearmApply
	}()

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	ordersCalls := 0
	mailboxCalls := 0
	degradadosCalls := 0
	autonomiaCalls := 0
	reanimacionesCalls := 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		ordersCalls++
		return 0, nil
	}
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		mailboxCalls++
		return 0, nil
	}
	runtimeProcessHygieneBatch = func() (int, error) { return 6, nil }
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{
			Count:                     4,
			GhostAssignmentsCompacted: 1,
			ReactivatedWithoutRuntime: 2,
			IdleAutoassigned:          1,
		}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 3, nil
	}
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		reanimacionesCalls++
		return apiRuntimeProcessReanimationsResponse{
			OK:                true,
			Count:             2,
			Reactivated:       2,
			CooldownSustained: 1,
		}
	}
	serverOperationalApplyNextActionFn = func(supervisor string) (map[string]any, error) {
		t.Fatalf("self_heal no deberia invocar rearm si el estado ya converge")
		return nil, nil
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
	if !payload.OK || payload.Hygiene.Count != 6 {
		t.Fatalf("payload self_heal inesperado: %+v", payload)
	}
	if !payload.Wake.Orders || !payload.Wake.Mailbox {
		t.Fatalf("wake self_heal inesperado: %+v", payload.Wake)
	}
	if payload.Operational.State != "ready" || !payload.Operational.Operational || payload.Operational.TasksInProgress != 3 {
		t.Fatalf("operational final inesperado: %+v", payload.Operational)
	}
	if ordersCalls != 0 || mailboxCalls != 0 {
		t.Fatalf("self_heal no deberia drenar runtime si ya converge: orders=%d mailbox=%d", ordersCalls, mailboxCalls)
	}
	if degradadosCalls != 0 || autonomiaCalls != 0 || reanimacionesCalls != 0 {
		t.Fatalf("self_heal no deberia ejecutar mantenimiento pesado si ya converge: degradados=%d autonomia=%d reanimaciones=%d", degradadosCalls, autonomiaCalls, reanimacionesCalls)
	}
	if payload.Drain != nil {
		t.Fatalf("self_heal no deberia incluir drain si ya converge: %+v", payload.Drain)
	}
	if payload.Degradados.Count != 0 || payload.Autonomia.Count != 0 || payload.Reanimaciones.Count != 0 {
		t.Fatalf("self_heal no deberia reportar mantenimiento pesado en estado sano: %+v", payload)
	}
	if !payload.Degradados.OK || !payload.Autonomia.OK || !payload.Reanimaciones.OK {
		t.Fatalf("self_heal deberia marcar como OK los carriles omitidos por short-circuit sano: %+v", payload)
	}
}

func TestMCPToolServerSelfHealAutoRearmaHastaConverger(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevRearmApply := serverOperationalApplyNextActionFn
	prevReview := serverOperationalReviewSnapshotFn
	prevLimit := mcpServerSelfHealRearmLimit
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		serverOperationalApplyNextActionFn = prevRearmApply
		serverOperationalReviewSnapshotFn = prevReview
		mcpServerSelfHealRearmLimit = prevLimit
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
	}()

	mcpServerSelfHealRearmLimit = 3
	mcpServerSelfHealDrainPassLimit = 1
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	serverOperationalReviewSnapshotFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"queue_kind": "safe",
			"next_safe_action": supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   "tarea:24",
				Action:   "reservar_tarea_libre",
				Priority: "baja",
				Assignee: "Codex10",
			},
		}, nil
	}
	operationalCalls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		operationalCalls++
		switch operationalCalls {
		case 1:
			return serverOperationalInfo{
				State:       "degraded",
				Operational: false,
				Reason:      "tasks_without_workers",
				Rearm: &serverOperationalRearmHint{
					Needed:     true,
					Available:  true,
					Supervisor: "OpenClaw",
				},
			}
		case 2:
			return serverOperationalInfo{
				State:       "degraded",
				Operational: false,
				Reason:      "workers_stuck",
				Rearm: &serverOperationalRearmHint{
					Needed:     true,
					Available:  true,
					Supervisor: "OpenClaw",
				},
			}
		default:
			return serverOperationalInfo{
				State:        "ready",
				Operational:  true,
				Reason:       "control_plane_responsive",
				ActiveAgents: 4,
			}
		}
	}

	var rearms []string
	serverOperationalApplyNextActionFn = func(supervisor string) (map[string]any, error) {
		rearms = append(rearms, supervisor)
		return map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"queue_kind": "safe",
			"task_id":    int64(len(rearms)),
		}, nil
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"orders":        false,
		"mailbox":       false,
		"warm":          false,
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras converger: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if !payload.OK || !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia converger tras rearm seguro: %+v", payload)
	}
	if len(rearms) < 1 {
		t.Fatalf("deberia aplicar al menos un rearm seguro antes de converger, got=%d", len(rearms))
	}
	if len(payload.RearmApplied) < 1 {
		t.Fatalf("rearmApplied inesperado: %+v", payload.RearmApplied)
	}
}

func TestMCPToolServerSelfHealMarcaErrorSiRearmNoConvergeDentroDelLimite(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevRearmApply := serverOperationalApplyNextActionFn
	prevReview := serverOperationalReviewSnapshotFn
	prevLimit := mcpServerSelfHealRearmLimit
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		serverOperationalApplyNextActionFn = prevRearmApply
		serverOperationalReviewSnapshotFn = prevReview
		mcpServerSelfHealRearmLimit = prevLimit
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	mcpServerSelfHealRearmLimit = 2
	mcpServerSelfHealDrainPassLimit = 1
	mcpServerSelfHealSettleAttempts = 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	serverOperationalReviewSnapshotFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"queue_kind": "safe",
			"next_safe_action": supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   "tarea:24",
				Action:   "reservar_tarea_libre",
				Priority: "baja",
				Assignee: "Codex10",
			},
		}, nil
	}
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:       "degraded",
			Operational: false,
			Reason:      "tasks_without_workers",
			Rearm: &serverOperationalRearmHint{
				Needed:     true,
				Available:  true,
				Supervisor: "OpenClaw",
			},
		}
	}
	attempts := 0
	serverOperationalApplyNextActionFn = func(supervisor string) (map[string]any, error) {
		attempts++
		return map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"queue_kind": "safe",
		}, nil
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"orders":        false,
		"mailbox":       false,
		"warm":          false,
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != true {
		t.Fatalf("server self_heal deberia marcar error si no converge el rearm: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.OK || attempts != 2 {
		t.Fatalf("self_heal deberia cortar en el limite de rearm: attempts=%d payload=%+v", attempts, payload)
	}
	if len(payload.Errors) == 0 || !strings.Contains(payload.Errors[len(payload.Errors)-1], "rearm: safe rearm limit reached before convergence") {
		t.Fatalf("error de rearm inesperado: %+v", payload.Errors)
	}
}

func TestMCPToolServerSelfHealReevaluaOperationalAntesDeResponder(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	mcpServerSelfHealDrainPassLimit = 1
	mcpServerSelfHealSettleAttempts = 2
	mcpServerSelfHealSettleDelay = 0
	calls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		calls++
		if calls == 1 {
			return serverOperationalInfo{
				State:       "degraded",
				Operational: false,
				Reason:      "tasks_without_workers",
			}
		}
		return serverOperationalInfo{
			State:        "ready",
			Operational:  true,
			Reason:       "control_plane_responsive",
			ActiveAgents: 2,
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"orders":        false,
		"mailbox":       false,
		"warm":          false,
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras reevaluar operational: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia devolver operational convergente tras settle: %+v", payload.Operational)
	}
}

func TestMCPToolServerSelfHealDrenaOrdersYMailboxHastaCerrarTasksWithoutWorkers(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	mcpServerSelfHealDrainPassLimit = 3
	mcpServerSelfHealSettleAttempts = 0
	mcpServerSelfHealSettleDelay = 0
	operationalCalls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		operationalCalls++
		if operationalCalls == 1 {
			return serverOperationalInfo{
				State:       "degraded",
				Operational: false,
				Reason:      "tasks_without_workers",
			}
		}
		return serverOperationalInfo{
			State:        "ready",
			Operational:  true,
			Reason:       "control_plane_responsive",
			ActiveAgents: 1,
		}
	}
	orderCalls := 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		orderCalls++
		if orderCalls == 1 {
			return 1, nil
		}
		return 0, nil
	}
	mailboxCalls := 0
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		mailboxCalls++
		if mailboxCalls == 1 {
			return 1, nil
		}
		return 0, nil
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras drenar recovery: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.Drain == nil || payload.Drain.Orders != 1 || payload.Drain.Mailbox != 1 || payload.Drain.Passes != 1 {
		t.Fatalf("drain inesperado: %+v", payload.Drain)
	}
	if !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia converger tras drenar orders/mailbox: %+v", payload.Operational)
	}
	if payload.NextRecoveryAction != nil || payload.NextRecoveryPlan != nil {
		t.Fatalf("no deberia quedar recovery pendiente tras converger: action=%+v plan=%+v", payload.NextRecoveryAction, payload.NextRecoveryPlan)
	}
}

func TestMCPToolServerSelfHealExponeNextRecoveryPlanSiSigueDegradado(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	mcpServerSelfHealDrainPassLimit = 1
	mcpServerSelfHealSettleAttempts = 0
	mcpServerSelfHealSettleDelay = 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:            "degraded",
			Operational:      false,
			Reason:           "workers_quota_blocked",
			NextQuotaResetAt: "2026-04-29T10:00:00Z",
			Recovery: &serverOperationalRecoveryHint{
				Kind:            "quota_cooldown",
				SuggestedAction: "wait_quota_reset",
				Detail:          "2 worker(s) bloqueados por cuota",
			},
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.NextRecoveryAction == nil || payload.NextRecoveryAction.Action != "wait_quota_reset" {
		t.Fatalf("next_recovery_action inesperada: %+v", payload.NextRecoveryAction)
	}
	if payload.NextRecoveryPlan == nil || payload.NextRecoveryPlan.Action != "wait_quota_reset" || payload.NextRecoveryPlan.NextQuotaResetAt != "2026-04-29T10:00:00Z" {
		t.Fatalf("next_recovery_plan inesperado: %+v", payload.NextRecoveryPlan)
	}
	if len(payload.NextRecoveryPlan.Steps) != 2 || payload.NextRecoveryPlan.Steps[0].Action != "wait_quota_reset" || payload.NextRecoveryPlan.Steps[1].Action != "server_operational_refresh" {
		t.Fatalf("next_recovery_plan inesperado: %+v", payload.NextRecoveryPlan)
	}
	if payload.Operational.NextRecoveryPlan == nil || payload.Operational.NextRecoveryPlan.Action != "wait_quota_reset" {
		t.Fatalf("operational sin recovery plan canónico: %+v", payload.Operational)
	}
}

func TestSettleMCPServerSelfHealOperationalOmiteEsperaEnRecoveryManual(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	calls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		calls++
		return serverOperationalInfo{
			State:       "degraded",
			Operational: false,
			Reason:      "workers_stuck",
			NextRecoveryPlan: &serverOperationalRecoveryPlan{
				Kind:           "stuck_workers",
				Action:         "inspect_stuck_workers",
				AutoExecutable: false,
			},
		}
	}
	mcpServerSelfHealSettleAttempts = 3
	mcpServerSelfHealSettleDelay = 0

	info := settleMCPServerSelfHealOperational(serverOperationalInfo{
		State:       "degraded",
		Operational: false,
		Reason:      "workers_stuck",
		NextRecoveryPlan: &serverOperationalRecoveryPlan{
			Kind:           "stuck_workers",
			Action:         "inspect_stuck_workers",
			AutoExecutable: false,
		},
	})
	if info.Reason != "workers_stuck" {
		t.Fatalf("settle no deberia mutar recovery manual: %+v", info)
	}
	if calls != 0 {
		t.Fatalf("settle no deberia relanzar operational cuando el recovery restante es manual, calls=%d", calls)
	}
}

func TestMCPToolServerSelfHealCortaMantenimientoPesadoEnWaitQuotaReset(t *testing.T) {
	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	runtimeProcessHygieneBatch = func() (int, error) { return 1, nil }

	ordersCalls := 0
	mailboxCalls := 0
	degradadosCalls := 0
	autonomiaCalls := 0
	reanimacionesCalls := 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		ordersCalls++
		return 0, nil
	}
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		mailboxCalls++
		return 0, nil
	}
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{Count: 99}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 77, nil
	}
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		reanimacionesCalls++
		return apiRuntimeProcessReanimationsResponse{OK: true, Count: 55}
	}
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:            "degraded",
			Operational:      false,
			Reason:           "tasks_without_workers",
			NextQuotaResetAt: "2026-04-29T10:00:00Z",
			QuotaAgents:      1,
			TasksInProgress:  2,
			ConnectedWorkers: 0,
			WorkingWorkers:   0,
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", nil)
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error en espera pasiva: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.NextRecoveryAction == nil || payload.NextRecoveryAction.Action != "wait_quota_reset" {
		t.Fatalf("next_recovery_action inesperada: %+v", payload.NextRecoveryAction)
	}
	if degradadosCalls != 0 || autonomiaCalls != 0 || reanimacionesCalls != 0 {
		t.Fatalf("self_heal no deberia ejecutar mantenimiento pesado en wait_quota_reset: degradados=%d autonomia=%d reanimaciones=%d", degradadosCalls, autonomiaCalls, reanimacionesCalls)
	}
	if ordersCalls != 0 || mailboxCalls != 0 {
		t.Fatalf("self_heal no deberia drenar runtime en wait_quota_reset pasivo: orders=%d mailbox=%d", ordersCalls, mailboxCalls)
	}
	if payload.Drain != nil {
		t.Fatalf("self_heal no deberia incluir drain en wait_quota_reset pasivo: %+v", payload.Drain)
	}
	if !payload.Degradados.OK || !payload.Autonomia.OK || !payload.Reanimaciones.OK {
		t.Fatalf("self_heal deberia marcar como OK los carriles omitidos en wait_quota_reset: %+v", payload)
	}
}

func TestMCPToolServerSelfHealCortaMantenimientoPesadoEnRecoveryManual(t *testing.T) {
	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimaciones := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	prevHygiene := runtimeProcessHygieneBatch
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimaciones
		mcpBuildServerOperationalInfoFn = prevOperational
		runtimeProcessHygieneBatch = prevHygiene
	}()

	runtimeProcessHygieneBatch = func() (int, error) { return 1, nil }

	ordersCalls := 0
	mailboxCalls := 0
	degradadosCalls := 0
	autonomiaCalls := 0
	reanimacionesCalls := 0
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		ordersCalls++
		return 0, nil
	}
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		mailboxCalls++
		return 0, nil
	}
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{Count: 99}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 77, nil
	}
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		reanimacionesCalls++
		return apiRuntimeProcessReanimationsResponse{OK: true, Count: 55}
	}
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:            "degraded",
			Operational:      false,
			Reason:           "workers_stuck",
			TasksInProgress:  1,
			ConnectedWorkers: 2,
			WorkingWorkers:   1,
			NextRecoveryPlan: &serverOperationalRecoveryPlan{Kind: "stuck_workers", Action: "inspect_stuck_workers", AutoExecutable: false},
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", nil)
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error en recovery manual: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.NextRecoveryPlan == nil || payload.NextRecoveryPlan.Action != "inspect_stuck_workers" {
		t.Fatalf("next_recovery_plan inesperado: %+v", payload.NextRecoveryPlan)
	}
	if degradadosCalls != 0 || autonomiaCalls != 0 || reanimacionesCalls != 0 {
		t.Fatalf("self_heal no deberia ejecutar mantenimiento pesado en recovery manual: degradados=%d autonomia=%d reanimaciones=%d", degradadosCalls, autonomiaCalls, reanimacionesCalls)
	}
	if ordersCalls != 0 || mailboxCalls != 0 {
		t.Fatalf("self_heal no deberia drenar runtime en recovery manual: orders=%d mailbox=%d", ordersCalls, mailboxCalls)
	}
	if !payload.Degradados.OK || !payload.Autonomia.OK || !payload.Reanimaciones.OK {
		t.Fatalf("self_heal deberia marcar como OK los carriles omitidos en recovery manual: %+v", payload)
	}
}

func TestMCPToolServerSelfHealAutoEjecutaCompactionDebt(t *testing.T) {
	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	runtimeProcessHygieneBatch = func() (int, error) { return 0, nil }
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{OK: true}
	}

	degradadosCalls := 0
	autonomiaCalls := 0
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{Count: 1, GhostAssignmentsCompacted: 1}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 1, nil
	}

	operationalCalls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		operationalCalls++
		if operationalCalls <= 2 {
			return serverOperationalInfo{
				State:                "degraded",
				Operational:          false,
				Reason:               "tasks_without_workers",
				NextQuotaResetAt:     "2026-04-29T10:00:00Z",
				TasksInProgress:      3,
				WorkingWorkers:       1,
				ConnectedWorkers:     1,
				CompactionDebtAgents: 1,
				CompactionDebtTasks:  1,
			}
		}
		return serverOperationalInfo{
			State:       "ready",
			Operational: true,
			Reason:      "control_plane_responsive",
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras autoejecutar compactacion: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if degradadosCalls < 2 || autonomiaCalls < 2 {
		t.Fatalf("self_heal deberia reintentar degradados/autonomia para compaction debt: degradados=%d autonomia=%d", degradadosCalls, autonomiaCalls)
	}
	if !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia converger tras compactacion autoejecutable: %+v", payload.Operational)
	}
	if payload.NextRecoveryAction != nil || payload.NextRecoveryPlan != nil {
		t.Fatalf("no deberia quedar recovery pendiente tras compactacion: action=%+v plan=%+v", payload.NextRecoveryAction, payload.NextRecoveryPlan)
	}
}

func TestMCPToolServerSelfHealAutoEjecutaBlockedFronts(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-selfheal-blocked-front",
		Nombre:  "Orquestador SelfHeal Blocked Front",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloq", "CodexIdle"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente bloqueado"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado self-heal",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloq"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloq"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "orquesta", "Agente CodexBloq en estado bloqueado: runtime stale sin worker"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexBloq", Activo: false, EstadoCuota: "activo"},
			Asignacion:      &db.Asignacion{Agente: "CodexBloq", ProyectoID: proyectoID, ProyectoSlug: "orquestador-selfheal-blocked-front", Estado: db.AsignacionActiva},
			EstadoOperativo: "bloqueado",
			BlockedTasks:    1,
			CurrentTask:     &agentesapp.TaskFocus{TaskID: tareaID, State: db.TareaBloqueada, ProjectID: &proyectoID},
		},
		{
			Agente:          &db.Agente{Nombre: "CodexIdle", Activo: false, EstadoCuota: "activo"},
			Asignacion:      &db.Asignacion{Agente: "CodexIdle", ProyectoID: proyectoID, ProyectoSlug: "orquestador-selfheal-blocked-front", Estado: db.AsignacionActiva},
			EstadoOperativo: "sin_tarea",
			OpenTasks:       0,
			BlockedTasks:    0,
			WorkerAlive:     false,
		},
	}, now)

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	runtimeProcessHygieneBatch = func() (int, error) { return 0, nil }
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{OK: true}
	}

	degradadosCalls := 0
	autonomiaCalls := 0
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{Count: 1}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 1, nil
	}

	operationalCalls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		operationalCalls++
		if operationalCalls <= 2 {
			return buildServerOperationalInfo(apiStatusResponse{
				AgentesActivos: []*db.Agente{
					{Nombre: "CodexIdle", Activo: true},
				},
				TareasActivas: []tareaLite{
					{ID: tareaID, Estado: db.TareaBloqueada, Agente: "CodexBloq"},
				},
				TareasPorEstado: map[string]int{
					string(db.TareaBloqueada): 1,
				},
			})
		}
		return serverOperationalInfo{
			State:       "ready",
			Operational: true,
			Reason:      "control_plane_responsive",
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras autoejecutar blocked fronts: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if degradadosCalls < 2 || autonomiaCalls < 2 {
		t.Fatalf("self_heal deberia reintentar degradados/autonomia para blocked fronts: degradados=%d autonomia=%d", degradadosCalls, autonomiaCalls)
	}
	if !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia converger tras blocked fronts autoejecutable: %+v", payload.Operational)
	}
	if payload.NextRecoveryAction != nil || payload.NextRecoveryPlan != nil {
		t.Fatalf("no deberia quedar recovery pendiente tras blocked fronts: action=%+v plan=%+v", payload.NextRecoveryAction, payload.NextRecoveryPlan)
	}
}

func TestMCPToolServerSelfHealAutoEjecutaStuckWorkers(t *testing.T) {
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevWakeOrders := apiRuntimeWakeOrdersFn
	prevWakeMailbox := apiRuntimeWakeMailboxFn
	prevWakeWarm := apiRuntimeWakeWarmFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevHygiene := runtimeProcessHygieneBatch
	prevDegradados := runtimeProcessDegradadosBatchDetailed
	prevAutonomia := runtimeProcessAutonomiaBatch
	prevReanimations := runtimeProcessReanimationsBatchFn
	prevOperational := mcpBuildServerOperationalInfoFn
	defer func() {
		apiRuntimeWakeOrdersFn = prevWakeOrders
		apiRuntimeWakeMailboxFn = prevWakeMailbox
		apiRuntimeWakeWarmFn = prevWakeWarm
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		runtimeProcessHygieneBatch = prevHygiene
		runtimeProcessDegradadosBatchDetailed = prevDegradados
		runtimeProcessAutonomiaBatch = prevAutonomia
		runtimeProcessReanimationsBatchFn = prevReanimations
		mcpBuildServerOperationalInfoFn = prevOperational
	}()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexTMUX", Activo: true, EstadoCuota: "activo"},
		Handle:          &db.RuntimeHandle{Estado: "activo", Transporte: "tmux"},
		EstadoOperativo: "atascado",
		OpenTasks:       1,
		WorkerAlive:     true,
		WorkerState:     "running",
		WorkerDriver:    "tmux_cli_session",
		WorkerHeartbeat: &now,
		WorkerUpdatedAt: &now,
		CurrentTask:     &agentesapp.TaskFocus{TaskID: 40, State: db.TareaEnProgreso},
	}}, now)

	apiRuntimeWakeOrdersFn = func() bool { return true }
	apiRuntimeWakeMailboxFn = func() bool { return true }
	apiRuntimeWakeWarmFn = func() bool { return false }
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	runtimeProcessHygieneBatch = func() (int, error) { return 0, nil }
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{OK: true}
	}

	degradadosCalls := 0
	autonomiaCalls := 0
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		degradadosCalls++
		return runtimeProcessDegradadosSummary{Count: 1}, nil
	}
	runtimeProcessAutonomiaBatch = func() (int, error) {
		autonomiaCalls++
		return 1, nil
	}

	operationalCalls := 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		operationalCalls++
		if operationalCalls <= 2 {
			return buildServerOperationalInfo(apiStatusResponse{
				AgentesActivos:    []*db.Agente{{Nombre: "CodexTMUX", Activo: true}},
				AgentesTrabajando: []*db.Agente{{Nombre: "CodexTMUX", Activo: true}},
				AgentesAtascados:  []*db.Agente{{Nombre: "CodexTMUX", Activo: true}},
				TareasEnProgreso:  []tareaLite{{ID: 40, Estado: db.TareaEnProgreso, Agente: "CodexTMUX"}},
				Autonomia:         autonomiaResumen{WorkConfirmed: 0},
			})
		}
		return serverOperationalInfo{
			State:       "ready",
			Operational: true,
			Reason:      "control_plane_responsive",
		}
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != false {
		t.Fatalf("server self_heal no deberia marcar error tras autoejecutar stuck workers: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if degradadosCalls < 2 || autonomiaCalls < 2 {
		t.Fatalf("self_heal deberia reintentar degradados/autonomia para stuck workers: degradados=%d autonomia=%d", degradadosCalls, autonomiaCalls)
	}
	if !payload.Operational.Operational || payload.Operational.State != "ready" {
		t.Fatalf("self_heal deberia converger tras stuck workers autoejecutable: %+v", payload.Operational)
	}
	if payload.NextRecoveryAction != nil || payload.NextRecoveryPlan != nil {
		t.Fatalf("no deberia quedar recovery pendiente tras stuck workers: action=%+v plan=%+v", payload.NextRecoveryAction, payload.NextRecoveryPlan)
	}
}

func TestNextSupervisorStartOrAssignActionUsaDispatchVisibleDelSnapshot(t *testing.T) {
	prevBuilder := supervisorActionSnapshotBuilder
	defer func() {
		supervisorActionSnapshotBuilder = prevBuilder
	}()
	supervisorActionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"action_queue": []supervisorRecommendedAction{{
				Kind:     "dispatch",
				Target:   "backlog:libre",
				Action:   "asignar_tarea_libre",
				Reason:   "hay frente arrancable sin worker",
				Priority: "media",
				Assignee: "Codex12",
			}},
			"safe_action_queue": []supervisorRecommendedAction{},
			"server_operational": serverOperationalInfo{
				State:       "degraded",
				Operational: false,
				Reason:      "tasks_without_workers",
			},
		}, nil
	}

	action, err := nextSupervisorStartOrAssignAction("OpenClaw")
	if err != nil {
		t.Fatalf("nextSupervisorStartOrAssignAction: %v", err)
	}
	if action == nil || action.Action != "asignar_tarea_libre" || action.Target != "backlog:libre" || !strings.EqualFold(strings.TrimSpace(action.Assignee), "Codex12") {
		t.Fatalf("accion start_or_assign inesperada: %+v", action)
	}
}

func TestMCPToolServerSelfHealMarcaErrorSiDrainOrdersFalla(t *testing.T) {
	prevOperational := mcpBuildServerOperationalInfoFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	prevSettleAttempts := mcpServerSelfHealSettleAttempts
	prevSettleDelay := mcpServerSelfHealSettleDelay
	defer func() {
		mcpBuildServerOperationalInfoFn = prevOperational
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
		mcpServerSelfHealSettleAttempts = prevSettleAttempts
		mcpServerSelfHealSettleDelay = prevSettleDelay
	}()

	mcpServerSelfHealDrainPassLimit = 1
	mcpServerSelfHealSettleAttempts = 0
	mcpServerSelfHealSettleDelay = 0
	mcpBuildServerOperationalInfoFn = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:       "degraded",
			Operational: false,
			Reason:      "tasks_without_workers",
		}
	}
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, fmt.Errorf("queue jammed") }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		t.Fatalf("mailbox no deberia ejecutarse si process-orders ya falló")
		return 0, nil
	}

	result, err := callMCPTool("orquesta.server.self_heal", map[string]any{
		"hygiene":       false,
		"degradados":    false,
		"autonomia":     false,
		"reanimaciones": false,
		"rearm":         false,
	})
	if err != nil {
		t.Fatalf("server self_heal MCP: %v", err)
	}
	if result["isError"] != true {
		t.Fatalf("server self_heal deberia marcar error si falla el drain: %#v", result)
	}
	payload, _ := result["structuredContent"].(apiRuntimeSelfHealResponse)
	if payload.OK || len(payload.Errors) == 0 || !strings.Contains(payload.Errors[0], "drain: orders: queue jammed") {
		t.Fatalf("error de drain inesperado: %+v", payload)
	}
}

func TestMCPToolServerSelfHealMarcaErroresParciales(t *testing.T) {
	prevHygiene := runtimeProcessHygieneBatch
	prevOperational := mcpBuildServerOperationalInfoFn
	prevOrdersExec := apiRuntimeProcessOrdersExecutor
	prevMailboxExec := apiRuntimeProcessMailboxExecutor
	prevDrainLimit := mcpServerSelfHealDrainPassLimit
	defer func() {
		runtimeProcessHygieneBatch = prevHygiene
		mcpBuildServerOperationalInfoFn = prevOperational
		apiRuntimeProcessOrdersExecutor = prevOrdersExec
		apiRuntimeProcessMailboxExecutor = prevMailboxExec
		mcpServerSelfHealDrainPassLimit = prevDrainLimit
	}()

	runtimeProcessHygieneBatch = func() (int, error) { return 0, fmt.Errorf("boom") }
	apiRuntimeProcessOrdersExecutor = func() (int, error) { return 0, nil }
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) { return 0, nil }
	mcpServerSelfHealDrainPassLimit = 1
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
