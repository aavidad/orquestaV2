package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0CierraPorDocsLocalesV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForStateSyncTestV0(t, projectDir, `# Backlog

## T17 autoprogramming-backlog-state-sync

Objetivo: sincronizar backlog.

Alcance:

- modulos/orquesta-server

## T18 siguiente

Objetivo: ejecutar trabajo pendiente.

Alcance:

- cmd/orquesta-server
`)
	localDoc := filepath.Join(projectDir, "modulos", "orquesta-server", "docs")
	if err := os.MkdirAll(localDoc, 0o700); err != nil {
		t.Fatalf("mkdir local docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localDoc, "tareas.md"), []byte(`## t17-autoprogramming-backlog-state-sync

Estado: completada con evidencia durable.
Evidencia focal: go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server.
`), 0o600); err != nil {
		t.Fatalf("write local doc: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 2,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t18-siguiente" {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EvidenciaAmbiguaGeneraRevisionDocumentalV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogForStateSyncTestV0(t, projectDir, `# Backlog

## T17 autoprogramming-backlog-state-sync

Objetivo: sincronizar backlog.

Alcance:

- cmd/orquesta-server
- modulos/orquesta-server

Evidencia focal: go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server.
`)

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				WriteSet:      []string{"cmd/orquesta-server", "modulos/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.FailureKind != "backlog_documentation_review" ||
		!containsStringForTestV0(request.ContextRefs, "backlog_state:ambiguous_evidence") {
		t.Fatalf("request=%+v", request)
	}
	for _, forbidden := range []string{"cmd/orquesta-server", "modulos/orquesta-server"} {
		if containsStringForTestV0(request.WriteSet, forbidden) {
			t.Fatalf("revision documental no debe abrir codigo: %+v", request.WriteSet)
		}
	}
	if !containsStringForTestV0(request.WriteSet, idleSelfImprovementBacklogDocRelV0) ||
		!containsStringForTestV0(request.WriteSet, "docs/runbooks") {
		t.Fatalf("write_set=%+v", request.WriteSet)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoAnadeScannerSiUnicaTareaYaVisibleV0(t *testing.T) {
	projectDir := t.TempDir()
	content := "## T08 runtime-neutral-e2e\n\nObjetivo: uno.\n"
	mustWriteBacklogForStateSyncTestV0(t, projectDir, content)
	section := parseIdleSelfImprovementBacklogSectionsV0(content)[0]

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:      2,
			Trigger:          "capacity_free",
			QueueSize:        1,
			FreeCapacity:     1,
			KnownRequestRefs: []string{idleSelfImprovementRequestRefForBacklogSectionV0(section)},
			BaseRequest:      orquestaserver.IdleSelfImprovementRequestV0{ProjectRef: "project-ref-orquesta"},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 0 || result.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("result=%+v", result)
	}
}

func mustWriteBacklogForStateSyncTestV0(t *testing.T, projectDir string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
}
