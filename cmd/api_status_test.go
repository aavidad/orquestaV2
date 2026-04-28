package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/db"
)

type stubStatusService struct {
	response apiStatusResponse
	err      error
}

func (s stubStatusService) FetchStatus() (apiStatusResponse, error) {
	return s.response, s.err
}

type blockingStatusService struct {
	release <-chan struct{}
}

func (s blockingStatusService) FetchStatus() (apiStatusResponse, error) {
	<-s.release
	return apiStatusResponse{}, nil
}

func TestAPIHandlerStatusReturnsPayload(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	status := apiStatusResponse{
		Agentes: []*db.Agente{{Nombre: "CodexX", Activo: true, EstadoCuota: "activo"}},
		AgentesActivos: []*db.Agente{
			{Nombre: "CodexX", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "CodexX", Activo: true, EstadoCuota: "activo"},
		},
		AgentesAuthManual: []*db.Agente{
			{Nombre: "CodexLogin"},
		},
		ConteoTareas: map[string]int{
			"asignada": 1,
		},
		TareasPorEstado: map[string]int{
			"asignada": 1,
		},
		TareasEnProgreso: []tareaLite{
			{ID: 7, Titulo: "En progreso", Estado: db.TareaEnProgreso, Agente: "CodexX"},
		},
		TareasReservadas: []tareaLite{
			{ID: 8, Titulo: "Reservada", Estado: db.TareaAsignada, Agente: "CodexX"},
		},
		PoolsLocales: []*capacidadapp.PoolLocalCompartido{
			{PoolSlug: "ollama-gemma4", ModeloPreferente: "gemma4:26b", SlotsMaximos: 1},
		},
		DeudaDispatch: deudaDispatchResumen{Total: 4, Pendientes: 1, Notificadas: 2, Fallidas: 1, WorkConfirmed: 3},
		Autonomia: autonomiaResumen{
			Supervisando:         1,
			Continuando:          2,
			ContinuidadPendiente: 3,
			Count:                2,
			ByKind:               map[string]int{"task_reassigned": 1, "handoff_completed": 1},
			Recent: []autonomyEventSummary{
				{Kind: "handoff_completed", CreatedAt: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC), TargetAgent: "CodexX"},
			},
		},
		AutonomySurface: &autonomySurface{
			Events: 2,
			ByKind: map[string]int{"handoff_completed": 1, "task_reassigned": 1},
			LastAt: &now,
			Highlights: []string{"task_reassigned=1", "integracion_bloqueada=10", "riesgo_top=orquestador(10)"},
			Recent: []autonomySurfaceRecentItem{
				{Project: "orquestador", autonomyEventSummary: autonomyEventSummary{Kind: "handoff_completed", TargetAgent: "CodexX", CreatedAt: now}},
			},
			Projects: []autonomyProjectSurface{{
				Project:    "orquestador",
				Events:     2,
				LastAt:     &now,
				Highlights: []string{"riesgo=critico", "integracion_bloqueada=10", "review_gates=1"},
			}},
		},
	}
	status.Autonomia.addSupervisorName("CodexSupervisor")
	statusFastFetcher = func() (apiStatusResponse, error) { return status, nil }
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) { return apiStatusResponse{}, false }
	rec := httptest.NewRecorder()
	writeAPIStatusPayload(rec, status)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(payload.Agentes) != 1 || payload.Agentes[0].Nombre != "CodexX" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.ResumenTareas["asignada"] != 1 {
		t.Fatalf("resumenTareas inesperado: %+v", payload.ResumenTareas)
	}
	if len(payload.AgentesTrabajando) != 1 || payload.AgentesTrabajando[0].Nombre != "CodexX" {
		t.Fatalf("agentesTrabajando inesperado: %+v", payload.AgentesTrabajando)
	}
	if len(payload.AgentesAuthManual) != 1 || payload.AgentesAuthManual[0].Nombre != "CodexLogin" {
		t.Fatalf("agentesAuthManual inesperado: %+v", payload.AgentesAuthManual)
	}
	if len(payload.TareasEnProgreso) != 1 || payload.TareasEnProgreso[0].ID != 7 {
		t.Fatalf("tareasEnProgreso inesperadas: %+v", payload.TareasEnProgreso)
	}
	if len(payload.TareasReservadas) != 1 || payload.TareasReservadas[0].ID != 8 {
		t.Fatalf("tareasReservadas inesperadas: %+v", payload.TareasReservadas)
	}
	if len(payload.PoolsLocales) != 1 || payload.PoolsLocales[0].PoolSlug != "ollama-gemma4" {
		t.Fatalf("poolsLocales inesperados: %+v", payload.PoolsLocales)
	}
	if payload.DeudaDispatch.Total != 4 || payload.DeudaDispatch.Pendientes != 1 || payload.DeudaDispatch.Notificadas != 2 || payload.DeudaDispatch.Fallidas != 1 || payload.DeudaDispatch.WorkConfirmed != 3 {
		t.Fatalf("deudaDispatch inesperada: %+v", payload.DeudaDispatch)
	}
	if payload.Autonomia.Supervisando != 1 || payload.Autonomia.Continuando != 2 || payload.Autonomia.ContinuidadPendiente != 3 {
		t.Fatalf("autonomia inesperada: %+v", payload.Autonomia)
	}
	if payload.Autonomia.Count != 2 || payload.Autonomia.ByKind["task_reassigned"] != 1 || len(payload.Autonomia.Recent) != 1 || payload.Autonomia.Recent[0].Kind != "handoff_completed" {
		t.Fatalf("autonomia reciente inesperada: %+v", payload.Autonomia)
	}
	if payload.WorkersConectados != 1 || payload.WorkersTrabajando != 1 || payload.SupervisoresActivos != 1 {
		t.Fatalf("counters de workers/supervisores inesperados: %+v", payload)
	}
	if payload.AutonomySurface == nil || payload.AutonomySurface.Events != 2 || payload.AutonomySurface.ByKind["handoff_completed"] != 1 {
		t.Fatalf("autonomySurface inesperada: %+v", payload.AutonomySurface)
	}
	if payload.CriticalProjectRisk == nil || payload.CriticalProjectRisk.Project != "orquestador" || payload.CriticalProjectRisk.Blocking != 10 {
		t.Fatalf("criticalProjectRisk inesperado: %+v", payload.CriticalProjectRisk)
	}
	if len(payload.AutonomyHighlights) == 0 || !containsStringWorkspace(payload.AutonomyHighlights, "riesgo_top=orquestador(10)") {
		t.Fatalf("autonomyHighlights inesperados: %+v", payload.AutonomyHighlights)
	}
}

