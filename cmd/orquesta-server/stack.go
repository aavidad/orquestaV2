package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
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
	appHandler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		return nil, err
	}
	supervisor := serverStackSupervisorV0{
		stack:          &stack,
		projectWorkDir: serverConfig.ProjectWorkDir,
		runtimeWorkDir: serverConfig.RuntimeWorkDir,
		stateDir:       serverConfig.StateDir,
	}
	return orquestaserver.NewRuntimeV0(serverConfig, orquestaserver.RuntimeDepsV0{
		AppHandler:   appHandler,
		Supervisor:   supervisor,
		StartupCheck: startupCheckFromEnvV0(stack, serverConfig),
	})
}

func buildServerAppHandlerV0(stack orquestaappcodexstack.StackV0) (http.Handler, error) {
	stack.MCPTransportBindings.WorkspaceTimeline = newServerWorkspaceTimelineSourceV0(stack.MCPTransportBindings)
	mcpHandler, err := newMCPRealHTTPHandlerV0(stack.MCPTransportBindings)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(mcpRealHTTPPathV0, mcpHandler)
	mux.Handle(orquestamcp.MCPWorkspaceTimelineEndpointV0, orquestamcp.NewMCPWorkspaceTimelineHTTPHandlerV0(
		stack.MCPTransportBindings.WorkspaceTimeline,
	))
	mux.Handle("/", withGovernanceCatalogRouteV0(stack.Handler))
	return mux, nil
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
	outboxLedger, err := orquestapersistence.NewFileOutboxLedgerV0(
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
	worktreeSnapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	stack, err := orquestaappcodexstack.BuildStackV0(orquestaappcodexstack.ConfigV0{
		Enabled:        true,
		Timeout:        30 * time.Second,
		DirectorLimits: directorLimitsV0(),
		Stores: orquestaappcodexstack.StoresV0{
			RunStore:                   stateStore,
			EventSink:                  stateStore,
			OutboxLedger:               outboxLedger,
			TaskStore:                  stateStore,
			WaitStateStore:             stateStore,
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
			MaxRunsPerTick: serverConfig.SupervisorCommand.MaxRunsPerTick,
			QueueLimit:     serverRunQueueLimitFromEnvV0(),
			DefaultPriorityScore: intEnvOrDefaultV0(
				envServerDefaultPriorityV0, defaultCodexServerDefaultPriorityV0,
			),
		},
		RunSupervisor: orquestaappcodexstack.RunSupervisorConfigV0{
			MaxTicks:      serverConfig.SupervisorCommand.MaxTicks,
			MaxExecutions: serverConfig.SupervisorCommand.MaxExecutions,
		},
		Codex: codexRuntimeConfigV0(
			serverConfig,
			processRuntime,
			codexUsageMetricsFromEnvV0(receiptStore),
		),
		Capacity: codexStackCapacityConfigFromEnvV0(),
		ReviewGate: orquestaappcodexstack.ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      boolEnvOrDefaultV0(envReviewGateStrictGoLineBudgetV0, false),
			LineBudgetSnapshotStore: worktreeSnapshotStore,
		},
		RequiredTests:            requiredTestRunner,
		DomainTests:              domainWorkRequiredTestConfigFromEnvV0(),
		AutoprogrammingPromotion: autoprogrammingPromotionConfigFromEnvV0(serverConfig),
		DomainWork:               domainWorkExecutor,
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: domainWorkDeliveryEnabledFromEnvV0(),
			Ledger:  domainDeliveryLedgerFromEnvV0(serverConfig),
		},
	})
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	stack.Handler = withFunctionContractRoutesV0(stack.Handler, stateStore)
	return stack, nil
}

func codexStackCapacityConfigFromEnvV0() orquestaappcodexstack.CapacityConfigV0 {
	envConfig := codexStackCapacityEnvConfigFromEnvV0()
	return orquestaappcodexstack.CapacityConfigV0{
		Tier:            envConfig.Tier,
		ReasoningEffort: envConfig.ReasoningEffort,
		OccurredAt:      time.Now().UTC().Format(time.RFC3339),
		RequestedBy:     "orquesta-server",
		Summary:         "Capacidad inicial del servidor residente.",
		EvidenceRefs:    []string{"evidence-ref-orquesta-server"},
	}
}

