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
