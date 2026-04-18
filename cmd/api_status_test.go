package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
			DeudaDispatch: deudaDispatchResumen{Total: 4, Pendientes: 1, Notificadas: 2, Fallidas: 1},
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
	if payload.DeudaDispatch.Total != 4 || payload.DeudaDispatch.Pendientes != 1 || payload.DeudaDispatch.Notificadas != 2 || payload.DeudaDispatch.Fallidas != 1 {
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
