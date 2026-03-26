package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestLenguajeUsaAPIParaLecturas(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/lenguaje/politica", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiLenguajePoliticaResponse{
			Politica: &db.LanguagePolicy{
				DefaultLanguage:          "es",
				DocumentationMultilang:   true,
				AppsMultilang:            true,
				DocumentationDefaultLang: "es",
				AppsDefaultLang:          "en",
				AllowedLanguages:         []string{"es", "en"},
			},
		})
	})
	mux.HandleFunc("/api/lenguaje/matriz", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiLenguajeMatrizResponse{
			Matriz: []*db.LanguageMatrixEntry{
				{Scope: "project", Selector: "orquestador", Context: "apps", Language: "fr"},
			},
		})
	})
	mux.HandleFunc("/api/lenguaje/resolver", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiLenguajeResolucionResponse{
			Resolucion: &db.LanguageResolution{
				Idioma:   "fr",
				Contexto: "apps",
				Origen:   "matriz",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	outPolitica := capturarStdout(t, func() {
		if err := lenguajePoliticaVerCmd.RunE(lenguajePoliticaVerCmd, nil); err != nil {
			t.Fatalf("lenguaje politica ver via API: %v", err)
		}
	})
	if !strings.Contains(outPolitica, "POLITICA GLOBAL DE LENGUAJE") || !strings.Contains(outPolitica, "en") {
		t.Fatalf("salida politica inesperada:\n%s", outPolitica)
	}

	outMatriz := capturarStdout(t, func() {
		if err := lenguajeMatrizListarCmd.RunE(lenguajeMatrizListarCmd, nil); err != nil {
			t.Fatalf("lenguaje matriz listar via API: %v", err)
		}
	})
	if !strings.Contains(outMatriz, "orquestador") || !strings.Contains(outMatriz, "fr") {
		t.Fatalf("salida matriz inesperada:\n%s", outMatriz)
	}

	if err := lenguajeResolverCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto: %v", err)
	}
	if err := lenguajeResolverCmd.Flags().Set("contexto", "apps"); err != nil {
		t.Fatalf("set contexto: %v", err)
	}
	outResolver := capturarStdout(t, func() {
		if err := lenguajeResolverCmd.RunE(lenguajeResolverCmd, nil); err != nil {
			t.Fatalf("lenguaje resolver via API: %v", err)
		}
	})
	if !strings.Contains(outResolver, "Idioma:   fr") || !strings.Contains(outResolver, "Origen:   matriz") {
		t.Fatalf("salida resolver inesperada:\n%s", outResolver)
	}
}

func TestLenguajeUsaAPIParaEscrituras(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/lenguaje/politica", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiLenguajePoliticaResponse{
				Politica: &db.LanguagePolicy{
					DefaultLanguage:          "es",
					DocumentationMultilang:   true,
					AppsMultilang:            true,
					DocumentationDefaultLang: "es",
					AppsDefaultLang:          "es",
					AllowedLanguages:         []string{"es", "en"},
				},
			})
		case http.MethodPost:
			var req apiLenguajePoliticaSetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode politica: %v", err)
			}
			if req.Politica == nil || req.Politica.DefaultLanguage != "en" || req.Por != "Codex1" {
				t.Fatalf("payload politica inesperado: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			t.Fatalf("metodo inesperado politica: %s", r.Method)
		}
	})
	mux.HandleFunc("/api/lenguaje/matriz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("metodo inesperado matriz: %s", r.Method)
		}
		var req apiLenguajeMatrizSetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode matriz: %v", err)
		}
		if req.Scope != "proyecto" || req.Selector != "orquestador" || req.Contexto != "apps" || req.Idioma != "fr" || req.Por != "Codex1" {
			t.Fatalf("payload matriz inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/lenguaje/matriz/borrar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("metodo inesperado matriz borrar: %s", r.Method)
		}
		var req apiLenguajeMatrizDeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode matriz borrar: %v", err)
		}
		if req.Scope != "proyecto" || req.Selector != "orquestador" || req.Contexto != "apps" {
			t.Fatalf("payload borrado inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := lenguajePoliticaSetCmd.Flags().Set("default", "en"); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if err := lenguajePoliticaSetCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por politica: %v", err)
	}
	outPolitica := capturarStdout(t, func() {
		if err := lenguajePoliticaSetCmd.RunE(lenguajePoliticaSetCmd, nil); err != nil {
			t.Fatalf("lenguaje politica fijar via API: %v", err)
		}
	})
	if !strings.Contains(outPolitica, "✓ Politica de lenguaje actualizada (en)") {
		t.Fatalf("salida politica set inesperada:\n%s", outPolitica)
	}

	if err := lenguajeMatrizFijarCmd.Flags().Set("contexto", "apps"); err != nil {
		t.Fatalf("set contexto fijar: %v", err)
	}
	if err := lenguajeMatrizFijarCmd.Flags().Set("razon", "demo"); err != nil {
		t.Fatalf("set razon fijar: %v", err)
	}
	if err := lenguajeMatrizFijarCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por fijar: %v", err)
	}
	outMatriz := capturarStdout(t, func() {
		if err := lenguajeMatrizFijarCmd.RunE(lenguajeMatrizFijarCmd, []string{"proyecto", "orquestador", "fr"}); err != nil {
			t.Fatalf("lenguaje matriz fijar via API: %v", err)
		}
	})
	if !strings.Contains(outMatriz, "✓ Matriz fijada: proyecto/orquestador [apps] = fr") {
		t.Fatalf("salida matriz fijar inesperada:\n%s", outMatriz)
	}

	if err := lenguajeMatrizBorrarCmd.Flags().Set("contexto", "apps"); err != nil {
		t.Fatalf("set contexto borrar: %v", err)
	}
	outBorrar := capturarStdout(t, func() {
		if err := lenguajeMatrizBorrarCmd.RunE(lenguajeMatrizBorrarCmd, []string{"proyecto", "orquestador"}); err != nil {
			t.Fatalf("lenguaje matriz borrar via API: %v", err)
		}
	})
	if !strings.Contains(outBorrar, "✓ Matriz borrada: proyecto/orquestador [apps]") {
		t.Fatalf("salida matriz borrar inesperada:\n%s", outBorrar)
	}
}

