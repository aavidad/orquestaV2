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
)

func TestMemoriaUsaAPICuandoHayServidor(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/memoria" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entidades": []map[string]any{{
					"id":                  7,
					"nombre":              "Core_API",
					"tipo":                "api",
					"valor_json":          `{"version":"v2"}`,
					"metadata_json":       `{"fuente":"manual"}`,
					"verificado_por":      "Codex1",
					"ultima_verificacion": "2026-03-23T10:00:00Z",
				}},
			})
		case r.URL.Path == "/api/memoria/Core_API" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entidad": map[string]any{
					"id":                  7,
					"nombre":              "Core_API",
					"tipo":                "api",
					"valor_json":          `{"version":"v2"}`,
					"metadata_json":       `{"fuente":"manual"}`,
					"verificado_por":      "Codex1",
					"ultima_verificacion": "2026-03-23T10:00:00Z",
				},
			})
		case r.URL.Path == "/api/memoria" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 8, "entidad": map[string]any{"nombre": "Core_API"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	outListar := capturarStdout(t, func() {
		if err := memoriaListarCmd.RunE(memoriaListarCmd, nil); err != nil {
			t.Fatalf("memoria listar via api: %v", err)
		}
	})
	if !strings.Contains(outListar, "Core_API") {
		t.Fatalf("salida listar via api inesperada:\n%s", outListar)
	}

	outVer := capturarStdout(t, func() {
		if err := memoriaVerCmd.RunE(memoriaVerCmd, []string{"Core_API"}); err != nil {
			t.Fatalf("memoria ver via api: %v", err)
		}
	})
	if !strings.Contains(outVer, "Entidad:      Core_API") {
		t.Fatalf("salida ver via api inesperada:\n%s", outVer)
	}

	if err := memoriaGuardarCmd.Flags().Set("valor", `{"version":"v3"}`); err != nil {
		t.Fatalf("set valor guardar api: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaGuardarCmd.Flags().Set("valor", "")
	})
	outGuardar := capturarStdout(t, func() {
		if err := memoriaGuardarCmd.RunE(memoriaGuardarCmd, []string{"Core_API", "api"}); err != nil {
			t.Fatalf("memoria guardar via api: %v", err)
		}
	})
	if !strings.Contains(outGuardar, "Entidad de memoria guardada: Core_API") {
		t.Fatalf("salida guardar via api inesperada:\n%s", outGuardar)
	}
}

func TestMemoriaRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(memoriaListarCmd)
	err := memoriaListarCmd.RunE(memoriaListarCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
