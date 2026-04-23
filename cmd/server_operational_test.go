package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

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
}

func TestBuildServerOperationalInfoFastFromDBPrefiereSnapshotFresco(t *testing.T) {
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
		Agentes:           []*db.Agente{{Nombre: "Codex1"}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1"}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1"}},
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
	prevNow := statusNowFunc
	defer func() {
		serverOperationalListAgentsFetcher = prevAgents
		serverOperationalCountTasksFetcher = prevTasks
		statusNowFunc = prevNow
	}()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
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