func TestBuildUltraLiteStatusReadOnlyIncluyeAutonomiaReciente(t *testing.T) {
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

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	statusAutonomyEventsWindow = 24 * time.Hour
	statusAutonomyEventsFetchLimit = 10
	statusAutonomyEventsRecentLimit = 4
	statusAutonomyEventsFetcher = func(since time.Time, limit int) ([]autonomyEventSummary, error) {
		return []autonomyEventSummary{
			{Kind: "worker_recovery_requested", CreatedAt: now.Add(-15 * time.Minute), Agent: "Codex1"},
		}, nil
	}

	status := buildUltraLiteStatusReadOnly(now,
		[]*db.Agente{{Nombre: "Codex1", Activo: true, EstadoCuota: "activo"}},
		map[string]int{"en_progreso": 1},
		[]tareaLite{{ID: 24, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
	)
	if status.Autonomia.Count != 1 {
		t.Fatalf("count autonomy inesperado: %+v", status.Autonomia)
	}
	if status.Autonomia.ByKind["worker_recovery_requested"] != 1 {
		t.Fatalf("by_kind autonomy inesperado: %+v", status.Autonomia.ByKind)
	}
	if len(status.Autonomia.Recent) != 1 || status.Autonomia.Recent[0].Kind != "worker_recovery_requested" {
		t.Fatalf("recent autonomy inesperado: %+v", status.Autonomia.Recent)
	}
	if status.CriticalProjectRisk != nil {
		t.Fatalf("criticalProjectRisk debería quedar nil sin autonomy surface: %+v", status.CriticalProjectRisk)
	}
	if len(status.AutonomyHighlights) != 0 {
		t.Fatalf("autonomyHighlights no deberían inventarse sin autonomy surface: %+v", status.AutonomyHighlights)
	}
	if status.AutonomySurface != nil {
		t.Fatalf("autonomySurface inesperada: %+v", status.AutonomySurface)
	}
}

func TestAPIHandlerStatusIgnoraSnapshotFrescoQueNecesitaRefreshInmediato(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:          []*db.Agente{{Nombre: "CodexStale"}},
		AgentesActivos:   []*db.Agente{{Nombre: "CodexStale"}},
		TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "CodexStale"}},
		Autonomia:        autonomiaResumen{ContinuidadPendiente: 2},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)

	statusFastFetcher = func() (apiStatusResponse, error) { return apiStatusResponse{}, errStatusFetchTimeout }
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) { return apiStatusResponse{}, false }
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{
				string(db.TareaEnProgreso): 1,
			}, []*db.Tarea{
				{ID: 7, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
			}, nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(payload.Agentes) != 1 || payload.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("deberia ignorar snapshot fresco que requiere refresh y usar read-only: %+v", payload)
	}
	if payload.Autonomia.ContinuidadPendiente != 0 {
		t.Fatalf("autonomia inesperada tras refresh directo: %+v", payload.Autonomia)
	}
}

