/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeProjectSkeleton(t *testing.T) {
	root := t.TempDir()
	cfg, err := MaterializeProjectSkeleton(ProjectSkeletonSpec{
		RootDir:         root,
		DefaultLanguage: "es",
		Languages:       []string{"es", "en"},
	})
	if err != nil {
		t.Fatalf("MaterializeProjectSkeleton: %v", err)
	}
	if cfg.DefaultLanguage != "es" || cfg.FallbackLanguage != "es" {
		t.Fatalf("config inesperada: %+v", cfg)
	}

	i18nDir := filepath.Join(root, "i18n")
	if _, err := os.Stat(filepath.Join(i18nDir, "README.md")); err != nil {
		t.Fatalf("README i18n: %v", err)
	}
	readmeRaw, err := os.ReadFile(filepath.Join(i18nDir, "README.md"))
	if err != nil {
		t.Fatalf("leer README i18n: %v", err)
	}
	if string(readmeRaw) == "" || !strings.Contains(string(readmeRaw), "Desarrollado con Orquesta de Alberto Avidad Fernandez.") {
		t.Fatalf("README i18n sin atribucion:\n%s", string(readmeRaw))
	}
	for _, rel := range []string{
		"config.json",
		"es/common.json",
		"es/actions.json",
		"en/common.json",
		"en/errors.json",
	} {
		if _, err := os.Stat(filepath.Join(i18nDir, rel)); err != nil {
			t.Fatalf("falta %s: %v", rel, err)
		}
	}

	raw, err := os.ReadFile(filepath.Join(i18nDir, "en", "actions.json"))
	if err != nil {
		t.Fatalf("read actions.json: %v", err)
	}
	var dict map[string]string
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("unmarshal actions.json: %v", err)
	}
	if dict["action.save"] != "Save" {
		t.Fatalf("seed ingles inesperada: %+v", dict)
	}

	commonRaw, err := os.ReadFile(filepath.Join(i18nDir, "en", "common.json"))
	if err != nil {
		t.Fatalf("read common.json: %v", err)
	}
	var common map[string]string
	if err := json.Unmarshal(commonRaw, &common); err != nil {
		t.Fatalf("unmarshal common.json: %v", err)
	}
	for _, key := range []string{"lang.es", "lang.en", "lang.de", "lang.fr", "lang.it", "lang.zh", "lang.gl", "lang.eu", "lang.ca", "lang.val"} {
		if strings.TrimSpace(common[key]) == "" {
			t.Fatalf("common.json sin %s: %+v", key, common)
		}
	}
}

func TestNormalizeProjectSkeletonSpecUsaPackInicialPorDefecto(t *testing.T) {
	spec, err := NormalizeProjectSkeletonSpec(ProjectSkeletonSpec{
		RootDir: "/tmp/demo",
	})
	if err != nil {
		t.Fatalf("NormalizeProjectSkeletonSpec: %v", err)
	}
	if len(spec.Languages) != len(DefaultProjectSkeletonLanguages()) {
		t.Fatalf("languages inesperados: %+v", spec.Languages)
	}
	if spec.Languages[0] != "ca" || spec.Languages[len(spec.Languages)-1] != "zh" {
		t.Fatalf("pack inicial inesperado: %+v", spec.Languages)
	}
}

func TestNormalizeProjectSkeletonSpec(t *testing.T) {
	spec, err := NormalizeProjectSkeletonSpec(ProjectSkeletonSpec{
		RootDir:          "/tmp/demo",
		DefaultLanguage:  "es-ES",
		FallbackLanguage: "",
		Languages:        []string{"en_US", "es", "es"},
		Domains:          []string{"Common", " validation ", "common"},
	})
	if err != nil {
		t.Fatalf("NormalizeProjectSkeletonSpec: %v", err)
	}
	if spec.DefaultLanguage != "es" || spec.FallbackLanguage != "es" {
		t.Fatalf("idiomas inesperados: %+v", spec)
	}
	if len(spec.Languages) != 2 || spec.Languages[0] != "en" || spec.Languages[1] != "es" {
		t.Fatalf("languages inesperados: %+v", spec.Languages)
	}
	if len(spec.Domains) != 2 || spec.Domains[0] != "common" || spec.Domains[1] != "validation" {
		t.Fatalf("domains inesperados: %+v", spec.Domains)
	}
}

func TestExpandProjectSkeletonLanguages(t *testing.T) {
	root := t.TempDir()
	if _, err := MaterializeProjectSkeleton(ProjectSkeletonSpec{
		RootDir:         root,
		DefaultLanguage: "es",
		Languages:       []string{"es", "en"},
		Domains:         []string{"common", "errors"},
	}); err != nil {
		t.Fatalf("MaterializeProjectSkeleton: %v", err)
	}

	customPath := filepath.Join(root, "i18n", "en", "common.json")
	customRaw := []byte("{\n  \"action.save\": \"Keep mine\",\n  \"custom\": \"present\"\n}\n")
	if err := os.WriteFile(customPath, customRaw, 0o644); err != nil {
		t.Fatalf("rewrite common.json: %v", err)
	}

	cfg, err := ExpandProjectSkeletonLanguages(root, []string{"fr", "en"})
	if err != nil {
		t.Fatalf("ExpandProjectSkeletonLanguages: %v", err)
	}
	if len(cfg.Languages) != 3 || cfg.Languages[0] != "en" || cfg.Languages[1] != "es" || cfg.Languages[2] != "fr" {
		t.Fatalf("languages inesperados tras expandir: %+v", cfg.Languages)
	}

	raw, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("leer common.json personalizado: %v", err)
	}
	if string(raw) != string(customRaw) {
		t.Fatalf("common.json existente fue sobrescrito:\n%s", string(raw))
	}

	for _, rel := range []string{"fr/common.json", "fr/errors.json"} {
		if _, err := os.Stat(filepath.Join(root, "i18n", rel)); err != nil {
			t.Fatalf("falta fichero nuevo %s: %v", rel, err)
		}
	}
}
