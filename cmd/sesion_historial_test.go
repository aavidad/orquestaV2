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
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestSesionHistorialYVer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-cli-001",
		ResumenContinuidad: "historial cli",
		Branch:             "main",
		Host:               "host-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if err := sesionHistorialCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := sesionHistorialCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := sesionHistorialCmd.Flags().Set("activa", "true"); err != nil {
		t.Fatalf("set activa: %v", err)
	}

	outHistorial := capturarStdout(t, func() {
		if err := sesionHistorialCmd.RunE(sesionHistorialCmd, nil); err != nil {
			t.Fatalf("run historial: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orquestador", "sess-cli-001"} {
		if !strings.Contains(outHistorial, token) {
			t.Fatalf("salida historial sin %q:\n%s", token, outHistorial)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := sesionVerCmd.RunE(sesionVerCmd, []string{itoa(sesion.ID)}); err != nil {
			t.Fatalf("run ver: %v", err)
		}
	})
	for _, token := range []string{"Sesión", "Codex1", "sess-cli-001", "historial cli"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida ver sin %q:\n%s", token, outVer)
		}
	}
}

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
