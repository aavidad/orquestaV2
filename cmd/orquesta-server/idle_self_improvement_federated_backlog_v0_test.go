package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0UsaIndiceFederadoLocalV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFederatedBacklogIndexTestV0(t, projectDir, "vigente",
		"go test -count=1 ./modulos/orquesta-autoprogramming")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md", `# Tareas

## Backlog

- APG-001: ampliar limites por contrato si apps grandes necesitan tareas largas.
`)

	result := mustPlanFederatedBacklogTestV0(t, projectDir, 1)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.FailureKind != "backlog_autoprogramming" ||
		request.SuggestedArea != "apg-001" ||
		request.WriteSet[0] != "modulos/orquesta-autoprogramming" ||
		request.RequiredTests[0] != "go test -count=1 ./modulos/orquesta-autoprogramming" {
		t.Fatalf("request=%+v", request)
	}
	for _, want := range []string{
		"backlog_doc:modulos/orquesta-autoprogramming/docs/tareas.md",
		"backlog_local_alias:APG-001",
		"backlog_federated_owner:modulos/orquesta-autoprogramming",
	} {
		if !containsStringForTestV0(request.ContextRefs, want) {
			t.Fatalf("missing %s in context_refs=%+v", want, request.ContextRefs)
		}
	}
	if len(request.BacklogScanDocs) != 1 ||
		request.BacklogScanDocs[0].Path != "modulos/orquesta-autoprogramming/docs/tareas.md" ||
		request.BacklogScanDocs[0].SectionRef != "apg-001" {
		t.Fatalf("scan docs=%+v", request.BacklogScanDocs)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0UsaIndiceFederadoLocalMultilineaV0(t *testing.T) {
	projectDir := t.TempDir()
	content := "# Backlog\n\n## Indice federado de backlog local\n\n" +
		"- source_path: `modulos/orquesta-autoprogramming/docs/tareas.md`; source_kind:\n" +
		"  module_tasks; owner: `modulos/orquesta-autoprogramming`; estado: vigente;\n" +
		"  aliases: APG-*; tests: `go test -count=1 ./modulos/orquesta-autoprogramming`.\n"
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md", `# Tareas

## Backlog

- APG-001: ampliar limites por contrato si apps grandes necesitan tareas largas.
`)

	result := mustPlanFederatedBacklogTestV0(t, projectDir, 1)
	request := result.Requests[0]
	if request.FailureKind != "backlog_autoprogramming" ||
		request.SuggestedArea != "apg-001" ||
		request.WriteSet[0] != "modulos/orquesta-autoprogramming" ||
		request.RequiredTests[0] != "go test -count=1 ./modulos/orquesta-autoprogramming" {
		t.Fatalf("request=%+v", request)
	}
	for _, want := range []string{
		"backlog_doc:modulos/orquesta-autoprogramming/docs/tareas.md",
		"backlog_federated_source_kind:module_tasks",
		"backlog_federated_owner:modulos/orquesta-autoprogramming",
		"backlog_federated_state:vigente",
		"backlog_federated_source_line:5",
		"backlog_local_alias:APG-001",
	} {
		if !containsStringForTestV0(request.ContextRefs, want) {
			t.Fatalf("missing %s in context_refs=%+v", want, request.ContextRefs)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0IgnoraFuenteFederadaHistoricaV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFederatedBacklogIndexTestV0(t, projectDir, "historico",
		"go test -count=1 ./modulos/orquesta-autoprogramming")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md",
		"- APG-001: no debe programarse desde fuente historica.\n")

	result := mustPlanFederatedBacklogTestV0(t, projectDir, 1)
	if len(result.Requests) != 1 ||
		result.Requests[0].SuggestedArea == "apg-001" {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0ClasificaLocalSinTestsV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFederatedBacklogIndexTestV0(t, projectDir, "vigente", "")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md",
		"- APG-001: requiere owner y pruebas antes de programar codigo.\n")

	result := mustPlanFederatedBacklogTestV0(t, projectDir, 1)
	request := result.Requests[0]
	if request.FailureKind != "backlog_documentation_review" ||
		containsStringForTestV0(request.WriteSet, "cmd/orquesta-server") ||
		!containsStringForTestV0(request.WriteSet, "modulos/orquesta-autoprogramming/docs/tareas.md") {
		t.Fatalf("request=%+v", request)
	}
}

func mustPlanFederatedBacklogTestV0(
	t *testing.T,
	projectDir string,
	maxRequests int,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: maxRequests,
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
	if len(result.Requests) == 0 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	return result
}

func mustWriteFederatedBacklogIndexTestV0(
	t *testing.T,
	projectDir string,
	state string,
	tests string,
) {
	t.Helper()
	testField := ""
	if tests != "" {
		testField = "; tests: `" + tests + "`"
	}
	content := "# Backlog\n\n## Indice federado de backlog local\n\n" +
		"- source_path: `modulos/orquesta-autoprogramming/docs/tareas.md`; " +
		"source_kind: module_tasks; owner: `modulos/orquesta-autoprogramming`; " +
		"estado: " + state + "; aliases: APG-*" + testField + "\n"
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
}

func mustWriteFederatedLocalDocTestV0(t *testing.T, projectDir string, rel string, content string) {
	t.Helper()
	path := filepath.Join(projectDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir local doc: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write local doc: %v", err)
	}
}
