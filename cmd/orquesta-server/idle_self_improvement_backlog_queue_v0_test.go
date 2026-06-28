package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0SaltaTareasYaEnColaSinAnadirScannerSiHayOwnerEjecutableV0(t *testing.T) {
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
				backlogRequestRefForTestV0(content, 0),
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
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t09-sanitizer" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	for _, request := range result.Requests {
		if request.FailureKind == "backlog_scan" {
			t.Fatalf("scanner repetido como relleno=%+v", result.Requests)
		}
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
			Trigger:      "idle_self_improvement",
			QueueSize:    2,
			FreeCapacity: 2,
			KnownRequestRefs: []string{
				backlogRequestRefForTestV0(content, 0),
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

func TestIdleSelfImprovementBacklogPlannerV0AnadeScannerSiTodoEstaEnColaPeroHayCapacidadV0(t *testing.T) {
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
			QueueSize:    1,
			FreeCapacity: 2,
			KnownRequestRefs: []string{
				backlogRequestRefForTestV0(content, 0),
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
	if len(result.Requests) != 1 || result.Requests[0].FailureKind != "backlog_scan" {
		t.Fatalf("result=%+v", result)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoRepiteRetryScannerCerradoSiHayOwnerEjecutableV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteFederatedBacklogIndexTestV0(t, projectDir, "vigente",
		"go test -count=1 ./modulos/orquesta-autoprogramming")
	mustWriteFederatedLocalDocTestV0(t, projectDir, "modulos/orquesta-autoprogramming/docs/tareas.md",
		"- APG-001: tarea ejecutable pendiente tras scanner no-op cerrado.\n")
	base := orquestaserver.IdleSelfImprovementRequestV0{
		RequestRef: "request-ref-base",
		ProjectRef: "project-ref-orquesta",
	}
	scannerRef := idleSelfImprovementBacklogScannerRequestV0(base, orquestaserver.IdleSelfImprovementPlanRequestV0{
		Trigger: "capacity_free",
	}).RequestRef
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: scannerRef + "-retry-deadbeef",
		Status: orquestarunqueue.RunStatusClosedV0,
	}}}
	supervisor := serverStackSupervisorV0{
		stack: &orquestaappcodexstack.StackV0{
			Stores:   orquestaappcodexstack.StoresV0{RunQueue: queue},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		projectWorkDir: projectDir,
	}

	result, err := supervisor.PlanIdleSelfImprovementV0(context.Background(), orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:  2,
		Trigger:      "capacity_free",
		QueueSize:    0,
		FreeCapacity: 2,
		BaseRequest:  base,
	})
	if err != nil {
		t.Fatalf("PlanIdleSelfImprovementV0: %v", err)
	}
	if len(result.Requests) != 1 ||
		result.Requests[0].FailureKind != "backlog_autoprogramming" ||
		result.Requests[0].SuggestedArea != "apg-001" ||
		!queue.lastRead.IncludeNonExecutable {
		t.Fatalf("result=%+v read=%+v", result, queue.lastRead)
	}
	for _, request := range result.Requests {
		if request.FailureKind == "backlog_scan" || request.RequestRef == scannerRef {
			t.Fatalf("scanner repetido=%+v result=%+v", request, result)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoExcluyeRunStaleRetryableV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T29 provider-usage-quota-accounting\n\nObjetivo: arreglar proveedor.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	requestRef := backlogRequestRefForTestV0(content, 0)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(context.Background(), orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         requestRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-autoprogramming-t29",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-autoprogramming-t29-g01"},
		FailedAgents:  []string{"agent-ref-task-autoprogramming-t29-g01"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{RunStore: runStore},
		},
		projectWorkDir: projectDir,
	}

	result, err := supervisor.PlanIdleSelfImprovementV0(context.Background(), orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:      1,
		KnownRunRefs:     []string{requestRef},
		KnownRequestRefs: []string{requestRef},
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef: "request-ref-base",
			ProjectRef: "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanIdleSelfImprovementV0: %v", err)
	}
	if len(result.Requests) != 1 || result.Requests[0].RequestRef != requestRef {
		t.Fatalf("result=%+v request_ref=%s", result, requestRef)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EstadoCubiertoNoRelanzaT29V0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `## T29 provider-usage-quota-accounting

Objetivo: convertir uso real de proveedor y cuota en puerto/adaptador explicito,
observable y redactado, sin meter proveedor/modelo/coste en el nucleo.

Estado: cubierto 2026-05-24 r2 para orquesta-runtime-worktree,
orquesta-runtime-codex, orquesta-app-codex-stack y cmd/orquesta-server.

Alcance:

- modulos/orquesta-orchestration-core
- cmd/orquesta-server

## T30 siguiente-pendiente

Objetivo: tarea pendiente real.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef: "request-ref-base",
				ProjectRef: "project-ref-orquesta",
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 ||
		result.Requests[0].SuggestedArea != "t30-siguiente-pendiente" ||
		strings.Contains(result.Requests[0].RequestRef, "t29-provider-usage-quota-accounting") {
		t.Fatalf("result=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0ExcluyeRunCerradoAunqueNoSeaVisibleV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T45 autoprogramming-go-file-line-budget-baseline\n\nObjetivo: uno.\n\n## T47 external-agent-required-test-receipts\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	closedRef := backlogRequestRefForTestV0(content, 0)
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: closedRef,
		Status: orquestarunqueue.RunStatusClosedV0,
	}}}
	supervisor := serverStackSupervisorV0{
		stack: &orquestaappcodexstack.StackV0{
			Stores:   orquestaappcodexstack.StoresV0{RunQueue: queue},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		projectWorkDir: projectDir,
	}

	result, err := supervisor.PlanIdleSelfImprovementV0(context.Background(), orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef: "request-ref-base",
			ProjectRef: "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanIdleSelfImprovementV0: %v", err)
	}
	if len(result.Requests) != 1 || result.Requests[0].RequestRef == closedRef ||
		result.Requests[0].SuggestedArea != "t47-external-agent-required-test-receipts" ||
		!queue.lastRead.IncludeNonExecutable {
		t.Fatalf("result=%+v closed_ref=%s read=%+v", result, closedRef, queue.lastRead)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoExcluyeRunConPendienteFantasmaSinRuntimeV0(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T29 provider-usage-quota-accounting\n\nObjetivo: arreglar proveedor.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	requestRef := backlogRequestRefForTestV0(content, 0)
	agentRef := "agent-ref-task-autoprogramming-t29-g01"
	ghostAgentRef := "agent-ref-task-autoprogramming-t29-g01-ghost"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(context.Background(), orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         requestRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-autoprogramming-t29",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-autoprogramming-t29-g01"},
		StartedAgents: []string{agentRef, ghostAgentRef},
		StoppedAgents: []string{agentRef},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-autoprogramming-t29-g01",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: agentRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		})},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stack: &orquestaappcodexstack.StackV0{
			Stores:              orquestaappcodexstack.StoresV0{RunStore: runStore},
			CodexRuntimeWorkDir: runtimeDir,
		},
		projectWorkDir: projectDir,
		runtimeWorkDir: runtimeDir,
	}

	result, err := supervisor.PlanIdleSelfImprovementV0(context.Background(), orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:      1,
		KnownRunRefs:     []string{requestRef},
		KnownRequestRefs: []string{requestRef},
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef: "request-ref-base",
			ProjectRef: "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanIdleSelfImprovementV0: %v", err)
	}
	if len(result.Requests) != 1 || result.Requests[0].RequestRef != requestRef {
		t.Fatalf("result=%+v request_ref=%s", result, requestRef)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0MantieneExclusionSiAgenteVivoPendienteV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T29 provider-usage-quota-accounting\n\nObjetivo: arreglar proveedor.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	requestRef := backlogRequestRefForTestV0(content, 0)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(context.Background(), orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         requestRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-autoprogramming-t29",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-autoprogramming-t29-g01"},
		StartedAgents: []string{"agent-ref-task-autoprogramming-t29-g01"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{RunStore: runStore},
		},
		projectWorkDir: projectDir,
	}

	result, err := supervisor.PlanIdleSelfImprovementV0(context.Background(), orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:      1,
		KnownRunRefs:     []string{requestRef},
		KnownRequestRefs: []string{requestRef},
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef: "request-ref-base",
			ProjectRef: "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanIdleSelfImprovementV0: %v", err)
	}
	if len(result.Requests) != 0 || result.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("result=%+v request_ref=%s", result, requestRef)
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
	runRef := backlogRequestRefForTestV0(content, 0) + "-retry-deadbeef"
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

func TestIdleSelfImprovementBacklogPlannerV0EstadoPendienteExplicitoMandaSobreACKRuntimeV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nEstado: pendiente. Falta evidencia real vigente.\n\nObjetivo: uno.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	runRef := backlogRequestRefForTestV0(content, 0) + "-retry-deadbeef"
	ackDir := filepath.Join(projectDir, ".orquesta-runtime", runRef, "agent-ref-t08")
	if err := os.MkdirAll(ackDir, 0o700); err != nil {
		t.Fatalf("mkdir ack: %v", err)
	}
	writeBacklogPlannerCorrelatedACKForTestV0(t, ackDir, runRef, []string{"go test -count=1 ./cmd/orquesta-server"}, "")

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
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t08-runtime-neutral-e2e" {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EstadoPendienteExplicitoReintentaRunCerradaConRefNuevaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nEstado: pendiente. Falta cierre canonico.\n\nObjetivo: uno.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	baseRef := backlogRequestRefForTestV0(content, 0)
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:      1,
			KnownRequestRefs: []string{baseRef},
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef: "request-ref-base",
				ProjectRef: "project-ref-orquesta",
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 ||
		result.Requests[0].SuggestedArea != "t08-runtime-neutral-e2e" ||
		!strings.HasPrefix(result.Requests[0].RequestRef, baseRef+"-retry-") ||
		!containsStringForTestV0(result.Requests[0].ContextRefs, "pending_canonical_overrides_closed_attempt:true") {
		t.Fatalf("requests=%+v base_ref=%s", result.Requests, baseRef)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0EstadoPendienteExplicitoReconciliacionTrasRetryCerradoV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nEstado: pendiente. Falta cierre canonico.\n\nObjetivo: uno.\n\nAlcance:\n\n- cmd/orquesta-server\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	baseRef := backlogRequestRefForTestV0(content, 0)
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			KnownRequestRefs: []string{
				baseRef,
				baseRef + "-retry-deadbeef",
			},
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
	if len(result.Requests) != 1 ||
		result.Requests[0].FailureKind != "backlog_documentation_review" ||
		result.Requests[0].SuggestedArea != "t08-runtime-neutral-e2e-reconciliacion" ||
		!strings.HasPrefix(result.Requests[0].RequestRef, baseRef+"-reconcile-") ||
		!containsStringForTestV0(result.Requests[0].ContextRefs, "pending_canonical_after_multiple_attempts:true") ||
		!containsStringForTestV0(result.Requests[0].WriteSet, idleSelfImprovementBacklogDocRelV0) {
		t.Fatalf("requests=%+v base_ref=%s", result.Requests, baseRef)
	}
	for _, criterion := range result.Requests[0].AcceptanceCriteria {
		if strings.Contains(criterion, "no relanzar otra implementacion padre") {
			return
		}
	}
	t.Fatalf("criterio anti-bucle ausente: %+v", result.Requests[0].AcceptanceCriteria)
}

func TestIdleSelfImprovementBacklogPlannerV0NoRelanzaReconciliacionCerradaV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nEstado: pendiente. Falta cierre canonico.\n\nObjetivo: uno.\n\nAlcance:\n\n- cmd/orquesta-server\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	baseRef := backlogRequestRefForTestV0(content, 0)
	reconcileRef := baseRef + "-reconcile-deadbeef"
	ackDir := filepath.Join(projectDir, ".orquesta-runtime", reconcileRef, "agent-ref-t08")
	if err := os.MkdirAll(ackDir, 0o700); err != nil {
		t.Fatalf("mkdir ack: %v", err)
	}
	writeBacklogPlannerCorrelatedACKForTestV0(t, ackDir, reconcileRef, []string{"go test -count=1 ./cmd/orquesta-server"}, "")

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 1,
			KnownRequestRefs: []string{
				baseRef,
				baseRef + "-retry-deadbeef",
				reconcileRef,
			},
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
	if len(result.Requests) != 1 || result.Requests[0].SuggestedArea != "t09-sanitizer" {
		t.Fatalf("debe saltar T08 reconciliada y avanzar: %+v", result.Requests)
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
