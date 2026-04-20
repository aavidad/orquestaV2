package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	prev := statusService
	defer func() { statusService = prev }()
	statusService = stubStatusService{
		response: apiStatusResponse{
			Agentes: []*db.Agente{{Nombre: "CodexX"}},
			AgentesActivos: []*db.Agente{
				{Nombre: "CodexX"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "CodexX"},
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
		},
	}
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
}

func TestAPIHandlerStatusPropagatesError(t *testing.T) {
	prev := statusService
	defer func() { statusService = prev }()
	statusService = stubStatusService{err: errors.New("boom")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAPIHandlerStatusReturns503OnTimeout(t *testing.T) {
	prev := statusService
	defer func() { statusService = prev }()
	statusService = stubStatusService{err: errStatusFetchTimeout}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIHandlerStatusReturns503WhenServiceHangs(t *testing.T) {
	prevService := statusService
	prevTimeout := apiStatusFetchTimeout
	defer func() {
		statusService = prevService
		apiStatusFetchTimeout = prevTimeout
	}()
	release := make(chan struct{})
	statusService = blockingStatusService{release: release}
	apiStatusFetchTimeout = 20 * time.Millisecond

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	apiHandlerStatus(rec, req)
	close(release)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
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