func TestAPIHandlerStatusReturnsDegradedPayloadWhenNoSnapshotNorFallback(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	statusFastFetcher = func() (apiStatusResponse, error) { return apiStatusResponse{}, errStatusFetchTimeout }
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) { return apiStatusResponse{}, false }
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if payload.Generado == "" || len(payload.Agentes) != 0 {
		t.Fatalf("payload degradado inesperado: %+v", payload)
	}
}

func TestAPIHandlerStatusReturnsDegradedPayloadOnTimeout(t *testing.T) {
	prev := statusService
	prevFast := statusFastFetcher
	defer func() {
		statusService = prev
		statusFastFetcher = prevFast
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	statusService = stubStatusService{err: errStatusFetchTimeout}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode degraded payload: %v", err)
	}
	for _, key := range []string{"agentes", "conteo_tareas", "resumenTareas", "generado", "tareasPorEstado", "agentesActivos", "propuestasAbiertas", "tareasActivas"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload degradado sin clave %q: %+v", key, payload)
		}
	}
}

func TestAPIHandlerStatusReturnsDegradedPayloadWhenServiceHangs(t *testing.T) {
	prevService := statusService
	prevTimeout := apiStatusFetchTimeout
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		statusService = prevService
		apiStatusFetchTimeout = prevTimeout
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	release := make(chan struct{})
	statusService = blockingStatusService{release: release}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusFetchTimeout = 20 * time.Millisecond

	start := time.Now()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	elapsed := time.Since(start)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("deberia degradar rapido sin esperar el servicio colgado, elapsed=%s", elapsed)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode degraded payload: %v", err)
	}
	if _, ok := payload["generado"]; !ok {
		t.Fatalf("payload degradado inesperado: %+v", payload)
	}
}

func TestAPIHandlerStatusUsaFallbackDirectoSiServiceTimeouta(t *testing.T) {
	prevService := statusService
	prevTimeout := apiStatusFetchTimeout
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		statusService = prevService
		apiStatusFetchTimeout = prevTimeout
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	release := make(chan struct{})
	statusService = blockingStatusService{release: release}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{
			Agentes:        []*db.Agente{{Nombre: "CodexFast"}},
			AgentesActivos: []*db.Agente{{Nombre: "CodexFast"}},
		}, nil
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusFetchTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(payload.Agentes) != 1 || payload.Agentes[0].Nombre != "CodexFast" {
		t.Fatalf("payload fallback inesperado: %+v", payload)
	}
}

func TestFetchStatusForAPIAllowDirectFallbackNoLanzaRefreshAntesDeDegradar(t *testing.T) {
	prevFresh := statusFreshFetcher
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevFreshTimeout := statusFreshTimeout
	prevFastTimeout := statusFastTimeout
	prevBackoff := statusFailureBackoffTTL
	resetStatusSnapshotCache()
	defer func() {
		statusFreshFetcher = prevFresh
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		statusFreshTimeout = prevFreshTimeout
		statusFastTimeout = prevFastTimeout
		statusFailureBackoffTTL = prevBackoff
		resetStatusSnapshotCache()
	}()

	statusFastTimeout = 20 * time.Millisecond
	statusFreshTimeout = 100 * time.Millisecond
	statusFailureBackoffTTL = 50 * time.Millisecond
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}

	started := make(chan struct{}, 1)
	statusFreshFetcher = func() (apiStatusResponse, error) {
		started <- struct{}{}
		return apiStatusResponse{Generado: "fresh"}, nil
	}

	start := time.Now()
	status, err := fetchStatusForAPIAllowDirectFallback(3*time.Second, statusFastTimeout)
	elapsed := time.Since(start)
	if !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("deberia degradar con timeout rapido, status=%+v err=%v", status, err)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("no deberia esperar el refresh fresco en frio, elapsed=%s", elapsed)
	}
	select {
	case <-started:
		t.Fatalf("no deberia lanzar refresh fresco desde el wrapper antes de degradar")
	default:
	}
}

