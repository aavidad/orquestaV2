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
	"time"

	"orquesta/db"
)

func TestSesionPresupuestoVerUsaAPI(t *testing.T) {
	checkedAt := time.Date(2026, 3, 24, 18, 40, 0, 0, time.UTC)
	remainingSeconds := int64(900)
	remainingMessages := int64(3)
	ratio := 0.08

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/presupuesto", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		if got := r.URL.Query().Get("agente"); got != "Codex1" {
			t.Fatalf("agente inesperado: %q", got)
		}
		_ = json.NewEncoder(w).Encode(apiSesionPresupuestoResponse{
			Presupuesto: &db.PresupuestoSesion{
				ID:                91,
				SesionID:          77,
				ModelSlug:         "gpt-5.4",
				WindowKind:        "rolling",
				RemainingSeconds:  &remainingSeconds,
				RemainingMessages: &remainingMessages,
				BudgetSource:      "api",
				CheckedAt:         checkedAt,
			},
			Evaluacion: &db.EvaluacionPresupuesto{
				Estado:         "handoff_preventivo",
				DebeHandoff:    true,
				Motivo:         "quedan 900 s",
				RemainingRatio: &ratio,
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = sesionPresupuestoVerCmd.Flags().Set("sesion", "0")
	_ = sesionPresupuestoVerCmd.Flags().Set("agente", "Codex1")

	out := capturarStdout(t, func() {
		if err := sesionPresupuestoVerCmd.RunE(sesionPresupuestoVerCmd, nil); err != nil {
			t.Fatalf("sesion presupuesto ver via API: %v", err)
		}
	})

	for _, token := range []string{
		"PRESUPUESTO — sesión #77",
		"Estado:         handoff_preventivo",
		"Budget source:  api",
		"Remaining secs: 900",
		"Ratio restante: 0.08",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestSesionPresupuestoRegistrarUsaAPI(t *testing.T) {
	checkedAt := time.Date(2026, 3, 24, 18, 45, 0, 0, time.UTC)
	remainingSeconds := int64(600)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/presupuesto", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiSesionPresupuestoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Agente != "Codex1" {
			t.Fatalf("agente inesperado: %q", req.Agente)
		}
		if req.BudgetSource != "api" {
			t.Fatalf("budget source inesperado: %q", req.BudgetSource)
		}
		_ = json.NewEncoder(w).Encode(apiSesionPresupuestoResponse{
			ID: 88,
			Presupuesto: &db.PresupuestoSesion{
				ID:               88,
				SesionID:         77,
				ModelSlug:        "gpt-5.4",
				WindowKind:       "rolling",
				RemainingSeconds: &remainingSeconds,
				BudgetSource:     "api",
				CheckedAt:        checkedAt,
			},
			Evaluacion: &db.EvaluacionPresupuesto{
				Estado:      "handoff_preventivo",
				DebeHandoff: true,
				Motivo:      "quedan 600 s",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = sesionPresupuestoRegistrarCmd.Flags().Set("sesion", "0")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("agente", "Codex1")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("pool-id", "0")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("model", "gpt-5.4")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("window-kind", "rolling")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("window-started-at", "")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("reset-at", "")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("remaining-seconds", "600")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("remaining-messages", "-1")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("remaining-tokens", "-1")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("remaining-credits", "-1")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("budget-source", "api")
	_ = sesionPresupuestoRegistrarCmd.Flags().Set("raw-snapshot", "{\"from\":\"test\"}")

	out := capturarStdout(t, func() {
		if err := sesionPresupuestoRegistrarCmd.RunE(sesionPresupuestoRegistrarCmd, nil); err != nil {
			t.Fatalf("sesion presupuesto registrar via API: %v", err)
		}
	})

	for _, token := range []string{
		"Presupuesto registrado (id: 88) para sesión #77",
		"Estado: handoff_preventivo",
		"handoff preventivo",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestSesionPresupuestoRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(sesionPresupuestoVerCmd)
	_ = sesionPresupuestoVerCmd.Flags().Set("agente", "Codex1")
	err := sesionPresupuestoVerCmd.RunE(sesionPresupuestoVerCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
