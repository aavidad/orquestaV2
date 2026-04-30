package cmd

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestAgenteCuentaComoConectadoRespetaEstadoCuotaVisible(t *testing.T) {
	if agenteCuentaComoConectado(nil) {
		t.Fatalf("nil no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex1", Activo: false, EstadoCuota: "activo"}) {
		t.Fatalf("un agente sin sesion activa no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{
		Nombre:       "CodexStale",
		Activo:       false,
		EstadoCuota:  "activo",
		EstadoSesion: "",
		UltimaSesion: timePtr(time.Now().UTC()),
	}) {
		t.Fatalf("una ultima sesion reciente sin sesion operativa no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex2", Activo: true, EstadoCuota: "agotado"}) {
		t.Fatalf("un agente agotado no deberia contar como conectado visible")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex3", Activo: true, EstadoCuota: "enfriamiento"}) {
		t.Fatalf("un agente en enfriamiento no deberia contar como conectado visible")
	}
	if !agenteCuentaComoConectado(&db.Agente{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}) {
		t.Fatalf("un agente activo con cuota activa deberia contar como conectado")
	}
	if !agenteCuentaComoConectado(&db.Agente{
		Nombre:                "antigravity",
		Activo:                true,
		EstadoCuota:           "agotado",
		PresupuestoSemanalPct: intPtr(69),
		CuotaRestantePct:      intPtr(69),
		PresupuestoVentana:    "weekly",
	}) {
		t.Fatalf("una cuenta solo semanal con saldo positivo no deberia caer por una cuota legacy")
	}
	if !agenteCuentaComoConectado(&db.Agente{
		Nombre:            "Gemma1",
		Activo:            true,
		EstadoCuota:       "enfriamiento",
		SinCuotaProveedor: true,
	}) {
		t.Fatalf("un agente local sin cuota de proveedor no deberia caer por enfriamiento legacy")
	}
	if !agenteCuentaComoConectado(&db.Agente{
		Nombre:      "Qwen1",
		Activo:      true,
		EstadoCuota: "enfriamiento",
	}) {
		t.Fatalf("un agente local detectado por nombre no deberia caer por enfriamiento legacy")
	}
}

func TestStatusSnapshotNeedsImmediateRefreshSiActivoVisibleVieneConFlagInactivo(t *testing.T) {
	if !statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		AgentesActivos: []*db.Agente{{Nombre: "CodexBudget", Activo: false, EstadoCuota: "activo"}},
	}) {
		t.Fatalf("un snapshot con agentesActivos pero flag Activo=false debe forzar refresh")
	}
}

func TestStatusSnapshotNeedsImmediateRefreshSiHayAtascadosVisibles(t *testing.T) {
	if !statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		AgentesActivos:   []*db.Agente{{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}},
		AgentesAtascados: []*db.Agente{{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}},
	}) {
		t.Fatalf("un snapshot con atascados visibles debe forzar refresh")
	}
}

func TestStatusSnapshotNeedsImmediateRefreshSiHayActivosPeroTrabajoSinWorkers(t *testing.T) {
	if !statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		AgentesActivos:  []*db.Agente{{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}},
		TareasPorEstado: map[string]int{string(db.TareaEnProgreso): 1},
	}) {
		t.Fatalf("un snapshot con activos pero trabajo sin workers visibles debe forzar refresh")
	}
}

func TestStatusSnapshotNeedsImmediateRefreshToleraActivosSinWorkersSiTrabajoYaConfirmado(t *testing.T) {
	if statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		AgentesActivos:  []*db.Agente{{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}},
		TareasPorEstado: map[string]int{string(db.TareaEnProgreso): 1},
		Autonomia:       autonomiaResumen{WorkConfirmed: 1},
	}) {
		t.Fatalf("si el trabajo ya está confirmado no deberia forzar refresh inmediato")
	}
}

func TestStatusSnapshotCanStayLightSiSoloHayCuotaBloqueando(t *testing.T) {
	now := time.Now().UTC()
	resetAt := now.Add(20 * time.Minute)
	status := apiStatusResponse{
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
	}
	if !statusSnapshotCanStayLight(status) {
		t.Fatalf("deberia poder quedarse en snapshot ligero: %+v", status)
	}
	if statusSnapshotNeedsImmediateRefresh(status) {
		t.Fatalf("no deberia forzar refresh inmediato en cuota bloqueada idle")
	}
}

func TestStatusSnapshotCanStayLightNoAplicaSiHayContinuidadPendiente(t *testing.T) {
	now := time.Now().UTC()
	resetAt := now.Add(20 * time.Minute)
	status := apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			Supervisando:         1,
			ContinuidadPendiente: 1,
		},
	}
	if statusSnapshotCanStayLight(status) {
		t.Fatalf("no deberia quedarse light si hay continuidad pendiente: %+v", status)
	}
}

func TestStatusSnapshotCanStayLightNoAplicaSiHayReservadasODispatch(t *testing.T) {
	now := time.Now().UTC()
	resetAt := now.Add(20 * time.Minute)
	base := apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "Codex1", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
	}
	withReserved := base
	withReserved.TareasReservadas = []tareaLite{{ID: 2, Estado: db.TareaAsignada}}
	if statusSnapshotCanStayLight(withReserved) {
		t.Fatalf("no deberia quedarse light con reservadas: %+v", withReserved)
	}
	withDispatch := base
	withDispatch.DeudaDispatch = deudaDispatchResumen{Pendientes: 1, Total: 1}
	if statusSnapshotCanStayLight(withDispatch) {
		t.Fatalf("no deberia quedarse light con dispatch pendiente: %+v", withDispatch)
	}
}

func TestNormalizarAgentesVisiblesStatusDescartaActivosInconsistentes(t *testing.T) {
	activos, trabajando, saturados := normalizarAgentesVisiblesStatus(
		[]*db.Agente{
			{Nombre: "CodexBudget", Activo: false, EstadoCuota: "activo"},
			{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		},
		[]*db.Agente{
			{Nombre: "CodexBudget", Activo: false, EstadoCuota: "activo"},
			{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"},
		},
		[]*db.Agente{
			{Nombre: "CodexBudget", Activo: false, EstadoCuota: "activo"},
		},
	)
	if len(activos) != 1 || activos[0].Nombre != "Codex1" {
		t.Fatalf("activos normalizados inesperados: %+v", activos)
	}
	if len(trabajando) != 1 || trabajando[0].Nombre != "Codex1" {
		t.Fatalf("trabajando normalizados inesperados: %+v", trabajando)
	}
	if len(saturados) != 0 {
		t.Fatalf("saturados normalizados inesperados: %+v", saturados)
	}
}

func TestStatusRowsForSnapshotPrefierePanelSnapshotFresco(t *testing.T) {
	prevFetcher := statusRowsFetcher
	defer func() {
		statusRowsFetcher = prevFetcher
		resetAgentPanelSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	storeAgentPanelSnapshot([]agentesapp.Row{{Agente: &db.Agente{Nombre: "CodexPanel"}, EstadoOperativo: "trabajando"}}, time.Now().UTC())
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{Agente: &db.Agente{Nombre: "CodexFetcher"}, EstadoOperativo: "trabajando"}}, nil
	}
	rows, err := statusRowsForSnapshot(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("statusRowsForSnapshot: %v", err)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "CodexPanel" {
		t.Fatalf("deberia preferir panel snapshot fresco, rows=%+v", rows)
	}
}

func TestStatusRowsForSnapshotUsaStatusLigeroEnQuotaBlockedIdle(t *testing.T) {
	prevFetcher := statusRowsFetcher
	defer func() {
		statusRowsFetcher = prevFetcher
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

	calls := 0
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		calls++
		return []agentesapp.Row{{Agente: &db.Agente{Nombre: "CodexFetcher"}, EstadoOperativo: "trabajando"}}, nil
	}

	rows, err := statusRowsForSnapshot(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("statusRowsForSnapshot: %v", err)
	}
	if calls != 0 {
		t.Fatalf("no deberia pedir filas ricas en quota_blocked idle, calls=%d", calls)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "Codex1" {
		t.Fatalf("rows ligeras inesperadas: %+v", rows)
	}
}

func TestStatusRowsForSnapshotPrefierePanelFrescoSobreStatusLigeroStale(t *testing.T) {
	prevFetcher := statusRowsFetcher
	defer func() {
		statusRowsFetcher = prevFetcher
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	resetAt := now.Add(20 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "CodexBudget", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		AgentesQuotaBlocked: []*db.Agente{
			{Nombre: "CodexBudget", EstadoCuota: "enfriamiento", ReanimarAt: &resetAt},
		},
		TareasEnProgreso: []tareaLite{{ID: 24, Estado: db.TareaEnProgreso}},
		Autonomia: autonomiaResumen{
			Supervisando:  1,
			WorkConfirmed: 1,
		},
	}, now, time.Minute)
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexLive", Activo: true, EstadoCuota: "activo"},
		EstadoOperativo: "trabajando",
		WorkerAlive:     true,
		WorkerState:     "running",
	}}, now)

	calls := 0
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		calls++
		return []agentesapp.Row{{Agente: &db.Agente{Nombre: "CodexFetcher"}, EstadoOperativo: "trabajando"}}, nil
	}

	rows, err := statusRowsForSnapshot(20 * time.Millisecond)
	if err != nil {
		t.Fatalf("statusRowsForSnapshot: %v", err)
	}
	if calls != 0 {
		t.Fatalf("no deberia pedir filas ricas si ya existe panel fresco, calls=%d", calls)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "CodexLive" {
		t.Fatalf("deberia preferir panel fresco sobre status ligero stale, rows=%+v", rows)
	}
}