func TestFetchStatusForAPIAllowDirectFallbackPrefiereReadOnlyAntesDeFastFallback(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	resetStatusSnapshotCache()
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		resetStatusSnapshotCache()
	}()

	block := make(chan struct{})
	statusFastFetcher = func() (apiStatusResponse, error) {
		<-block
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		<-block
		return apiStatusResponse{}, false
	}
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 1}, []*db.Tarea{
			{ID: 5, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}

	start := time.Now()
	status, err := fetchStatusForAPIAllowDirectFallback(3*time.Second, 50*time.Millisecond)
	elapsed := time.Since(start)
	close(block)
	if err != nil {
		t.Fatalf("deberia resolver por read-only directo: %v", err)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("no deberia esperar al fast fallback bloqueado, elapsed=%s", elapsed)
	}
	if len(status.Agentes) != 1 || status.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("status read-only inesperado: %+v", status)
	}
}

func TestFetchStatusFallbackRacePrefiereFastSiLlegaTrasGraciaCorta(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevGrace := apiStatusFallbackRichGrace
	t.Cleanup(func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusFallbackRichGrace = prevGrace
	})

	apiStatusFallbackRichGrace = 40 * time.Millisecond
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{
			Generado:         "ultra",
			AgentesActivos:   []*db.Agente{{Nombre: "CodexLite", Activo: true}},
			TareasEnProgreso: []tareaLite{},
		}, true
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		time.Sleep(10 * time.Millisecond)
		return apiStatusResponse{
			Generado:          "fast",
			AgentesActivos:    []*db.Agente{{Nombre: "CodexFast", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "CodexFast", Activo: true}},
			TareasEnProgreso:  []tareaLite{{ID: 26, Estado: db.TareaEnProgreso, Agente: "CodexFast"}},
		}, nil
	}

	status, err := fetchStatusFallbackRace(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchStatusFallbackRace: %v", err)
	}
	if status.Generado != "fast" || len(status.AgentesTrabajando) != 1 || status.AgentesTrabajando[0].Nombre != "CodexFast" {
		t.Fatalf("deberia preferir resultado fast: %+v", status)
	}
}

func TestFetchStatusFallbackRaceDevuelveUltraLiteSiFastNoLlegaEnGracia(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevGrace := apiStatusFallbackRichGrace
	t.Cleanup(func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusFallbackRichGrace = prevGrace
	})

	apiStatusFallbackRichGrace = 15 * time.Millisecond
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{
			Generado:       "ultra",
			AgentesActivos: []*db.Agente{{Nombre: "CodexLite", Activo: true}},
		}, true
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		time.Sleep(40 * time.Millisecond)
		return apiStatusResponse{
			Generado:          "fast",
			AgentesActivos:    []*db.Agente{{Nombre: "CodexFast", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "CodexFast", Activo: true}},
		}, nil
	}

	status, err := fetchStatusFallbackRace(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("fetchStatusFallbackRace: %v", err)
	}
	if status.Generado != "ultra" || len(status.AgentesActivos) != 1 || status.AgentesActivos[0].Nombre != "CodexLite" {
		t.Fatalf("deberia caer a ultra-lite si fast no llega a tiempo: %+v", status)
	}
}