func TestLenguajeEsqueletoUsaPoliticaYMaterializaEstructura(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/lenguaje/politica", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiLenguajePoliticaResponse{
			Politica: &db.LanguagePolicy{
				DefaultLanguage:          "es",
				DocumentationMultilang:   true,
				AppsMultilang:            true,
				DocumentationDefaultLang: "es",
				AppsDefaultLang:          "es",
				AllowedLanguages:         []string{"es", "en"},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	root := filepath.Join(t.TempDir(), "demo")
	out := capturarStdout(t, func() {
		if err := lenguajeEsqueletoCmd.RunE(lenguajeEsqueletoCmd, []string{root}); err != nil {
			t.Fatalf("lenguaje esqueleto: %v", err)
		}
	})
	if !strings.Contains(out, "Esqueleto i18n creado") {
		t.Fatalf("salida esqueleto inesperada:\n%s", out)
	}

	for _, rel := range []string{
		"i18n/config.json",
		"i18n/README.md",
		"i18n/es/common.json",
		"i18n/en/errors.json",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("falta %s: %v", rel, err)
		}
	}
}

func TestLenguajeRequiereServidorParaPoliticaYMatriz(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	if err := lenguajePoliticaVerCmd.RunE(lenguajePoliticaVerCmd, nil); err == nil {
		t.Fatalf("se esperaba error sin servidor en politica ver")
	}

	resetCommandFlags(lenguajeResolverCmd)
	_ = lenguajeResolverCmd.Flags().Set("proyecto", "orquestador")
	err := lenguajeResolverCmd.RunE(lenguajeResolverCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor en resolver")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