func TestStatusServiceCacheaSnapshotCorto(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = 2 * time.Second
	statusAsyncRefresh = true
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errors.New("sin fallback")
	}

	calls := 0
	statusFreshFetcher = func() (apiStatusResponse, error) {
		calls++
		return apiStatusResponse{Generado: current.Format(time.RFC3339)}, nil
	}

	service := dbStatusService{}
	first, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	second, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if calls != 1 {
		t.Fatalf("la cache deberia evitar una segunda carga, calls=%d", calls)
	}
	if first.Generado != second.Generado {
		t.Fatalf("snapshot cacheado inesperado: %q vs %q", first.Generado, second.Generado)
	}

	current = current.Add(3 * time.Second)
	third, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("third fetch: %v", err)
	}
	if calls != 1 {
		t.Fatalf("tras expirar la cache deberia devolver stale y refrescar en background, calls=%d", calls)
	}
	if third.Generado != second.Generado {
		t.Fatalf("deberia devolver el snapshot stale mientras refresca en background")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		statusCacheState.mu.Lock()
		refreshed := !statusCacheState.refreshing && statusCacheState.ok && statusCacheState.value.Generado != second.Generado
		statusCacheState.mu.Unlock()
		if refreshed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	statusCacheState.mu.Lock()
	refreshed := statusCacheState.value.Generado
	callsAfter := calls
	statusCacheState.mu.Unlock()
	if callsAfter != 2 {
		t.Fatalf("deberia refrescar en background tras expirar la cache, calls=%d", callsAfter)
	}
	if refreshed == second.Generado {
		t.Fatalf("la cache no se actualizo en background")
	}
}

func TestStatusServiceNoRetieneSnapshotVacioSiHayTareasEnProgreso(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = time.Minute
	statusAsyncRefresh = true
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errors.New("sin fallback")
	}

	calls := 0
	statusFreshFetcher = func() (apiStatusResponse, error) {
		calls++
		if calls == 1 {
			return apiStatusResponse{
				Generado:         "stale",
				AgentesActivos:   nil,
				TareasEnProgreso: []tareaLite{{ID: 501, Estado: db.TareaEnProgreso, Agente: "Codex2"}},
			}, nil
		}
		return apiStatusResponse{
			Generado:         "fresh",
			AgentesActivos:   []*db.Agente{{Nombre: "Codex2"}},
			TareasEnProgreso: []tareaLite{{ID: 501, Estado: db.TareaEnProgreso, Agente: "Codex2"}},
		}, nil
	}

	service := dbStatusService{}
	first, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if first.Generado != "stale" {
		t.Fatalf("primer snapshot inesperado: %+v", first)
	}
	second, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if second.Generado != "fresh" {
		t.Fatalf("deberia refrescar de inmediato el snapshot inconsistente: %+v", second)
	}
	if calls != 2 {
		t.Fatalf("deberia forzar un segundo fetch, calls=%d", calls)
	}
}

func TestStatusServiceSerializaRefreshSincronoEnCacheMiss(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = time.Minute
	statusAsyncRefresh = true
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errors.New("sin fallback")
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	statusFreshFetcher = func() (apiStatusResponse, error) {
		if calls.Add(1) == 1 {
			started <- struct{}{}
		}
		<-release
		return apiStatusResponse{Generado: "fresh"}, nil
	}

	service := dbStatusService{}
	var wg sync.WaitGroup
	results := make(chan apiStatusResponse, 2)
	errs := make(chan error, 2)
	callFetch := func() {
		defer wg.Done()
		status, err := service.FetchStatus()
		results <- status
		errs <- err
	}

	wg.Add(1)
	go callFetch()
	<-started

	wg.Add(1)
	go callFetch()

	time.Sleep(50 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("solo deberia haber un refresh en vuelo, calls=%d", got)
	}

	close(release)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("fetch concurrente fallo: %v", err)
		}
	}
	for status := range results {
		if status.Generado != "fresh" {
			t.Fatalf("snapshot inesperado: %+v", status)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("solo deberia ejecutarse un refresh, calls=%d", got)
	}
}

func TestStatusServiceTimeoutUsaFallbackRapido(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		resetStatusSnapshotCache()
	}()

	blockFresh := make(chan struct{})
	statusFreshTimeout = 20 * time.Millisecond
	statusFastTimeout = 100 * time.Millisecond
	statusFreshFetcher = func() (apiStatusResponse, error) {
		<-blockFresh
		return apiStatusResponse{Generado: "fresh tardio"}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{Generado: "fallback"}, nil
	}

	start := time.Now()
	status, err := dbStatusService{}.FetchStatus()
	elapsed := time.Since(start)
	close(blockFresh)
	if err != nil {
		t.Fatalf("fetch status con fallback: %v", err)
	}
	if status.Generado != "fallback" {
		t.Fatalf("deberia devolver fallback rapido, got=%+v", status)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("deberia degradar rapido, elapsed=%s", elapsed)
	}
}

func TestStatusServiceNoEsperaIndefinidamenteRefreshEnVuelo(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	prevRefreshWaitTimeout := statusRefreshWaitTimeout
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		statusRefreshWaitTimeout = prevRefreshWaitTimeout
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = time.Minute
	statusAsyncRefresh = true
	statusFreshTimeout = 20 * time.Millisecond
	statusFastTimeout = 100 * time.Millisecond

	blockFresh := make(chan struct{})
	statusFreshFetcher = func() (apiStatusResponse, error) {
		<-blockFresh
		return apiStatusResponse{Generado: "fresh"}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{Generado: "fallback_wait"}, nil
	}

	statusCacheState.mu.Lock()
	statusCacheState.refreshing = true
	statusCacheState.waitCh = blockFresh
	statusCacheState.ok = false
	statusCacheState.mu.Unlock()

	start := time.Now()
	status, err := dbStatusService{}.FetchStatus()
	elapsed := time.Since(start)
	close(blockFresh)
	if err != nil {
		t.Fatalf("fetch status con refresh en vuelo: %v", err)
	}
	if status.Generado != "fallback_wait" {
		t.Fatalf("deberia devolver fallback al agotarse waitCh, got=%+v", status)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("no deberia esperar indefinidamente a waitCh, elapsed=%s", elapsed)
	}
}

func TestStatusServiceTimeoutAcotadoSiFastYRefreshSiguenBloqueados(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	prevRefreshWaitTimeout := statusRefreshWaitTimeout
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		statusRefreshWaitTimeout = prevRefreshWaitTimeout
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = time.Minute
	statusAsyncRefresh = true
	statusFreshTimeout = 20 * time.Millisecond
	statusFastTimeout = 20 * time.Millisecond
	statusRefreshWaitTimeout = 30 * time.Millisecond

	blockFresh := make(chan struct{})
	statusFreshFetcher = func() (apiStatusResponse, error) {
		<-blockFresh
		return apiStatusResponse{Generado: "fresh"}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		<-blockFresh
		return apiStatusResponse{Generado: "fallback_wait"}, nil
	}

	statusCacheState.mu.Lock()
	statusCacheState.refreshing = true
	statusCacheState.waitCh = blockFresh
	statusCacheState.ok = false
	statusCacheState.mu.Unlock()

	start := time.Now()
	status, err := dbStatusService{}.FetchStatus()
	elapsed := time.Since(start)
	close(blockFresh)
	if !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("deberia devolver timeout acotado, status=%+v err=%v", status, err)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("no deberia esperar indefinidamente con fast y refresh bloqueados, elapsed=%s", elapsed)
	}
}

