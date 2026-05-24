package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
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

func TestIdleSelfImprovementBacklogPlannerV0SaltaTareasYaEnColaYAnadeScannerV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nObjetivo: uno.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:  3,
			Trigger:      "capacity_free",
			QueueSize:    2,
			FreeCapacity: 3,
			KnownRequestRefs: []string{
				"request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-51b03c45",
			},
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
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	if result.Requests[0].SuggestedArea != "t09-sanitizer" {
		t.Fatalf("first=%+v", result.Requests[0])
	}
	scanner := result.Requests[1]
	if scanner.SuggestedArea != "backlog-scan" ||
		scanner.FailureKind != "backlog_scan" ||
		!containsStringForTestV0(scanner.WriteSet, idleSelfImprovementBacklogDocRelV0) ||
		!containsStringForTestV0(scanner.ContextRefs, "trigger:capacity_free") {
		t.Fatalf("scanner=%+v", scanner)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoInventaFallbackSiTodoEstaEnColaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nObjetivo: uno.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:  2,
			Trigger:      "capacity_free",
			QueueSize:    2,
			FreeCapacity: 2,
			KnownRequestRefs: []string{
				"request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-51b03c45",
				"request-ref-autoprogramming-backlog-scanner-d64fa5bb",
			},
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef: "request-ref-base",
				ProjectRef: "project-ref-orquesta",
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

func TestIdleSelfImprovementBacklogPlannerV0SaltaACKCompletadosDelRuntimeV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nObjetivo: uno.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	ackDir := filepath.Join(
		projectDir,
		".orquesta-runtime",
		"request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-51b03c45-retry-deadbeef",
		"agent-ref-task",
	)
	if err := os.MkdirAll(ackDir, 0o700); err != nil {
		t.Fatalf("mkdir ack: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ackDir, "agent_ack.json"), []byte(`{"status":"completed"}`), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 2,
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
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t09-sanitizer" {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NormalizaBackticksDeAlcanceV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T01 markdown-scope\n\nObjetivo: normalizar alcance.\n\nAlcance: `scripts` y `docs/runbooks`\n\n- Revisar `cmd/orquesta-server` y `modulos/orquesta-server`.\n- docs/rail_errors_observados_2026-05-23.md.\n"
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
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	writeSet := result.Requests[0].WriteSet
	if len(writeSet) != 5 ||
		writeSet[0] != "scripts" ||
		writeSet[1] != "docs/runbooks" ||
		writeSet[2] != "cmd/orquesta-server" ||
		writeSet[3] != "modulos/orquesta-server" ||
		writeSet[4] != "docs/rail_errors_observados_2026-05-23.md" {
		t.Fatalf("write_set=%+v", writeSet)
	}
}

func TestIdleSelfImprovementStackV0RechazaCandidatoNoEjecutableV0(t *testing.T) {
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: "run-ref-1", Status: orquestarunqueue.RunStatusPausedV0,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores:   orquestaappcodexstack.StoresV0{RunQueue: queue},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: "run-ref-1"},
	)
	if result.Accepted || result.Message != "idle_self_improvement_queue_candidate_not_executable" {
		t.Fatalf("result=%+v", result)
	}
}

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type fakeIdleSelfRunQueueV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
}

func (fake *fakeIdleSelfRunQueueV0) ListRunSchedulingCandidatesV0(
	context.Context,
	orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

func (fake *fakeIdleSelfRunQueueV0) SetRunPriorityV0(
	context.Context,
	orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	return orquestarunqueue.RunSchedulingCandidateV0{}, nil
}
