package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0DeduplicaPropuestasPorFingerprintV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T250 backlog-proposal-deduplication-fingerprint

Objetivo: evitar que scanners creen propuestas duplicadas para el mismo owner.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`" + `

## T251 scanner-proposal-deduplication-preflight

Objetivo: evitar que scanners creen propuestas duplicadas para el mismo owner.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planProposalDedupeBacklogForTestV0(t, projectDir, nil)
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t250-backlog-proposal-deduplication-fingerprint" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	if !containsStringPrefixForBacklogProposalDedupeTestV0(result.Requests[0].ContextRefs, "backlog_proposal_fingerprint:") {
		t.Fatalf("context_refs=%+v", result.Requests[0].ContextRefs)
	}
	if collision := backlogCollisionForTestV0(result.Collisions, "duplicate_backlog_proposal_fingerprint"); collision.SectionRef != "t251-scanner-proposal-deduplication-preflight" {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0BloqueaFingerprintSiDuplicadoEstaVivoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T250 backlog-proposal-deduplication-fingerprint

Objetivo: evitar que scanners creen propuestas duplicadas para el mismo owner.

Alcance:

- ` + "`cmd/orquesta-server`" + `

## T251 scanner-proposal-deduplication-preflight

Objetivo: evitar que scanners creen propuestas duplicadas para el mismo owner.

Alcance:

- ` + "`cmd/orquesta-server`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planProposalDedupeBacklogForTestV0(t, projectDir, []string{
		"request-ref-autoprogramming-backlog-t251-scanner-proposal-deduplication-preflight-live",
	})
	if len(result.Requests) != 0 || result.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("result=%+v", result)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0RespetaSubshardIntencionalV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T260 server-status-split-a

Objetivo: dividir servidor residente por owner local.

Alcance:

- ` + "`modulos/orquesta-server`" + `

## T261 server-status-split-b

Objetivo: dividir servidor residente por owner local.

Alcance:

- ` + "`modulos/orquesta-server`" + `

Criterios:

- subshard intencional con frontera reducida, criterio no cubierto y motivo causal documentado.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result := planProposalDedupeBacklogForTestV0(t, projectDir, nil)
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v collisions=%+v", result.Requests, result.Collisions)
	}
}

func planProposalDedupeBacklogForTestV0(
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
				RequestRef: "request-ref-base",
				ProjectRef: "project-ref-orquesta",
				WriteSet:   []string{"cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	return result
}

func containsStringPrefixForBacklogProposalDedupeTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
