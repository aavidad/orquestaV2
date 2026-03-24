/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestProgresoVerUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/progreso", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
			t.Fatalf("proyecto inesperado: %q", got)
		}
		_ = json.NewEncoder(w).Encode(apiProgresoResumenResponse{
			Resumen: &db.ResumenProgresoProyecto{
				Proyecto:          "orquestador",
				ProgresoPct:       62.5,
				TareasTotales:     8,
				TareasCompletadas: 5,
				Fases: []*db.FaseProgresoDetalle{
					{Fase: &db.FaseProyecto{Orden: 10, Nombre: "Analisis"}, ProgresoPct: 80, TareasCompletadas: 4, TareasTotales: 5},
				},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := progresoVerCmd.RunE(progresoVerCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("progreso ver via API: %v", err)
		}
	})
	for _, token := range []string{"Proyecto: orquestador", "Progreso: 62.5%", "[10] Analisis"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestProgresoFaseListarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/progreso/fases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(apiProgresoFasesResponse{
			Fases: []*db.FaseProyecto{
				{ID: 7, Proyecto: "orquestador", Nombre: "Implementacion", Orden: 20, Peso: 2, Estado: "activa"},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := progresoFaseListarCmd.RunE(progresoFaseListarCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("progreso fase listar via API: %v", err)
		}
	})
	if !strings.Contains(out, "Implementacion") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestProgresoFaseRegistrarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/progreso/fases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiProgresoFaseRegistrarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Proyecto != "orquestador" || req.Nombre != "QA" {
			t.Fatalf("request inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiProgresoFaseResponse{ID: 12})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = progresoFaseRegistrarCmd.Flags().Set("nombre", "QA")
	_ = progresoFaseRegistrarCmd.Flags().Set("descripcion", "pruebas")
	_ = progresoFaseRegistrarCmd.Flags().Set("orden", "30")
	_ = progresoFaseRegistrarCmd.Flags().Set("peso", "1.5")
	_ = progresoFaseRegistrarCmd.Flags().Set("estado", "activa")

	out := capturarStdout(t, func() {
		if err := progresoFaseRegistrarCmd.RunE(progresoFaseRegistrarCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("progreso fase registrar via API: %v", err)
		}
	})
	if !strings.Contains(out, "Fase #12 registrada") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestProgresoTareaRegistrarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/progreso/tareas/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiProgresoTareaRegistrarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Proyecto != "orquestador" || req.ActualizadoPor != "Codex1" || req.ProgresoPct != 55 {
			t.Fatalf("request inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = progresoTareaRegistrarCmd.Flags().Set("proyecto", "orquestador")
	_ = progresoTareaRegistrarCmd.Flags().Set("fase", "0")
	_ = progresoTareaRegistrarCmd.Flags().Set("pct", "55")
	_ = progresoTareaRegistrarCmd.Flags().Set("agente", "Codex1")

	out := capturarStdout(t, func() {
		if err := progresoTareaRegistrarCmd.RunE(progresoTareaRegistrarCmd, []string{"42"}); err != nil {
			t.Fatalf("progreso tarea registrar via API: %v", err)
		}
	})
	if !strings.Contains(out, "Avance registrado para tarea #42") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}
