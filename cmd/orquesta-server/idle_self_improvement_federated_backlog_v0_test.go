package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestIdleSelfImprovementBacklogPlannerV0AcotaEpochAFuentesFederadasEjecutablesV0(t *testing.T) {
	projectDir := t.TempDir()
	content := "# Backlog\n\n## Indice federado de backlog local\n\n" +
		"- source_path: `modulos/orquesta-autoprogramming/docs/tareas.md`; " +
		"source_kind: module_tasks; owner: `modulos/orquesta-autoprogramming`; " +
		"estado: vigente; aliases: APG-*; tests: `go test -count=1 ./modulos/orquesta-autoprogramming`\n" +
		"- source_path: `docs/legacy_orquestador_externo.md`; " +
		"source_kind: legacy_docs; owner: `docs`; estado: quarantine; aliases: LEG-*\n"
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md",
		"- APG-001: tarea ejecutable.\n")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/legacy_orquestador_externo.md",
		"- LEG-001: legado cuarentenado.\n")

	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	first := planner.withBacklogScannerMergeLeaseV0(orquestaserver.IdleSelfImprovementRequestV0{
		WriteSet: []string{"cmd/orquesta-server"},
	})
	if !backlogScanDocsContainPathTestV0(first.BacklogScanDocs, "modulos/orquesta-autoprogramming/docs/tareas.md") ||
		backlogScanDocsContainPathTestV0(first.BacklogScanDocs, "docs/legacy_orquestador_externo.md") {
		t.Fatalf("scan docs=%+v", first.BacklogScanDocs)
	}
	if !strings.Contains(strings.Join(first.ContextRefs, "\n"),
		"federated_backlog_source_not_executable:path=docs/legacy_orquestador_externo.md;state=quarantine") {
		t.Fatalf("context_refs=%+v", first.ContextRefs)
	}
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/legacy_orquestador_externo.md",
		"- LEG-001: legado cuarentenado cambiado sin entrar al epoch.\n")

	second := planner.withBacklogScannerMergeLeaseV0(orquestaserver.IdleSelfImprovementRequestV0{
		WriteSet: []string{"cmd/orquesta-server"},
	})
	if first.BacklogScanEpoch == "" || first.BacklogScanEpoch != second.BacklogScanEpoch {
		t.Fatalf("epochs first=%s second=%s", first.BacklogScanEpoch, second.BacklogScanEpoch)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0AcotaEpochAFuentesStaleHistoricasV0(t *testing.T) {
	projectDir := t.TempDir()
	content := "# Backlog\n\n## Indice federado de backlog local\n\n" +
		"- source_path: `modulos/orquesta-autoprogramming/docs/tareas.md`; " +
		"source_kind: module_tasks; owner: `modulos/orquesta-autoprogramming`; " +
		"estado: active; aliases: APG-*; tests: `go test -count=1 ./modulos/orquesta-autoprogramming`\n" +
		"- source_path: `docs/stale.md`; source_kind: legacy_docs; owner: `docs`; estado: stale; aliases: OLD-*\n" +
		"- source_path: `docs/historico.md`; source_kind: legacy_docs; owner: `docs`; estado: historico; aliases: OLD-*\n"
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md",
		"- APG-001: tarea ejecutable.\n")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/stale.md", "- OLD-001: stale inicial.\n")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/historico.md", "- OLD-002: historico inicial.\n")

	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	first := planner.withBacklogScannerMergeLeaseV0(orquestaserver.IdleSelfImprovementRequestV0{
		WriteSet: []string{"cmd/orquesta-server"},
	})
	if backlogScanDocsContainPathTestV0(first.BacklogScanDocs, "docs/stale.md") ||
		backlogScanDocsContainPathTestV0(first.BacklogScanDocs, "docs/historico.md") {
		t.Fatalf("scan docs=%+v", first.BacklogScanDocs)
	}
	context := strings.Join(first.ContextRefs, "\n")
	for _, want := range []string{
		"federated_backlog_source_not_executable:path=docs/stale.md;state=stale",
		"federated_backlog_source_not_executable:path=docs/historico.md;state=historico",
	} {
		if !strings.Contains(context, want) {
			t.Fatalf("missing %s in context_refs=%+v", want, first.ContextRefs)
		}
	}
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/stale.md", "- OLD-001: stale cambiado.\n")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "docs/historico.md", "- OLD-002: historico cambiado.\n")

	second := planner.withBacklogScannerMergeLeaseV0(orquestaserver.IdleSelfImprovementRequestV0{
		WriteSet: []string{"cmd/orquesta-server"},
	})
	if first.BacklogScanEpoch == "" || first.BacklogScanEpoch != second.BacklogScanEpoch {
		t.Fatalf("epochs first=%s second=%s", first.BacklogScanEpoch, second.BacklogScanEpoch)
	}
}

func TestIdleSelfImprovementFederatedBacklogV0IgnoraDirectoriosSinContratoEfectivoV0(t *testing.T) {
	content := "# Backlog\n\n## Indice federado de backlog local\n\n" +
		"- source_path: `modulos/orquesta-work-profiles`; source_kind: module_placeholder; " +
		"owner: `modulos/orquesta-work-profiles`; estado: vigente; aliases: WPR-*; " +
		"tests: `go test -count=1 ./modulos/orquesta-core-workflow`.\n"

	sources := idleSelfImprovementParseFederatedBacklogSourcesV0(content)
	if len(sources) != 0 {
		t.Fatalf("directorios sin contrato efectivo no deben ser fuentes federadas: %+v", sources)
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

func backlogScanDocsContainPathTestV0(docs []orquestaserver.BacklogScanDocumentV0, path string) bool {
	for _, doc := range docs {
		if doc.Path == path {
			return true
		}
	}
	return false
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
