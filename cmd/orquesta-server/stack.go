package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
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
	supervisorWakeup := &serverSupervisorWakeupRelayV0{}
	goalBackends, err := serverCodexGoalBackendsFromEnvV0(serverConfig)
	if err != nil {
		return nil, err
	}
	serverConfig = serverConfigWithCodexGoalBackendDiagnosticsV0(serverConfig, goalBackends)
	stack, err := buildStackFromEnvWithGoalBackendV0(serverConfig, goalBackends.AppGoal, supervisorWakeup)
	if err != nil {
		return nil, err
	}
	appHandler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		return nil, err
	}
	baseSupervisor := serverStackSupervisorV0{
		stack:                   &stack,
		projectWorkDir:          serverConfig.IdleSelfImprovementProjectWorkDir,
		runtimeWorkDir:          serverConfig.RuntimeWorkDir,
		stateDir:                serverConfig.StateDir,
		selfAuditBacklogEnabled: serverConfig.SelfAuditBacklogEnabled,
	}
	supervisor := serverSupervisorWithCodexGoalBackendV0(baseSupervisor, goalBackends.IdleGoal)
	residentDirector := newServerResidentDirectorV0(&stack, serverConfig)
	runtime, err := orquestaserver.NewRuntimeV0(serverConfig, orquestaserver.RuntimeDepsV0{
		AppHandler:       appHandler,
		Supervisor:       supervisor,
		ResidentDirector: residentDirector,
		RouteManifest:    serverRouteManifestResourcesV0(),
		GoalStateStore:   stack.Stores.AppGoalStateStore,
		GoalFingerprint:  serverGoalObservationFingerprintFromBackendV0(goalBackends.AppGoal, serverGoalObserverFingerprintEnabledFromEnvV0()),
		ShutdownHooks:    serverGoalShutdownHooksFromBackendsV0(goalBackends.AppGoal, goalBackends.IdleGoal),
		StartupCheck:     startupCheckFromEnvV0(stack, serverConfig),
		SelfWatchdog: orquestaserver.NewProcessSelfWatchdogObserverV0(
			orquestaserver.NewProcSelfCPUSamplerV0(),
		),
	})
	if err != nil {
		return nil, err
	}
	supervisorWakeup.bindRuntimeV0(runtime)
	return runtime, nil
}

func buildServerAppHandlerV0(stack orquestaappcodexstack.StackV0) (http.Handler, error) {
	eventReader, _ := stack.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	stack.MCPTransportBindings.WorkspaceTimeline = newServerWorkspaceTimelineSourceWithEventsV0(
		stack.MCPTransportBindings,
		eventReader,
	)
	mcpHandler, err := newMCPRealHTTPHandlerV0(stack.MCPTransportBindings)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(mcpRealHTTPPathV0, mcpHandler)
	mux.Handle(orquestamcp.MCPWorkspaceTimelineEndpointV0, newServerWorkspaceTimelineHTTPHandlerV0(
		stack.MCPTransportBindings.WorkspaceTimeline,
	))
	observedWeb := orquestaappgateway.ObserveWebHTMLRenderErrorsV0(
		stack.Handler,
		serverWebHTMLRenderObserverV0(),
	)
	mux.Handle("/", withGovernanceCatalogRouteV0(observedWeb))
	return orquestahttpgateway.NewControlPlaneHTTPHeadersV0(mux), nil
}

func serverWebHTMLRenderObserverV0() orquestaobservability.WebHTMLRenderObserverV0 {
	return orquestaobservability.NewInMemoryWebHTMLRenderObserverV0()
}

func buildStackFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	supervisorWakeups ...*serverSupervisorWakeupRelayV0,
) (orquestaappcodexstack.StackV0, error) {
	return buildStackFromEnvWithGoalBackendV0(serverConfig, serverCodexGoalBackendV0{}, supervisorWakeups...)
}

