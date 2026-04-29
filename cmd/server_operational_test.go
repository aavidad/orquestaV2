package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func ptrTimeServerOperational(v time.Time) *time.Time { return &v }

func TestBuildServerOperationalInfoToleraTrabajoConfirmadoSinWorkingAgents(t *testing.T) {
	info := buildServerOperationalInfo(apiStatusResponse{
		TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			WorkConfirmed: 1,
		},
	})
	if !info.Operational || info.State != "ready" {
		t.Fatalf("no deberia degradar si la autonomia ya confirmo trabajo: %+v", info)
	}
}

func TestBuildServerOperationalInfoDegradaSiSoloQuedaSupervisorConTareasEnProgreso(t *testing.T) {
	info := buildServerOperationalInfo(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex2", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex2", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			Supervisando: 1,
		},
	})
	if info.Operational || info.State != "degraded" || info.Reason != "tasks_without_workers" {
		t.Fatalf("deberia degradar si solo queda supervisor sin workers reales: %+v", info)
	}
}

func TestBuildServerOperationalInfoDegradaReservadasSiSoloQuedaSupervisor(t *testing.T) {
	info := buildServerOperationalInfo(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex2", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex2", Activo: true}},
		TareasReservadas:  []tareaLite{{ID: 2, Estado: db.TareaAsignada}},
		Autonomia: autonomiaResumen{
			Supervisando: 1,
		},
	})
	if info.Operational || info.State != "degraded" || info.Reason != "reserved_without_connected_workers" {
		t.Fatalf("deberia degradar reservadas si solo queda supervisor: %+v", info)
	}
}

func TestBuildServerOperationalInfoPreservaWorkersSiSupervisorRealNoEstaEnSliceVisible(t *testing.T) {
	autonomia := autonomiaResumen{Supervisando: 1, WorkConfirmed: 2}
	autonomia.addSupervisorName("Codex2")
	info := buildServerOperationalInfo(apiStatusResponse{
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex1", Activo: true},
			{Nombre: "Codex4", Activo: true},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex1", Activo: true},
			{Nombre: "Codex4", Activo: true},
		},
		TareasEnProgreso: []tareaLite{
			{ID: 24, Estado: db.TareaEnProgreso, Agente: "Codex4"},
			{ID: 25, Estado: db.TareaEnProgreso, Agente: "Codex1"},
		},
		Autonomia: autonomia,
	})
	if info.ConnectedWorkers != 2 || info.WorkingWorkers != 2 {
		t.Fatalf("deberia preservar workers visibles reales aunque haya supervisor fuera del slice: %+v", info)
	}
}

func TestBuildServerOperationalInfoExponeQuotaBlockedSinWorkersActivos(t *testing.T) {
	reset := time.Now().UTC().Add(30 * time.Minute)
	info := buildServerOperationalInfo(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
	})
	if !info.Operational {
		t.Fatalf("el control plane sigue operativo aunque la flota este en cuota: %+v", info)
	}
	if info.State != "idle" || info.Reason != "workers_quota_blocked" || info.QuotaAgents != 1 {
		t.Fatalf("estado operativo inesperado: %+v", info)
	}
	if info.NextQuotaResetAt == "" {
		t.Fatalf("deberia exponer el siguiente quota reset visible: %+v", info)
	}
}

func TestBuildServerOperationalInfoNoMarcaIdleSiQuedaTrabajoSinWorkersAunqueHayaCuota(t *testing.T) {
	reset := time.Now().UTC().Add(30 * time.Minute)
	info := buildServerOperationalInfo(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
		TareasEnProgreso: []tareaLite{{ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	})
	if info.Operational || info.State != "degraded" || info.Reason != "tasks_without_workers" {
		t.Fatalf("no deberia marcar idle si queda trabajo en progreso sin workers: %+v", info)
	}
}

func TestBuildServerOperationalInfoPreservaRiesgoCanonicoEstructuradoDelStatus(t *testing.T) {
	info := buildServerOperationalInfo(apiStatusResponse{
		AutonomyHighlights: []string{"integracion_bloqueada=9", "riesgo_top=infra(9)"},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "infra",
			Blocking:   9,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=9", "runtime_orders=1"},
		},
	})
	if info.CriticalProjectRisk == nil || info.CriticalProjectRisk.Project != "infra" || info.CriticalProjectRisk.Blocking != 9 {
		t.Fatalf("deberia preservar el riesgo crítico estructurado: %+v", info.CriticalProjectRisk)
	}
	if len(info.AutonomyHighlights) != 2 || info.AutonomyHighlights[0] != "integracion_bloqueada=9" {
		t.Fatalf("deberia preservar highlights canónicos: %+v", info.AutonomyHighlights)
	}
}

