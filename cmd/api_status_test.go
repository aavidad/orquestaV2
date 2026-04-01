package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
	if len(payload.TareasEnProgreso) != 1 || payload.TareasEnProgreso[0].ID != 7 {
		t.Fatalf("tareasEnProgreso inesperadas: %+v", payload.TareasEnProgreso)
	}
	if len(payload.TareasReservadas) != 1 || payload.TareasReservadas[0].ID != 8 {
		t.Fatalf("tareasReservadas inesperadas: %+v", payload.TareasReservadas)
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
