package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0UsaCanonicaParaDuplicadosV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T102 legacy-http-json-boundary-policy

Objetivo: unificar frontera JSON publica.

Alcance:

- ` + "`cmd/orquesta-server`" + `

Criterios:

- criterio canonico.
- Tests: ` + "`go test -count=1 ./cmd/orquesta-server`" + `

## T137 public-http-request-body-bounds

Objetivo: unificar limites y decodificacion HTTP publica.

Fusionada_con: T102 legacy-http-json-boundary-policy

Alcance:

- ` + "`modulos/orquesta-mcp`" + `

Criterios:

- criterio duplicado conservado.
- Tests: ` + "`go test -count=1 ./modulos/orquesta-mcp`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planCanonicalBacklogForTestV0(t, projectDir, nil)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.SuggestedArea != "t102-legacy-http-json-boundary-policy" ||
		!containsStringForTestV0(request.WriteSet, "cmd/orquesta-server") ||
		!containsStringForTestV0(request.WriteSet, "modulos/orquesta-mcp") ||
		!containsStringForTestV0(request.AcceptanceCriteria, "criterio duplicado conservado.") ||
		!containsStringForTestV0(request.RequiredTests, "go test -count=1 ./modulos/orquesta-mcp") {
		t.Fatalf("request=%+v", request)
	}
	if collision := backlogCollisionForTestV0(result.Collisions, "duplicate_backlog_task"); collision.SectionRef != "t137-public-http-request-body-bounds" {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0BloqueaCanonicaSiAliasEstaVivoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T102 legacy-http-json-boundary-policy

Objetivo: unificar frontera JSON publica.

## T137 public-http-request-body-bounds

Objetivo: unificar limites y decodificacion HTTP publica.

Fusionada_con: T102 legacy-http-json-boundary-policy
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planCanonicalBacklogForTestV0(t, projectDir, []string{
		"request-ref-autoprogramming-backlog-t137-public-http-request-body-bounds-live",
	})
	if len(result.Requests) != 0 || result.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("result=%+v", result)
	}
	if collision := backlogCollisionForTestV0(result.Collisions, "duplicate_backlog_task"); collision.SectionRef == "" {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0CanonicalizaSolapePorFirmaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T248 backlog-scan-section-id-and-sequence-policy

Objetivo: dar identidad estable y verificable a cada bloque de escaneo de backlog.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `

Criterios:

- distinguir scan_ref, lineas y hashes.
- Tests: ` + "`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`" + `

## T249 backlog-scan-entry-identity-and-order-policy

Objetivo: dar identidad estable y validable a las entradas documentales de escaneo de backlog.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `

Criterios:

- detectar orden, saltos y duplicados por scan_entry_ref.
- Tests: ` + "`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planCanonicalBacklogForTestV0(t, projectDir, nil)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.SuggestedArea != "t248-backlog-scan-section-id-and-sequence-policy" ||
		!containsStringForTestV0(request.AcceptanceCriteria, "detectar orden, saltos y duplicados por scan_entry_ref.") ||
		!containsStringForTestV0(request.RequiredTests, "go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server") {
		t.Fatalf("request=%+v", request)
	}
	collision := backlogCollisionForTestV0(result.Collisions, "backlog_task_overlap_canonicalization_required")
	if collision.SectionRef != "t249-backlog-scan-entry-identity-and-order-policy" ||
		!strings.Contains(collision.Message, "covered_by=t248-backlog-scan-section-id-and-sequence-policy") {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoFusionaFronterasVecinasV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T240 backlog-doc-read-path-budget-policy

Objetivo: acotar path y budget de lectura documental del scanner.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `

Criterios:

- controlar presupuesto de lectura.
- Tests: ` + "`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`" + `

## T249 backlog-scan-entry-identity-and-order-policy

Objetivo: dar identidad estable y validable a las entradas documentales de escaneo de backlog.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `

Criterios:

- detectar orden, saltos y duplicados por scan_entry_ref.
- Tests: ` + "`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planCanonicalBacklogForTestV0(t, projectDir, nil)
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v collisions=%+v", result.Requests, result.Collisions)
	}
	if collision := backlogCollisionForTestV0(result.Collisions, "backlog_task_overlap_canonicalization_required"); collision.SectionRef != "" {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func planCanonicalBacklogForTestV0(
	t *testing.T,
	projectDir string,
	knownRequestRefs []string,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:      5,
			KnownRequestRefs: knownRequestRefs,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	return result
}

func backlogCollisionForTestV0(
	collisions []orquestaserver.BacklogScanCollisionV0,
	code string,
) orquestaserver.BacklogScanCollisionV0 {
	for _, collision := range collisions {
		if collision.Code == code {
			return collision
		}
	}
	return orquestaserver.BacklogScanCollisionV0{}
}
