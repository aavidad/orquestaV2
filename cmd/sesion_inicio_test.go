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
	"orquesta/sesionesapp"
)

func TestSesionInicioUsaAPIBriefing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/inicio", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSesionInicioResponse{
			Sesion: &db.Sesion{
				ID:             77,
				Agente:         "Codex1",
				ProyectoSlug:   "orquestador",
				ProyectoNombre: "Orquestador",
				CWD:            "/tmp/orquestador",
				Herramienta:    "codex-cli",
				Branch:         "main",
			},
			Rol: "programador",
			PropuestasPendientes: []*sesionesapp.Propuesta{
				{Codigo: "OP-082", Titulo: "Observabilidad pasiva"},
			},
			Reglas: []*sesionesapp.Regla{
				{Categoria: "sesion", Titulo: "Fuente de verdad", Descripcion: "Usar Orquesta"},
			},
			Skills: []*sesionesapp.Skill{
				{Nombre: "fix-bug", CuandoUsar: "Cuando hay un bug confirmado"},
			},
			Workflow: &sesionesapp.Workflow{
				Nombre: "inicio-sesion",
				Pasos:  `["1. Iniciar","2. Trabajar"]`,
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = sesionInicioCmd.Flags().Set("nuevo-codex", "false")
	_ = sesionInicioCmd.Flags().Set("proyecto", "")
	_ = sesionInicioCmd.Flags().Set("conector", "")
	_ = sesionInicioCmd.Flags().Set("cwd", "")
	_ = sesionInicioCmd.Flags().Set("herramienta", "")
	_ = sesionInicioCmd.Flags().Set("branch", "")
	_ = sesionInicioCmd.Flags().Set("external-session-id", "")
	_ = sesionInicioCmd.Flags().Set("resume-payload", "")
	_ = sesionInicioCmd.Flags().Set("resumen", "")
	_ = sesionInicioCmd.Flags().Set("host", "")
	_ = sesionInicioCmd.Flags().Set("pid", "0")

	out := capturarStdout(t, func() {
		if err := sesionInicioCmd.RunE(sesionInicioCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("sesion inicio via API: %v", err)
		}
	})

	for _, token := range []string{
		"SESIÓN INICIADA — agente: Codex1",
		"PROPUESTAS PENDIENTES DE TU VOTO",
		"REGLAS ACTIVAS (PROGRAMADOR)",
		"SKILLS DISPONIBLES",
		"WORKFLOW — INICIO-SESION",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestSesionNuevoCodexUsaAPIBriefing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/inicio", func(w http.ResponseWriter, r *http.Request) {
		var req apiSesionInicioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode sesion inicio: %v", err)
		}
		if !req.NuevoCodex {
			t.Fatalf("esperaba NuevoCodex=true: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiSesionInicioResponse{
			Sesion: &db.Sesion{
				ID:             88,
				Agente:         "Codex9",
				ProyectoSlug:   "orquestador",
				ProyectoNombre: "Orquestador",
				CWD:            "/tmp/orquestador",
				Herramienta:    "codex-cli",
				Branch:         "main",
			},
			Rol: "programador",
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := sesionNuevoCodexCmd.RunE(sesionNuevoCodexCmd, nil); err != nil {
			t.Fatalf("sesion nuevo-codex via API: %v", err)
		}
	})

	for _, token := range []string{
		"Nuevo agente Codex registrado como: Codex9",
		"SESIÓN INICIADA — agente: Codex9",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida sin %q:\n%s", token, out)
		}
	}
}

func TestSesionInicioYNuevoCodexExigenServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(sesionInicioCmd)
	if err := sesionInicioCmd.RunE(sesionInicioCmd, []string{"Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("sesion inicio deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(sesionNuevoCodexCmd)
	if err := sesionNuevoCodexCmd.RunE(sesionNuevoCodexCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("sesion nuevo-codex deberia exigir servidor, err=%v", err)
	}
}

func TestSesionInicioArranqueLimpioEnviaBanderaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/inicio", func(w http.ResponseWriter, r *http.Request) {
		var req apiSesionInicioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode sesion inicio: %v", err)
		}
		if !req.ArranqueLimpio {
			t.Fatalf("esperaba arranque_limpio=true: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiSesionInicioResponse{
			Sesion: &db.Sesion{ID: 90, Agente: "Codex1"},
			Rol:    "programador",
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(sesionInicioCmd)
	_ = sesionInicioCmd.Flags().Set("arranque-limpio", "true")
	if err := sesionInicioCmd.RunE(sesionInicioCmd, []string{"Codex1"}); err != nil {
		t.Fatalf("sesion inicio arranque limpio via API: %v", err)
	}
}
