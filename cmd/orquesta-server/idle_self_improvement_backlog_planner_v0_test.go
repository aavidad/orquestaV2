package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0GeneraTareasPendientesV0(t *testing.T) {
	projectDir := t.TempDir()
	docsDir := filepath.Join(projectDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 server-autonomy

Objetivo: reforzar el servidor residente para autoprogramacion desatendida.

Alcance:

- ` + "`cmd/orquesta-server`" + `
- ` + "`modulos/orquesta-server`" + `

Criterios:

- Revisar backlog y generar tareas concretas.
- Tests: ` + "`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`" + `

## T05 web-autoprogramming

Objetivo: mejorar la web.

Estado: completada para cliente/proyeccion fina.

## T07 opes-consumer-smoke

Objetivo: cerrar el smoke OPES aislado pendiente como consumidor.

Alcance:

- scripts
- docs/runbooks

Criterios:

- No tocar OPES productivo.
- Tests: ` + "`go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	result, err := planner.PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 5,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef:         "request-ref-idle-base",
			CorrelationID:      "corr-idle-base",
			ProjectRef:         "project-ref-orquesta",
			WorktreeRef:        "worktree-ref-orquesta",
			BranchRef:          "branch-ref-orquesta",
			RequestedBy:        "test",
			Source:             "orquesta-server",
			FailureKind:        "idle_capacity",
			FailureSummary:     "cola sin ejecuciones",
			SuggestedArea:      "automejora",
			WriteSet:           []string{"modulos/orquesta-server"},
			RequiredTests:      []string{"go test -count=1 ./..."},
			AcceptanceCriteria: []string{"base"},
			CompactRules:       []string{"comunicacion compacta"},
			ContextRefs:        []string{"trigger:test"},
			EvidenceRefs:       []string{"evidence-ref-base"},
			OccurredAt:         "2026-05-23T10:00:00Z",
			PriorityScore:      10,
		},
	})
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	first := result.Requests[0]
	if first.RequestRef == "request-ref-idle-base" ||
		first.CorrelationID != "corr-"+first.RequestRef ||
		first.FailureKind != "backlog_autoprogramming" ||
		first.SuggestedArea != "t01-server-autonomy" ||
		len(first.WriteSet) != 2 ||
		first.WriteSet[0] != "cmd/orquesta-server" ||
		first.WriteSet[1] != "modulos/orquesta-server" ||
		first.RequiredTests[0] != "go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server" ||
		!containsStringForTestV0(first.ContextRefs, "backlog-doc-autoprogramacion-2026-05-23") ||
		!containsStringForTestV0(first.ContextRefs, "backlog_section:t01-server-autonomy") {
		t.Fatalf("first=%+v", first)
	}
	if result.Requests[1].SuggestedArea != "t07-opes-consumer-smoke" {
		t.Fatalf("second=%+v", result.Requests[1])
	}
}

func TestIdleSelfImprovementBacklogPlannerV0RespetaMaxRequestsV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T01 a\n\nObjetivo: uno.\n\n## T02 b\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
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
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t01-a" {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0FallbackIlegibleEsScannerDocumentalV0(t *testing.T) {
	projectDir := t.TempDir()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			Trigger:     "capacity_free",
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet: []string{
					"cmd/orquesta-server",
					"modulos/orquesta-server",
				},
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
	if request.FailureKind != "backlog_scan" ||
		request.SuggestedArea != "backlog-scan" ||
		!containsStringForTestV0(request.ContextRefs, "backlog_planner_fallback:autoprogramming_backlog_doc_unavailable") ||
		!containsStringForTestV0(request.EvidenceRefs, "evidence-ref-autoprogramming-backlog-planner-fallback") {
		t.Fatalf("request=%+v", request)
	}
	if containsStringForTestV0(request.WriteSet, "cmd/orquesta-server") ||
		containsStringForTestV0(request.WriteSet, "modulos/orquesta-server") ||
		!containsStringForTestV0(request.WriteSet, idleSelfImprovementBacklogDocRelV0) {
		t.Fatalf("write_set=%+v", request.WriteSet)
	}
	if request.BacklogScanEpoch == "" || len(request.BacklogScanDocs) == 0 {
		t.Fatalf("scan=%s docs=%+v", request.BacklogScanEpoch, request.BacklogScanDocs)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0FallbackVacioNoAbreCodigoGenericoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte("# Backlog\n\nSin secciones Txx.\n"), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
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
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.FailureKind != "backlog_scan" ||
		containsStringForTestV0(request.WriteSet, "cmd/orquesta-server") ||
		!containsStringForTestV0(request.ContextRefs, "backlog_planner_fallback:backlog_sin_tareas_pendientes_detectadas") {
		t.Fatalf("request=%+v", request)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0PriorizaNucleoDirectorAutomejoraAntesQueSecundariasV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 web-polish

Objetivo: mejorar la experiencia web.

Alcance:

- ` + "`modulos/orquesta-web`" + `

## T02 opes-consumer-smoke

Objetivo: cerrar smoke OPES temporal.

Alcance:

- ` + "`modulos/orquesta-opes-bridge`" + `

## T03 mcp-claude-connector

Objetivo: crear conector de agente Claude reutilizando adaptadores existentes.

Alcance:

- ` + "`modulos/orquesta-runtime`" + `
- ` + "`modulos/orquesta-mcp`" + `

## T04 director-causal-close

Objetivo: reforzar el Director Operativo y cierre causal.

Alcance:

- ` + "`modulos/orquesta-director-operativo`" + `

## T05 automejora-backlog

Objetivo: mejorar automejora del servidor residente.

Alcance:

- ` + "`cmd/orquesta-server`" + `

## T06 core-workflow-replay

Objetivo: endurecer nucleo de workflow.

Alcance:

- ` + "`modulos/orquesta-core-workflow`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 5,
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
	got := make([]string, 0, len(result.Requests))
	for _, request := range result.Requests {
		got = append(got, request.SuggestedArea)
	}
	want := []string{
		"t04-director-causal-close",
		"t05-automejora-backlog",
		"t06-core-workflow-replay",
		"t03-mcp-claude-connector",
		"t01-web-polish",
	}
	if len(got) != len(want) {
		t.Fatalf("areas=%+v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("areas=%+v want=%+v", got, want)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0DemoraMejorasFuturasExperimentalesV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## T01 rotacion-sesiones-director

Objetivo: mejora futura experimental para rotacion de sesiones del director y agentes.

Alcance:

- ` + "`modulos/orquesta-director-runner`" + `

## T02 web-followup

Objetivo: completar vista web.

Alcance:

- ` + "`modulos/orquesta-web`" + `

## T03 core-gap

Objetivo: cerrar hueco del nucleo.

Alcance:

- ` + "`modulos/orquesta-core-workflow`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 3,
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
	got := make([]string, 0, len(result.Requests))
	for _, request := range result.Requests {
		got = append(got, request.SuggestedArea)
	}
	want := []string{"t03-core-gap", "t02-web-followup", "t01-rotacion-sesiones-director"}
	if len(got) != len(want) {
		t.Fatalf("areas=%+v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("areas=%+v want=%+v", got, want)
		}
	}
}
