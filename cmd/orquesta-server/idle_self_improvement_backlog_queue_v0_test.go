package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

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
	if len(result.Requests) != 2 || result.Requests[0].SuggestedArea != "t09-sanitizer" {
		t.Fatalf("requests=%+v", result.Requests)
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
	runRef := "request-ref-autoprogramming-backlog-t08-runtime-neutral-e2e-51b03c45-retry-deadbeef"
	ackDir := filepath.Join(projectDir, ".orquesta-runtime", runRef, "agent-ref-t08")
	if err := os.MkdirAll(ackDir, 0o700); err != nil {
		t.Fatalf("mkdir ack: %v", err)
	}
	writeBacklogPlannerCorrelatedACKForTestV0(t, ackDir, runRef, []string{"go test -count=1 ./cmd/orquesta-server"}, "")

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
	writeSet := result.Requests[0].WriteSet
	if len(writeSet) != 5 || writeSet[0] != "scripts" || writeSet[1] != "docs/runbooks" ||
		writeSet[2] != "cmd/orquesta-server" || writeSet[3] != "modulos/orquesta-server" ||
		writeSet[4] != "docs/rail_errors_observados_2026-05-23.md" {
		t.Fatalf("write_set=%+v", writeSet)
	}
}
