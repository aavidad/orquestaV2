package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestServerConfigProjectionSettingsForMCPV0ProyectaEffectiveConfigV0(t *testing.T) {
	got := serverConfigProjectionSettingsForMCPV0(orquestaserver.ConfigV0{
		EffectiveConfig: orquestaserver.ServerEffectiveConfigV0{
			Settings: []orquestaserver.ServerConfigSettingV0{{
				Key:   "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS",
				Value: "450000",
			}, {
				Key:       "ORQUESTA_CODEX_CODE_HOME",
				Value:     "codex-code-home-configured",
				Sensitive: true,
			}},
		},
	})

	if len(got) != 2 ||
		got[0].Key != "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS" ||
		got[0].Value != "450000" ||
		got[1].Key != "ORQUESTA_CODEX_CODE_HOME" ||
		got[1].Value != "codex-code-home-configured" ||
		!got[1].Sensitive {
		t.Fatalf("projection=%+v", got)
	}
}

func TestServerRuntimeDepsFromStackV0CableaGoalRequiredTestSpecBinderV0(t *testing.T) {
	binder := serverGoalRequiredTestSpecBinderForTestV0{}
	deps := serverRuntimeDepsFromStackV0(
		orquestaserver.ConfigV0{},
		nil,
		nil,
		nil,
		orquestaappcodexstack.StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalRequiredTestSpecBinder: binder,
		}},
		serverCodexGoalBackendsV0{},
	)
	if deps.GoalRequiredTestSpecBinder == nil {
		t.Fatal("GoalRequiredTestSpecBinder no llega al runtime residente")
	}
}

type serverGoalRequiredTestSpecBinderForTestV0 struct{}

func (serverGoalRequiredTestSpecBinderForTestV0) BindGoalRequiredTestSpecV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	spec.ImplementerAgentRef = "agent-ref-server-wiring-test"
	spec.ImplementerCredentialRef = "credential-ref-server-wiring-test"
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = "trust-policy-ref-server-wiring-test"
	return spec, nil
}

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
	deps := serverRuntimeDepsFromStackV0(
		config,
		nil,
		nil,
		nil,
		stack,
		serverCodexGoalBackendsV0{},
	)
	if deps.MaterialProgressStore == nil {
		t.Fatal("MaterialProgressStore debe reutilizar el state store durable")
	}
	if deps.MaterialProgressEvidence == nil {
		t.Fatal("MaterialProgressEvidence debe quedar cableado en la composicion Codex")
	}
	if stack.MCPTransportBindings.AutoprogrammingMaterialProgressStateReader == nil {
		t.Fatal("MCP debe proyectar el mismo estado durable de progreso material")
	}
	runControl, ok := stack.MCPTransportBindings.RunControl.(orquestamcp.MCPRunControlToolExecutorV0)
	if !ok || runControl.MaterialProgressStateReader == nil {
		t.Fatal("runs/control debe usar la decision durable de progreso material")
	}
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

func TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	if err := writeServerCodeContextFixtureV0(projectDir); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")

	backend := &serverCodeContextGoalStarterForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter: backend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	if stack.CodeContext == nil ||
		stack.MCPTransportBindings.CodebaseQuery == nil {
		t.Fatalf("broker de contexto no cableado: stack.CodeContext=%T binding=%T", stack.CodeContext, stack.MCPTransportBindings.CodebaseQuery)
	}
	if stack.Ports.AutonomousDirectorPolicy == nil {
		t.Fatalf("politica autonoma no cableada en stack desde env")
	}

	mcpResult, err := stack.MCPTransportBindings.CodebaseQuery.Execute(
		context.Background(),
		serverCodeContextQueryForTestV0("request-ref-codebase-wiring-mcp"),
	)
	if err != nil {
		t.Fatalf("CodebaseQuery.Execute: %v", err)
	}
	assertServerCodeContextResultV0(t, mcpResult)

	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	httpResult := postServerCodeContextQueryForTestV0(t, handler, "request-ref-codebase-wiring-http")
	assertServerCodeContextResultV0(t, httpResult)

	receipt, err := stack.Ports.GoalLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-codebase-wiring-001",
		RunRef:       "run-ref-codebase-wiring-001",
		ProjectRef:   "repo-ref-codebase-wiring",
		Objective:    "Modificar el broker de contexto desde el stack real.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "cmd/demo",
			Purpose: "codigo de prueba del broker",
		}},
		RuleRefs: []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.ContextBudget.CodeContextCacheStatus == "" {
		t.Fatalf("receipt sin cache_status de broker: %+v", receipt.ContextBudget)
	}
	if backend.calls != 1 ||
		!serverGoalContextRefPrefixForTestV0(backend.packet.ContextRefs, "code_context_prepared:repo_map:") {
		t.Fatalf("goal sin repo_map precargado: calls=%d packet=%+v", backend.calls, backend.packet)
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
	assertTypeV0[*orquestastatefile.StoreV0](t, "GoalRequiredTestAttestationStore", stack.Stores.GoalRequiredTestAttestationStore)
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

type serverCodeContextGoalStarterForTestV0 struct {
	calls  int
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0
}

func (starter *serverCodeContextGoalStarterForTestV0) StartCodexGoalV0(
	_ context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	starter.calls++
	starter.packet = packet
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: "thread-ref-codebase-wiring-001",
		EvidenceRefs:    []string{"evidence-ref-codebase-wiring-launch"},
	}, nil
}

func writeServerCodeContextFixtureV0(root string) error {
	path := filepath.Join(root, "cmd", "demo", "service.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(`package demo

func CodebaseWiringDemoV0() string {
	return "codebase wiring demo"
}
`), 0o644)
}

func serverCodeContextQueryForTestV0(requestRef string) orquestamcp.MCPCodebaseQueryToolInputV0 {
	return orquestamcp.MCPCodebaseQueryToolInputV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:    requestRef,
		RepositoryRef: "repo-ref-codebase-wiring",
		QueryKind:     orquestacontext.CodeContextQueryKindRepoMapV0,
		Query:         "CodebaseWiringDemoV0",
		Scope:         []string{"cmd/demo"},
		MaxResults:    4,
		MaxBytes:      4000,
		RequestedBy:   "cmd-orquesta-server-stack-wiring-test",
	}
}

func postServerCodeContextQueryForTestV0(
	t *testing.T,
	handler http.Handler,
	requestRef string,
) orquestamcp.MCPCodebaseQueryToolResultV0 {
	t.Helper()
	body, err := json.Marshal(serverCodeContextQueryForTestV0(requestRef))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPCodebaseQueryHTTPPathV0, bytes.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST %s status=%d body=%s", orquestamcp.MCPCodebaseQueryHTTPPathV0, rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPCodebaseQueryToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
}

func assertServerCodeContextResultV0(t *testing.T, result orquestamcp.MCPCodebaseQueryToolResultV0) {
	t.Helper()
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 ||
		result.ProviderKind != orquestacontext.CodeContextProviderKindFallbackRGV0 ||
		result.IndexerPolicy != orquestacontext.CodeContextProviderPolicyCentralOnlyV0 ||
		len(result.Results) == 0 {
		t.Fatalf("code context result inesperado: %+v", result)
	}
	found := false
	for _, hit := range result.Results {
		if strings.Contains(hit.Path, "cmd/demo/service.go") ||
			strings.Contains(hit.Symbol, "CodebaseWiringDemoV0") ||
			strings.Contains(hit.Snippet, "CodebaseWiringDemoV0") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("resultado no incluye fixture cmd/demo/service.go: %+v", result.Results)
	}
}

func serverGoalContextRefPrefixForTestV0(refs []orquestagoal.GoalContextRefV0, prefix string) bool {
	for _, ref := range refs {
		if strings.HasPrefix(strings.TrimSpace(ref.Ref), prefix) {
			return true
		}
	}
	return false
}
