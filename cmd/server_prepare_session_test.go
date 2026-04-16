package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func intPtr(v int) *int { return &v }

func TestLimpiarFlotaFueraDePoolCierraNoPoolYReseteaPausaOperativa(t *testing.T) {
	now := time.Now().UTC()
	var resets []string
	var finishes []string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/agentes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAgentesResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "antigravity", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento", EstadoSesion: "esperando", MotivoPausa: "pausa manual", ReanimarAt: &now, PresupuestoSemanalPct: intPtr(69)},
				{Nombre: "Codex6", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento", EstadoSesion: "pausada", MotivoPausa: "usage limit", ReanimarAt: &now, PresupuestoCheckedAt: &now, PresupuestoSemanalPct: intPtr(0)},
				{Nombre: "CodexBusy", Rol: "programador", Activo: true, EstadoCuota: "activo", EstadoSesion: "programando"},
			},
		})
	})
	mux.HandleFunc("/api/agentes/antigravity/reset-reanimacion", func(w http.ResponseWriter, r *http.Request) {
		resets = append(resets, "antigravity")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/agentes/Codex6/reset-reanimacion", func(w http.ResponseWriter, r *http.Request) {
		resets = append(resets, "Codex6")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/fin", func(w http.ResponseWriter, r *http.Request) {
		var req apiSesionFinRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		finishes = append(finishes, strings.TrimSpace(req.Agente))
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	status := &estadoResumen{
		TareasActivas: []tareaLite{
			{ID: 410, Agente: "CodexBusy", Estado: db.TareaEnProgreso, Titulo: "ocupado"},
		},
	}
	allowed := map[string]struct{}{
		"codex3": {},
		"codex4": {},
	}

	out, err := limpiarFlotaFueraDePool(srv.URL, allowed, status)
	if err != nil {
		t.Fatalf("limpiarFlotaFueraDePool: %v", err)
	}
	if out == nil {
		t.Fatalf("respuesta nil")
	}
	if out.ReanimationsReset != 1 {
		t.Fatalf("reanimations_reset=%d, want 1", out.ReanimationsReset)
	}
	if out.SessionsFinished != 2 {
		t.Fatalf("sessions_finished=%d, want 2", out.SessionsFinished)
	}
	if got := strings.Join(out.Agents, ","); got != "Codex6,antigravity" {
		t.Fatalf("agents=%q, want %q", got, "Codex6,antigravity")
	}
	if got := strings.Join(resets, ","); got != "antigravity" {
		t.Fatalf("resets=%q, want %q", got, "antigravity")
	}
	if got := strings.Join(finishes, ","); got != "antigravity,Codex6" && got != "Codex6,antigravity" {
		t.Fatalf("finishes=%q", got)
	}
}

func TestServerPrepararSesionAllowedAgentsIncluyeFlotaOficial(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]string{
				"server_autobootstrap_supervisor_agent": "antigravity",
				"server_autobootstrap_worker_agents":    "Codex2,Codex3,Claude2,Codex4,Codex5",
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	allowed, err := serverPrepararSesionAllowedAgents(srv.URL)
	if err != nil {
		t.Fatalf("serverPrepararSesionAllowedAgents: %v", err)
	}
	for _, nombre := range []string{"codex2", "codex3", "codex4", "codex5"} {
		if _, ok := allowed[nombre]; !ok {
			t.Fatalf("faltaba %s en la flota oficial: %#v", nombre, allowed)
		}
	}
	if _, ok := allowed["codex1"]; ok {
		t.Fatalf("no deberia conservar supervisor no oficial como fallback implicito: %#v", allowed)
	}
	if _, ok := allowed["antigravity"]; ok {
		t.Fatalf("antigravity no deberia pertenecer a la flota oficial: %#v", allowed)
	}
	if _, ok := allowed["claude2"]; ok {
		t.Fatalf("claude2 no deberia pertenecer a la flota oficial: %#v", allowed)
	}
}
