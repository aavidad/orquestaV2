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