func TestFetchStatusForAPIAllowDirectFallbackLanzaRefreshAsyncTrasReadOnly(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevFresh := statusFreshFetcher
	prevFreshTimeout := statusFreshTimeout
	prevBackoff := statusFailureBackoffTTL
	resetStatusSnapshotCache()
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusFreshFetcher = prevFresh
		statusFreshTimeout = prevFreshTimeout
		statusFailureBackoffTTL = prevBackoff
		resetStatusSnapshotCache()
	}()

	statusFastFetcher = func() (apiStatusResponse, error) { return apiStatusResponse{}, errStatusFetchTimeout }
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) { return apiStatusResponse{}, false }
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{
				string(db.TareaEnProgreso): 1,
			}, []*db.Tarea{
				{ID: 5, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
			}, nil
	}
	started := make(chan struct{}, 1)
	statusFreshFetcher = func() (apiStatusResponse, error) {
		started <- struct{}{}
		return apiStatusResponse{Agentes: []*db.Agente{{Nombre: "CodexFresh"}}}, nil
	}
	statusFreshTimeout = 100 * time.Millisecond
	statusFailureBackoffTTL = 20 * time.Millisecond

	status, err := fetchStatusForAPIAllowDirectFallback(3*time.Second, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("fetchStatusForAPIAllowDirectFallback: %v", err)
	}
	if len(status.Agentes) != 1 || status.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("status read-only inesperado: %+v", status)
	}
	select {
	case <-started:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("deberia lanzar refresh async tras servir snapshot read-only")
	}
}

func TestFetchStatusForAPIAllowDirectFallbackIgnoraSnapshotFrescoQueNecesitaRefresh(t *testing.T) {
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	resetStatusSnapshotCache()
	defer func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		resetStatusSnapshotCache()
	}()

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:          []*db.Agente{{Nombre: "CodexStale"}},
		AgentesActivos:   []*db.Agente{{Nombre: "CodexStale"}},
		TareasEnProgreso: []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "CodexStale"}},
		Autonomia:        autonomiaResumen{ContinuidadPendiente: 1},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)

	statusFastFetcher = func() (apiStatusResponse, error) { return apiStatusResponse{}, errStatusFetchTimeout }
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) { return apiStatusResponse{}, false }
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{
				string(db.TareaEnProgreso): 1,
			}, []*db.Tarea{
				{ID: 5, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
			}, nil
	}

	status, err := fetchStatusForAPIAllowDirectFallback(3*time.Second, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("fetchStatusForAPIAllowDirectFallback: %v", err)
	}
	if len(status.Agentes) != 1 || status.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("deberia ignorar snapshot fresco que requiere refresh: %+v", status)
	}
	if status.Autonomia.ContinuidadPendiente != 0 {
		t.Fatalf("autonomia inesperada tras refresh directo: %+v", status.Autonomia)
	}
}

func TestAPIHandlerStatusUsaFallbackUltraLiteTrasTimeoutDirecto(t *testing.T) {
	prevService := statusService
	prevTimeout := apiStatusFetchTimeout
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	defer func() {
		statusService = prevService
		apiStatusFetchTimeout = prevTimeout
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	release := make(chan struct{})
	statusService = blockingStatusService{release: release}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{
			Agentes:           []*db.Agente{{Nombre: "CodexLite"}},
			AgentesActivos:    []*db.Agente{{Nombre: "CodexLite"}},
			ConteoTareas:      map[string]int{"en_progreso": 2},
			TareasPorEstado:   map[string]int{"en_progreso": 2},
			TareasEnProgreso:  make([]tareaLite, 2),
			TareasReservadas:  []tareaLite{},
			AgentesTrabajando: []*db.Agente{{Nombre: "CodexLite"}},
			Generado:          "ultralite",
		}, true
	}
	apiStatusReadOnlyLiteFetcher = nil
	apiStatusFetchTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if payload.Generado != "ultralite" || len(payload.Agentes) != 1 || payload.Agentes[0].Nombre != "CodexLite" {
		t.Fatalf("payload ultralite inesperado: %+v", payload)
	}
	if len(payload.TareasEnProgreso) != 0 {
		t.Fatalf("tareasEnProgreso ultralite deberian filtrar placeholders vacios: %+v", payload.TareasEnProgreso)
	}
	if len(payload.TareasReservadas) != 0 {
		t.Fatalf("tareasReservadas ultralite deberian filtrar placeholders vacios: %+v", payload.TareasReservadas)
	}
	if payload.WorkersConectados != 1 || payload.WorkersTrabajando != 1 || payload.SupervisoresActivos != 0 {
		t.Fatalf("counters ultralite inesperados: %+v", payload)
	}
}