func buildStackFromEnvWithGoalBackendV0(
	serverConfig orquestaserver.ConfigV0,
	goalBackend serverCodexGoalBackendV0,
	supervisorWakeups ...*serverSupervisorWakeupRelayV0,
) (orquestaappcodexstack.StackV0, error) {
	var supervisorWakeup *serverSupervisorWakeupRelayV0
	if len(supervisorWakeups) > 0 {
		supervisorWakeup = supervisorWakeups[0]
	}
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
	var outboxLedgerPort orquestaappcodexstack.OutboxLedgerPortV0 = outboxLedger
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
	operatorConnector, err := hermesOperatorConnectorFromEnvV0()
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	requiredTestRunner, err := requiredTestRunnerFromEnvV0(serverConfig, stateStore)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	egressSanitizer, err := egressSanitizerConfigWithSidecarPortFromEnvV0()
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	worktreeSnapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	runStore := orquestacionnucleoapp.RunStorePortV0(stateStore)
	runQueue := orquestarunqueue.RunQueuePortV0(runFileStore)
	runControl := orquestaruncontrol.RunControlPortV0(runFileStore)
	receiptStorePort := orquestaappcodexstack.CodexReceiptStorePortV0(receiptStore)
	appChangeStore := orquestaappchange.AppChangeRecordStorePortV0(runFileStore)
	var appGoalStateStore orquestagoal.GoalWorkStateStorePortV0 = stateStore
	domainDeliveryLedger := domainDeliveryLedgerFromEnvV0(serverConfig)
	if supervisorWakeup != nil {
		runStore = serverWakeupRunStoreV0{inner: stateStore, wakeup: supervisorWakeup}
		runQueue = serverWakeupRunQueueV0{inner: runFileStore, wakeup: supervisorWakeup}
		runControl = serverWakeupRunControlV0{inner: runFileStore, wakeup: supervisorWakeup}
		appChangeStore = serverWakeupAppChangeStoreV0{
			inner:  appChangeStore,
			wakeup: supervisorWakeup,
		}
		receiptStorePort = serverWakeupCodexReceiptStoreV0{
			inner:  receiptStorePort,
			wakeup: supervisorWakeup,
		}
		appGoalStateStore = serverWakeupGoalStateStoreV0{
			inner:  stateStore,
			wakeup: supervisorWakeup,
		}
		outboxLedgerPort = serverWakeupDirectorCycleOutboxLedgerV0{
			inner:  outboxLedgerPort,
			wakeup: supervisorWakeup,
		}
		domainDeliveryLedger = serverWakeupDomainWorkArtifactSubmissionLedgerV0{
			inner:  domainDeliveryLedger,
			wakeup: supervisorWakeup,
		}
	}
	codeContextWiring := codeContextBrokerWiringFromEnvV0(serverConfig)
	stack, err := orquestaappcodexstack.BuildStackV0(orquestaappcodexstack.ConfigV0{
		Enabled:        true,
		Timeout:        30 * time.Second,
		DirectorLimits: directorLimitsV0(),
		Stores: orquestaappcodexstack.StoresV0{
			RunStore:                   runStore,
			EventSink:                  stateStore,
			OutboxLedger:               outboxLedgerPort,
			TaskStore:                  stateStore,
			WaitStateStore:             stateStore,
			OperationalPlanStateWriter: stateStore,
			OperationalPlanStateStore:  stateStore,
			RequiredTestEvidenceStore:  stateStore,
			AppChangeStore:             appChangeStore,
			ReceiptStore:               receiptStorePort,
			ProgressState:              progressStore,
			ProcessRegistry:            stateStore,
			RunControl:                 runControl,
			RunQueue:                   runQueue,
			AppGoalStateStore:          appGoalStateStore,
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
			codexUsageMetricsFromEnvV0(receiptStorePort),
		),
		Gemini:                geminiRuntimeConfigV0(serverConfig),
		Claude:                claudeRuntimeConfigV0(serverConfig),
		EgressSanitizer:       egressSanitizer,
		Capacity:              codexStackCapacityConfigFromEnvV0(),
		AppGoalLauncher:       serverGoalWorkLauncherFromBackendV0(goalBackend),
		AppGoalReworkLauncher: serverGoalWorkLauncherFromBackendV0(goalBackend),
		AppGoalObserver:       serverGoalWorkObserverFromBackendV0(goalBackend),
		RuntimeModels:         runtimeModelManagerFromEnvV0(),
		ReviewGate: orquestaappcodexstack.ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      boolEnvOrDefaultV0(envReviewGateStrictGoLineBudgetV0, false),
			LineBudgetSnapshotStore: worktreeSnapshotStore,
			SnapshotReadBudget:      codexServerWorktreeSnapshotReadBudgetFromEnvV0(),
		},
		RequiredTests:                         requiredTestRunner,
		DomainTests:                           domainWorkRequiredTestConfigFromEnvV0(),
		AutoprogrammingPromotion:              autoprogrammingPromotionConfigFromEnvV0(serverConfig),
		DomainWork:                            domainWorkExecutor,
		CodeContext:                           codeContextWiring.Query,
		CodeContextToolLeases:                 codeContextWiring.ToolLeases,
		ExternalWorkRunGuard:                  externalWorkRunProjectWorkDirGuardConfigFromEnvV0(serverConfig),
		GoalObserverResidentEnabled:           serverConfig.GoalObserverEnabled,
		PromoteMaterializedArtifactWithoutAck: codexPromoteMaterializedArtifactWithoutAckFromEnvV0(),
		AllowLegacyAutoprogrammingRun: boolEnvOrDefaultV0(
			envAutoprogrammingLegacyDirectorLoopV0,
			false,
		),
		AllowLegacyExternalWorkRun: boolEnvOrDefaultV0(
			envExternalWorkLegacyDirectorLoopV0,
			false,
		),
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: domainWorkDeliveryEnabledFromEnvV0(),
			Ledger:  domainDeliveryLedger,
		},
	})
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	if operatorConnector != nil {
		stack.MCPTransportBindings.OperatorConnector = operatorConnector
	}
	stack.Handler = withFunctionContractRoutesV0(stack.Handler, stateStore)
	return stack, nil
}

