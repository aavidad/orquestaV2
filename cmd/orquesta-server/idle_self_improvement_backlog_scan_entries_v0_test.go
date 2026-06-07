package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestBacklogParserV0IgnoraSeccionesEscaneoComoTareaV0(t *testing.T) {
	content := `# Backlog

## Escaneo backlog 2026-05-27 primera pasada

Evidencia revisada.

## T249 backlog-scan-entry-identity-and-order-policy

Objetivo: dar identidad estable.
`
	sections := parseIdleSelfImprovementBacklogSectionsV0(content)
	if len(sections) != 1 ||
		sections[0].Ref != "t249-backlog-scan-entry-identity-and-order-policy" {
		t.Fatalf("sections=%+v", sections)
	}
}

func TestBacklogScanEntryRefsV0DetectaDuplicadoYOrdenAmbiguoV0(t *testing.T) {
	content := `# Backlog

## Escaneo backlog 2026-05-27 segunda pasada

Evidencia 2.

## Escaneo backlog 2026-05-27 primera pasada

Evidencia 1.

## Escaneo backlog 2026-05-27 primera pasada

Evidencia duplicada.
`
	entries, issues := backlogScanEntryRefsForDocumentV0(
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
		content,
	)
	if len(entries) != 3 || backlogScanEntryDigestV0(entries) == "" {
		t.Fatalf("entries=%+v", entries)
	}
	if !hasBacklogScanIssueForTestV0(issues, "backlog_scan_entry_duplicate") ||
		!hasBacklogScanIssueForTestV0(issues, "backlog_scan_entry_order_ambiguous") {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestBacklogScanEntryOrdinalV0ReconoceSecuenciasHistoricasV0(t *testing.T) {
	cases := map[string]int{
		"vigesimoprimera pasada":        21,
		"vigesimosegunda pasada":        22,
		"vigesima tercera pasada":       23,
		"vigesimoctava pasada":          28,
		"trigesimoprimera pasada":       31,
		"quincuagesima sexta pasada":    56,
		"sexagesima septima pasada":     67,
		"septuagesima tercera pasada":   73,
		"undecima pasada":               11,
		"retry 42fbc4 burst 002 abd199": 0,
	}
	for value, want := range cases {
		if got := backlogScanEntryOrdinalV0(value); got != want {
			t.Fatalf("value=%q got=%d want=%d", value, got, want)
		}
	}
}

func TestBacklogScanRequestV0IncluyeIdentidadEIssuesDeEscaneoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## Escaneo backlog 2026-05-27 primera pasada

Evidencia 1.

## Escaneo backlog 2026-05-27 primera pasada

Evidencia duplicada.

## T249 backlog-scan-entry-identity-and-order-policy

Objetivo: dar identidad estable.
`
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte(content),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	request := idleSelfImprovementBacklogScannerRequestV0(
		orquestaserver.IdleSelfImprovementRequestV0{ProjectRef: "project-ref-orquesta"},
		orquestaserver.IdleSelfImprovementPlanRequestV0{Trigger: "capacity_free"},
	)
	request = planner.withBacklogScannerMergeLeaseV0(request)

	if len(request.BacklogScanDocs) == 0 ||
		len(request.BacklogScanDocs[0].ScanEntries) != 2 ||
		request.BacklogScanDocs[0].ScanEntries[0].Ref == "" ||
		request.BacklogScanDocs[0].ScanEntries[0].Line != 3 ||
		request.BacklogScanDocs[0].ScanEntryDigest == "" ||
		request.BacklogScanDocs[0].ScanEntryCount != 2 ||
		!hasBacklogScanIssueForTestV0(
			request.BacklogScanDocs[0].ScanEntryIssues,
			"backlog_scan_entry_duplicate",
		) {
		t.Fatalf("docs=%+v", request.BacklogScanDocs)
	}
	if request.BacklogScanRef == "" ||
		!containsStringForTestV0(request.ContextRefs, "backlog_scan_ref:"+request.BacklogScanRef) ||
		!containsStringForTestV0(request.AcceptanceCriteria, "backlog_scan_ref:"+request.BacklogScanRef) ||
		!containsStringPrefixForBacklogScanEntriesTestV0(request.ContextRefs, "backlog_scan_entry:") ||
		!containsStringPrefixForBacklogScanEntriesTestV0(request.ContextRefs, "backlog_scan_entries:") ||
		!containsStringPrefixForBacklogScanEntriesTestV0(request.AcceptanceCriteria, "backlog_scan_entry_issue:") ||
		!containsStringForTestV0(
			request.AcceptanceCriteria,
			"si hay backlog_scan_entry_issue, conservar propuesta como borrador y anotar rebase/merge pendiente",
		) {
		t.Fatalf("context=%+v criteria=%+v", request.ContextRefs, request.AcceptanceCriteria)
	}
}

func hasBacklogScanIssueForTestV0(
	issues []orquestaserver.BacklogScanEntryIssueV0,
	code string,
) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func containsStringPrefixForBacklogScanEntriesTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
