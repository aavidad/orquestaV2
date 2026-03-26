package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProyectoFabricarAppValidaFlagsMinimos(t *testing.T) {
	resetCommandFlags(proyectoFabricarAppCmd)
	err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"})
	if err == nil {
		t.Fatalf("se esperaba error")
	}
	if !strings.Contains(err.Error(), "--tipo es obligatorio") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestProyectoFabricarAppUsaAPICuandoHayServidor(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/proyectos/mi-app/fabricar-app", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("metodo inesperado: %s", r.Method)
		}
		var req apiProyectoFabricarAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode fabricar-app: %v", err)
		}
		if req.Tipo != "web_api" || req.Nombre != "Mi App" || req.Por != "Codex1" {
			t.Fatalf("payload fabricar-app inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiProyectoFabricarAppResponse{
			OK:      true,
			Slug:    "mi-app",
			Tipo:    req.Tipo,
			Created: 8,
			Backlog: 3,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(proyectoFabricarAppCmd)
	_ = proyectoFabricarAppCmd.Flags().Set("tipo", "web_api")
	_ = proyectoFabricarAppCmd.Flags().Set("nombre", "Mi App")
	_ = proyectoFabricarAppCmd.Flags().Set("por", "Codex1")

	out := capturarStdout(t, func() {
		if err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"}); err != nil {
			t.Fatalf("proyecto fabricar-app via api: %v", err)
		}
	})
	if !strings.Contains(out, "Backlog de app generado para mi-app") || !strings.Contains(out, "Tareas creadas: 8") {
		t.Fatalf("salida fabricar-app via api inesperada:\n%s", out)
	}
}

func TestProyectoFabricarAppRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(proyectoFabricarAppCmd)
	_ = proyectoFabricarAppCmd.Flags().Set("tipo", "web_api")
	err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"})
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