func TestAPIHandlerStatusUsaFallbackReadOnlyDirectoTrasTimeout(t *testing.T) {
	prevService := statusService
	prevTimeout := apiStatusFetchTimeout
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	defer func() {
		statusService = prevService
		apiStatusFetchTimeout = prevTimeout
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		resetStatusSnapshotCache()
	}()
	resetStatusSnapshotCache()
	release := make(chan struct{})
	statusService = blockingStatusService{release: release}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 1}, []*db.Tarea{
			{ID: 7, Titulo: "Tarea viva", Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}
	apiStatusFetchTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(payload.Agentes) != 1 || payload.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("payload read-only inesperado: %+v", payload)
	}
	if len(payload.TareasEnProgreso) != 1 || payload.TareasEnProgreso[0].Agente != "CodexRO" {
		t.Fatalf("tareasEnProgreso read-only inesperadas: %+v", payload.TareasEnProgreso)
	}
	if len(payload.AgentesTrabajando) != 1 || payload.AgentesTrabajando[0].Nombre != "CodexRO" {
		t.Fatalf("agentesTrabajando read-only inesperados: %+v", payload.AgentesTrabajando)
	}
	if payload.WorkersConectados != 1 || payload.WorkersTrabajando != 1 || payload.SupervisoresActivos != 0 {
		t.Fatalf("counters read-only inesperados: %+v", payload)
	}
}

func TestFetchStatusUltraLiteFallbackPrefiereReadOnlyFetcher(t *testing.T) {
	prevRO := apiStatusReadOnlyLiteFetcher
	prevList := statusListAgentsFetcher
	prevCount := statusCountTasksFetcher
	defer func() {
		apiStatusReadOnlyLiteFetcher = prevRO
		statusListAgentsFetcher = prevList
		statusCountTasksFetcher = prevCount
	}()

	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 3}, []*db.Tarea{
			{ID: 7, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}
	statusListAgentsFetcher = func() ([]*db.Agente, error) {
		time.Sleep(50 * time.Millisecond)
		return nil, errors.New("slow primary")
	}
	statusCountTasksFetcher = func() (map[string]int, error) {
		time.Sleep(50 * time.Millisecond)
		return nil, errors.New("slow primary")
	}

	status, ok := fetchStatusUltraLiteFallback(100 * time.Millisecond)
	if !ok {
		t.Fatalf("deberia construir fallback ultraligero desde read-only")
	}
	if len(status.Agentes) != 1 || status.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("agentes inesperados: %+v", status.Agentes)
	}
	if status.ConteoTareas["en_progreso"] != 1 {
		t.Fatalf("conteo inesperado: %+v", status.ConteoTareas)
	}
	if len(status.TareasEnProgreso) != 1 || status.TareasEnProgreso[0].Agente != "CodexRO" {
		t.Fatalf("tareasEnProgreso inesperadas: %+v", status.TareasEnProgreso)
	}
	if len(status.AgentesTrabajando) != 1 || status.AgentesTrabajando[0].Nombre != "CodexRO" {
		t.Fatalf("agentesTrabajando inesperados: %+v", status.AgentesTrabajando)
	}
	if status.WorkersConectados != 1 || status.WorkersTrabajando != 1 || status.SupervisoresActivos != 0 {
		t.Fatalf("counters fallback inesperados: %+v", status)
	}
}

func TestFetchStatusUltraLiteFallbackNoDuplicaTimeoutsEnSerie(t *testing.T) {
	prevRO := apiStatusReadOnlyLiteFetcher
	prevList := statusListAgentsFetcher
	prevCount := statusCountTasksFetcher
	defer func() {
		apiStatusReadOnlyLiteFetcher = prevRO
		statusListAgentsFetcher = prevList
		statusCountTasksFetcher = prevCount
	}()

	block := make(chan struct{})
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		<-block
		return nil, nil, nil, errors.New("timeout")
	}
	statusListAgentsFetcher = func() ([]*db.Agente, error) {
		<-block
		return nil, errors.New("timeout")
	}
	statusCountTasksFetcher = func() (map[string]int, error) {
		<-block
		return nil, errors.New("timeout")
	}

	start := time.Now()
	_, ok := fetchStatusUltraLiteFallback(50 * time.Millisecond)
	elapsed := time.Since(start)
	close(block)
	if ok {
		t.Fatalf("no deberia resolver fallback")
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("no deberia pagar timeouts en serie, elapsed=%s", elapsed)
	}
}