func TestServerOperationalRiskContextNormalizaHighlightsDesdeRiesgoEstructurado(t *testing.T) {
	highlights, risk := serverOperationalRiskContext(&serverOperationalInfo{
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "infra",
			Blocking:   11,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=11", "runtime_orders=2"},
		},
	})
	if risk == nil || risk.Project != "infra" || risk.Blocking != 11 {
		t.Fatalf("riesgo crítico inesperado: %+v", risk)
	}
	if len(highlights) == 0 {
		t.Fatalf("deberia derivar highlights canónicos desde riesgo estructurado")
	}
	if !containsStringWorkspace(highlights, "integracion_bloqueada=11") || !containsStringWorkspace(highlights, "riesgo_top=infra(11)") {
		t.Fatalf("highlights canónicos inesperados: %+v", highlights)
	}
}

func TestServerOperationalRiskContextPreservaHighlightsCanonicosExistentes(t *testing.T) {
	highlights, risk := serverOperationalRiskContext(&serverOperationalInfo{
		AutonomyHighlights: []string{"integracion_bloqueada=8", "riesgo_top=orquestador(8)"},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "orquestador",
			Blocking:   8,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=8"},
		},
	})
	if risk == nil || risk.Project != "orquestador" || risk.Blocking != 8 {
		t.Fatalf("riesgo crítico inesperado: %+v", risk)
	}
	if len(highlights) != 2 || highlights[0] != "integracion_bloqueada=8" || highlights[1] != "riesgo_top=orquestador(8)" {
		t.Fatalf("deberia preservar highlights ya normalizados: %+v", highlights)
	}
}

func TestNormalizeServerOperationalInfoDerivaHighlightsDesdeRiesgoEstructurado(t *testing.T) {
	info := normalizeServerOperationalInfo(serverOperationalInfo{
		State:       "degraded",
		Operational: false,
		Reason:      "status_temporarily_degraded",
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "infra",
			Blocking:   12,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=12", "runtime_orders=3"},
		},
	})
	if info.CriticalProjectRisk == nil || info.CriticalProjectRisk.Project != "infra" || info.CriticalProjectRisk.Blocking != 12 {
		t.Fatalf("riesgo crítico inesperado: %+v", info.CriticalProjectRisk)
	}
	if !containsStringWorkspace(info.AutonomyHighlights, "integracion_bloqueada=12") || !containsStringWorkspace(info.AutonomyHighlights, "riesgo_top=infra(12)") {
		t.Fatalf("highlights canónicos inesperados: %+v", info.AutonomyHighlights)
	}
}

func TestNormalizeServerOperationalInfoDerivaRecoveryHintParaTasksWithoutWorkers(t *testing.T) {
	prevReview := serverOperationalReviewSnapshotFn
	t.Cleanup(func() {
		serverOperationalReviewSnapshotFn = prevReview
	})
	serverOperationalReviewSnapshotFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"supervisor": supervisor,
			"next_safe_action": supervisorRecommendedAction{
				Target: "tarea:24",
				Action: "reanudar_worker",
				Reason: "worker parado con backlog pendiente",
			},
		}, nil
	}

	info := normalizeServerOperationalInfo(serverOperationalInfo{
		State:            "degraded",
		Operational:      false,
		Reason:           "tasks_without_workers",
		TasksInProgress:  3,
		WorkingWorkers:   1,
		QuotaAgents:      1,
		NextQuotaResetAt: "2026-04-29T10:30:00Z",
	})
	if info.Recovery == nil {
		t.Fatalf("recovery hint ausente: %+v", info)
	}
	if info.Recovery.Kind != "worker_gap" || info.Recovery.MissingWorkers != 2 || info.Recovery.AffectedTasks != 3 {
		t.Fatalf("recovery hint inesperado: %+v", info.Recovery)
	}
	if info.Recovery.SuggestedAction != "server_rearm" || !info.Recovery.RearmAvailable {
		t.Fatalf("recovery hint sin accion canónica: %+v", info.Recovery)
	}
}

