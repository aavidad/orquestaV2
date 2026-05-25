package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0PreservaTestsNoGoOrdenadosV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir,
		"# Backlog\n\n"+
			"## T37 parser-fidelity\n\n"+
			"Objetivo: preservar verificaciones declaradas.\n\n"+
			"Alcance:\n\n"+
			"- cmd/orquesta-server\n\n"+
			"Criterios:\n\n"+
			"- Tests: `git diff --check` y `bash -n scripts/*.sh` y "+
			"`./scripts/test_rails_fast.sh` y "+
			"`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server` "+
			"y prueba documental focal que busque contradicciones.\n")

	result := mustPlanParserFidelityTestV0(t, projectDir,
		[]string{"go test -count=1 ./cmd/orquesta-server"},
	)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	wantTests := []string{
		"git diff --check",
		"bash -n scripts/*.sh",
		"./scripts/test_rails_fast.sh",
		"go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server",
	}
	if !stringSlicesEqualForParserFidelityTestV0(request.RequiredTests, wantTests) {
		t.Fatalf("tests=%+v want=%+v", request.RequiredTests, wantTests)
	}
	if !containsStringForTestV0(
		request.AcceptanceCriteria,
		"verificacion manual requerida: prueba documental focal que busque contradicciones",
	) {
		t.Fatalf("criteria=%+v", request.AcceptanceCriteria)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0ManualTestNoUsaBaseV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## T38 manual-only

Objetivo: verificar documentacion.

Alcance:

- docs/autoprogramacion_orquesta_pendientes_2026-05-23.md

Criterios:

- Tests: prueba documental focal que busque contradicciones.
`)

	result := mustPlanParserFidelityTestV0(t, projectDir,
		[]string{"go test -count=1 ./cmd/orquesta-server"},
	)
	request := result.Requests[0]
	if len(request.RequiredTests) != 0 {
		t.Fatalf("manual verification must not become base required test: %+v", request.RequiredTests)
	}
	if !containsStringForTestV0(
		request.ContextRefs,
		"backlog_manual_verification:prueba documental focal que busque contradicciones",
	) {
		t.Fatalf("context_refs=%+v", request.ContextRefs)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0RevalidacionFinalNoOcultaSeccionV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## T39 ambiguous-state

Objetivo: corregir tarea pendiente.

Alcance:

- cmd/orquesta-server

Revalidacion final: pendiente de ejecutar.
`)

	result := mustPlanParserFidelityTestV0(t, projectDir,
		[]string{"go test -count=1 ./cmd/orquesta-server"},
	)
	request := result.Requests[0]
	if request.FailureKind != "backlog_autoprogramming" ||
		request.SuggestedArea != "t39-ambiguous-state" {
		t.Fatalf("request=%+v", request)
	}
}

func mustPlanParserFidelityTestV0(
	t *testing.T,
	projectDir string,
	baseTests []string,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: baseTests,
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) == 0 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	return result
}

func mustWriteBacklogForParserFidelityTestV0(t *testing.T, projectDir string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
}

func stringSlicesEqualForParserFidelityTestV0(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
