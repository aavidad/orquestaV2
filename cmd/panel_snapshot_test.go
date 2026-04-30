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

func TestFetchAgentPanelRowsReadOnlyCachedNoBloqueaPorRefrescoInmediato(t *testing.T) {
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

	release := make(chan struct{})
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		<-release
		return []agentesapp.Row{{EstadoOperativo: "trabajando"}}, nil
	}

	start := time.Now()
	rows, err := fetchAgentPanelRowsReadOnlyCached(20 * time.Millisecond)
	elapsed := time.Since(start)
	close(release)

	if err != nil {
		t.Fatalf("fetchAgentPanelRowsReadOnlyCached: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("read-only panel no deberia bloquearse por refresh inmediato, elapsed=%s", elapsed)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "Codex1" {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
}

func TestFetchAgentPanelRowsReadOnlyCachedRefrescaSnapshotFreshConFilasRicasSiEntraEnBudget(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Codex20", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		EstadoOperativo: "trabajando",
		OpenTasks:       2,
		CurrentTask:     &agentesapp.TaskFocus{TaskID: 29, State: db.TareaEnProgreso},
	}}, now)

	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex20", Activo: true, Habilitado: true, EstadoCuota: "activo"},
			EstadoOperativo: "trabajando",
			OpenTasks:       1,
			CurrentTask:     &agentesapp.TaskFocus{TaskID: 4, State: db.TareaEnProgreso},
			WorkerState:     "ready",
			WorkerAlive:     true,
			Runtime:         &db.RuntimeInstance{Agente: "Codex20", LogicalState: "activo"},
			Handle:          &db.RuntimeHandle{Agente: "Codex20", Estado: "activo"},
		}}, nil
	}

	rows, err := fetchAgentPanelRowsReadOnlyCached(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsReadOnlyCached: %v", err)
	}
	if len(rows) != 1 || rows[0].CurrentTask == nil || rows[0].CurrentTask.TaskID != 4 || !rows[0].WorkerAlive || rows[0].Runtime == nil || rows[0].Handle == nil {
		t.Fatalf("deberia refrescar snapshot fresh a filas ricas, rows=%+v", rows)
	}
}

func TestFetchAgentPanelRowsCachedRefrescaSnapshotSinteticaConAtascadoHeredado(t *testing.T) {
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
		Agentes:          []*db.Agente{{Nombre: "CodexStuck", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		AgentesAtascados: []*db.Agente{{Nombre: "CodexStuck", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)

	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:           &db.Agente{Nombre: "CodexStuck", Activo: true, Habilitado: true, EstadoCuota: "activo"},
			EstadoOperativo:  "atascado",
			DetalleOperativo: "mailbox sin drenar",
			WorkerState:      "waiting_input",
			WorkerAlive:      true,
			Runtime:          &db.RuntimeInstance{Agente: "CodexStuck", LogicalState: "running"},
			Handle:           &db.RuntimeHandle{Agente: "CodexStuck", Estado: "activo"},
		}}, nil
	}

	rows, err := fetchAgentPanelRowsCached(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 1 || rows[0].DetalleOperativo != "mailbox sin drenar" || rows[0].Runtime == nil || rows[0].Handle == nil {
		t.Fatalf("deberia refrescar atascado heredado a filas ricas, rows=%+v", rows)
	}
}

func TestFetchAgentPanelRowsCachedRefrescaSnapshotSinteticaConRuntimeBlockedHeredado(t *testing.T) {
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
		Agentes:           []*db.Agente{{Nombre: "GeminiAuth", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		AgentesAuthManual: []*db.Agente{{Nombre: "GeminiAuth", Activo: true, Habilitado: true, EstadoCuota: "activo"}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:           &db.Agente{Nombre: "GeminiAuth", Activo: true, Habilitado: true, EstadoCuota: "activo"},
			EstadoOperativo:  "bloqueado_por_runtime",
			DetalleOperativo: "autenticacion manual requerida",
			WorkerState:      "blocked_auth",
			WorkerAlive:      true,
			Runtime:          &db.RuntimeInstance{Agente: "GeminiAuth", LogicalState: "blocked_auth"},
			Handle:           &db.RuntimeHandle{Agente: "GeminiAuth", Estado: "activo"},
		}}, nil
	}

	rows, err := fetchAgentPanelRowsCached(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchAgentPanelRowsCached: %v", err)
	}
	if len(rows) != 1 || rows[0].WorkerState != "blocked_auth" || rows[0].DetalleOperativo != "autenticacion manual requerida" {
		t.Fatalf("deberia refrescar runtime blocked heredado a filas ricas, rows=%+v", rows)
	}
}