func TestNormalizeServerOperationalInfoDerivaRecoveryHintDeCuotaSinRearm(t *testing.T) {
	info := normalizeServerOperationalInfo(serverOperationalInfo{
		State:            "idle",
		Operational:      true,
		Reason:           "workers_quota_blocked",
		QuotaAgents:      2,
		NextQuotaResetAt: "2026-04-29T11:00:00Z",
	})
	if info.Recovery == nil {
		t.Fatalf("recovery hint ausente: %+v", info)
	}
	if info.Recovery.Kind != "quota_cooldown" || info.Recovery.QuotaBlockedAgents != 2 {
		t.Fatalf("recovery quota inesperado: %+v", info.Recovery)
	}
	if info.Recovery.SuggestedAction != "wait_quota_reset" {
		t.Fatalf("accion recovery inesperada: %+v", info.Recovery)
	}
}

func TestControlPlaneQuotaBlockedIdleGuardSaltaSinWorkersUtiles(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	resetAt := now.Add(30 * time.Minute)
	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 2}, nil
	}

	skip, motivo, err := controlPlaneQuotaBlockedIdleGuard()
	if err != nil {
		t.Fatalf("controlPlaneQuotaBlockedIdleGuard: %v", err)
	}
	if !skip || motivo != "workers_quota_blocked" {
		t.Fatalf("guard inesperado skip=%v motivo=%q", skip, motivo)
	}
}

func TestControlPlaneQuotaBlockedIdleGuardNoSaltaSiHayWorkerConectado(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	skip, motivo, err := controlPlaneQuotaBlockedIdleGuard()
	if err != nil {
		t.Fatalf("controlPlaneQuotaBlockedIdleGuard: %v", err)
	}
	if skip || motivo != "" {
		t.Fatalf("no deberia saltar con worker conectado: skip=%v motivo=%q", skip, motivo)
	}
}

func TestControlPlaneQuotaBlockedIdleGuardNoSaltaSiHayReservadas(t *testing.T) {
	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento"},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento"},
		},
		TareasReservadas: []tareaLite{{ID: 2, Estado: db.TareaAsignada}},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)
	defer resetStatusSnapshotCache()

	skip, motivo, err := controlPlaneQuotaBlockedIdleGuard()
	if err != nil {
		t.Fatalf("controlPlaneQuotaBlockedIdleGuard: %v", err)
	}
	if skip || motivo != "" {
		t.Fatalf("no deberia saltar con reservadas: skip=%v motivo=%q", skip, motivo)
	}
}

func TestControlPlaneQuotaBlockedIdleGuardNoSaltaSiHayDispatchPendiente(t *testing.T) {
	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento"},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento"},
		},
		DeudaDispatch: deudaDispatchResumen{Pendientes: 1, Total: 1},
		Generado:      now.Format(time.RFC3339),
	}, now, time.Minute)
	defer resetStatusSnapshotCache()

	skip, motivo, err := controlPlaneQuotaBlockedIdleGuard()
	if err != nil {
		t.Fatalf("controlPlaneQuotaBlockedIdleGuard: %v", err)
	}
	if skip || motivo != "" {
		t.Fatalf("no deberia saltar con dispatch pendiente: skip=%v motivo=%q", skip, motivo)
	}
}

