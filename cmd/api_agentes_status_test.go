package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orquesta/db"
)

func TestAPIAgentesUsaSnapshotDeStatusCuandoEstaDisponible(t *testing.T) {
	prev := statusService
	statusService = stubStatusService{response: apiStatusResponse{
		Agentes: []*db.Agente{{
			Nombre:      "Codex3",
			EstadoCuota: "activo",
			Activo:      false,
		}},
	}}
	t.Cleanup(func() {
		statusService = prev
	})

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	if len(resp.Agentes) != 1 || resp.Agentes[0] == nil {
		t.Fatalf("agentes inesperados: %+v", resp.Agentes)
	}
	if resp.Agentes[0].Nombre != "Codex3" || resp.Agentes[0].EstadoCuota != "activo" {
		t.Fatalf("deberia exponer la misma foto que status: %+v", resp.Agentes[0])
	}
}