func codexPromoteMaterializedArtifactWithoutAckFromEnvV0() bool {
	return boolEnvOrDefaultV0(envCodexPromoteMaterializedArtifactWithoutAckV0, true)
}

func codexStackCapacityConfigFromEnvV0() orquestaappcodexstack.CapacityConfigV0 {
	envConfig := codexStackCapacityEnvConfigFromEnvV0()
	return orquestaappcodexstack.CapacityConfigV0{
		Tier:            envConfig.Tier,
		ReasoningEffort: envConfig.ReasoningEffort,
		PolicyRef:       envConfig.PolicyRef,
		PoolRef:         envConfig.PoolRef,
		ModelRef:        envConfig.ModelRef,
		QuotaRef:        envConfig.QuotaRef,
		OccurredAt:      time.Now().UTC().Format(time.RFC3339),
		RequestedBy:     "orquesta-server",
		Summary:         "Capacidad inicial del servidor residente.",
		EvidenceRefs:    []string{"evidence-ref-orquesta-server"},
	}
}

type codexStackCapacityEnvConfigV0 struct {
	Tier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	ReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	PolicyRef       string
	PoolRef         string
	ModelRef        string
	QuotaRef        string
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
		PolicyRef: envOrDefaultV0(envCapacityPolicyRefV0, "capacity-policy-ref-server-default-v0"),
		PoolRef:   envOrDefaultV0(envCapacityPoolRefV0, "capacity-pool-ref-server-default-v0"),
		ModelRef:  envOrDefaultV0(envCapacityModelRefV0, "capacity-model-ref-server-default-v0"),
		QuotaRef:  envOrDefaultV0(envCapacityQuotaRefV0, "capacity-quota-ref-server-default-v0"),
	}
}

func (supervisor serverStackSupervisorV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request orquestaserver.IdleSelfImprovementRequestFilterRequestV0,
) (orquestaserver.IdleSelfImprovementRequestFilterResultV0, error) {
	out := make([]orquestaserver.IdleSelfImprovementRequestV0, 0, len(request.Requests))
	dropped := 0
	for _, candidate := range request.Requests {
		if serverStackIdleSelfImprovementBacklogRequestAllowedV0(candidate) {
			out = append(out, candidate)
			continue
		}
		dropped++
	}
	result := orquestaserver.IdleSelfImprovementRequestFilterResultV0{Requests: out}
	if dropped > 0 {
		result.EvidenceRefs = []string{"evidence-ref-idle-self-improvement-backlog-non-task-section-filtered"}
		result.Message = "backlog_non_task_sections_filtered"
	}
	return result, nil
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

func serverRunQueueLimitFromEnvV0() int {
	return intEnvOrDefaultV0(envServerQueueLimitV0, defaultCodexServerQueueLimitV0)
}

type codexDirectorWaveLimitsEnvConfigV0 struct {
	Agents               int
	MaxSubagentsPerAgent int
	RecursiveAgentBudget int
}

func codexDirectorWaveLimitsEnvConfigFromEnvV0() codexDirectorWaveLimitsEnvConfigV0 {
	executionMode := codexExecutionModeFromEnvV0()
	return codexDirectorWaveLimitsEnvConfigV0{
		Agents:               codexExecutionModeCapPositiveV0(executionMode, intEnvOrDefaultV0(envCodexDirectorWaveAgentsV0, defaultCodexDirectorWaveAgentsV0)),
		MaxSubagentsPerAgent: intEnvOrDefaultV0(envCodexDirectorMaxSubagentsPerAgentV0, defaultCodexDirectorMaxSubagentsPerAgentV0),
		RecursiveAgentBudget: intEnvOrDefaultV0(envCodexDirectorRecursiveAgentBudgetV0, defaultCodexDirectorRecursiveAgentBudgetV0),
	}
}
