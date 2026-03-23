package cmd

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestMemoriaGuardarListarYVer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if proyectoID == 0 {
		t.Fatalf("proyecto id inesperado")
	}

	if err := memoriaGuardarCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("valor", `{"version":"v2","grpc":true}`); err != nil {
		t.Fatalf("set valor guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("metadata", `{"fuente":"manual"}`); err != nil {
		t.Fatalf("set metadata guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("verificado-por", "Codex1"); err != nil {
		t.Fatalf("set verificado-por guardar: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaGuardarCmd.Flags().Set("proyecto", "")
		_ = memoriaGuardarCmd.Flags().Set("valor", "")
		_ = memoriaGuardarCmd.Flags().Set("metadata", "{}")
		_ = memoriaGuardarCmd.Flags().Set("verificado-por", "")
	})

	outGuardar := capturarStdout(t, func() {
		if err := memoriaGuardarCmd.RunE(memoriaGuardarCmd, []string{"Core_API", "api"}); err != nil {
			t.Fatalf("memoria guardar: %v", err)
		}
	})
	if !strings.Contains(outGuardar, "Entidad de memoria #") {
		t.Fatalf("salida guardar inesperada:\n%s", outGuardar)
	}

	if err := memoriaListarCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto listar: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaListarCmd.Flags().Set("proyecto", "")
	})
	outListar := capturarStdout(t, func() {
		if err := memoriaListarCmd.RunE(memoriaListarCmd, nil); err != nil {
			t.Fatalf("memoria listar: %v", err)
		}
	})
	if !strings.Contains(outListar, "Core_API") || !strings.Contains(outListar, `"version":"v2"`) {
		t.Fatalf("salida listar inesperada:\n%s", outListar)
	}

	if err := memoriaVerCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto ver: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaVerCmd.Flags().Set("proyecto", "")
	})
	outVer := capturarStdout(t, func() {
		if err := memoriaVerCmd.RunE(memoriaVerCmd, []string{"Core_API"}); err != nil {
			t.Fatalf("memoria ver: %v", err)
		}
	})
	if !strings.Contains(outVer, "Entidad:      Core_API") || !strings.Contains(outVer, "Codex1") {
		t.Fatalf("salida ver inesperada:\n%s", outVer)
	}
}

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
