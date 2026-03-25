package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxLenguajeWeb() *http.ServeMux {
	mux := http.NewServeMux()
	registrarRutasServe(mux)
	return mux
}

func TestWebLenguajePaginaMuestraPoliticaMatrizYResolucion(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.SetLanguagePolicy(&db.LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "en",
		AllowedLanguages:         []string{"es", "en", "fr"},
		Notes:                    "politica web",
	}, "Codex3"); err != nil {
		t.Fatalf("SetLanguagePolicy: %v", err)
	}
	if _, err := db.SetLanguageMatrixEntry("project", "orquestador", "apps", "fr", "demo web", "Codex3"); err != nil {
		t.Fatalf("SetLanguageMatrixEntry: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/lenguaje?proyecto=orquestador&contexto=apps", nil)
	testMuxLenguajeWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"Lenguaje e i18n",
		"politica web",
		"orquestador",
		"demo web",
		"fr",
		"/lenguaje/politica",
		"/lenguaje/matriz",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("pagina de lenguaje sin %q:\n%s", token, body)
		}
	}
}

func TestWebLenguajePaginaRespetaIdiomaDelRequest(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.SetLanguagePolicy(&db.LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "en",
		AllowedLanguages:         []string{"es", "en"},
		Notes:                    "policy",
	}, "Codex3"); err != nil {
		t.Fatalf("SetLanguagePolicy: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/lenguaje?lang=en", nil)
	testMuxLenguajeWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Language and i18n") {
		t.Fatalf("pagina de lenguaje no traducida al ingles:\n%s", body)
	}
	if !strings.Contains(body, "<html lang=\"en\">") {
		t.Fatalf("html lang inesperado:\n%s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q, want en", got)
	}
}

func TestWebLenguajePoliticaYMatrizMutanEstado(t *testing.T) {
	prepararDBTemporalCmd(t)

	policyForm := url.Values{
		"default_language":               {"en"},
		"documentation_default_language": {"es"},
		"apps_default_language":          {"fr"},
		"allowed_languages":              {"es,en,fr"},
		"notes":                          {"actualizada desde web"},
		"documentation_multilang":        {"on"},
		"apps_multilang":                 {"on"},
	}
	recPolicy := httptest.NewRecorder()
	reqPolicy := httptest.NewRequest(http.MethodPost, "/lenguaje/politica", strings.NewReader(policyForm.Encode()))
	reqPolicy.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxLenguajeWeb().ServeHTTP(recPolicy, reqPolicy)
	if recPolicy.Code != http.StatusSeeOther {
		t.Fatalf("status politica inesperado: %d body=%s", recPolicy.Code, recPolicy.Body.String())
	}

	policy, err := db.GetLanguagePolicy()
	if err != nil {
		t.Fatalf("GetLanguagePolicy: %v", err)
	}
	if policy.DefaultLanguage != "en" || policy.AppsDefaultLang != "fr" || policy.Notes != "actualizada desde web" {
		t.Fatalf("politica inesperada: %+v", policy)
	}

	matrixForm := url.Values{
		"scope":    {"project"},
		"selector": {"orquestador"},
		"context":  {"apps"},
		"language": {"fr"},
		"reason":   {"preferencia web"},
	}
	recMatrix := httptest.NewRecorder()
	reqMatrix := httptest.NewRequest(http.MethodPost, "/lenguaje/matriz", strings.NewReader(matrixForm.Encode()))
	reqMatrix.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxLenguajeWeb().ServeHTTP(recMatrix, reqMatrix)
	if recMatrix.Code != http.StatusSeeOther {
		t.Fatalf("status matriz inesperado: %d body=%s", recMatrix.Code, recMatrix.Body.String())
	}

	resolucion, err := db.ResolveLanguage("orquestador", nil, "apps")
	if err != nil {
		t.Fatalf("ResolveLanguage: %v", err)
	}
	if resolucion.Idioma != "fr" || resolucion.Entrada == nil || resolucion.Entrada.Reason != "preferencia web" {
		t.Fatalf("resolucion inesperada: %+v", resolucion)
	}

	deleteForm := url.Values{
		"scope":    {"project"},
		"selector": {"orquestador"},
		"context":  {"apps"},
	}
	recDelete := httptest.NewRecorder()
	reqDelete := httptest.NewRequest(http.MethodPost, "/lenguaje/matriz/borrar", strings.NewReader(deleteForm.Encode()))
	reqDelete.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxLenguajeWeb().ServeHTTP(recDelete, reqDelete)
	if recDelete.Code != http.StatusSeeOther {
		t.Fatalf("status borrar inesperado: %d body=%s", recDelete.Code, recDelete.Body.String())
	}

	matriz, err := db.ListLanguageMatrixEntries()
	if err != nil {
		t.Fatalf("ListLanguageMatrixEntries: %v", err)
	}
	if len(matriz) != 0 {
		t.Fatalf("se esperaba matriz vacia y hay %d entradas", len(matriz))
	}
}
