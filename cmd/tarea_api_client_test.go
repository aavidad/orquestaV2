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

func TestTareaListarExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	db.Close()
	defer db.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	err := tareaListarCmd.RunE(tareaListarCmd, nil)
	if err == nil {
		t.Fatalf("tarea listar deberia exigir servidor o recuperacion local explicita")
	}
	if !strings.Contains(err.Error(), "ORQUESTA_FORCE_LOCAL_DB=1") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestTareaTomarExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	db.Close()
	defer db.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	err := tareaTomar.RunE(tareaTomar, []string{"12", "Codex1"})
	if err == nil {
		t.Fatalf("tarea tomar deberia exigir servidor o recuperacion local explicita")
	}
	if !strings.Contains(err.Error(), "ORQUESTA_FORCE_LOCAL_DB=1") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestTareaNotasUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas/12", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiTareaResponse{
			Tarea: &db.Tarea{
				ID:     12,
				Titulo: "Cliente fino",
				Notas:  "nota uno\nnota dos",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := tareaNotasCmd.RunE(tareaNotasCmd, []string{"12"}); err != nil {
			t.Fatalf("tarea notas via API: %v", err)
		}
	})
	for _, token := range []string{"Cliente fino", "nota uno", "nota dos"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida notas sin %q:\n%s", token, out)
		}
	}
}

func TestTareaCancelarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas/12/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiTareaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req cancelar: %v", err)
		}
		if req.Accion != "cancelar" || req.Agente != "Codex1" || req.Motivo != "duplicada" {
			t.Fatalf("payload inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := tareaCancelarCmd.RunE(tareaCancelarCmd, []string{"12", "Codex1", "duplicada"}); err != nil {
			t.Fatalf("tarea cancelar via API: %v", err)
		}
	})
	if !strings.Contains(out, "Tarea #12 cancelada: duplicada") {
		t.Fatalf("salida cancelar inesperada:\n%s", out)
	}
}

func TestTareaRefineriaUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas/21/refineria", func(w http.ResponseWriter, r *http.Request) {
		var req apiRefineriaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req refineria: %v", err)
		}
		if req.Agente != "Codex1" || req.Rama != "feature/refineria" || req.Dir != "/tmp/orquesta" || req.CmdTest != "go test ./cmd" {
			t.Fatalf("payload refineria inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"solicitud": &db.RefineriaSolicitud{
				ID:         44,
				TareaID:    21,
				Agente:     "Codex1",
				Rama:       "feature/refineria",
				DirTrabajo: "/tmp/orquesta",
				CmdTest:    "go test ./cmd",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	tareaRefineriaCmd.Flags().Set("rama", "feature/refineria")
	tareaRefineriaCmd.Flags().Set("dir", "/tmp/orquesta")
	tareaRefineriaCmd.Flags().Set("cmd", "go test ./cmd")
	defer tareaRefineriaCmd.Flags().Set("rama", "")
	defer tareaRefineriaCmd.Flags().Set("dir", "")
	defer tareaRefineriaCmd.Flags().Set("cmd", "go test ./...")

	out := capturarStdout(t, func() {
		if err := tareaRefineriaCmd.RunE(tareaRefineriaCmd, []string{"21", "Codex1"}); err != nil {
			t.Fatalf("tarea refineria via API: %v", err)
		}
	})
	for _, token := range []string{"Tarea #21 enviada a Refinería", "solicitud #44", "feature/refineria", "go test ./cmd"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida refineria sin %q:\n%s", token, out)
		}
	}
}
