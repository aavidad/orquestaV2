package cmd

import (
	"encoding/json"
	"net/http"
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
