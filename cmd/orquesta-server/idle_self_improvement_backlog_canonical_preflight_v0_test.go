package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0BloqueaDuplicadoDeCanonicaCerradaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := canonicalPreflightBacklogContentTestV0("cerrada")
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:  2,
			Trigger:      "capacity_free",
			FreeCapacity: 2,
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
	if len(result.Requests) != 0 ||
		result.Message != "backlog_scanner_canonical_preflight_required" ||
		backlogCollisionForTestV0(result.Collisions, "backlog_scanner_canonical_preflight_required").Code == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestIdleSelfImprovementBacklogScannerV0IncluyePreflightCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte(canonicalPreflightBacklogContentTestV0("cerrada")), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	request := idleSelfImprovementBacklogScannerRequestV0(
		orquestaserver.IdleSelfImprovementRequestV0{RequestRef: "request-ref-base", ProjectRef: "project-ref-orquesta"},
		orquestaserver.IdleSelfImprovementPlanRequestV0{Trigger: "capacity_free"},
	)
	request = planner.withBacklogScannerMergeLeaseV0(request)
	if !containsStringForTestV0(request.ContextRefs, "backlog_scanner_canonical_preflight:required") ||
		!containsStringForTestV0(request.AcceptanceCriteria, "backlog_scanner_canonical_preflight_required") ||
		!containsStringForTestV0(request.EvidenceRefs, "evidence-ref-autoprogramming-backlog-scanner-canonical-preflight") {
		t.Fatalf("request=%+v", request)
	}
	if !containsStringPrefixForTaskIDLeaseTestV0(request.ContextRefs, "backlog_canonical_preflight_entry:") {
		t.Fatalf("context_refs=%+v", request.ContextRefs)
	}
}

func canonicalPreflightBacklogContentTestV0(canonicalState string) string {
	return `## T140 backlog-overlap-canonicalization

Objetivo: impedir duplicados canonicos de scanner.
Estado: ` + canonicalState + `.

Alcance:

- ` + "`cmd/orquesta-server`" + `

Criterios:

- Tests: ` + "`go test -count=1 ./cmd/orquesta-server`" + `

## T251 backlog-scanner-canonical-preflight

Objetivo: impedir duplicados canonicos de scanner.
Estado: pendiente.

Alcance:

- ` + "`cmd/orquesta-server`" + `

Criterios:

- Tests: ` + "`go test -count=1 ./cmd/orquesta-server`" + `
`
}
