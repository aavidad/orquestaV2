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
			ConteoTareas: map[string]int{
				"asignada": 1,
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