func TestControlPlaneQuotaBlockedIdleGuardUsaFetcherLigeroInyectable(t *testing.T) {
	prev := controlPlaneOperationalInfoFetcher
	defer func() { controlPlaneOperationalInfoFetcher = prev }()

	controlPlaneOperationalInfoFetcher = func() (serverOperationalInfo, error) {
		return serverOperationalInfo{
			State:            "idle",
			Reason:           "workers_quota_blocked",
			ConnectedWorkers: 0,
			WorkingWorkers:   0,
			StuckAgents:      0,
			AuthAgents:       0,
			ReservedTasks:    0,
			BlockedTasks:     0,
		}, nil
	}

	skip, motivo, err := controlPlaneQuotaBlockedIdleGuard()
	if err != nil {
		t.Fatalf("controlPlaneQuotaBlockedIdleGuard: %v", err)
	}
	if !skip || motivo != "workers_quota_blocked" {
		t.Fatalf("guard inesperado con fetcher inyectable skip=%v motivo=%q", skip, motivo)
	}
}

func TestBuildServerOperationalInfoUsaCuotaVisibleAunqueSnapshotNoTraigaQuotaBlockedExplicito(t *testing.T) {
	reset := time.Now().UTC().Add(20 * time.Minute)
	info := buildServerOperationalInfo(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &reset},
		},
	})
	if info.QuotaAgents != 1 {
		t.Fatalf("deberia derivar quota agents desde el snapshot visible: %+v", info)
	}
	if info.NextQuotaResetAt == "" {
		t.Fatalf("deberia derivar el próximo reset visible aunque no venga agentesQuotaBlocked: %+v", info)
	}
}

func TestFormatServerOperationalSummaryIncluyeDispatch(t *testing.T) {
	summary := formatServerOperationalSummary(&serverOperationalInfo{
		ActiveAgents:        2,
		RegisteredAgents:    4,
		DispatchPending:     1,
		DispatchNotified:    2,
		DispatchFailed:      3,
		DispatchConfirmed:   4,
		AutonomySupervising: 1,
		AutonomyContinuing:  2,
		AutonomyPending:     3,
		AutonomyConfirmed:   4,
		AutonomyHandoffs:    5,
		QuotaAgents:         6,
		NextQuotaResetAt:    "2026-04-20T21:05:00Z",
		Recovery: &serverOperationalRecoveryHint{
			Kind:            "worker_gap",
			SuggestedAction: "server_rearm",
			MissingWorkers:  2,
		},
	})
	if !strings.Contains(summary, "dispatch p:1 n:2 f:3 c:4") {
		t.Fatalf("summary sin dispatch confirmado: %s", summary)
	}
	if !strings.Contains(summary, "autonomia s:1 c:2 p:3 ok:4 h:5") {
		t.Fatalf("summary sin resumen de autonomia: %s", summary)
	}
	if !strings.Contains(summary, "6 bloqueados_cuota") || !strings.Contains(summary, "quota_reset 2026-04-20T21:05:00Z") {
		t.Fatalf("summary sin informacion de cuota visible: %s", summary)
	}
	if !strings.Contains(summary, "recovery worker_gap->server_rearm") || !strings.Contains(summary, "faltan_workers:2") {
		t.Fatalf("summary sin pista recovery: %s", summary)
	}
}

