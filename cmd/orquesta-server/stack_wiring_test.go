package main

import (
	"context"
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestBuildStackFromEnvV0UsaConectoresDurablesFileBased(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}

	assertServerStackDurableStoresV0(t, stack)
}

func TestBuildStackFromEnvV0CableaOperationalClosureSourceV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.Ports.OperationalClosureSource == nil {
		t.Fatalf("OperationalClosureSource debe quedar cableado en el stack servidor")
	}
}

func TestBuildStackFromEnvV0DirectorStatsExponeProgressSourceConfiguradoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	runRef := "run-ref-server-progress-source-configured-001"
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-server-progress-source-configured-001",
		AppSpecRef:    "app-spec-ref-server-progress-source-configured-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		}},
		Tasks:         []string{"task-ref-server-progress-source-configured-001"},
		Agents:        []string{"agent-ref-server-progress-source-configured-001"},
		StartedAgents: []string{"agent-ref-server-progress-source-configured-001"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	result, err := stack.MCPTransportBindings.DirectorStats.Execute(
		context.Background(),
		orquestamcp.MCPDirectorStatsToolInputV0{
			RunRef:               runRef,
			IncludeAgentProgress: true,
		},
	)
	if err != nil {
		t.Fatalf("DirectorStats.Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 || result.Stats == nil {
		t.Fatalf("result=%+v", result)
	}
	if result.Stats.Progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceLoadedV0 {
		t.Fatalf("source_status=%s issues=%+v", result.Stats.Progress.SourceStatus, result.Stats.Progress.Issues)
	}
}

func TestBuildStackFromEnvV0CableaDomainWorkFileOptIn(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork file no cableado")
	}
	if !stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery debe activarse con domain-work-file porque ya tiene submitter local")
	}
	result, err := stack.DomainWork.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "request-ref-stack-file-001",
			CorrelationID:  "corr-stack-file-001",
			IdempotencyKey: "idem-stack-file-001",
			RequestedBy:    "test",
			DomainRef:      "dominio-demo",
			WorkKind:       "generate_content_package",
			Objective:      "crear job generico desde stack",
		},
	})
	if err != nil {
		t.Fatalf("DomainWork.Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef == "" {
		t.Fatalf("result=%+v", result)
	}
	submitted, err := stack.DomainWork.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "request-ref-stack-file-artifact-001",
			CorrelationID:  "corr-stack-file-001",
			IdempotencyKey: "idem-stack-file-artifact-001",
			RequestedBy:    "test",
			DomainRef:      "dominio-demo",
			JobRef:         result.Job.JobRef,
			ArtifactRef:    "artifact-ref-stack-file-001",
			ArtifactType:   "content_package",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "body",
				Value: "contenido",
			}},
			CompleteJob: true,
		},
	})
	if err != nil {
		t.Fatalf("DomainWork.Execute submit: %v", err)
	}
	if submitted.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		submitted.Receipt == nil ||
		submitted.Receipt.ReceiptRef == "" {
		t.Fatalf("submitted=%+v", submitted)
	}
}

func TestBuildStackFromEnvV0CableaOPESFallbackParaDomainWorkYDeliveryV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_OPES_TEMPORAL_CONFIRM", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork OPES fallback no cableado")
	}
	if !stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery debe activarse con OPES_BASE_URL fallback")
	}
}

func TestBuildStackFromEnvV0CableaHTTPNeutralParaDomainWorkYDeliveryV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "http://127.0.0.1:18083")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "smoke_local")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork HTTP neutral no cableado")
	}
	if !stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery debe activarse con HTTP neutral opt-in")
	}
}

func TestBuildStackFromEnvV0NoCableaRunnerLocalPorDefectoPeroAceptaReceiptsACK(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", "")
	t.Setenv("ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS", "")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	localRunner, err := requiredTestRunnerFromEnvV0(config, stack.Stores.RequiredTestEvidenceStore)
	if err != nil {
		t.Fatalf("requiredTestRunnerFromEnvV0: %v", err)
	}
	if localRunner != nil {
		t.Fatalf("runner local de comandos debe quedar apagado por defecto")
	}
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("runner de receipts ACK debe quedar disponible por defecto")
	}
}

func TestBuildStackFromEnvV0CableaRequiredTestRunnerOptIn(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", filepath.Join(projectDir, "go-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", filepath.Join(t.TempDir(), "test-output"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_ENV", "")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("RequiredTestRunner opt-in no cableado")
	}
}

func assertServerStackDurableStoresV0(
	t *testing.T,
	stack orquestaappcodexstack.StackV0,
) {
	t.Helper()
	assertTypeV0[*orquestastatefile.StoreV0](t, "RunStore", stack.Stores.RunStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "EventSink", stack.Stores.EventSink)
	assertTypeV0[*orquestastatefile.StoreV0](t, "TaskStore", stack.Stores.TaskStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "WaitStateStore", stack.Stores.WaitStateStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "OperationalPlanStateWriter", stack.Stores.OperationalPlanStateWriter)
	assertTypeV0[*orquestastatefile.StoreV0](t, "OperationalPlanStateStore", stack.Stores.OperationalPlanStateStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "ProcessRegistry", stack.Stores.ProcessRegistry)
	assertTypeV0[*orquestapersistence.FileOutboxLedgerV0](t, "OutboxLedger", stack.Stores.OutboxLedger)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "AppChangeStore", stack.Stores.AppChangeStore)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunControl", stack.Stores.RunControl)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunQueue", stack.Stores.RunQueue)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexReceiptDescriptorStoreV0](t, "ReceiptStore", stack.Stores.ReceiptStore)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexProgressStateStoreV0](t, "ProgressState", stack.Stores.ProgressState)
}

func assertTypeV0[T any](t *testing.T, name string, value any) {
	t.Helper()
	if _, ok := value.(T); !ok {
		t.Fatalf("%s usa %T, debe usar %T", name, value, *new(T))
	}
}
