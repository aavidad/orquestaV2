package main

import (
	"strings"
	"testing"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func TestCodexDirectorChildRoleProfileV0UsaRolesTecnicosParaProgramacion(t *testing.T) {
	plan := orquestadirectoroperativo.OperationalDirectorPlanV0{
		Mode:       orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		ProjectRef: "orquesta",
		Objective:  "Autoprogramar Orquesta como ola complementaria con servidor y panel web.",
		DomainRefs: []string{"self_improvement"},
	}

	profile := codexDirectorChildRoleProfileV0(plan, nil)
	role := codexDirectorChildRoleV0(profile, 3)

	if profile != codexDirectorChildRoleProfileProgrammingV0 {
		t.Fatalf("profile=%q want %q", profile, codexDirectorChildRoleProfileProgrammingV0)
	}
	if strings.Contains(role, "A1") || strings.Contains(role, "teorico") {
		t.Fatalf("rol de programacion contaminado por perfil documental: %q", role)
	}
	if !strings.Contains(role, "implementacion") {
		t.Fatalf("rol tecnico inesperado: %q", role)
	}
}

func TestCodexDirectorChildRoleProfileV0ConservaRolesDocumentalesParaDomainWork(t *testing.T) {
	plan := orquestadirectoroperativo.OperationalDirectorPlanV0{
		Mode:       orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0,
		ProjectRef: "opes",
		Objective:  "Preparar document_plan con tono A1.",
		DomainRefs: []string{"opes"},
	}

	profile := codexDirectorChildRoleProfileV0(plan, nil)
	role := codexDirectorChildRoleV0(profile, 3)

	if profile != codexDirectorChildRoleProfileDocumentDomainV0 {
		t.Fatalf("profile=%q want %q", profile, codexDirectorChildRoleProfileDocumentDomainV0)
	}
	if !strings.Contains(role, "A1") {
		t.Fatalf("rol documental inesperado: %q", role)
	}
}

func TestCodexDirectorChildRoleProfileV0NoTrataA1ComoSubcadenaV0(t *testing.T) {
	profile := codexDirectorChildRoleProfileV0(
		orquestadirectoroperativo.OperationalDirectorPlanV0{
			Mode:       orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
			ProjectRef: "orquesta-a10-self-topup",
			Objective:  "Autoprogramar Orquesta con 10 padres y 6 subagentes por padre.",
			DomainRefs: []string{"autoprogramming"},
		},
		[]codexDirectorDomainContextBlockV0{{
			SourceRef: "self-topup-a10",
			Text:      "revision tecnica de capacidad 10x6",
		}},
	)

	if profile != codexDirectorChildRoleProfileProgrammingV0 {
		t.Fatalf("profile=%s", profile)
	}
}

func TestCodexDirectorChildRoleIndexV0ReservaUltimoHijoParaRevisionV0(t *testing.T) {
	for _, tc := range []struct {
		childIndex int
		childTotal int
		want       int
	}{
		{childIndex: 1, childTotal: 2, want: 1},
		{childIndex: 2, childTotal: 2, want: 6},
		{childIndex: 5, childTotal: 5, want: 6},
		{childIndex: 6, childTotal: 6, want: 6},
	} {
		if got := codexDirectorChildRoleIndexV0(tc.childIndex, tc.childTotal); got != tc.want {
			t.Fatalf("child=%d/%d got=%d want=%d", tc.childIndex, tc.childTotal, got, tc.want)
		}
	}
}
