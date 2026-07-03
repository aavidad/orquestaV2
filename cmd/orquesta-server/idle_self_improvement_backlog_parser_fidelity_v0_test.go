package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestIdleSelfImprovementBacklogPlannerV0ScannerDocumentalNoHeredaGoTestGlobalV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, "# Backlog\n\nSin secciones Txx.\n")

	result := mustPlanParserFidelityWithRequestTestV0(t, projectDir,
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			Trigger:     "backlog_scan",
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{
					"go test -count=1 ./...",
					idleSelfImprovementRefOnlyRequiredTestV0,
				},
			},
		},
	)
	request := result.Requests[0]
	if containsStringForTestV0(request.RequiredTests, "go test -count=1 ./...") ||
		!containsStringForTestV0(request.RequiredTests, idleSelfImprovementBacklogDocumentalRequiredTestV0) ||
		!containsStringForTestV0(request.RequiredTests, idleSelfImprovementRefOnlyRequiredTestV0) {
		t.Fatalf("required_tests=%+v", request.RequiredTests)
	}
	for _, want := range []string{
		"required_test_origin:global_policy:documental_focal",
		"required_test_origin:ref_only_guard",
		"required_test_scope_policy:global_go_test_all_omitted_for_doc_only_scanner",
	} {
		if !containsStringForTestV0(request.ContextRefs, want) {
			t.Fatalf("missing %s in context_refs=%+v", want, request.ContextRefs)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0IgnoraSeccionesNarrativasTNoNumericasV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## Tareas narrativas

Este bloque resume contexto y no es una tarea ejecutable.

## Trazas de auditoria

Tampoco debe convertirse en run de automejora.

## T41 ejecutable-real

Objetivo: corregir una tarea real.

Alcance:

- cmd/orquesta-server
`)

	result := mustPlanParserFidelityTestV0(t, projectDir,
		[]string{"go test -count=1 ./cmd/orquesta-server"},
	)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.SuggestedArea != "t41-ejecutable-real" ||
		strings.Contains(request.FailureSummary, "Tareas narrativas") ||
		strings.Contains(request.FailureSummary, "Trazas de auditoria") {
		t.Fatalf("request=%+v", request)
	}
}

func TestIdleSelfImprovementBacklogParserV0WriteSetAliasesAlimentanScopeV0(t *testing.T) {
	baseContent := func(label string) string {
		return `# Backlog

## T42 alias-scope

Objetivo: comprobar alias de alcance.

` + label + `

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `
`
	}
	canonical := parseIdleSelfImprovementBacklogSectionsV0(baseContent("Alcance:"))
	if len(canonical) != 1 {
		t.Fatalf("canonical sections=%+v", canonical)
	}
	for _, label := range []string{"Write-set previsto:", "Write-set:"} {
		sections := parseIdleSelfImprovementBacklogSectionsV0(baseContent(label))
		if len(sections) != 1 {
			t.Fatalf("%s sections=%+v", label, sections)
		}
		if !stringSlicesEqualForParserFidelityTestV0(sections[0].Scope, canonical[0].Scope) {
			t.Fatalf("%s scope=%+v want=%+v", label, sections[0].Scope, canonical[0].Scope)
		}
		if len(sections[0].DiagnosticEvidenceRefs) != 0 {
			t.Fatalf("%s diagnostics=%+v", label, sections[0].DiagnosticEvidenceRefs)
		}
	}
}

