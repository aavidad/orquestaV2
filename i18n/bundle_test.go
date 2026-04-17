/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package i18n

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBundleReloadAndTranslate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "es.json"), []byte(`{"hola":"hola","estado":"estado"}`), 0o644); err != nil {
		t.Fatalf("es.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{"hola":"hello"}`), 0o644); err != nil {
		t.Fatalf("en.json: %v", err)
	}

	b := NewBundle(dir, "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if got := b.T("en", "hola"); got != "hello" {
		t.Fatalf("traduccion inesperada: %q", got)
	}
	if got := b.T("en", "estado"); got != "estado" {
		t.Fatalf("fallback inesperado: %q", got)
	}
	if got := b.T("fr", "hola"); got != "hola" {
		t.Fatalf("fallback de idioma inesperado: %q", got)
	}
}

func TestNormalizeLang(t *testing.T) {
	cases := map[string]string{
		"es":    "es",
		"ES":    "es",
		"es-ES": "es",
		"en_US": "en",
		"  ":    "",
	}
	for in, want := range cases {
		if got := NormalizeLang(in); got != want {
			t.Fatalf("NormalizeLang(%q)=%q want %q", in, got, want)
		}
	}
}

func TestBundleReloadSoportaEsqueletoPorIdiomaYDominio(t *testing.T) {
	root := t.TempDir()
	if _, err := MaterializeProjectSkeleton(ProjectSkeletonSpec{
		RootDir:         root,
		DefaultLanguage: "es",
		Languages:       []string{"es", "en"},
		Domains:         []string{"common", "actions"},
	}); err != nil {
		t.Fatalf("MaterializeProjectSkeleton: %v", err)
	}

	b := NewBundle(filepath.Join(root, "i18n"), "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if !b.HasLang("en") {
		t.Fatalf("debería cargar idioma en desde subdirectorio")
	}
	if got := b.T("en", "action.save"); got != "Save" {
		t.Fatalf("traduccion desde dominio inesperada: %q", got)
	}
	if got := b.T("en", "status.ready"); got != "Ready" {
		t.Fatalf("traduccion common inesperada: %q", got)
	}
	if got := b.T("fr", "action.save"); got != "Guardar" {
		t.Fatalf("fallback al idioma por defecto inesperado: %q", got)
	}
}

func TestBundleReloadUsaConfigJsonComoFallbackDeIdiomas(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(filepath.Join(i18nDir, "en"), 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(i18nDir, "fr"), 0o755); err != nil {
		t.Fatalf("mkdir fr: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "config.json"), []byte(`{
  "version": 1,
  "default_language": "en",
  "fallback_language": "fr",
  "languages": ["en", "fr"],
  "domains": ["common"],
  "domain_mode": "file_per_domain",
  "path_pattern": "i18n/<lang>/<domain>.json"
}`), 0o644); err != nil {
		t.Fatalf("config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "en", "common.json"), []byte(`{"hello":"hello"}`), 0o644); err != nil {
		t.Fatalf("en/common.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "fr", "common.json"), []byte(`{"missing":"bonjour"}`), 0o644); err != nil {
		t.Fatalf("fr/common.json: %v", err)
	}

	b := NewBundle(i18nDir, "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload con config.json: %v", err)
	}

	if got := b.ResolveLang(""); got != "en" {
		t.Fatalf("default lang inesperado: %q", got)
	}
	if got := b.T("de", "hello"); got != "hello" {
		t.Fatalf("fallback a default_language inesperado: %q", got)
	}
	if got := b.T("de", "missing"); got != "bonjour" {
		t.Fatalf("fallback a fallback_language inesperado: %q", got)
	}
}

func TestBundleReloadMantieneCompatibilidadLegacyConContratoCanonico(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(filepath.Join(i18nDir, "en"), 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "config.json"), []byte(`{
  "version": 1,
  "default_language": "en",
  "fallback_language": "en",
  "languages": ["en"],
  "domains": ["common"],
  "domain_mode": "file_per_domain",
  "path_pattern": "i18n/<lang>/<domain>.json"
}`), 0o644); err != nil {
		t.Fatalf("config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "en", "common.json"), []byte(`{"hello":"hello"}`), 0o644); err != nil {
		t.Fatalf("en/common.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "en.json"), []byte(`{"status.ready":"ready"}`), 0o644); err != nil {
		t.Fatalf("en.json: %v", err)
	}

	b := NewBundle(i18nDir, "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload con formato mixto: %v", err)
	}
	if got := b.T("en", "hello"); got != "hello" {
		t.Fatalf("clave canónica inesperada: %q", got)
	}
	if got := b.T("en", "status.ready"); got != "ready" {
		t.Fatalf("clave legacy inesperada: %q", got)
	}
}

func TestBundleReloadIgnoraConfigJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"default_language":"es","languages":["es","en"]}`), 0o644); err != nil {
		t.Fatalf("config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "es.json"), []byte(`{"hola":"hola"}`), 0o644); err != nil {
		t.Fatalf("es.json: %v", err)
	}

	b := NewBundle(dir, "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload con config.json: %v", err)
	}
	if got := b.T("es", "hola"); got != "hola" {
		t.Fatalf("traduccion inesperada: %q", got)
	}
}

func TestBundleRepoIncluyePackInicialBase(t *testing.T) {
	b := NewBundle(".", "es")
	if err := b.Reload(); err != nil {
		t.Fatalf("Reload repo i18n: %v", err)
	}

	got := b.Languages()
	want := append([]string{}, DefaultProjectSkeletonLanguages()...)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("idiomas base inesperados: got=%v want=%v", got, want)
	}
}
