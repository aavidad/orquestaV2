package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestSesionGuardarContinuarFinYListarUsanAPI(t *testing.T) {
	sesion := &db.Sesion{
		ID:                 88,
		Agente:             "Codex1",
		ProyectoSlug:       "orquestador",
		CWD:                "/tmp/orquestador",
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-guardar-001",
		ResumenContinuidad: "seguir por api",
		Branch:             "main",
		Estado:             "pausada",
		Host:               "host-api",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sesiones/guardar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSesionResponse{Sesion: sesion})
	})
	mux.HandleFunc("/api/sesiones/continuar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSesionResponse{Sesion: sesion})
	})
	mux.HandleFunc("/api/sesiones/fin", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/agentes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAgentesResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(sesionGuardarCmd)
	_ = sesionGuardarCmd.Flags().Set("proyecto", "orquestador")
	_ = sesionGuardarCmd.Flags().Set("cwd", "/tmp/orquestador")
	_ = sesionGuardarCmd.Flags().Set("herramienta", "codex-cli")
	_ = sesionGuardarCmd.Flags().Set("external-session-id", "sess-guardar-001")
	_ = sesionGuardarCmd.Flags().Set("resumen", "seguir por api")
	outGuardar := capturarStdout(t, func() {
		if err := sesionGuardarCmd.RunE(sesionGuardarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("sesion guardar via api: %v", err)
		}
	})
	if !strings.Contains(outGuardar, "Contexto de sesión guardado para Codex1") {
		t.Fatalf("salida guardar inesperada:\n%s", outGuardar)
	}

	resetCommandFlags(sesionContinuarCmd)
	_ = sesionContinuarCmd.Flags().Set("proyecto", "orquestador")
	outContinuar := capturarStdout(t, func() {
		if err := sesionContinuarCmd.RunE(sesionContinuarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("sesion continuar via api: %v", err)
		}
	})
	for _, token := range []string{"Sesión 88", "Codex1", "sess-guardar-001", "seguir por api"} {
		if !strings.Contains(outContinuar, token) {
			t.Fatalf("salida continuar sin %q:\n%s", token, outContinuar)
		}
	}

	outFin := capturarStdout(t, func() {
		if err := sesionFinCmd.RunE(sesionFinCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("sesion fin via api: %v", err)
		}
	})
	if !strings.Contains(outFin, "Sesión cerrada") {
		t.Fatalf("salida fin inesperada:\n%s", outFin)
	}

	outListar := capturarStdout(t, func() {
		if err := sesionListarCmd.RunE(sesionListarCmd, nil); err != nil {
			t.Fatalf("sesion listar via api: %v", err)
		}
	})
	for _, token := range []string{"AGENTE", "Codex1", "programador", "SÍ"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida listar sin %q:\n%s", token, outListar)
		}
	}
}