type codexStackCapacityEnvConfigV0 struct {
	Tier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	ReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

func codexStackCapacityEnvConfigFromEnvV0() codexStackCapacityEnvConfigV0 {
	return codexStackCapacityEnvConfigV0{
		Tier: capacityRecommendationEnvOrDefaultV0(
			envCapacityTierV0,
			orquestacoreworkflow.OrchestrationCapacityMediumV0,
		),
		ReasoningEffort: capacityRecommendationEnvOrDefaultV0(
			envCapacityReasoningEffortV0,
			capacityRecommendationEnvOrDefaultV0(
				envCodexReasoningEffortV0,
				orquestacoreworkflow.OrchestrationCapacityMediumV0,
			),
		),
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
	path := strings.TrimSpace(os.Getenv(envDomainDeliveryLedgerPathV0))
	if path == "" {
		path = filepath.Join(serverConfig.StateDir, "domain-work-artifact-ledger.json")
	}
	return orquestaappcodexstack.NewFileDomainWorkArtifactSubmissionLedgerV0(path)
}

func directorLimitsV0() orquestaweb.WebArrancarDirectorAppLimitsV0 {
	return orquestaweb.WebArrancarDirectorAppLimitsV0{
		MaxBursts:            intEnvOrDefaultV0(envDirectorMaxBurstsV0, 16),
		MaxStepsPerBurst:     intEnvOrDefaultV0(envDirectorMaxStepsV0, 12),
		MaxDispatchesPerWait: intEnvOrDefaultV0(envDirectorMaxDispatchesV0, 8),
		MaxCommands:          intEnvOrDefaultV0(envDirectorMaxCommandsV0, 20),
		MaxOutboxPerCycle:    intEnvOrDefaultV0(envDirectorMaxOutboxV0, 8),
		MaxExternalWaits:     intEnvOrDefaultV0(envDirectorMaxExternalWaitsV0, 120),
	}
}

func codexRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	usageMetrics ...orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0,
) orquestaappcodexstack.CodexRuntimeConfigV0 {
	var usageSource orquestaappcodexstack.CodexStackAgentUsageMetricsProviderPortV0
	if len(usageMetrics) > 0 {
		usageSource = usageMetrics[0]
	}
	envConfig := codexRuntimeEnvConfigFromEnvV0()
	return orquestaappcodexstack.CodexRuntimeConfigV0{
		CommandPath:              envConfig.CommandPath,
		ProjectWorkDir:           serverConfig.ProjectWorkDir,
		RuntimeWorkDir:           serverConfig.RuntimeWorkDir,
		CodeHomeDir:              envConfig.CodeHomeDir,
		HomeDir:                  envConfig.HomeDir,
		PathEnv:                  envConfig.PathEnv,
		Model:                    envConfig.Model,
		ReasoningEffort:          envConfig.ReasoningEffort,
		Profile:                  envConfig.Profile,
		Sandbox:                  envConfig.Sandbox,
		ApprovalPolicy:           envConfig.ApprovalPolicy,
		DirectorSandbox:          envConfig.DirectorSandbox,
		DirectorApprovalPolicy:   envConfig.DirectorApprovalPolicy,
		InteractiveApprovalOptIn: envConfig.InteractiveApprovalOptIn,
		ExtraArgs:                envConfig.ExtraArgs,
		PromptHints:              codexServerPromptHintsV0(serverConfig),
		Runtime:                  processRuntime,
		ProcessStopper:           processRuntime,
		SnapshotSource:           processRuntime,
		MaxBatchReady:            envConfig.Limits.MaxBatchReady,
		MaxConcurrency:           envConfig.Limits.MaxLiveProcesses,
		WaitInterval:             envConfig.WaitInterval,
		ProgressPolicy:           envConfig.ProgressPolicy,
		ProgressBudget:           envConfig.ProgressBudget,
		UsageMetrics:             usageSource,
	}
}

type codexRuntimeEnvConfigV0 struct {
	CommandPath              string
	CodeHomeDir              string
	HomeDir                  string
	PathEnv                  string
	Model                    string
	ReasoningEffort          string
	Profile                  string
	Sandbox                  string
	ApprovalPolicy           string
	DirectorSandbox          string
	DirectorApprovalPolicy   string
	InteractiveApprovalOptIn bool
	ExtraArgs                []string
	Limits                   codexRuntimeLimitsV0
	WaitInterval             time.Duration
	ProgressPolicy           orquestaruntime.AgentProgressHeartbeatPolicyV0
	ProgressBudget           orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0
}

func codexRuntimeEnvConfigFromEnvV0() codexRuntimeEnvConfigV0 {
	return codexRuntimeEnvConfigV0{
		CommandPath: codexCommandPathV0(),
		CodeHomeDir: codeHomeDirV0(),
		HomeDir:     homeDirV0(),
		PathEnv:     envOrDefaultV0(envCodexPathV0, os.Getenv("PATH")),
		Model:       strings.TrimSpace(os.Getenv(envCodexModelV0)),
		ReasoningEffort: codexReasoningEffortPolicyV0(envOrDefaultV0(
			envCodexReasoningEffortV0,
			string(orquestacoreworkflow.OrchestrationCapacityMediumV0),
		)),
		Profile:                  strings.TrimSpace(os.Getenv(envCodexProfileV0)),
		Sandbox:                  codexSandboxFromEnvV0(envCodexSandboxV0, "danger-full-access"),
		ApprovalPolicy:           envOrDefaultV0(envCodexApprovalPolicyV0, "never"),
		DirectorSandbox:          codexOptionalSandboxFromEnvV0(envCodexDirectorSandboxV0),
		DirectorApprovalPolicy:   strings.TrimSpace(os.Getenv(envCodexDirectorApprovalPolicyV0)),
		InteractiveApprovalOptIn: boolEnvOrDefaultV0(envCodexAllowInteractiveApprovalV0, false),
		ExtraArgs:                strings.Fields(os.Getenv(envCodexExtraArgsV0)),
		Limits:                   codexRuntimeLimitsFromEnvV0(),
		WaitInterval:             time.Duration(intEnvOrDefaultV0(envCodexWaitIntervalMSV0, defaultCodexWaitIntervalMSV0)) * time.Millisecond,
		ProgressPolicy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: intEnvOrDefaultV0(envCodexStalledTicksV0, defaultCodexStalledTicksV0),
			LoopAfterRepeatedActions:    intEnvOrDefaultV0(envCodexLoopTicksV0, defaultCodexLoopTicksV0),
		},
		ProgressBudget: orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0{
			MaxExpected:     time.Duration(intEnvOrDefaultV0(envCodexMaxExpectedSecondsV0, defaultCodexMaxExpectedSecondsV0)) * time.Second,
			NoActivityLimit: time.Duration(intEnvOrDefaultV0(envCodexNoActivitySecondsV0, defaultCodexNoActivitySecondsV0)) * time.Second,
		},
	}
}

