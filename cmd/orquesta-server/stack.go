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
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	orquestastatefileoutbox "orquesta/modulos/orquesta-state-file/outbox"
	orquestaweb "orquesta/modulos/orquesta-web"
)

const (
	defaultCodexWaitIntervalMSV0        = 2000
	defaultCodexStalledTicksV0          = 300
	defaultCodexLoopTicksV0             = 300
	defaultCodexMaxExpectedSecondsV0    = 1200
	defaultCodexNoActivitySecondsV0     = 600
	defaultCodexMaxBatchReadyV0         = 4
	defaultCodexMaxConcurrencyV0        = 4
	defaultCodexServerMaxRunsPerTickV0  = 2
	defaultCodexServerQueueLimitV0      = 20
	defaultCodexServerDefaultPriorityV0 = 50
	defaultCodexServerMaxExecutionsV0   = 2
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
		AppHandler:   stack.Handler,
		Supervisor:   stack,
		StartupCheck: startupCheckFromEnvV0(stack, serverConfig),
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
	domainWorkExecutor, err := domainWorkExecutorFromEnvV0(serverConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	requiredTestRunner, err := requiredTestRunnerFromEnvV0(serverConfig, stateStore)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	return orquestaappcodexstack.BuildStackV0(orquestaappcodexstack.ConfigV0{
		Enabled:        true,
		Timeout:        30 * time.Second,
		DirectorLimits: directorLimitsV0(),
		Stores: orquestaappcodexstack.StoresV0{
			RunStore:                   stateStore,
			EventSink:                  stateStore,
			OutboxLedger:               outboxLedger,
			TaskStore:                  stateStore,
			OperationalPlanStateWriter: stateStore,
			OperationalPlanStateStore:  stateStore,
			RequiredTestEvidenceStore:  stateStore,
			AppChangeStore:             runFileStore,
			ReceiptStore:               receiptStore,
			ProgressState:              progressStore,
			ProcessRegistry:            stateStore,
			RunControl:                 runFileStore,
			RunQueue:                   runFileStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{
			QueueRef:       "global",
			MaxRunsPerTick: intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", defaultCodexServerMaxRunsPerTickV0),
			QueueLimit:     intEnvOrDefaultV0("ORQUESTA_SERVER_QUEUE_LIMIT", defaultCodexServerQueueLimitV0),
			DefaultPriorityScore: intEnvOrDefaultV0(
				"ORQUESTA_SERVER_DEFAULT_PRIORITY", defaultCodexServerDefaultPriorityV0,
			),
		},
		RunSupervisor: orquestaappcodexstack.RunSupervisorConfigV0{
			MaxTicks:      1,
			MaxExecutions: intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", defaultCodexServerMaxExecutionsV0),
		},
		Codex:    codexRuntimeConfigV0(serverConfig, processRuntime),
		Capacity: codexStackCapacityConfigFromEnvV0(),
		ReviewGate: orquestaappcodexstack.ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
		RequiredTests: requiredTestRunner,
		DomainWork:    domainWorkExecutor,
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL") != "",
			Ledger:  domainDeliveryLedgerFromEnvV0(serverConfig),
		},
	})
}

func codexStackCapacityConfigFromEnvV0() orquestaappcodexstack.CapacityConfigV0 {
	tier := capacityRecommendationEnvOrDefaultV0(
		"ORQUESTA_CAPACITY_TIER",
		orquestacoreworkflow.OrchestrationCapacityXHighV0,
	)
	reasoningEffort := capacityRecommendationEnvOrDefaultV0(
		"ORQUESTA_CAPACITY_REASONING_EFFORT",
		capacityRecommendationEnvOrDefaultV0(
			"ORQUESTA_CODEX_REASONING_EFFORT",
			orquestacoreworkflow.OrchestrationCapacityXHighV0,
		),
	)
	return orquestaappcodexstack.CapacityConfigV0{
		Tier:            tier,
		ReasoningEffort: reasoningEffort,
		OccurredAt:      time.Now().UTC().Format(time.RFC3339),
		RequestedBy:     "orquesta-server",
		Summary:         "Capacidad inicial del servidor residente.",
		EvidenceRefs:    []string{"evidence-ref-orquesta-server"},
	}
}

func capacityRecommendationEnvOrDefaultV0(
	key string,
	fallback orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	switch value := strings.TrimSpace(os.Getenv(key)); value {
	case string(orquestacoreworkflow.OrchestrationCapacityLowV0):
		return orquestacoreworkflow.OrchestrationCapacityLowV0
	case string(orquestacoreworkflow.OrchestrationCapacityMediumV0):
		return orquestacoreworkflow.OrchestrationCapacityMediumV0
	case string(orquestacoreworkflow.OrchestrationCapacityHighV0):
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	case string(orquestacoreworkflow.OrchestrationCapacityXHighV0):
		return orquestacoreworkflow.OrchestrationCapacityXHighV0
	default:
		return fallback
	}
}

func domainDeliveryLedgerFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
) orquestaappcodexstack.DomainWorkArtifactSubmissionLedgerPortV0 {
	path := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH"))
	if path == "" {
		path = filepath.Join(serverConfig.StateDir, "domain-work-artifact-ledger.json")
	}
	return orquestaappcodexstack.NewFileDomainWorkArtifactSubmissionLedgerV0(path)
}