func TestAPIAgenteOverviewReturns503WhenBuilderHangs(t *testing.T) {
	prevBuilder := apiAgentDetailBuilder
	prevTimeout := apiAgentOverviewTimeout
	defer func() {
		apiAgentDetailBuilder = prevBuilder
		apiAgentOverviewTimeout = prevTimeout
	}()
	release := make(chan struct{})
	apiAgentDetailBuilder = func(nombre string, compact bool) (*agentesapp.Detail, error) {
		<-release
		return &agentesapp.Detail{}, nil
	}
	apiAgentOverviewTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes/Codex2/overview?compact=true", nil)
	apiRouterAgentes(rec, req)
	close(release)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAgentesPanelReturnsEmptyFallbackWhenBuilderHangs(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevTimeout := apiAgentsPanelTimeout
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		apiAgentsPanelTimeout = prevTimeout
		apiStatusUltraLiteFetcher = prevUltraLite
		resetAgentPanelSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	release := make(chan struct{})
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		<-release
		return []agentesapp.Row{}, nil
	}
	apiAgentsPanelTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel", nil)
	apiHandlerAgentes(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "{\"rows\":[]}" {
		t.Fatalf("respuesta inesperada: %s", rec.Body.String())
	}
}

func TestAPIAgentesPanelUsesCachedSnapshotWhenBuilderHangs(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevTimeout := apiAgentsPanelTimeout
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		apiAgentsPanelTimeout = prevTimeout
		resetAgentPanelSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	storeAgentPanelSnapshot([]agentesapp.Row{{EstadoOperativo: "trabajando"}}, time.Now().UTC())
	release := make(chan struct{})
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		<-release
		return []agentesapp.Row{}, nil
	}
	apiAgentsPanelTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel", nil)
	apiHandlerAgentes(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"rows\"") {
		t.Fatalf("respuesta inesperada: %s", rec.Body.String())
	}
}

func TestAPIAgentesPanelUsesStatusSnapshotFallbackWhenNoPanelSnapshot(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevTimeout := apiAgentsPanelTimeout
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		apiAgentsPanelTimeout = prevTimeout
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex2", Activo: true}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex2", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex2", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Agente: "Codex2", Estado: db.TareaEnProgreso}},
		Generado:          time.Now().UTC().Format(time.RFC3339),
	}, time.Now().UTC(), time.Minute)
	release := make(chan struct{})
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		<-release
		return []agentesapp.Row{}, nil
	}
	apiAgentsPanelTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel", nil)
	apiHandlerAgentes(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Codex2") || !strings.Contains(rec.Body.String(), "trabajando") {
		t.Fatalf("respuesta inesperada: %s", rec.Body.String())
	}
}

func TestAPIAgentesPanelUsesUltraLiteStatusFallbackWhenNoSnapshots(t *testing.T) {
	prevBuilder := apiAgentPanelRowsBuilder
	prevTimeout := apiAgentsPanelTimeout
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		apiAgentPanelRowsBuilder = prevBuilder
		apiAgentsPanelTimeout = prevTimeout
		apiStatusUltraLiteFetcher = prevUltraLite
		resetAgentPanelSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetAgentPanelSnapshotCache()
	resetStatusSnapshotCache()
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{
			Agentes:           []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesActivos:    []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex3", Activo: true}},
			TareasEnProgreso:  []tareaLite{{ID: 1, Agente: "Codex3", Estado: db.TareaEnProgreso}},
			Generado:          time.Now().UTC().Format(time.RFC3339),
		}, true
	}
	release := make(chan struct{})
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		<-release
		return []agentesapp.Row{}, nil
	}
	apiAgentsPanelTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel", nil)
	apiHandlerAgentes(rec, req)
	close(release)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"rows\"") {
		t.Fatalf("respuesta inesperada: %s", rec.Body.String())
	}
}

