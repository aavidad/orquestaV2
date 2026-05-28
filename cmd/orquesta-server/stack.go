package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
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
			SnapshotReadBudget:      codexServerWorktreeSnapshotReadBudgetFromEnvV0(),
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