func TestBuildServerOperationalInfoFastFromDBPrefiereSnapshotFresco(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevSurface := statusAutonomySurfaceFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusAutonomySurfaceFetcher = prevSurface
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) { return nil, nil }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.ActiveAgents != 1 || info.WorkingAgents != 1 || info.TasksInProgress != 1 {
		t.Fatalf("deberia usar snapshot fresco: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBUsaDBSiSnapshotExpira(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:          []*db.Agente{{Nombre: "Stale"}},
		TareasEnProgreso: []tareaLite{{ID: 99, Estado: db.TareaEnProgreso, Agente: "Stale"}},
		Generado:         now.Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)

	agentsCalls := 0
	tasksCalls := 0
	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		agentsCalls++
		return []*db.Agente{{Nombre: "Codex2", Activo: true}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		tasksCalls++
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if agentsCalls != 1 || tasksCalls != 1 {
		t.Fatalf("deberia consultar DB si snapshot expira, agentCalls=%d taskCalls=%d", agentsCalls, tasksCalls)
	}
	if info.RegisteredAgents != 1 || info.TasksInProgress != 1 {
		t.Fatalf("deberia caer a DB si snapshot expira: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBPrefierePanelFrescoSobreSnapshotFrescoUsable(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 29, 11, 15, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	resetAt := now.Add(30 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaEnProgreso): 1,
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Autonomia: autonomiaResumen{
			Supervisando:  1,
			WorkConfirmed: 1,
		},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		EstadoOperativo:  "bloqueado_por_runtime",
		DetalleOperativo: "autenticacion manual requerida",
		WorkerAlive:      true,
		WorkerState:      "blocked_auth",
		OpenTasks:        1,
	}}, now)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes con snapshot fresco usable")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas con snapshot fresco usable")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.State != "degraded" || info.Reason != "workers_require_manual_auth" || info.Operational {
		t.Fatalf("deberia preferir panel fresco sobre snapshot fresco usable: %+v", info)
	}
	if info.AuthAgents != 1 || info.QuotaAgents != 0 {
		t.Fatalf("clasificacion auth/quota inesperada: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBDerivaRiesgoCanonicoSinStatus(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevSurface := statusAutonomySurfaceFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusAutonomySurfaceFetcher = prevSurface
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "Codex1", Activo: true}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) {
		return &autonomySurface{
			Events:     2,
			LastAt:     ptrTimeServerOperational(now.Add(-2 * time.Minute)),
			Highlights: []string{"integracion_bloqueada=9", "riesgo_top=infra(9)"},
			Projects: []autonomyProjectSurface{
				{
					Project:    "infra",
					Events:     2,
					LastAt:     ptrTimeServerOperational(now.Add(-2 * time.Minute)),
					Highlights: []string{"riesgo=alto", "integracion_bloqueada=9", "runtime_orders=1"},
				},
			},
		}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.CriticalProjectRisk == nil || info.CriticalProjectRisk.Project != "infra" {
		t.Fatalf("deberia derivar riesgo crítico canónico en ruta operativa ligera: %+v", info.CriticalProjectRisk)
	}
	if info.CriticalProjectRisk.Blocking <= 0 {
		t.Fatalf("deberia derivar blocking > 0: %+v", info.CriticalProjectRisk)
	}
	if len(info.AutonomyHighlights) == 0 {
		t.Fatalf("deberia derivar autonomy highlights canónicos: %+v", info.AutonomyHighlights)
	}
}

func TestBuildServerOperationalInfoFastFromDBToleraAutonomySurfaceLenta(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevSurface := statusAutonomySurfaceFetcher
	prevTimeout := serverOperationalOptionalTimeout
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusAutonomySurfaceFetcher = prevSurface
		serverOperationalOptionalTimeout = prevTimeout
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	serverOperationalOptionalTimeout = 20 * time.Millisecond

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "Codex1", Activo: true}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) {
		time.Sleep(150 * time.Millisecond)
		return &autonomySurface{
			Projects: []autonomyProjectSurface{{Project: "infra"}},
		}, nil
	}

	start := time.Now()
	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 120*time.Millisecond {
		t.Fatalf("la ruta operativa ligera no deberia bloquearse por autonomy surface lenta: %s", elapsed)
	}
	if info.TasksInProgress != 1 || info.ActiveAgents != 1 {
		t.Fatalf("snapshot operativo inesperado: %+v", info)
	}
	if info.CriticalProjectRisk != nil {
		t.Fatalf("no deberia esperar riesgo crítico cuando autonomy surface expira: %+v", info.CriticalProjectRisk)
	}
}