func TestNormalizeAgentPanelRowsForAPICompactaContinuidadAbsorbida(t *testing.T) {
	now := time.Now().UTC()
	rows := normalizeAgentPanelRowsForAPI([]agentesapp.Row{{
		Agente:                   &db.Agente{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		EstadoOperativo:          "trabajando",
		OpenTasks:                1,
		MailboxPending:           1,
		MailboxContinuityPending: 1,
		LastAutonomyAction:       "continuar_trabajo",
		LastAutonomySource:       "work_queue",
		LastAutonomyState:        "pending",
		WorkerAlive:              true,
		WorkerHeartbeat:          &now,
	}}, now)
	if len(rows) != 1 {
		t.Fatalf("rows inesperadas: %+v", rows)
	}
	if rows[0].MailboxPending != 0 || rows[0].MailboxContinuityPending != 0 {
		t.Fatalf("panel API deberia compactar continuidad absorbida: %+v", rows[0])
	}
}

func TestAPIAgentesPresupuestoRefrescarReturns503WhenRefreshHangs(t *testing.T) {
	prevExecutor := apiAgentBudgetRefreshExecutor
	prevTimeout := apiAgentBudgetRefreshTimeout
	defer func() {
		apiAgentBudgetRefreshExecutor = prevExecutor
		apiAgentBudgetRefreshTimeout = prevTimeout
	}()
	release := make(chan struct{})
	apiAgentBudgetRefreshExecutor = func(string) (int, error) {
		<-release
		return 0, nil
	}
	apiAgentBudgetRefreshTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/presupuesto/refrescar", bytes.NewReader([]byte(`{"agente":"Codex2"}`)))
	req.Header.Set("Content-Type", "application/json")
	apiHandlerAgentesPresupuestoRefrescar(rec, req)
	close(release)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAgentePrepararReturns503WhenBuilderHangs(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	prevBuilder := apiAgentPrepareBuilder
	prevTimeout := apiAgentPrepareTimeout
	defer func() {
		apiAgentPrepareBuilder = prevBuilder
		apiAgentPrepareTimeout = prevTimeout
	}()
	release := make(chan struct{})
	apiAgentPrepareBuilder = func(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error) {
		<-release
		return &agentesapp.PrepareOutput{}, nil
	}
	apiAgentPrepareTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agente/preparar?agente=Codex2&proyecto=orquestador", nil)
	apiHandlerAgentePreparar(rec, req)
	close(release)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRunAPIAgentPrepareLimitedTimeoutNoDejaTrabajoEnCola(t *testing.T) {
	prevLimiter := apiAgentPrepareLimiter
	apiAgentPrepareLimiter = make(chan struct{}, 1)
	defer func() { apiAgentPrepareLimiter = prevLimiter }()

	apiAgentPrepareLimiter <- struct{}{}
	defer func() {
		select {
		case <-apiAgentPrepareLimiter:
		default:
		}
	}()

	var calls atomic.Int32
	_, err := runAPIAgentPrepareLimited(20*time.Millisecond, func() (*agentesapp.PrepareOutput, error) {
		calls.Add(1)
		return &agentesapp.PrepareOutput{}, nil
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	<-apiAgentPrepareLimiter
	time.Sleep(40 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("builder no deberia ejecutarse tras timeout de cola, got=%d", calls.Load())
	}
}

func TestAPIAgenteTickReturns503WhenProcessorHangs(t *testing.T) {
	prevProcessor := apiAgentTickProcessor
	prevTimeout := apiAgentTickTimeout
	prevLimiter := apiAgentTickLimiter
	defer func() {
		apiAgentTickProcessor = prevProcessor
		apiAgentTickTimeout = prevTimeout
		apiAgentTickLimiter = prevLimiter
	}()

	release := make(chan struct{})
	apiAgentTickProcessor = func(input agentesapp.TickInput) (*agentesapp.TickOutput, error) {
		<-release
		return &agentesapp.TickOutput{}, nil
	}
	apiAgentTickTimeout = 20 * time.Millisecond
	apiAgentTickLimiter = make(chan struct{}, 1)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/tick", bytes.NewReader([]byte(`{"agente":"Codex2","proyecto":"orquestador"}`)))
	req.Header.Set("Content-Type", "application/json")
	apiHandlerAgenteTick(rec, req)
	close(release)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRunAPIAgentTickLimitedTimeoutNoDejaTrabajoEnCola(t *testing.T) {
	prevLimiter := apiAgentTickLimiter
	apiAgentTickLimiter = make(chan struct{}, 1)
	defer func() { apiAgentTickLimiter = prevLimiter }()

	apiAgentTickLimiter <- struct{}{}
	defer func() {
		select {
		case <-apiAgentTickLimiter:
		default:
		}
	}()

	var calls atomic.Int32
	_, err := runAPIAgentTickLimited(20*time.Millisecond, func() (*agentesapp.TickOutput, error) {
		calls.Add(1)
		return &agentesapp.TickOutput{}, nil
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	<-apiAgentTickLimiter
	time.Sleep(40 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("processor no deberia ejecutarse tras timeout de cola, got=%d", calls.Load())
	}
}