func TestStatusServiceDevuelveSnapshotValidaSinEsperarRefreshEnVuelo(t *testing.T) {
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusNowFunc = prevNow
		statusSnapshotTTL = prevTTL
		statusAsyncRefresh = prevAsync
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusSnapshotTTL = time.Minute
	statusAsyncRefresh = true

	waitCh := make(chan struct{})
	statusCacheState.mu.Lock()
	statusCacheState.value = apiStatusResponse{Generado: "cached"}
	statusCacheState.expires = current.Add(time.Minute)
	statusCacheState.ok = true
	statusCacheState.refreshing = true
	statusCacheState.waitCh = waitCh
	statusCacheState.mu.Unlock()

	start := time.Now()
	status, err := dbStatusService{}.FetchStatus()
	elapsed := time.Since(start)
	close(waitCh)
	if err != nil {
		t.Fatalf("fetch status con snapshot valida: %v", err)
	}
	if status.Generado != "cached" {
		t.Fatalf("deberia devolver snapshot cacheada, got=%+v", status)
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("no deberia esperar refresh en vuelo cuando ya hay snapshot valida, elapsed=%s", elapsed)
	}
}

func TestStatusServiceAplicaBackoffTrasTimeoutFresco(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	prevFailureBackoff := statusFailureBackoffTTL
	prevNow := statusNowFunc
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		statusFailureBackoffTTL = prevFailureBackoff
		statusNowFunc = prevNow
		resetStatusSnapshotCache()
	}()

	current := time.Date(2026, 4, 21, 11, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return current }
	statusFreshTimeout = 20 * time.Millisecond
	statusFastTimeout = 100 * time.Millisecond
	statusFailureBackoffTTL = time.Second

	blockFresh := make(chan struct{})
	var freshCalls atomic.Int32
	statusFreshFetcher = func() (apiStatusResponse, error) {
		freshCalls.Add(1)
		<-blockFresh
		return apiStatusResponse{Generado: "fresh tardio"}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{
			Generado:         "fallback",
			TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex2"}},
		}, nil
	}

	service := dbStatusService{}
	first, err := service.FetchStatus()
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if first.Generado != "fallback" {
		t.Fatalf("fallback inesperado: %+v", first)
	}
	statusCacheState.mu.Lock()
	retryAfter := statusCacheState.retryAfter
	okCached := statusCacheState.ok
	generadoCached := statusCacheState.value.Generado
	statusCacheState.mu.Unlock()
	if !okCached || generadoCached != "fallback" || !retryAfter.After(current) {
		t.Fatalf("cache/backoff inesperado: ok=%t generado=%q retryAfter=%s now=%s", okCached, generadoCached, retryAfter, current)
	}
	second, err := service.FetchStatus()
	close(blockFresh)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if second.Generado != "fallback" {
		t.Fatalf("deberia reutilizar fallback durante backoff: %+v", second)
	}
	if got := freshCalls.Load(); got > 1 {
		t.Fatalf("no deberia relanzar fetch fresco durante backoff, calls=%d", got)
	}
}

func TestStatusServicePrefiereFallbackRapidoAntesDeFetchFrescoEnFrio(t *testing.T) {
	prevFetcher := statusFreshFetcher
	prevFastFetcher := statusFastFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	prevFailureBackoff := statusFailureBackoffTTL
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		statusFailureBackoffTTL = prevFailureBackoff
		resetStatusSnapshotCache()
	}()

	statusFreshTimeout = 100 * time.Millisecond
	statusFastTimeout = 20 * time.Millisecond
	statusFailureBackoffTTL = time.Second

	blockFresh := make(chan struct{})
	statusFreshFetcher = func() (apiStatusResponse, error) {
		<-blockFresh
		return apiStatusResponse{Generado: "fresh tardio"}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{Generado: "fallback"}, nil
	}

	start := time.Now()
	status, err := (dbStatusService{}).FetchStatus()
	elapsed := time.Since(start)
	close(blockFresh)
	if err != nil {
		t.Fatalf("fetch status: %v", err)
	}
	if status.Generado != "fallback" {
		t.Fatalf("deberia devolver fallback rapido, got=%+v", status)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("no deberia esperar al fetch fresco en frio, elapsed=%s", elapsed)
	}
}

func TestStoreStatusSnapshotNoGuardaSnapshotDegenerada(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	storeStatusSnapshot(apiStatusResponse{Generado: "empty"}, time.Now().UTC())
	if _, ok := readStatusSnapshotAny(); ok {
		t.Fatalf("no deberia guardar snapshot degenerada")
	}
}

func TestReadStatusSnapshotFreshIgnoraSnapshotDegenerada(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	statusCacheState.mu.Lock()
	statusCacheState.value = apiStatusResponse{Generado: "empty"}
	statusCacheState.expires = time.Now().Add(time.Minute)
	statusCacheState.ok = true
	statusCacheState.mu.Unlock()

	if _, ok := readStatusSnapshotFresh(); ok {
		t.Fatalf("no deberia reutilizar snapshot degenerada como fresca")
	}
	if _, ok := readStatusSnapshotAny(); ok {
		t.Fatalf("no deberia reutilizar snapshot degenerada")
	}
}

func TestAgentRowsForStatusWithinTimeoutDegradaRapido(t *testing.T) {
	prevFetcher := statusRowsFetcher
	prevTimeout := statusRowsTimeout
	defer func() {
		statusRowsFetcher = prevFetcher
		statusRowsTimeout = prevTimeout
		resetStatusRowsFlightState()
	}()
	resetStatusRowsFlightState()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	statusRowsTimeout = 20 * time.Millisecond

	start := time.Now()
	rows, err := agentRowsForStatusWithinTimeout(statusRowsTimeout)
	elapsed := time.Since(start)
	close(block)
	if !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("deberia devolver timeout, rows=%+v err=%v", rows, err)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("deberia degradar rapido, elapsed=%s", elapsed)
	}
}

func TestAgentRowsForStatusWithinTimeoutCoalesceFetchEnVuelo(t *testing.T) {
	prevFetcher := statusRowsFetcher
	prevTimeout := statusRowsTimeout
	defer func() {
		statusRowsFetcher = prevFetcher
		statusRowsTimeout = prevTimeout
		resetStatusRowsFlightState()
	}()
	resetStatusRowsFlightState()

	block := make(chan struct{})
	var calls atomic.Int32
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		calls.Add(1)
		<-block
		return []agentesapp.Row{{Agente: &db.Agente{Nombre: "Codex10"}}}, nil
	}
	statusRowsTimeout = 20 * time.Millisecond

	if _, err := agentRowsForStatusWithinTimeout(statusRowsTimeout); !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("primera llamada debería degradar por timeout, err=%v", err)
	}
	if _, err := agentRowsForStatusWithinTimeout(statusRowsTimeout); !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("segunda llamada debería esperar el mismo vuelo y degradar por timeout, err=%v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("no debería lanzar fetch duplicado mientras el primero sigue en vuelo, calls=%d", got)
	}
	close(block)
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		statusRowsFlightState.mu.Lock()
		running := statusRowsFlightState.running
		statusRowsFlightState.mu.Unlock()
		if !running {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	rows, err := agentRowsForStatusWithinTimeout(statusRowsTimeout)
	if err != nil {
		t.Fatalf("tercera llamada debería volver a poder resolver rows, err=%v", err)
	}
	if len(rows) != 1 || rows[0].Agente == nil || rows[0].Agente.Nombre != "Codex10" {
		t.Fatalf("rows inesperadas tras completar el vuelo: %+v", rows)
	}
	if got := calls.Load(); got > 2 {
		t.Fatalf("no debería disparar más de un fetch adicional tras completar el vuelo, calls=%d", got)
	}
}