func TestBuildServerOperationalInfoFastFromDBIgnoraSnapshotFrescoQueNecesitaRefresh(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:          []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		AgentesActivos:   []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		TareasEnProgreso: []tareaLite{{ID: 99, Estado: db.TareaEnProgreso, Agente: "CodexStale"}},
		Autonomia:        autonomiaResumen{ContinuidadPendiente: 1},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)

	agentsCalls := 0
	tasksCalls := 0
	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		agentsCalls++
		return []*db.Agente{{Nombre: "CodexDB", Activo: true}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		tasksCalls++
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if agentsCalls != 1 || tasksCalls != 1 {
		t.Fatalf("deberia ignorar snapshot fresco que requiere refresh, agentCalls=%d taskCalls=%d", agentsCalls, tasksCalls)
	}
	if info.RegisteredAgents != 1 || info.TasksInProgress != 1 {
		t.Fatalf("deberia reconstruir desde DB: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBNoHeredaAtascadosDeSnapshotStale(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 28, 19, 30, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		AgentesActivos:    []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		AgentesAtascados:  []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "CodexStale", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 99, Estado: db.TareaEnProgreso, Agente: "CodexStale"}},
		TareasPorEstado:   map[string]int{string(db.TareaEnProgreso): 1},
		Autonomia:         autonomiaResumen{WorkConfirmed: 1},
		Generado:          now.Add(-10 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "CodexDB", Activo: true, EstadoCuota: "activo"}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.StuckAgents != 0 || info.Reason == "workers_stuck" {
		t.Fatalf("no deberia heredar atascados de snapshot stale: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBNoHeredaTrabajoNiDispatchDeSnapshotQueRequiereRefresh(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevDispatch := serverOperationalDispatchFetcher
	prevTimeout := serverOperationalOptionalTimeout
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		serverOperationalDispatchFetcher = prevDispatch
		serverOperationalOptionalTimeout = prevTimeout
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 29, 10, 30, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	serverOperationalOptionalTimeout = 20 * time.Millisecond
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "CodexStale", Activo: true, EstadoCuota: "activo"},
		},
		AgentesActivos: []*db.Agente{
			{Nombre: "CodexStale", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "CodexStale", Activo: true, EstadoCuota: "activo"},
		},
		TareasEnProgreso: []tareaLite{
			{ID: 99, Estado: db.TareaEnProgreso, Agente: "CodexStale"},
		},
		DeudaDispatch: deudaDispatchResumen{Pendientes: 3, Notificadas: 1, Total: 4},
		Autonomia:     autonomiaResumen{ContinuidadPendiente: 1, Handoffs: 2},
		Generado:      now.Format(time.RFC3339),
	}, now, time.Minute)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "CodexDB", Activo: true, EstadoCuota: "activo"}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}
	serverOperationalDispatchFetcher = func() (statusDispatchSummary, error) {
		return statusDispatchSummary{}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.WorkingAgents != 0 || info.WorkingWorkers != 0 {
		t.Fatalf("no deberia heredar working agents stale: %+v", info)
	}
	if info.DispatchPending != 0 || info.DispatchNotified != 0 || info.DispatchConfirmed != 0 {
		t.Fatalf("no deberia heredar deuda dispatch stale: %+v", info)
	}
	if info.AutonomyPending != 0 || info.AutonomyHandoffs != 0 {
		t.Fatalf("no deberia heredar autonomia stale: %+v", info)
	}
	if info.Operational || info.Reason != "tasks_without_workers" {
		t.Fatalf("deberia degradar por trabajo sin workers reales: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBCargaDispatchLigeroSinSnapshot(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevDispatch := serverOperationalDispatchFetcher
	prevTimeout := serverOperationalOptionalTimeout
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		serverOperationalDispatchFetcher = prevDispatch
		serverOperationalOptionalTimeout = prevTimeout
	}()

	serverOperationalOptionalTimeout = 20 * time.Millisecond
	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"}}, nil
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return map[string]int{string(db.TareaEnProgreso): 1}, nil
	}
	serverOperationalDispatchFetcher = func() (statusDispatchSummary, error) {
		return statusDispatchSummary{
			Deuda:    deudaDispatchResumen{Pendientes: 2, Notificadas: 1, Total: 3},
			Handoffs: 4,
		}, nil
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.DispatchPending != 2 || info.DispatchNotified != 1 {
		t.Fatalf("deberia cargar dispatch ligero canónico: %+v", info)
	}
	if info.AutonomyHandoffs != 4 {
		t.Fatalf("deberia cargar handoffs desde dispatch summary: %+v", info)
	}
}

func TestStatusSnapshotWarmLoopIntervalSigueFallbackTTL(t *testing.T) {
	if got := statusSnapshotWarmLoopInterval(); got != statusFallbackTTL {
		t.Fatalf("intervalo warm inesperado: got=%s want=%s", got, statusFallbackTTL)
	}
}

func TestBuildServerOperationalInfoFastFromDBReutilizaSnapshotStaleSinIrADB(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevSurface := statusAutonomySurfaceFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusAutonomySurfaceFetcher = prevSurface
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) { return nil, nil }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"}},
		TareasPorEstado: map[string]int{
			string(db.TareaEnProgreso): 1,
		},
		TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Generado:         now.Add(-10 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes con snapshot stale reutilizable")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas con snapshot stale reutilizable")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.ActiveAgents != 1 || info.WorkingAgents != 1 || info.TasksInProgress != 1 {
		t.Fatalf("deberia reutilizar snapshot stale sin ir a DB: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBRefrescaSnapshotStaleConPanelFresco(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	resetAt := now.Add(30 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaEnProgreso): 1,
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Autonomia: autonomiaResumen{
			Supervisando:  1,
			WorkConfirmed: 1,
		},
		Generado: now.Add(-15 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		EstadoOperativo: "trabajando",
		WorkerAlive:     true,
		WorkerState:     "running",
		OpenTasks:       1,
	}}, now)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes con snapshot stale reutilizable")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas con snapshot stale reutilizable")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.State != "ready" || !info.Operational {
		t.Fatalf("deberia converger con panel fresco y dejar de anunciar quota_blocked idle: %+v", info)
	}
	if info.ActiveAgents != 1 || info.WorkingAgents != 1 || info.ConnectedWorkers != 1 || info.WorkingWorkers != 1 {
		t.Fatalf("deberia reflejar workers del panel fresco: %+v", info)
	}
	if info.QuotaAgents != 0 || info.Reason != "control_plane_responsive" {
		t.Fatalf("no deberia arrastrar cuota stale desde status: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBPrefiereAuthManualDelPanelSobreCuotaStale(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 29, 11, 5, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	resetAt := now.Add(30 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaEnProgreso): 1,
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Autonomia: autonomiaResumen{
			Supervisando:  1,
			WorkConfirmed: 1,
		},
		Generado: now.Add(-15 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		EstadoOperativo:  "bloqueado_por_runtime",
		DetalleOperativo: "autenticacion manual requerida",
		WorkerAlive:      true,
		WorkerState:      "blocked_auth",
		OpenTasks:        1,
	}}, now)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes con snapshot stale reutilizable")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas con snapshot stale reutilizable")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.State != "degraded" || info.Reason != "workers_require_manual_auth" || info.Operational {
		t.Fatalf("deberia preferir auth manual del panel sobre cuota stale: %+v", info)
	}
	if info.AuthAgents != 1 || info.QuotaAgents != 0 {
		t.Fatalf("clasificacion auth/quota inesperada: %+v", info)
	}
}

