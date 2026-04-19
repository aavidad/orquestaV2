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
	prevNow := statusNowFunc
	prevTTL := statusSnapshotTTL
	prevAsync := statusAsyncRefresh
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFetcher
		statusFastFetcher = prevFastFetcher
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
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

func TestAgentRowsForStatusWithinTimeoutDegradaRapido(t *testing.T) {
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