func directorLimitsV0() orquestaweb.WebArrancarDirectorAppLimitsV0 {
	return orquestaweb.WebArrancarDirectorAppLimitsV0{
		MaxBursts:            intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_BURSTS", 16),
		MaxStepsPerBurst:     intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_STEPS", 12),
		MaxDispatchesPerWait: intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_DISPATCHES", 8),
		MaxCommands:          intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_COMMANDS", 20),
		MaxOutboxPerCycle:    intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_OUTBOX", 8),
		MaxExternalWaits:     intEnvOrDefaultV0("ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS", 120),
	}
}

func codexRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaappcodexstack.CodexRuntimeConfigV0 {
	return orquestaappcodexstack.CodexRuntimeConfigV0{
		CommandPath:    codexCommandPathV0(),
		ProjectWorkDir: serverConfig.ProjectWorkDir,
		RuntimeWorkDir: serverConfig.RuntimeWorkDir,
		CodeHomeDir:    codeHomeDirV0(),
		HomeDir:        homeDirV0(),
		PathEnv:        envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH")),
		Model:          strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_MODEL")),
		ReasoningEffort: envOrDefaultV0(
			"ORQUESTA_CODEX_REASONING_EFFORT",
			string(orquestacoreworkflow.OrchestrationCapacityXHighV0),
		),
		Profile:        strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROFILE")),
		Sandbox:        envOrDefaultV0("ORQUESTA_CODEX_SANDBOX", "workspace-write"),
		ApprovalPolicy: envOrDefaultV0("ORQUESTA_CODEX_APPROVAL_POLICY", "never"),
		DirectorSandbox: strings.TrimSpace(
			os.Getenv("ORQUESTA_CODEX_DIRECTOR_SANDBOX"),
		),
		DirectorApprovalPolicy: strings.TrimSpace(
			os.Getenv("ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY"),
		),
		ExtraArgs:      strings.Fields(os.Getenv("ORQUESTA_CODEX_EXTRA_ARGS")),
		Runtime:        processRuntime,
		ProcessStopper: processRuntime,
		SnapshotSource: processRuntime,
		MaxBatchReady:  intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_BATCH_READY", defaultCodexMaxBatchReadyV0),
		MaxConcurrency: intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_CONCURRENCY", defaultCodexMaxConcurrencyV0),
		WaitInterval:   time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_WAIT_INTERVAL_MS", defaultCodexWaitIntervalMSV0)) * time.Millisecond,
		ProgressPolicy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: intEnvOrDefaultV0("ORQUESTA_CODEX_STALLED_TICKS", defaultCodexStalledTicksV0),
			LoopAfterRepeatedActions:    intEnvOrDefaultV0("ORQUESTA_CODEX_LOOP_TICKS", defaultCodexLoopTicksV0),
		},
		ProgressBudget: orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0{
			MaxExpected:     time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", defaultCodexMaxExpectedSecondsV0)) * time.Second,
			NoActivityLimit: time.Duration(intEnvOrDefaultV0("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", defaultCodexNoActivitySecondsV0)) * time.Second,
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

func domainWorkExecutorFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
) (orquestamcp.MCPDomainWorkExecutorPortV0, error) {
	baseURL := firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL")
	fileDir := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_FILE_DIR"))
	fileEnabled := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED")) == "1" ||
		fileDir != ""
	if baseURL != "" && fileEnabled {
		return nil, fmt.Errorf("domain_work_backend_ambiguous")
	}
	if baseURL != "" {
		client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
			BaseURL: baseURL,
			HTTPClient: &http.Client{
				Timeout: time.Duration(intEnvOrDefaultV0("ORQUESTA_OPES_TIMEOUT_SECONDS", 30)) * time.Second,
			},
			DefaultMaxAttempts: intEnvOrDefaultV0("ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS", 1),
		})
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if !fileEnabled {
		return nil, nil
	}
	if fileDir == "" {
		fileDir = filepath.Join(serverConfig.StateDir, "domain-work-jobs")
	}
	fileDir, err := filepath.Abs(fileDir)
	if err != nil {
		return nil, fmt.Errorf("domain_work_file_dir_invalid")
	}
	creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(fileDir)
	if err != nil {
		return nil, err
	}
	return orquestamcp.NewMCPDomainWorkToolExecutorV0(creator, nil), nil
}

func requiredTestRunnerFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	evidenceWriter orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0,
) (orquestacionnucleoapp.RequiredTestRunnerPortV0, error) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED")) != "1" {
		return nil, nil
	}
	if evidenceWriter == nil {
		return nil, fmt.Errorf("required_test_evidence_writer_required")
	}
	allowed, err := requiredTestAllowedCommandsFromEnvV0()
	if err != nil {
		return nil, err
	}
	absOutputDir, err := requiredTestOutputDirFromEnvV0(serverConfig)
	if err != nil {
		return nil, err
	}
	env, err := requiredTestEnvFromEnvV0(allowed, absOutputDir)
	if err != nil {
		return nil, err
	}
	return orquestacionnucleoapp.RequiredTestRunnerV0{
		Executor: orquestaruntimerequiredtest.LocalCommandExecutorV0{
			ProjectWorkDir:  serverConfig.ProjectWorkDir,
			OutputDir:       absOutputDir,
			AllowedCommands: allowed,
			Env:             env,
			MaxOutputBytes:  int64(intEnvOrDefaultV0("ORQUESTA_REQUIRED_TEST_MAX_OUTPUT_BYTES", 1024*1024)),
		},
		EvidenceWriter: evidenceWriter,
	}, nil
}

func requiredTestAllowedCommandsFromEnvV0() (map[string]string, error) {
	allowed := map[string]string{}
	if goCommand := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND")); goCommand != "" {
		abs, err := filepath.Abs(goCommand)
		if err != nil {
			return nil, fmt.Errorf("required_test_go_command_invalid")
		}
		allowed["go"] = abs
	}
	for _, item := range strings.Split(os.Getenv("ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		name, path, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("required_test_allowed_commands_invalid")
		}
		abs, err := filepath.Abs(strings.TrimSpace(path))
		if err != nil {
			return nil, fmt.Errorf("required_test_allowed_commands_invalid")
		}
		allowed[strings.TrimSpace(name)] = abs
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("required_test_allowed_commands_required")
	}
	return allowed, nil
}

func requiredTestOutputDirFromEnvV0(serverConfig orquestaserver.ConfigV0) (string, error) {
	outputDir := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"))
	if outputDir == "" {
		outputDir = filepath.Join(serverConfig.StateDir, "required-test-output")
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("required_test_output_dir_invalid")
	}
	return absOutputDir, nil
}

func requiredTestEnvFromEnvV0(
	allowed map[string]string,
	outputDir string,
) ([]string, error) {
	env := []string(nil)
	provided := map[string]bool{}
	for _, item := range strings.Split(os.Getenv("ORQUESTA_REQUIRED_TEST_ENV"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, _, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("required_test_env_invalid")
		}
		provided[strings.ToUpper(strings.TrimSpace(key))] = true
		env = append(env, item)
	}
	if _, ok := allowed["go"]; ok {
		goEnv := map[string]string{
			"GOCACHE":    filepath.Join(outputDir, "go-build-cache"),
			"GOPATH":     filepath.Join(outputDir, "go-path"),
			"GOMODCACHE": filepath.Join(outputDir, "go-mod-cache"),
		}
		for _, key := range []string{"GOCACHE", "GOPATH", "GOMODCACHE"} {
			if provided[key] {
				continue
			}
			dir := goEnv[key]
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("required_test_go_env_unavailable")
			}
			env = append(env, key+"="+dir)
		}
	}
	return env, nil
}

func validateCodexCommandAvailableV0() error {
	if !filepath.IsAbs(codexCommandPathV0()) {
		return fmt.Errorf("codex_command_unavailable")
	}
	return nil
}