func TestFetchStatusFreshNoReintentaRowsSinTimeout(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	agente := "Codex2"
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Cerrar hot path",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Estado:      db.TareaEnProgreso,
		Agente:      &agente,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "test",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	prevFetcher := statusRowsFetcher
	prevTimeout := statusRowsTimeout
	defer func() {
		statusRowsFetcher = prevFetcher
		statusRowsTimeout = prevTimeout
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	statusRowsTimeout = 20 * time.Millisecond

	start := time.Now()
	status, err := fetchStatusFresh()
	elapsed := time.Since(start)
	close(block)
	if err != nil {
		t.Fatalf("fetchStatusFresh: %v", err)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("deberia degradar rapido sin relanzar rows sin timeout, elapsed=%s", elapsed)
	}
	if len(status.AgentesActivos) != 1 || status.AgentesActivos[0].Nombre != "Codex2" {
		t.Fatalf("agentes activos ligeros inesperados: %+v", status.AgentesActivos)
	}
}

func TestFetchStatusFreshNoPromocionaResiduoSinSesionOperativaPorEstadoCrudo(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.DB.Exec(`UPDATE agentes SET activo=0, habilitado=1, estado_sesion='disponible', ultima_sesion=? WHERE nombre='CodexBudget'`, now); err != nil {
		t.Fatalf("forzar estado crudo: %v", err)
	}

	prevFetcher := statusRowsFetcher
	prevTimeout := statusRowsTimeout
	defer func() {
		statusRowsFetcher = prevFetcher
		statusRowsTimeout = prevTimeout
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	statusRowsTimeout = 20 * time.Millisecond

	status, err := fetchStatusFresh()
	close(block)
	if err != nil {
		t.Fatalf("fetchStatusFresh: %v", err)
	}
	if len(status.AgentesActivos) != 0 {
		t.Fatalf("un residuo sin sesion operativa no deberia promocionarse por estado crudo: %+v", status.AgentesActivos)
	}
}

func TestAgentesVisiblesLigeroMarcaTrabajandoPorTareaEnProgreso(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex2", Activo: true, EstadoCuota: "activo"}
	activos, trabajando, quota := agentesVisiblesLigero(
		[]*db.Agente{agente},
		[]tareaLite{{ID: 7, Estado: db.TareaEnProgreso, Agente: "Codex2"}},
	)
	if len(activos) != 1 || activos[0].Nombre != "Codex2" {
		t.Fatalf("activos ligeros inesperados: %+v", activos)
	}
	if len(trabajando) != 1 || trabajando[0].Nombre != "Codex2" {
		t.Fatalf("trabajando ligeros inesperados: %+v", trabajando)
	}
	if len(quota) != 0 {
		t.Fatalf("quota bloqueados inesperados: %+v", quota)
	}
}

func TestFetchStatusFreshToleraSeccionesOpcionalesLentas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	prevProjects := statusListProjectsFetcher
	prevTimeout := statusOptionalSectionTimeout
	defer func() {
		statusListProjectsFetcher = prevProjects
		statusOptionalSectionTimeout = prevTimeout
	}()

	block := make(chan struct{})
	statusListProjectsFetcher = func() ([]*db.Proyecto, error) {
		<-block
		return nil, nil
	}
	statusOptionalSectionTimeout = 20 * time.Millisecond

	start := time.Now()
	status, err := fetchStatusFresh()
	elapsed := time.Since(start)
	close(block)
	if err != nil {
		t.Fatalf("fetchStatusFresh: %v", err)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("deberia degradar rapido con seccion opcional lenta, elapsed=%s", elapsed)
	}
	found := false
	for _, agente := range status.Agentes {
		if agente != nil && agente.Nombre == "Codex2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("deberia conservar a Codex2 en snapshot degradado, agentes=%+v", status.Agentes)
	}
}

func TestFetchStatusFreshReutilizaResumenDispatchParaEvitarBarridosDuplicados(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	prevSummary := statusDispatchSummaryFetcher
	prevDebt := statusDispatchDebtFetcher
	prevHandoffs := statusHandoffsFetcher
	prevRows := statusRowsFetcher
	prevIncludeProposalSections := statusIncludeProposalSections
	defer func() {
		statusDispatchSummaryFetcher = prevSummary
		statusDispatchDebtFetcher = prevDebt
		statusHandoffsFetcher = prevHandoffs
		statusRowsFetcher = prevRows
		statusIncludeProposalSections = prevIncludeProposalSections
	}()

	statusDispatchSummaryFetcher = func() (statusDispatchSummary, error) {
		return statusDispatchSummary{
			Deuda: deudaDispatchResumen{
				Pendientes:    1,
				WorkConfirmed: 2,
				Total:         3,
			},
			Handoffs: 4,
		}, nil
	}
	statusDispatchDebtFetcher = func() (deudaDispatchResumen, error) {
		t.Fatal("no deberia consultar deuda dispatch separada si ya existe resumen combinado")
		return deudaDispatchResumen{}, nil
	}
	statusHandoffsFetcher = func() (int, error) {
		t.Fatal("no deberia consultar handoffs separados si ya existe resumen combinado")
		return 0, nil
	}
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		return nil, errors.New("rows degradado")
	}
	statusIncludeProposalSections = false

	status, err := fetchStatusFresh()
	if err != nil {
		t.Fatalf("fetchStatusFresh: %v", err)
	}
	if status.DeudaDispatch.WorkConfirmed != 2 || status.DeudaDispatch.Pendientes != 1 || status.DeudaDispatch.Total != 3 {
		t.Fatalf("deuda dispatch inesperada: %+v", status.DeudaDispatch)
	}
	if status.Autonomia.Handoffs != 4 {
		t.Fatalf("handoffs inesperados: %+v", status.Autonomia)
	}
}

func TestFetchStatusFastFallbackCompletaHandoffsDesdeFetcher(t *testing.T) {
	prevRowsFetcher := statusRowsFetcher
	prevHandoffs := statusHandoffsFetcher
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		statusHandoffsFetcher = prevHandoffs
	}()

	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		return nil, errors.New("rows degradado")
	}
	statusHandoffsFetcher = func() (int, error) {
		return 2, nil
	}

	status, err := fetchStatusFastFallback()
	if err != nil {
		t.Fatalf("fetchStatusFastFallback: %v", err)
	}
	if status.Autonomia.Handoffs != 2 {
		t.Fatalf("deberia completar handoffs desde el fetcher ligero, got=%+v", status.Autonomia)
	}
}

func TestFetchStatusFastFallbackIncluyeAutonomySurfaceCanonica(t *testing.T) {
	prevRowsFetcher := statusRowsFetcher
	prevSurfaceFetcher := statusAutonomySurfaceFetcher
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		statusAutonomySurfaceFetcher = prevSurfaceFetcher
	}()

	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		return nil, errors.New("rows degradado")
	}
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) {
		now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
		return &autonomySurface{
			Events: 1,
			ByKind: map[string]int{"task_reassigned": 1},
			LastAt: &now,
			Recent: []autonomySurfaceRecentItem{
				{Project: "orquestador", autonomyEventSummary: autonomyEventSummary{Kind: "task_reassigned", Agent: "Codex1", CreatedAt: now}},
			},
		}, nil
	}

	status, err := fetchStatusFastFallback()
	if err != nil {
		t.Fatalf("fetchStatusFastFallback: %v", err)
	}
	if status.AutonomySurface == nil || status.AutonomySurface.Events != 1 || status.AutonomySurface.ByKind["task_reassigned"] != 1 {
		t.Fatalf("autonomySurface inesperada: %+v", status.AutonomySurface)
	}
}

func TestFetchStatusFastFallbackIncluyeRiesgoGlobalCanonico(t *testing.T) {
	prevRowsFetcher := statusRowsFetcher
	prevSurfaceFetcher := statusAutonomySurfaceFetcher
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		statusAutonomySurfaceFetcher = prevSurfaceFetcher
	}()

	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		return nil, errors.New("rows degradado")
	}
	statusAutonomySurfaceFetcher = func() (*autonomySurface, error) {
		now := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
		return &autonomySurface{
			Highlights: []string{"frentes_bloqueantes=1", "integracion_bloqueada=10", "riesgo_top=orquestador(10)"},
			Projects: []autonomyProjectSurface{{
				Project:    "orquestador",
				Events:     2,
				LastAt:     &now,
				Highlights: []string{"riesgo=critico", "integracion_bloqueada=10", "review_gates=1", "runtime_orders=1"},
			}},
		}, nil
	}

	status, err := fetchStatusFastFallback()
	if err != nil {
		t.Fatalf("fetchStatusFastFallback: %v", err)
	}
	if status.CriticalProjectRisk == nil || status.CriticalProjectRisk.Project != "orquestador" || status.CriticalProjectRisk.Blocking != 10 {
		t.Fatalf("criticalProjectRisk inesperado: %+v", status.CriticalProjectRisk)
	}
	if status.AutonomySurface == nil || !containsStringWorkspace(status.AutonomySurface.Highlights, "riesgo_top=orquestador(10)") {
		t.Fatalf("autonomySurface sin riesgo canónico: %+v", status.AutonomySurface)
	}
	if len(status.AutonomyHighlights) == 0 || !containsStringWorkspace(status.AutonomyHighlights, "integracion_bloqueada=10") {
		t.Fatalf("autonomyHighlights sin riesgo canónico: %+v", status.AutonomyHighlights)
	}
}