type codexRuntimeLimitsV0 struct {
	MaxBatchReady    int
	MaxLiveProcesses int
}

func codexRuntimeLimitsFromEnvV0() codexRuntimeLimitsV0 {
	return codexRuntimeLimitsV0{
		MaxBatchReady:    intEnvOrDefaultV0(envCodexMaxBatchReadyV0, defaultCodexMaxBatchReadyV0),
		MaxLiveProcesses: intEnvOrDefaultV0(envCodexMaxConcurrencyV0, defaultCodexMaxConcurrencyV0),
	}
}

func serverRunQueueLimitFromEnvV0() int {
	return intEnvOrDefaultV0(envServerQueueLimitV0, defaultCodexServerQueueLimitV0)
}

type codexDirectorWaveLimitsEnvConfigV0 struct {
	Agents               int
	MaxSubagentsPerAgent int
	RecursiveAgentBudget int
}

func codexDirectorWaveLimitsEnvConfigFromEnvV0() codexDirectorWaveLimitsEnvConfigV0 {
	return codexDirectorWaveLimitsEnvConfigV0{
		Agents:               intEnvOrDefaultV0(envCodexDirectorWaveAgentsV0, defaultCodexDirectorWaveAgentsV0),
		MaxSubagentsPerAgent: intEnvOrDefaultV0(envCodexDirectorMaxSubagentsPerAgentV0, defaultCodexDirectorMaxSubagentsPerAgentV0),
		RecursiveAgentBudget: intEnvOrDefaultV0(envCodexDirectorRecursiveAgentBudgetV0, defaultCodexDirectorRecursiveAgentBudgetV0),
	}
}
