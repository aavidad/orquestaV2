package cmd

import (
	"context"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestFetchAgentPanelRowsCachedNoDisparaWarmFrioSinSnapshotPrevio(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		apiStatusUltraLiteFetcher = prevUltraLite
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	calls := 0
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		calls++
		return []agentesapp.Row{{EstadoOperativo: "disponible"}}, nil
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}

	rows, err := fetchAgentPanelRowsCached(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	time.Sleep(40 * time.Millisecond)
	if calls != 0 {
		t.Fatalf("no deberia disparar warm frio inmediato, calls=%d", calls)
	}
}

func TestLaunchAgentPanelSnapshotWarmLoopOmiteWarmSinSnapshot(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevDelay := agentPanelSnapshotWarmDelay
	prevTicker := agentPanelSnapshotWarmTicker
	prevTimeout := agentPanelSnapshotWarmTimeo
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		agentPanelSnapshotWarmDelay = prevDelay
		agentPanelSnapshotWarmTicker = prevTicker
		agentPanelSnapshotWarmTimeo = prevTimeout
		resetAgentPanelSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()

	calls := 0
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		calls++
		return []agentesapp.Row{{EstadoOperativo: "disponible"}}, nil
	}
	agentPanelSnapshotWarmDelay = 10 * time.Millisecond
	agentPanelSnapshotWarmTicker = 10 * time.Millisecond
	agentPanelSnapshotWarmTimeo = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	launchAgentPanelSnapshotWarmLoop(ctx, nil)
	time.Sleep(45 * time.Millisecond)

	if calls != 0 {
		t.Fatalf("warm loop no deberia construir panel sin snapshot previa, calls=%d", calls)
	}
}

func TestLaunchAgentPanelSnapshotWarmLoopOmiteWarmEnQuotaBlockedIdle(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevDelay := agentPanelSnapshotWarmDelay
	prevTicker := agentPanelSnapshotWarmTicker
	prevTimeout := agentPanelSnapshotWarmTimeo
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		agentPanelSnapshotWarmDelay = prevDelay
		agentPanelSnapshotWarmTicker = prevTicker
		agentPanelSnapshotWarmTimeo = prevTimeout
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	resetAt := now.Add(20 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			Supervisando:  1,
			WorkConfirmed: 1,
		},
	}, now, time.Minute)
	storeAgentPanelSnapshotWithTTL([]agentesapp.Row{{Agente: &db.Agente{Nombre: "Codex1"}, EstadoOperativo: "sin_tarea"}}, now, 0, agentPanelSnapshotStaleTTL)

	calls := 0
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		calls++
		return []agentesapp.Row{{EstadoOperativo: "disponible"}}, nil
	}
	agentPanelSnapshotWarmDelay = 10 * time.Millisecond
	agentPanelSnapshotWarmTicker = 10 * time.Millisecond
	agentPanelSnapshotWarmTimeo = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	launchAgentPanelSnapshotWarmLoop(ctx, nil)
	time.Sleep(45 * time.Millisecond)

	if calls != 0 {
		t.Fatalf("warm loop no deberia construir panel en quota_blocked idle, calls=%d", calls)
	}
}

func TestFetchAgentPanelRowsCachedDesdeStatusNoQuedaFresh(t *testing.T) {
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()
	defer resetAgentPanelSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:        []*db.Agente{{Nombre: "Codex2", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		AgentesActivos: []*db.Agente{{Nombre: "Codex2", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
	}, now, time.Minute)

	rows, err := fetchAgentPanelRowsCached(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "Codex2" {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if _, ok := readAgentPanelSnapshotFresh(); ok {
		t.Fatalf("snapshot sintetica derivada de status no deberia quedar fresh")
	}
	if _, ok := readAgentPanelSnapshotAny(); !ok {
		t.Fatalf("snapshot sintetica derivada de status deberia quedar disponible como stale")
	}
}

func TestFetchAgentPanelRowsCachedUltraLiteNoQuedaFresh(t *testing.T) {
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		apiStatusUltraLiteFetcher = prevUltraLite
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{
			Agentes:        []*db.Agente{{Nombre: "Codex4", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
			AgentesActivos: []*db.Agente{{Nombre: "Codex4", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		}, true
	}

	rows, err := fetchAgentPanelRowsCached(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "Codex4" {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if _, ok := readAgentPanelSnapshotFresh(); ok {
		t.Fatalf("snapshot sintetica derivada de ultra-lite no deberia quedar fresh")
	}
	if _, ok := readAgentPanelSnapshotAny(); !ok {
		t.Fatalf("snapshot sintetica derivada de ultra-lite deberia quedar disponible como stale")
	}
}

func TestFetchAgentPanelRowsCachedRefrescaSnapshotSinteticaConTrabajoVivo(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		TareasEnProgreso:  []tareaLite{{ID: 26, Agente: "Codex1", Estado: db.TareaEnProgreso}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:           &db.Agente{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"},
			EstadoOperativo:  "arrancando",
			DetalleOperativo: "worker starting",
			OpenTasks:        1,
			WorkerState:      "starting",
			WorkerAlive:      true,
			Runtime:          &db.RuntimeInstance{Agente: "Codex1", LogicalState: "pensando"},
			Handle:           &db.RuntimeHandle{Agente: "Codex1", Estado: "activo"},
		}}, nil
	}

	rows, err := fetchAgentPanelRowsCached(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 1 || rows[0].WorkerState != "starting" || rows[0].Runtime == nil || rows[0].Handle == nil {
		t.Fatalf("deberia refrescar a filas ricas, rows=%+v", rows)
	}
}