func TestBuildServerOperationalInfoFastFromDBReutilizaSnapshotConRiesgoCanonico(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevAgents := serverOperationalListAgentsFetcher
	prevTasks := serverOperationalCountTasksFetcher
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		AutonomyHighlights: []string{
			"integracion_bloqueada=8",
			"riesgo_top=infra(8)",
		},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "infra",
			Blocking:   8,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=8"},
		},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)

	serverOperationalListAgentsFetcher = func() ([]*db.Agente, error) {
		return nil, errors.New("no deberia consultar agentes con snapshot fresco")
	}
	serverOperationalCountTasksFetcher = func() (map[string]int, error) {
		return nil, errors.New("no deberia consultar tareas con snapshot fresco")
	}

	info, err := buildServerOperationalInfoFastFromDB()
	if err != nil {
		t.Fatalf("buildServerOperationalInfoFastFromDB: %v", err)
	}
	if info.CriticalProjectRisk == nil || info.CriticalProjectRisk.Project != "infra" || info.CriticalProjectRisk.Blocking != 8 {
		t.Fatalf("deberia reutilizar riesgo canónico desde snapshot: %+v", info.CriticalProjectRisk)
	}
	if len(info.AutonomyHighlights) != 2 {
		t.Fatalf("deberia reutilizar highlights canónicos desde snapshot: %+v", info.AutonomyHighlights)
	}
}
