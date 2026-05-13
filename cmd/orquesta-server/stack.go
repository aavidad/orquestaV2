package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	orquestastatefileoutbox "orquesta/modulos/orquesta-state-file/outbox"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func buildRuntimeFromEnvV0() (*orquestaserver.RuntimeV0, error) {
	serverConfig, err := serverConfigFromEnvV0()
	if err != nil {
		return nil, err
	}
	stack, err := buildStackFromEnvV0(serverConfig)
	if err != nil {
		return nil, err
	}
	return orquestaserver.NewRuntimeV0(serverConfig, orquestaserver.RuntimeDepsV0{
		AppHandler: stack.Handler,
		Supervisor: stack,
	})
}

func buildStackFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
) (orquestaappcodexstack.StackV0, error) {
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	receiptStore, err := orquestaruntimecodexdelivery.NewFileCodexReceiptDescriptorStoreV0(serverConfig.StateDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	progressStore, err := orquestaruntimecodexdelivery.NewFileCodexProgressStateStoreV0(serverConfig.StateDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
		RootDir: filepath.Join(serverConfig.StateDir, "orchestration-state"),
	})
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	outboxLedger, err := orquestastatefileoutbox.NewFileOutboxLedgerV0(
		filepath.Join(serverConfig.StateDir, "outbox-state"),
	)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	runFileStore, err := orquestarunfile.NewRunFileStoreV0(
		filepath.Join(serverConfig.StateDir, "run-state"),
	)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	return orquestaappcodexstack.BuildStackV0(orquestaappcodexstack.ConfigV0{
		Enabled:        true,
		Timeout:        30 * time.Second,
		DirectorLimits: directorLimitsV0(),
		Stores: orquestaappcodexstack.StoresV0{
			RunStore:        stateStore,
			EventSink:       stateStore,
			OutboxLedger:    outboxLedger,
			TaskStore:       stateStore,
			AppChangeStore:  runFileStore,
			ReceiptStore:    receiptStore,
			ProgressState:   progressStore,
			ProcessRegistry: stateStore,
			RunControl:      runFileStore,
			RunQueue:        runFileStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{
			QueueRef:       "global",
			MaxRunsPerTick: intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", 2),
			QueueLimit:     intEnvOrDefaultV0("ORQUESTA_SERVER_QUEUE_LIMIT", 20),
			DefaultPriorityScore: intEnvOrDefaultV0(
				"ORQUESTA_SERVER_DEFAULT_PRIORITY", 50,
			),
		},
		RunSupervisor: orquestaappcodexstack.RunSupervisorConfigV0{
			MaxTicks:      1,
			MaxExecutions: intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", 2),
		},
		Codex: codexRuntimeConfigV0(serverConfig, processRuntime),
		Capacity: orquestaappcodexstack.CapacityConfigV0{
			Tier:            orquestacoreworkflow.OrchestrationCapacityXHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityXHighV0,
			OccurredAt:      time.Now().UTC().Format(time.RFC3339),
			RequestedBy:     "orquesta-server",
			Summary:         "Capacidad inicial del servidor residente.",
			EvidenceRefs:    []string{"evidence-ref-orquesta-server"},
		},
		ReviewGate: orquestaappcodexstack.ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
		DomainWork: domainWorkExecutorFromEnvV0(),
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "",
		},
	})
}

func directorLimitsV0() orquestaweb.WebArrancarDirectorAppLimitsV0 {
	return orquestaweb.WebArrancarDirectorAppLimitsV0{
		MaxBursts:            intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_BURSTS", 16),
		MaxStepsPerBurst:     intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_STEPS", 12),
		MaxDispatchesPerWait: intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_DISPATCHES", 8),
		MaxCommands:          intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_COMMANDS", 20),
		MaxOutboxPerCycle:    intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_OUTBOX", 8),
		MaxExternalWaits:     intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS", 2),
	}
}

func codexRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaappcodexstack.CodexRuntimeConfigV0 {
	return orquestaappcodexstack.CodexRuntimeConfigV0{
		CommandPath:     codexCommandPathV0(),
		ProjectWorkDir:  serverConfig.ProjectWorkDir,
		RuntimeWorkDir:  serverConfig.RuntimeWorkDir,
		CodeHomeDir:     codeHomeDirV0(),
		HomeDir:         homeDirV0(),
		PathEnv:         envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH")),
		Model:           envOrDefaultV0("ORQUESTA_CODEX_MODEL", "gpt-5.5"),
		ReasoningEffort: envOrDefaultV0("ORQUESTA_CODEX_REASONING_EFFORT", "xhigh"),
		Profile:         strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROFILE")),
		Sandbox:         envOrDefaultV0("ORQUESTA_CODEX_SANDBOX", "workspace-write"),
		ApprovalPolicy:  envOrDefaultV0("ORQUESTA_CODEX_APPROVAL_POLICY", "never"),
		ExtraArgs:       strings.Fields(os.Getenv("ORQUESTA_CODEX_EXTRA_ARGS")),
		Runtime:         processRuntime,
		ProcessStopper:  processRuntime,
		SnapshotSource:  processRuntime,
		MaxBatchReady:   intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_BATCH_READY", 4),
		MaxConcurrency:  intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_CONCURRENCY", 4),
		WaitInterval:    time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_WAIT_INTERVAL_MS", 2000)) * time.Millisecond,
		ProgressPolicy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: intEnvOrDefaultV0("ORQUESTA_CODEX_STALLED_TICKS", 8),
			LoopAfterRepeatedActions:    intEnvOrDefaultV0("ORQUESTA_CODEX_LOOP_TICKS", 12),
		},
		ProgressBudget: orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0{
			MaxExpected:     time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", 1200)) * time.Second,
			NoActivityLimit: time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", 90)) * time.Second,
		},
	}
}

func codexCommandPathV0() string {
	raw := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_COMMAND"))
	if raw == "" {
		raw = "codex"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		return raw
	}
	return path
}

func codeHomeDirV0() string {
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_CODE_HOME"))
	if value != "" {
		return value
	}
	return filepath.Join(homeDirV0(), ".codex")
}

func homeDirV0() string {
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_HOME"))
	if value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func domainWorkExecutorFromEnvV0() orquestamcp.MCPDomainWorkExecutorPortV0 {
	baseURL := strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL"))
	if baseURL == "" {
		return nil
	}
	client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(intEnvOrDefaultV0("ORQUESTA_OPES_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		DefaultMaxAttempts: intEnvOrDefaultV0("ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS", 1),
	})
	return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client)
}

func validateCodexCommandAvailableV0() error {
	if !filepath.IsAbs(codexCommandPathV0()) {
		return fmt.Errorf("codex_command_unavailable")
	}
	return nil
}