func TestFetchStatusWorkspaceRiskSummaryUsaCockpitCanonicoSinRecursion(t *testing.T) {
	prevListProjects := workspaceControlListProjects
	prevCockpitBuilder := workspaceControlCockpitBuilder
	defer func() {
		workspaceControlListProjects = prevListProjects
		workspaceControlCockpitBuilder = prevCockpitBuilder
	}()

	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "infra"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		if slug != "infra" {
			t.Fatalf("slug inesperado: %q", slug)
		}
		return &apiProyectoCockpit{
			Proyecto:                &db.Proyecto{Slug: "infra"},
			AutonomyEvents:          1,
			AutonomyByKind:          map[string]int{"handoff_failed": 1},
			TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
			ReviewGatesAbiertas:     1,
			RuntimeOrdersAbiertas:   1,
			RuntimeMailboxPendiente: 1,
		}, nil
	}

	out, err := fetchStatusWorkspaceRiskSummary()
	if err != nil {
		t.Fatalf("fetchStatusWorkspaceRiskSummary: %v", err)
	}
	if out.CriticalProjectRisk == nil || out.CriticalProjectRisk.Project != "infra" || out.CriticalProjectRisk.Blocking != 14 {
		t.Fatalf("critical project risk inesperado: %+v", out.CriticalProjectRisk)
	}
	for _, token := range []string{"frentes_bloqueantes=1", "integracion_bloqueada=14", "riesgo_top=infra(14)"} {
		if !containsStringWorkspace(out.Highlights, token) {
			t.Fatalf("faltan highlights canónicos %q: %+v", token, out.Highlights)
		}
	}
}

func TestStoreStatusSnapshotUsaTTLReducidoSiNoHayActivosYHayTrabajo(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	storeStatusSnapshot(apiStatusResponse{
		TareasEnProgreso: []tareaLite{{ID: 502, Estado: db.TareaEnProgreso, Agente: "Codex3"}},
	}, now)

	statusCacheState.mu.Lock()
	expires := statusCacheState.expires
	statusCacheState.mu.Unlock()
	if got := expires.Sub(now); got != statusFallbackTTL {
		t.Fatalf("ttl inesperado para snapshot inconsistente: %s", got)
	}
}

func TestInvalidateStatusSnapshotCacheConservaValorPeroMarcaStale(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	prevNow := statusNowFunc
	statusNowFunc = func() time.Time { return now }
	defer func() { statusNowFunc = prevNow }()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Generado:          now.Format(time.RFC3339),
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1"}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1"}},
	}, now, time.Minute)

	invalidateStatusSnapshotCache()

	statusCacheState.mu.Lock()
	defer statusCacheState.mu.Unlock()
	if !statusCacheState.ok {
		t.Fatal("deberia conservar snapshot previa tras invalidacion suave")
	}
	if statusCacheState.value.Generado != now.Format(time.RFC3339) {
		t.Fatalf("snapshot conservada inesperada: %+v", statusCacheState.value)
	}
	if got := statusCacheState.expires.Sub(now); got != statusSoftInvalidateTTL {
		t.Fatalf("la invalidacion suave deberia recortar ttl a %s, got=%s", statusSoftInvalidateTTL, got)
	}
}

