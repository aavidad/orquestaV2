package skillsapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"orquesta/db"
)

type fakeFetcher struct {
	meta *RemoteSkill
	err  error
}

func (f fakeFetcher) Fetch(context.Context, SourceSpec) (*RemoteSkill, error) {
	return f.meta, f.err
}

func TestImportFromWebCreaSkillExternaPendiente(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	svc := NewService(Repository{}, fakeFetcher{
		meta: &RemoteSkill{
			Name:         "openai-docs",
			Description:  "Accede a documentacion actual de OpenAI",
			Repo:         "openai/skills",
			SkillRef:     "openai-docs",
			CanonicalURL: "https://skills.sh/openai/skills/openai-docs",
			SourceKind:   "skills.sh",
		},
	})
	result, err := svc.ImportFromWeb(context.Background(), ImportInput{
		Actor:      "Codex1",
		TipoAgente: "programador",
		URL:        "https://skills.sh/openai/skills/openai-docs",
	})
	if err != nil {
		t.Fatalf("ImportFromWeb: %v", err)
	}
	if result.Existente || result.ID <= 0 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if result.Skill == nil || result.Skill.Origen != "third_party" || result.Skill.Activa || !result.Skill.RequiereAprobacion {
		t.Fatalf("skill importada inesperada: %+v", result.Skill)
	}
}

func TestImportFromWebDevuelveExistenteSiYaHayEquivalente(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	id, err := db.CrearSkill("Codex1", &db.Skill{
		TipoAgente:       "programador",
		Nombre:           "openai-docs",
		Descripcion:      "docs",
		CuandoUsar:       "consultar docs",
		Escenario:        "skills.sh",
		AliasesJSON:      `["openai/skills/openai-docs"]`,
		HerramientasJSON: "[]",
		Origen:           "third_party",
		Activa:           false,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	svc := NewService(Repository{}, fakeFetcher{
		meta: &RemoteSkill{
			Name:         "openai-docs",
			Description:  "docs",
			Repo:         "openai/skills",
			SkillRef:     "openai-docs",
			CanonicalURL: "https://skills.sh/openai/skills/openai-docs",
			SourceKind:   "skills.sh",
		},
	})
	result, err := svc.ImportFromWeb(context.Background(), ImportInput{
		Actor:      "Codex1",
		TipoAgente: "programador",
		URL:        "https://skills.sh/openai/skills/openai-docs",
	})
	if err != nil {
		t.Fatalf("ImportFromWeb: %v", err)
	}
	if !result.Existente || result.ID != id {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestResolveSourceSpecSkillsSh(t *testing.T) {
	spec, err := resolveSourceSpec(SourceSpec{
		URL: "https://skills.sh/openai/skills/openai-docs",
	})
	if err != nil {
		t.Fatalf("resolveSourceSpec: %v", err)
	}
	if spec.Repo != "openai/skills" || spec.Skill != "openai-docs" {
		t.Fatalf("spec inesperada: %+v", spec)
	}
}

func TestParseSkillFrontmatter(t *testing.T) {
	name, description, err := parseSkillFrontmatter(`---
name: openai-docs
description: >
  Access current OpenAI developer documentation
  with live checks.
metadata:
  short-description: docs
---

# OpenAI Docs`)
	if err != nil {
		t.Fatalf("parseSkillFrontmatter: %v", err)
	}
	if name != "openai-docs" {
		t.Fatalf("name inesperado: %s", name)
	}
	if description != "Access current OpenAI developer documentation with live checks." {
		t.Fatalf("description inesperada: %s", description)
	}
}