func TestIdleSelfImprovementBacklogParserV0ValidacionAliasAlimentaTestsV0(t *testing.T) {
	baseContent := func(label string) string {
		return `# Backlog

## T43 alias-tests

Objetivo: comprobar alias de validacion.

Alcance:

- cmd/orquesta-server

` + label + ` ` + "`go test -count=1 ./cmd/orquesta-server`" + ` y prueba manual focal.
`
	}
	canonical := parseIdleSelfImprovementBacklogSectionsV0(baseContent("Tests:"))
	aliased := parseIdleSelfImprovementBacklogSectionsV0(baseContent("Validacion:"))
	if len(canonical) != 1 || len(aliased) != 1 {
		t.Fatalf("canonical=%+v aliased=%+v", canonical, aliased)
	}
	if !stringSlicesEqualForParserFidelityTestV0(aliased[0].Tests, canonical[0].Tests) {
		t.Fatalf("tests=%+v want=%+v", aliased[0].Tests, canonical[0].Tests)
	}
	if !stringSlicesEqualForParserFidelityTestV0(aliased[0].ManualVerifications, canonical[0].ManualVerifications) {
		t.Fatalf("manual=%+v want=%+v", aliased[0].ManualVerifications, canonical[0].ManualVerifications)
	}
	if len(aliased[0].DiagnosticEvidenceRefs) != 0 {
		t.Fatalf("diagnostics=%+v", aliased[0].DiagnosticEvidenceRefs)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EtiquetaDesconocidaViajaEnEvidenceRefsV0(t *testing.T) {
	for _, tc := range []struct {
		label    string
		evidence string
	}{
		{label: "Scope:", evidence: "section_label_unrecognized:Scope"},
		{label: "Alcance previsto:", evidence: "section_label_unrecognized:Alcance previsto"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			projectDir := t.TempDir()
			mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## T44 etiqueta-desconocida

Objetivo: diagnosticar etiqueta parecida.

`+tc.label+`

- modulos/orquesta-server

Criterios:

- emitir evidencia diagnostica.

Tests:

- `+"`go test -count=1 ./cmd/orquesta-server`"+`
`)

			result := mustPlanParserFidelityWithRequestTestV0(t, projectDir,
				orquestaserver.IdleSelfImprovementPlanRequestV0{
					MaxRequests: 1,
					BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
						RequestRef:    "request-ref-base",
						CorrelationID: "corr-request-ref-base",
						WriteSet:      []string{"base-write-set"},
					},
				},
			)
			request := result.Requests[0]
			if !containsStringForTestV0(request.EvidenceRefs, tc.evidence) {
				t.Fatalf("evidence_refs=%+v want=%s", request.EvidenceRefs, tc.evidence)
			}
			if containsStringForTestV0(request.WriteSet, "modulos/orquesta-server") ||
				!containsStringForTestV0(request.WriteSet, "base-write-set") {
				t.Fatalf("write_set=%+v", request.WriteSet)
			}
		})
	}
}

func TestIdleSelfImprovementBacklogParserV0EtiquetaDesconocidaNoDiagnosticaConCanonicoV0(t *testing.T) {
	sections := parseIdleSelfImprovementBacklogSectionsV0(`# Backlog

## T45 canonico-presente

Objetivo: no ensuciar seccion correcta.

Alcance:

- cmd/orquesta-server

Scope:

- modulos/orquesta-server

Criterios:

- conservar diagnostico vacio si el campo canonico esta presente.

Tests:

- ` + "`go test -count=1 ./cmd/orquesta-server`" + `
`)
	if len(sections) != 1 {
		t.Fatalf("sections=%+v", sections)
	}
	if len(sections[0].DiagnosticEvidenceRefs) != 0 {
		t.Fatalf("diagnostics=%+v", sections[0].DiagnosticEvidenceRefs)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0PreservaGoTestGlobalDeclaradoEnSeccionV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## T40 declarada

Objetivo: preservar pruebas explicitas.

Alcance:

- docs/autoprogramacion_orquesta_pendientes_2026-05-23.md

Criterios:

- Tests: `+"`go test -count=1 ./...`"+`
`)

	result := mustPlanParserFidelityTestV0(t, projectDir,
		[]string{"go test -count=1 ./cmd/orquesta-server"},
	)
	request := result.Requests[0]
	if !containsStringForTestV0(request.RequiredTests, "go test -count=1 ./...") ||
		!containsStringForTestV0(request.ContextRefs, "required_test_origin:section_declared") {
		t.Fatalf("request=%+v", request)
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

func TestIdleSelfImprovementBacklogPlannerV0EstadoFechadoConFechaNoRelanzaAliasV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, `# Backlog

## T256 server-shutdown-usecase-file-split

Objetivo: dividir shutdown_v0.go.

Estado 2026-05-27: T256 queda resuelto dentro de orquesta-server-shutdown.

Alcance:

- modulos/orquesta-server-shutdown
`)

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			KnownRequestRefs: []string{
				"request-ref-autoprogramming-backlog-t256-server-shutdown-usecase-file-split-before-growth-live",
			},
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef: "request-ref-base",
				WriteSet:   []string{"modulos/orquesta-server-shutdown"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 0 || result.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("result=%+v", result)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EstadoSincronizadoNoOpNoRelanzaT260V0(t *testing.T) {
	projectDir := t.TempDir()
	content := `# Backlog

## T260 corregir-estados-falsos-running-en-agentes-externos

Objetivo: evitar estados falsos running en agentes externos.

Estado: sincronizado/no-op documental; cerrado localmente y pendiente solo validacion opt-in futura.

Alcance:

- cmd/orquesta-server
- modulos/orquesta-app-codex-stack
`
	mustWriteBacklogForParserFidelityTestV0(t, projectDir, content)

	sections := parseIdleSelfImprovementBacklogSectionsV0(content)
	if len(sections) != 1 || !sections[0].Completed || sections[0].PendingExplicit {
		t.Fatalf("sections=%+v", sections)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef: "request-ref-base",
				WriteSet:   []string{"cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	for _, request := range result.Requests {
		if request.FailureKind == "backlog_autoprogramming" ||
			request.SuggestedArea == "t260-corregir-estados-falsos-running-en-agentes-externos" {
			t.Fatalf("result=%+v", result)
		}
	}
	if len(result.Requests) > 1 {
		t.Fatalf("result=%+v", result)
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

func mustPlanParserFidelityWithRequestTestV0(
	t *testing.T,
	projectDir string,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(request)
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