func TestInvalidateStatusSnapshotCacheHardInutilizaSnapshotHastaRefresh(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	prevNow := statusNowFunc
	statusNowFunc = func() time.Time { return now }
	defer func() { statusNowFunc = prevNow }()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Generado:          now.Format(time.RFC3339),
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1"}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1"}},
	}, now, time.Minute)

	if _, ok := readStatusSnapshotFresh(); !ok {
		t.Fatal("snapshot deberia existir fresco antes de invalidacion dura")
	}
	if _, ok := readStatusSnapshotAny(); !ok {
		t.Fatal("snapshot deberia existir por fallback antes de invalidacion dura")
	}

	invalidateStatusSnapshotCacheHard()

	if _, ok := readStatusSnapshotFresh(); ok {
		t.Fatal("snapshot no deberia seguir fresco tras invalidacion dura")
	}
	if _, ok := readStatusSnapshotAny(); ok {
		t.Fatal("snapshot no deberia seguir disponible por fallback tras invalidacion dura")
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsUsaEstadoOperativoCanónico(t *testing.T) {
	agentes := []*db.Agente{
		{Nombre: "Codex6", Rol: "programador"},
		{Nombre: "Codex7", Rol: "programador"},
		{Nombre: "Codex8", Rol: "programador"},
		{Nombre: "Codex9", Rol: "programador"},
		{Nombre: "Codex10", Rol: "programador"},
		{Nombre: "Codex11", Rol: "programador"},
	}
	rows := []agentesapp.Row{
		{Agente: agentes[0], EstadoOperativo: "arrancando"},
		{Agente: agentes[1], EstadoOperativo: "trabajando"},
		{Agente: agentes[2], EstadoOperativo: "disponible"},
		{Agente: agentes[3], EstadoOperativo: "bloqueado_por_runtime"},
		{Agente: agentes[4], EstadoOperativo: "saturado"},
		{Agente: agentes[5], EstadoOperativo: "atascado"},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows(agentes, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 4 {
		t.Fatalf("activos inesperados: %+v", activos)
	}
	if len(trabajando) != 2 {
		t.Fatalf("trabajando inesperados: %+v", trabajando)
	}
	if len(saturados) != 1 || saturados[0].Nombre != "Codex10" {
		t.Fatalf("saturados inesperados: %+v", saturados)
	}
	if len(atascados) != 1 || atascados[0].Nombre != "Codex11" {
		t.Fatalf("atascados inesperados: %+v", atascados)
	}
	if len(authManual) != 0 {
		t.Fatalf("auth manual inesperados: %+v", authManual)
	}
	if len(quotaBlocked) != 0 {
		t.Fatalf("quota bloqueados inesperados: %+v", quotaBlocked)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsNoCuentaArrancandoComoTrabajando(t *testing.T) {
	agente := &db.Agente{Nombre: "CodexStart", Rol: "programador"}
	rows := []agentesapp.Row{
		{Agente: agente, EstadoOperativo: "arrancando"},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 1 || activos[0].Nombre != "CodexStart" {
		t.Fatalf("activos inesperados: %+v", activos)
	}
	if len(trabajando) != 0 {
		t.Fatalf("arrancando no deberia contar como trabajando: %+v", trabajando)
	}
	if len(saturados) != 0 || len(atascados) != 0 || len(authManual) != 0 || len(quotaBlocked) != 0 {
		t.Fatalf("listas inesperadas: saturados=%+v atascados=%+v auth=%+v quota=%+v", saturados, atascados, authManual, quotaBlocked)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsCanonicalizaAliasCodex(t *testing.T) {
	agenteCanonico := &db.Agente{Nombre: "Codex2", Rol: "programador"}
	agenteAlias := &db.Agente{Nombre: "codex2", Rol: "programador"}
	rows := []agentesapp.Row{
		{Agente: &db.Agente{Nombre: "Codex2", Rol: "programador"}, EstadoOperativo: "atascado"},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agenteCanonico, agenteAlias}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(atascados) != 1 || atascados[0].Nombre != "Codex2" {
		t.Fatalf("atascados inesperados: %+v", atascados)
	}
	if len(activos) > 1 || len(trabajando) > 0 || len(saturados) > 0 || len(authManual) > 0 || len(quotaBlocked) > 0 {
		t.Fatalf("listas inesperadas: activos=%+v trabajando=%+v saturados=%+v auth=%+v quota=%+v", activos, trabajando, saturados, authManual, quotaBlocked)
	}
}

func TestAgenteBloqueadoPorCuotaVisibleSinSenalesNoBloquea(t *testing.T) {
	if agenteBloqueadoPorCuotaVisible(&db.Agente{Nombre: "Codex2", Rol: "programador"}) {
		t.Fatal("un agente sin ninguna señal de cuota no debería quedar bloqueado por defecto")
	}
}

func TestResumirAutonomiaLigeraDerivaSupervisorYTrabajoConfirmado(t *testing.T) {
	out := resumirAutonomiaLigera(
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
			{Nombre: "Codex2", Rol: "programador"},
			{Nombre: "Codex3", Rol: "programador"},
		},
		[]*db.Agente{
			{Nombre: "Codex2", Rol: "programador"},
		},
		[]tareaLite{
			{ID: 17, Estado: db.TareaEnProgreso, Agente: "Codex2"},
		},
		time.Now().UTC(),
	)
	if out.Supervisando != 1 {
		t.Fatalf("supervisando inesperado: %+v", out)
	}
	if out.Continuando != 0 {
		t.Fatalf("continuando inesperado: %+v", out)
	}
	if out.WorkConfirmed != 1 {
		t.Fatalf("work_confirmed inesperado: %+v", out)
	}
	if out.ContinuidadPendiente != 0 || out.Handoffs != 0 {
		t.Fatalf("resumen ligero no deberia inflar pendientes/handoffs: %+v", out)
	}
}

func TestResumirAutonomiaLigeraCuentaSupervisorTrabajandoComoConfirmado(t *testing.T) {
	out := resumirAutonomiaLigera(
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
		},
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
		},
		[]tareaLite{
			{ID: 22, Estado: db.TareaEnProgreso, Agente: "Codex1"},
		},
		time.Now().UTC(),
	)
	if out.Supervisando != 0 || out.Continuando != 0 || out.WorkConfirmed != 1 {
		t.Fatalf("resumen ligero inesperado: %+v", out)
	}
}

func TestResumirAutonomiaLigeraCanonicalizaAliasCodex(t *testing.T) {
	out := resumirAutonomiaLigera(
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
			{Nombre: "Codex2", Rol: "programador"},
		},
		[]*db.Agente{
			{Nombre: "codex2", Rol: "programador"},
		},
		[]tareaLite{
			{ID: 22, Estado: db.TareaEnProgreso, Agente: "Codex2"},
			{ID: 23, Estado: db.TareaEnProgreso, Agente: "codex2"},
		},
		time.Now().UTC(),
	)
	if out.Supervisando != 1 || out.Continuando != 0 || out.WorkConfirmed != 1 {
		t.Fatalf("resumen ligero inesperado con alias canonico: %+v", out)
	}
}

func TestStatusVisibleWorkerCountersFiltraSupervisorRealContado(t *testing.T) {
	autonomia := autonomiaResumen{Supervisando: 1}
	autonomia.addSupervisorName("Codex2")
	workersConectados, workersTrabajando, supervisoresActivos := statusVisibleWorkerCounters(
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
			{Nombre: "Codex4", Rol: "programador"},
		},
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
			{Nombre: "Codex4", Rol: "programador"},
		},
		autonomia,
	)
	if workersConectados != 2 || workersTrabajando != 2 || supervisoresActivos != 1 {
		t.Fatalf("contadores visibles inesperados: conectados=%d trabajando=%d supervisores=%d", workersConectados, workersTrabajando, supervisoresActivos)
	}
}

func TestStatusVisibleWorkerCountersUsaSupervisorConfiguradoAntesDeDescontarPorConteo(t *testing.T) {
	prevConfig := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfig
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "OpenClaw", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	workersConectados, workersTrabajando, supervisoresActivos := statusVisibleWorkerCounters(
		[]*db.Agente{{Nombre: "Codex10", Rol: "programador"}},
		[]*db.Agente{{Nombre: "Codex10", Rol: "programador"}},
		autonomiaResumen{Supervisando: 1},
	)
	if workersConectados != 1 || workersTrabajando != 1 || supervisoresActivos != 1 {
		t.Fatalf("contadores visibles inesperados con supervisor configurado externo: conectados=%d trabajando=%d supervisores=%d", workersConectados, workersTrabajando, supervisoresActivos)
	}
}

func TestReconciledVisibleWorkerCountersRecomponeWorkersConSupervisorStale(t *testing.T) {
	autonomia := autonomiaResumen{Supervisando: 1}
	autonomia.addSupervisorName("CodexSupervisor")

	workersConectados, workersTrabajando, supervisoresActivos := reconciledVisibleWorkerCounters(
		0,
		0,
		1,
		[]*db.Agente{{Nombre: "Codex10", Rol: "programador"}},
		[]*db.Agente{{Nombre: "Codex10", Rol: "programador"}},
		autonomia,
	)
	if workersConectados != 1 || workersTrabajando != 1 || supervisoresActivos != 1 {
		t.Fatalf("contadores reconciliados inesperados: conectados=%d trabajando=%d supervisores=%d", workersConectados, workersTrabajando, supervisoresActivos)
	}
}

func TestReconciledVisibleWorkerCountersConservaSupervisorSinWorkers(t *testing.T) {
	autonomia := autonomiaResumen{Supervisando: 1}
	autonomia.addSupervisorName("CodexSupervisor")

	workersConectados, workersTrabajando, supervisoresActivos := reconciledVisibleWorkerCounters(
		0,
		0,
		1,
		[]*db.Agente{{Nombre: "CodexSupervisor", Rol: "supervisor"}},
		[]*db.Agente{{Nombre: "CodexSupervisor", Rol: "supervisor"}},
		autonomia,
	)
	if workersConectados != 0 || workersTrabajando != 0 || supervisoresActivos != 1 {
		t.Fatalf("contadores reconciliados inesperados sin workers: conectados=%d trabajando=%d supervisores=%d", workersConectados, workersTrabajando, supervisoresActivos)
	}
}

func TestResumirAutonomiaEventosRecientesCompactaResumenGlobal(t *testing.T) {
	prevFetcher := statusAutonomyEventsFetcher
	prevWindow := statusAutonomyEventsWindow
	prevLimit := statusAutonomyEventsFetchLimit
	prevRecent := statusAutonomyEventsRecentLimit
	defer func() {
		statusAutonomyEventsFetcher = prevFetcher
		statusAutonomyEventsWindow = prevWindow
		statusAutonomyEventsFetchLimit = prevLimit
		statusAutonomyEventsRecentLimit = prevRecent
	}()

	now := time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC)
	statusAutonomyEventsWindow = 24 * time.Hour
	statusAutonomyEventsFetchLimit = 20
	statusAutonomyEventsRecentLimit = 2
	statusAutonomyEventsFetcher = func(since time.Time, limit int) ([]autonomyEventSummary, error) {
		if !since.Equal(now.Add(-24 * time.Hour)) {
			t.Fatalf("since inesperado: %s", since)
		}
		if limit != 20 {
			t.Fatalf("limit inesperado: %d", limit)
		}
		return []autonomyEventSummary{
			{Kind: "task_reassigned", CreatedAt: now.Add(-20 * time.Minute), TargetAgent: "Codex4"},
			{Kind: "handoff_completed", CreatedAt: now.Add(-5 * time.Minute), TargetAgent: "Codex1"},
			{Kind: "task_reassigned", CreatedAt: now.Add(-10 * time.Minute), TargetAgent: "Codex2"},
		}, nil
	}

	out, err := resumirAutonomiaEventosRecientes(autonomiaResumen{Supervisando: 1}, now)
	if err != nil {
		t.Fatalf("resumirAutonomiaEventosRecientes: %v", err)
	}
	if out.Supervisando != 1 {
		t.Fatalf("deberia conservar resumen base: %+v", out)
	}
	if out.Count != 3 {
		t.Fatalf("count inesperado: %+v", out)
	}
	if out.ByKind["task_reassigned"] != 2 || out.ByKind["handoff_completed"] != 1 {
		t.Fatalf("by_kind inesperado: %+v", out.ByKind)
	}
	if out.LastAt == nil || !out.LastAt.Equal(now.Add(-5*time.Minute)) {
		t.Fatalf("last_at inesperado: %+v", out.LastAt)
	}
	if len(out.Recent) != 2 || out.Recent[0].Kind != "handoff_completed" || out.Recent[1].Kind != "task_reassigned" {
		t.Fatalf("recent inesperado: %+v", out.Recent)
	}
}

func TestStatusSnapshotIsDegenerateNoDescartaAutonomiaReciente(t *testing.T) {
	if statusSnapshotIsDegenerate(apiStatusResponse{
		Autonomia: autonomiaResumen{
			Count:  1,
			ByKind: map[string]int{"task_reassigned": 1},
			Recent: []autonomyEventSummary{{Kind: "task_reassigned", CreatedAt: time.Now().UTC()}},
		},
	}) {
		t.Fatal("autonomia reciente no deberia considerarse snapshot degenerado")
	}
}

func TestResumirAutonomiaLigeraNoCuentaComoSupervisorAConfiguradoSiEstaEjecutandoTarea(t *testing.T) {
	prevAutonomy := statusListAutonomyFetcher
	prevConfig := statusConfigGet
	defer func() {
		statusListAutonomyFetcher = prevAutonomy
		statusConfigGet = prevConfig
	}()

	statusConfigGet = func(string) (string, error) { return "Codex1", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) {
		return []*db.ProyectoAutonomia{{
			ProyectoID:        1,
			Enabled:           true,
			SupervisorAgente:  "Codex2",
			ReserveSupervisor: true,
			EstadoAutonomia:   db.AutonomiaProyectoActiva,
		}}, nil
	}

	out := resumirAutonomiaLigera(
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
			{Nombre: "Codex2", Rol: "programador"},
			{Nombre: "Codex4", Rol: "programador"},
		},
		[]*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
		},
		[]tareaLite{
			{ID: 26, Estado: db.TareaEnProgreso, Agente: "Codex1"},
		},
		time.Now().UTC(),
	)
	if out.Supervisando != 1 || out.Continuando != 0 || out.WorkConfirmed != 1 {
		t.Fatalf("el supervisor configurado con tarea activa debe contar como worker y no inflar supervisando: %+v", out)
	}
}

