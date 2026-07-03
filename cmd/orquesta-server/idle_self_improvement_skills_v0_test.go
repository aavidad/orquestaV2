package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerStackSupervisorV0FiltroAnadeSkillCuradaCasadaV0(t *testing.T) {
	supervisor := serverStackSupervisorV0{
		curatedSkills: []orquestaautoprogramming.AutoprogrammingCuratedSkillV0{{
			Name:        "orquesta-programacion-tests",
			Description: "Pruebas focales y suite completa.",
			Tags:        []string{"tests"},
		}},
	}

	result, err := supervisor.FilterIdleSelfImprovementRequestsV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementRequestFilterRequestV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{{
				RequestRef:     "request-ref-autoprogramming-backlog-t290",
				SuggestedArea:  "modulos-orquesta-autoprogramming",
				FailureSummary: "Anadir pruebas de matcher.",
				RequiredTests:  []string{"go test -count=1 ./modulos/orquesta-autoprogramming"},
			}},
		},
	)
	if err != nil {
		t.Fatalf("FilterIdleSelfImprovementRequestsV0: %v", err)
	}
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	got := result.Requests[0]
	if !containsServerTestStringV0(got.SkillRefs, "skill-ref-orquesta-programacion-tests-v0") {
		t.Fatalf("skill_refs=%+v", got.SkillRefs)
	}
	if !containsServerTestStringV0(got.ContextRefs, "skill_ref:skill-ref-orquesta-programacion-tests-v0") {
		t.Fatalf("context_refs=%+v", got.ContextRefs)
	}
}

func TestServerCuratedSkillsFromProjectV0DirectorioVacioNoCambiaV0(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "skills"), 0o700); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if skills := serverCuratedSkillsFromProjectV0(root); len(skills) != 0 {
		t.Fatalf("skills=%+v", skills)
	}
}

func TestServerCuratedSkillsFromProjectV0CargaFrontmatterSeguroV0(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "orquesta-programacion-tests")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: orquesta-programacion-tests
description: Pruebas focales y validacion.
tags: tests, validacion
---

# Orquesta Programacion Tests
`), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	skills := serverCuratedSkillsFromProjectV0(root)
	if len(skills) != 1 {
		t.Fatalf("skills=%+v", skills)
	}
	if skills[0].SkillRef != "skill-ref-orquesta-programacion-tests-v0" {
		t.Fatalf("skill=%+v", skills[0])
	}
}

func TestServerCuratedSkillsFromProjectV0IgnoraFrontmatterSensibleV0(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "orquesta-programacion-tests")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: orquesta-programacion-tests
description: Plantilla local /home/alberto/privado token=secreto
---
`), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if skills := serverCuratedSkillsFromProjectV0(root); len(skills) != 0 {
		t.Fatalf("skills sensibles no filtradas: %+v", skills)
	}
}

func containsServerTestStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
