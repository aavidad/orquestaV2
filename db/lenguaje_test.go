/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestLanguagePolicyDefaults(t *testing.T) {
	prepararDBTemporal(t)

	p, err := GetLanguagePolicy()
	if err != nil {
		t.Fatalf("GetLanguagePolicy: %v", err)
	}
	if p.DefaultLanguage != "es" {
		t.Fatalf("idioma por defecto inesperado: %+v", p)
	}
	if !p.DocumentationMultilang || !p.AppsMultilang {
		t.Fatalf("multilenguaje por defecto inesperado: %+v", p)
	}
}

func TestLanguageMatrixAndResolution(t *testing.T) {
	prepararDBTemporal(t)

	if err := SetLanguagePolicy(&LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "es",
		AllowedLanguages:         []string{"es", "en"},
	}, "Codex3"); err != nil {
		t.Fatalf("SetLanguagePolicy: %v", err)
	}

	if _, err := SetLanguageMatrixEntry("project", "orquestador", "docs", "en", "docs en ingles", "Codex3"); err != nil {
		t.Fatalf("SetLanguageMatrixEntry proyecto: %v", err)
	}
	if _, err := SetLanguageMatrixEntry("task", "214", "docs", "es", "task local", "Codex3"); err != nil {
		t.Fatalf("SetLanguageMatrixEntry tarea: %v", err)
	}

	res, err := ResolveLanguage("orquestador", nil, "docs")
	if err != nil {
		t.Fatalf("ResolveLanguage proyecto: %v", err)
	}
	if res.Idioma != "en" || res.Entrada == nil {
		t.Fatalf("resolucion de proyecto inesperada: %+v", res)
	}

	taskID := int64(214)
	res, err = ResolveLanguage("orquestador", &taskID, "docs")
	if err != nil {
		t.Fatalf("ResolveLanguage tarea: %v", err)
	}
	if res.Idioma != "es" || res.Entrada == nil || res.Entrada.Scope != "task" {
		t.Fatalf("resolucion de tarea inesperada: %+v", res)
	}
}

func TestLanguageMatrixRejectsUnsupportedLanguage(t *testing.T) {
	prepararDBTemporal(t)

	if _, err := SetLanguageMatrixEntry("project", "orquestador", "docs", "fr", "no soportado", "Codex3"); err == nil {
		t.Fatalf("se esperaba error por idioma no permitido")
	}
}