func TestBuildUltraLiteStatusReadOnlyCuentaSupervisorDesdePolicyProyecto(t *testing.T) {
	prevAutonomy := statusListAutonomyFetcher
	prevConfig := statusConfigGet
	defer func() {
		statusListAutonomyFetcher = prevAutonomy
		statusConfigGet = prevConfig
	}()

	statusConfigGet = func(string) (string, error) { return "Codex1", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) {
		return []*db.ProyectoAutonomia{{
			ProyectoID:           1,
			Enabled:              true,
			SupervisorAgente:     "Codex2",
			ReserveSupervisor:    true,
			EstadoAutonomia:      db.AutonomiaProyectoActiva,
			ObjetivoGeneral:      "Terminar la app",
			DefinitionOfDoneJSON: "{}",
		}}, nil
	}

	status := buildUltraLiteStatusReadOnly(time.Now().UTC(),
		[]*db.Agente{
			{Nombre: "Codex2", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"},
		},
		map[string]int{"en_progreso": 1},
		[]tareaLite{{ID: 31, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	)
	if status.Autonomia.Supervisando != 1 {
		t.Fatalf("deberia contar supervisor reservado por policy de proyecto: %+v", status.Autonomia)
	}
	if status.Autonomia.Continuando != 0 {
		t.Fatalf("continuando inesperado: %+v", status.Autonomia)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsCuentaAtascadoSoloSiTieneRuntimeUtil(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex11", Rol: "programador"}
	rows := []agentesapp.Row{
		{
			Agente:          agente,
			EstadoOperativo: "atascado",
			Handle:          &db.RuntimeHandle{Estado: "activo"},
		},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 1 || activos[0].Nombre != "Codex11" {
		t.Fatalf("el atascado con handle activo debe seguir contando como conectado: %+v", activos)
	}
	if len(trabajando) != 0 || len(saturados) != 0 {
		t.Fatalf("trabajando/saturados inesperados: %+v %+v", trabajando, saturados)
	}
	if len(atascados) != 1 || atascados[0].Nombre != "Codex11" {
		t.Fatalf("atascados inesperados: %+v", atascados)
	}
	if len(authManual) != 0 {
		t.Fatalf("auth manual inesperados: %+v", authManual)
	}
	if len(quotaBlocked) != 0 {
		t.Fatalf("quota bloqueados inesperados: %+v", quotaBlocked)
	}
}

func TestResumirAutonomiaRowsCuentaSupervisorConNotaSupervisionAutomatica(t *testing.T) {
	now := time.Now().UTC()
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente:             &db.Agente{Nombre: "Codex2"},
		Asignacion:         &db.Asignacion{Agente: "Codex2", ProyectoID: 1, Estado: db.AsignacionActiva, Nota: "supervision_automatica"},
		WorkerAlive:        true,
		WorkerHeartbeat:    ptrTimeStatus(now),
		WorkerUpdatedAt:    ptrTimeStatus(now),
		EstadoOperativo:    "trabajando",
		LastAutonomyAction: "continuar_trabajo",
	}}, now)
	if resumen.Supervisando != 1 {
		t.Fatalf("deberia contar supervisor con nota supervision_automatica: %+v", resumen)
	}
	if resumen.Continuando != 0 {
		t.Fatalf("no deberia contar como continuando a la vez: %+v", resumen)
	}
}

func TestResumirAutonomiaRowsMantieneSupervisorConHeartbeatQuietoSiRuntimeSigueFresco(t *testing.T) {
	now := time.Now().UTC()
	hb := now.Add(-70 * time.Second)
	lastSeen := now.Add(-20 * time.Second)
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Codex2"},
		Asignacion:      &db.Asignacion{Agente: "Codex2", ProyectoID: 1, Estado: db.AsignacionActiva, Nota: "supervision_automatica"},
		WorkerAlive:     true,
		WorkerState:     "ready",
		WorkerHeartbeat: &hb,
		Handle:          &db.RuntimeHandle{Estado: "activo", LastSeenAt: &lastSeen},
		EstadoOperativo: "disponible",
	}}, now)
	if resumen.Supervisando != 1 {
		t.Fatalf("deberia mantener supervisor con runtime fresco aunque heartbeat este quieto: %+v", resumen)
	}
}

func TestResumirAutonomiaRowsCuentaSupervisorBloqueadoPorRuntimeSiSigueVivo(t *testing.T) {
	now := time.Now().UTC()
	hb := now.Add(-70 * time.Second)
	lastSeen := now.Add(-20 * time.Second)
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente:             &db.Agente{Nombre: "Codex2"},
		Asignacion:         &db.Asignacion{Agente: "Codex2", ProyectoID: 1, Estado: db.AsignacionActiva, Nota: "supervision_automatica"},
		WorkerAlive:        true,
		WorkerState:        "ready",
		WorkerHeartbeat:    &hb,
		Handle:             &db.RuntimeHandle{Estado: "activo", LastSeenAt: &lastSeen},
		EstadoOperativo:    "bloqueado_por_runtime",
		DetalleOperativo:   "heartbeat worker retrasado",
		LastAutonomyAction: "supervisar_proyecto",
		LastAutonomySource: "work_queue",
		LastAutonomyState:  "work_confirmed",
		LastAutonomyMoment: &now,
	}}, now)
	if resumen.Supervisando != 1 {
		t.Fatalf("deberia contar supervisor vivo aunque su estado operativo sea bloqueado_por_runtime: %+v", resumen)
	}
	if resumen.Continuando != 0 {
		t.Fatalf("no deberia contar como worker continuando: %+v", resumen)
	}
}

func TestResumirAutonomiaRowsNoCuentaContinuityPendingSiWorkerYaTieneTareaActiva(t *testing.T) {
	now := time.Now().UTC()
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente:                   &db.Agente{Nombre: "Codex1"},
		Handle:                   &db.RuntimeHandle{Estado: "activo", LastSeenAt: &now},
		WorkerAlive:              true,
		WorkerHeartbeat:          &now,
		LastAutonomyAction:       "continuar_trabajo",
		LastAutonomySource:       "work_queue",
		LastAutonomyState:        "pending",
		MailboxContinuityPending: 1,
		OpenTasks:                1,
	}}, now)
	if resumen.Continuando != 1 {
		t.Fatalf("deberia seguir contando como continuando: %+v", resumen)
	}
	if resumen.ContinuidadPendiente != 0 {
		t.Fatalf("no deberia contar continuity_pending si el worker ya absorbio el trabajo: %+v", resumen)
	}
}

func TestResumirAutonomiaRowsNoCuentaContinuityPendingSiSupervisorSoloInspeccionaSignalDeBajoValor(t *testing.T) {
	now := time.Now().UTC()
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente: &db.Agente{Nombre: "Codex2"},
		Asignacion: &db.Asignacion{
			Agente: "Codex2",
			Estado: db.AsignacionActiva,
			Nota:   "supervision_automatica",
		},
		Handle:                   &db.RuntimeHandle{Estado: "activo", LastSeenAt: &now},
		WorkerAlive:              true,
		WorkerHeartbeat:          &now,
		WorkerUpdatedAt:          &now,
		EstadoOperativo:          "trabajando",
		LastAutonomyAction:       "inspeccionar_transcript_signal",
		LastAutonomySource:       "work_queue",
		LastAutonomyState:        "pending",
		LastAutonomyReason:       "agente=Codex1 signal=tool_exploration transcript=123",
		MailboxContinuityPending: 1,
	}}, now)
	if resumen.Supervisando != 1 {
		t.Fatalf("deberia seguir contando supervisor: %+v", resumen)
	}
	if resumen.ContinuidadPendiente != 0 {
		t.Fatalf("no deberia contar continuity_pending por signal de bajo valor: %+v", resumen)
	}
}

