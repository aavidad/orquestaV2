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

func TestSesionHistorialYVerUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSesionesInspeccionResponse{
			Sesiones: []*db.Sesion{
				{
					ID:                91,
					Agente:            "Codex1",
					ProyectoSlug:      "orquestador",
					Activa:            true,
					Estado:            "activa",
					Herramienta:       "codex-cli",
					ExternalSessionID: "sess-api-001",
				},
			},
		})
	})
	mux.HandleFunc("/api/sesiones/91", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSesionResponse{
			Sesion: &db.Sesion{
				ID:                 91,
				Agente:             "Codex1",
				ProyectoSlug:       "orquestador",
				Activa:             true,
				Estado:             "activa",
				Herramienta:        "codex-cli",
				ExternalSessionID:  "sess-api-001",
				ResumenContinuidad: "historial via api",
				Host:               "host-api",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = sesionHistorialCmd.Flags().Set("agente", "Codex1")
	_ = sesionHistorialCmd.Flags().Set("proyecto", "orquestador")
	_ = sesionHistorialCmd.Flags().Set("activa", "true")
	_ = sesionHistorialCmd.Flags().Set("estado", "activa")

	outHistorial := capturarStdout(t, func() {
		if err := sesionHistorialCmd.RunE(sesionHistorialCmd, nil); err != nil {
			t.Fatalf("run historial via api: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orquestador", "sess-api-001"} {
		if !strings.Contains(outHistorial, token) {
			t.Fatalf("salida historial via api sin %q:\n%s", token, outHistorial)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := sesionVerCmd.RunE(sesionVerCmd, []string{"91"}); err != nil {
			t.Fatalf("run ver via api: %v", err)
		}
	})
	for _, token := range []string{"Sesión 91", "Codex1", "sess-api-001", "historial via api", "host-api"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida ver via api sin %q:\n%s", token, outVer)
		}
	}
}

func TestSesionHistorialRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(sesionHistorialCmd)
	err := sesionHistorialCmd.RunE(sesionHistorialCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