func TestResumirAutonomiaRowsNoCuentaContinuityPendingSiSupervisorSoloArrastraGuidanceDurableEnInbox(t *testing.T) {
	now := time.Now().UTC()
	resumen := resumirAutonomiaRows([]agentesapp.Row{{
		Agente: &db.Agente{Nombre: "Codex2"},
		Asignacion: &db.Asignacion{
			Agente: "Codex2",
			Estado: db.AsignacionActiva,
			Nota:   "supervision_automatica",
		},
		Handle:                   &db.RuntimeHandle{Estado: "activo", LastSeenAt: &now},
		WorkerAlive:              true,
		WorkerHeartbeat:          &now,
		WorkerUpdatedAt:          &now,
		EstadoOperativo:          "trabajando",
		LastAutonomyAction:       "inspeccionar_transcript_signal",
		LastAutonomySource:       "work_queue",
		LastAutonomyState:        "pending",
		LastAutonomyReason:       "guidance durable escrita en inbox",
		MailboxContinuityPending: 1,
	}}, now)
	if resumen.Supervisando != 1 {
		t.Fatalf("deberia seguir contando supervisor: %+v", resumen)
	}
	if resumen.ContinuidadPendiente != 0 {
		t.Fatalf("no deberia contar continuity_pending por guidance durable en inbox: %+v", resumen)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsOmiteResiduoPausadoSinTrabajo(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex4", Rol: "programador"}
	rows := []agentesapp.Row{
		{
			Agente:          agente,
			EstadoOperativo: "atascado",
			Asignacion:      &db.Asignacion{Estado: db.AsignacionPausada, Nota: "handoff_cedido"},
			Handle:          &db.RuntimeHandle{Estado: "activo"},
			Runtime:         &db.RuntimeInstance{LogicalState: "activo"},
		},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 0 || len(trabajando) != 0 || len(saturados) != 0 || len(atascados) != 0 || len(authManual) != 0 || len(quotaBlocked) != 0 {
		t.Fatalf("el residuo pausado sin trabajo no deberia contar como visible: activos=%+v trabajando=%+v saturados=%+v atascados=%+v auth=%+v quota=%+v", activos, trabajando, saturados, atascados, authManual, quotaBlocked)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsSeparaAutenticacionManual(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex2", Rol: "programador"}
	rows := []agentesapp.Row{
		{
			Agente:           agente,
			EstadoOperativo:  "bloqueado_por_runtime",
			DetalleOperativo: "worker requiere autenticacion manual",
		},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 0 || len(trabajando) != 0 || len(saturados) != 0 || len(atascados) != 0 {
		t.Fatalf("no deberia contar el agente auth como activo: activos=%+v trabajando=%+v saturados=%+v atascados=%+v", activos, trabajando, saturados, atascados)
	}
	if len(authManual) != 1 || authManual[0].Nombre != "Codex2" {
		t.Fatalf("auth manual inesperados: %+v", authManual)
	}
	if len(quotaBlocked) != 0 {
		t.Fatalf("quota bloqueados inesperados: %+v", quotaBlocked)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsSeparaBloqueadoPorCuota(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex2", Rol: "programador"}
	rows := []agentesapp.Row{
		{
			Agente:           agente,
			EstadoOperativo:  "bloqueado_por_cuota",
			DetalleOperativo: "worker bloqueado por cuota",
			WorkerState:      "blocked_quota",
		},
	}

	activos, trabajando, saturados, atascados, authManual, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 0 || len(trabajando) != 0 || len(saturados) != 0 || len(atascados) != 0 {
		t.Fatalf("no deberia contar el agente bloqueado por cuota como activo: activos=%+v trabajando=%+v saturados=%+v atascados=%+v", activos, trabajando, saturados, atascados)
	}
	if len(authManual) != 0 {
		t.Fatalf("auth manual inesperados: %+v", authManual)
	}
	if len(quotaBlocked) != 1 || quotaBlocked[0].Nombre != "Codex2" {
		t.Fatalf("quota bloqueados inesperados: %+v", quotaBlocked)
	}
}

func TestAplicarVisibilidadOperativaAgentesSincronizaCuotaBloqueadaDerivada(t *testing.T) {
	agente := &db.Agente{Nombre: "Codex3", Activo: true, EstadoCuota: "activo"}
	rows := []agentesapp.Row{
		{
			Agente:           &db.Agente{Nombre: "Codex3"},
			EstadoOperativo:  "bloqueado_por_cuota",
			DetalleOperativo: "Presupuesto agotado observado",
		},
	}

	aplicarVisibilidadOperativaAgentes([]*db.Agente{agente}, rows)

	if agente.Activo {
		t.Fatalf("no deberia seguir activo tras bloqueo por cuota: %+v", agente)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado cuota inesperado: %+v", agente)
	}
	if agente.MotivoPausa != "Presupuesto agotado observado" {
		t.Fatalf("motivo pausa inesperado: %+v", agente)
	}
}

func TestAgentesVisiblesPorEstadoOperativoRowsOmiteActivoSiAgenteSigueEnCuotaVisible(t *testing.T) {
	agente := &db.Agente{
		Nombre:             "Codex4",
		Rol:                "programador",
		EstadoCuota:        "enfriamiento",
		ReanimarAt:         timePtr(time.Now().UTC().Add(30 * time.Minute)),
		PresupuestoResetAt: nil,
	}
	rows := []agentesapp.Row{
		{
			Agente:             agente,
			EstadoOperativo:    "trabajando",
			DetalleOperativo:   "worker running",
			LastAutonomyAction: "continuar_trabajo",
			LastAutonomyState:  "work_confirmed",
			OpenTasks:          1,
		},
	}

	activos, trabajando, _, _, _, quotaBlocked, ok := agentesVisiblesPorEstadoOperativoRows([]*db.Agente{agente}, rows)
	if !ok {
		t.Fatalf("debería resolver filas operativas")
	}
	if len(activos) != 0 || len(trabajando) != 0 {
		t.Fatalf("no deberia seguir activo visible en cuota: activos=%+v trabajando=%+v", activos, trabajando)
	}
	if len(quotaBlocked) != 1 || quotaBlocked[0].Nombre != "Codex4" {
		t.Fatalf("quota bloqueados inesperados: %+v", quotaBlocked)
	}
	if got := resumirAutonomiaRows(rows, time.Now().UTC()); got.Continuando != 0 || got.WorkConfirmed != 0 {
		t.Fatalf("autonomia no deberia contar trabajo en cuota visible: %+v", got)
	}
}

func TestReconciliarConteoTareasActivasVisibleUsaSoloLaListaVisible(t *testing.T) {
	cuentas := map[string]int{
		string(db.TareaEnProgreso): 4,
		string(db.TareaAsignada):   2,
		string(db.TareaBloqueada):  3,
		string(db.TareaCompletada): 7,
	}
	tareasActivas := []tareaLite{
		{ID: 478, Estado: db.TareaEnProgreso, Agente: "alberto"},
		{ID: 492, Estado: db.TareaEnProgreso, Agente: "alberto"},
		{ID: 502, Estado: db.TareaEnProgreso, Agente: "alberto"},
		{ID: 511, Estado: db.TareaBloqueada, Agente: "Codex2"},
	}

	got := reconciliarConteoTareasActivasVisible(cuentas, tareasActivas)
	if got[string(db.TareaEnProgreso)] != 3 {
		t.Fatalf("en_progreso visible inesperado: %+v", got)
	}
	if got[string(db.TareaAsignada)] != 0 {
		t.Fatalf("asignada visible inesperada: %+v", got)
	}
	if got[string(db.TareaBloqueada)] != 1 {
		t.Fatalf("bloqueada visible inesperada: %+v", got)
	}
	if got[string(db.TareaCompletada)] != 7 {
		t.Fatalf("completada no deberia cambiar: %+v", got)
	}
}

func TestFiltrarTareasActivasVisiblesOcultaAgenteDeshabilitadoYPreservaOrquesta(t *testing.T) {
	agentes := []*db.Agente{
		{Nombre: "Codex1", Habilitado: true},
		{Nombre: "antigravity", Habilitado: false},
	}
	tareas := []tareaLite{
		{ID: 535, Estado: db.TareaEnProgreso, Agente: "antigravity", Titulo: "legacy"},
		{ID: 548, Estado: db.TareaEnProgreso, Agente: "Codex1", Titulo: "real"},
		{ID: 628, Estado: db.TareaEnProgreso, Agente: "orquesta", Titulo: "interna"},
	}

	got := filtrarTareasActivasVisibles(tareas, agentes)
	if len(got) != 2 {
		t.Fatalf("tareas visibles inesperadas: %+v", got)
	}
	if got[0].ID != 548 || got[1].ID != 628 {
		t.Fatalf("orden/filtrado inesperado: %+v", got)
	}
}

func TestFiltrarTareasActivasVisiblesOcultaAgenteFueraDeFlotaOficial(t *testing.T) {
	agentes := []*db.Agente{
		{Nombre: "Codex1", Habilitado: true},
		{Nombre: "Claude2", Habilitado: true},
	}
	tareas := []tareaLite{
		{ID: 548, Estado: db.TareaEnProgreso, Agente: "Codex1", Titulo: "real"},
		{ID: 549, Estado: db.TareaEnProgreso, Agente: "Claude2", Titulo: "fuera"},
	}

	got := filtrarTareasActivasVisibles(tareas, agentes)
	if len(got) != 1 || got[0].ID != 548 {
		t.Fatalf("tareas visibles inesperadas: %+v", got)
	}
}
