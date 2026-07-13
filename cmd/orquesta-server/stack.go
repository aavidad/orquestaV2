package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
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
	orquestatoolcapabilityfile "orquesta/modulos/orquesta-tool-capability-file"
	orquestaweb "orquesta/modulos/orquesta-web"
)

var serverGoalRequiredTestClaimOwnerRefV0 = fmt.Sprintf(
	"owner-ref-orquesta-server-goal-observer-%d-%d",
	os.Getpid(),
	time.Now().UTC().UnixNano(),
)

func buildRuntimeFromConfigV0(serverConfig orquestaserver.ConfigV0) (*orquestaserver.RuntimeV0, error) {
	supervisorWakeup := &serverSupervisorWakeupRelayV0{}
	goalBackends, err := serverCodexGoalBackendsFromEnvV0(serverConfig)
	if err != nil {
		return nil, err
	}
	serverConfig = serverConfigWithCodexGoalBackendDiagnosticsV0(serverConfig, goalBackends)
	projectConfig, err := projectConfigForBuildStackV0(serverConfig)
	if err != nil {
		return nil, err
	}
	stack, err := buildStackFromProjectConfigWithGoalBackendsV0(
		serverConfig,
		goalBackends.AppGoal,
		goalBackends.AutoprogrammingGoal,
		projectConfig,
		supervisorWakeup,
	)
	if err != nil {
		return nil, err
	}
	requiredTestResourceHook := serverRequiredTestResourceShutdownHookFromStackV0(stack)
	resourcesTransferred := false
	defer func() {
		if !resourcesTransferred {
			_ = requiredTestResourceHook.ShutdownV0(context.Background())
		}
	}()
	appHandler, err := buildServerAppHandlerV0(stack, serverConfig)
	if err != nil {
		return nil, err
	}
	baseSupervisor := serverStackSupervisorV0{
		stack:                   &stack,
		projectWorkDir:          serverConfig.IdleSelfImprovementProjectWorkDir,
		runtimeWorkDir:          serverConfig.RuntimeWorkDir,
		stateDir:                serverConfig.StateDir,
		selfAuditBacklogEnabled: serverConfig.SelfAuditBacklogEnabled,
		curatedSkills:           serverCuratedSkillsFromProjectV0(serverConfig.IdleSelfImprovementProjectWorkDir),
		operatorNotifier:        operatorTaskTerminalNotifierFromProjectConfigFileV0(projectConfig),
	}
	supervisor := serverSupervisorWithCodexGoalBackendV0(baseSupervisor, goalBackends.IdleGoal)
	residentDirector := newServerResidentDirectorV0(&stack, serverConfig)
	runtime, err := orquestaserver.NewRuntimeV0(serverConfig, serverRuntimeDepsFromStackV0(
		serverConfig,
		appHandler,
		supervisor,
		residentDirector,
		stack,
		goalBackends,
		requiredTestResourceHook,
	))
	if err != nil {
		return nil, err
	}
	supervisorWakeup.bindRuntimeV0(runtime)
	resourcesTransferred = true
	return runtime, nil
}

func serverRuntimeDepsFromStackV0(
	serverConfig orquestaserver.ConfigV0,
	appHandler http.Handler,
	supervisor orquestaserver.SupervisorPortV0,
	residentDirector orquestaserver.ResidentDirectorPortV0,
	stack orquestaappcodexstack.StackV0,
	goalBackends serverCodexGoalBackendsV0,
	requiredTestResourceHooks ...*serverRequiredTestResourceShutdownHookV0,
) orquestaserver.RuntimeDepsV0 {
	shutdownHooks := serverGoalShutdownHooksFromBackendsV0(
		goalBackends.AppGoal,
		goalBackends.AutoprogrammingGoal,
		goalBackends.IdleGoal,
	)
	var requiredTestResourceHook *serverRequiredTestResourceShutdownHookV0
	if len(requiredTestResourceHooks) > 0 {
		requiredTestResourceHook = requiredTestResourceHooks[0]
	} else {
		requiredTestResourceHook = serverRequiredTestResourceShutdownHookFromStackV0(stack)
	}
	if requiredTestResourceHook != nil {
		shutdownHooks = append(shutdownHooks, requiredTestResourceHook)
	}
	return orquestaserver.RuntimeDepsV0{
		AppHandler:                 appHandler,
		Supervisor:                 supervisor,
		ResidentDirector:           residentDirector,
		RouteManifest:              serverRouteManifestResourcesV0(),
		GoalStateStore:             stack.Stores.AppGoalStateStore,
		GoalRequiredTestSpecBinder: stack.Ports.GoalRequiredTestSpecBinder,
		GoalFingerprint: serverGoalObservationFingerprintForAppAndAutoprogrammingV0(
			goalBackends.AppGoal,
			goalBackends.AutoprogrammingGoal,
			serverGoalObserverFingerprintEnabledFromEnvV0(),
		),
		GoalStopper:              serverGoalCooperativeStopperFromRunControlV0(stack.Stores.RunControl, stack.MCPTransportBindings.RunControl),
		MaterialProgressStore:    serverMaterialProgressStoreFromStackV0(stack),
		MaterialProgressEvidence: orquestaappcodexstack.CodexStackMaterialProgressEvidenceV0{Stack: &stack},
		EstadoVivoSource:         stack.MCPTransportBindings.AutoprogrammingEstadoVivoSource,
		ShutdownSnapshot:         serverShutdownSnapshotFromStackV0(stack, goalBackends),
		ShutdownHooks:            shutdownHooks,
		BackgroundWorkers:        serverBackgroundWorkersFromStackV0(stack),
		StartupCheck:             startupCheckFromEnvV0(stack, serverConfig),
		SelfWatchdog: orquestaserver.NewProcessSelfWatchdogObserverV0(
			orquestaserver.NewProcSelfCPUSamplerV0(),
		),
		ForceExit: serverForceExitPortV0{},
	}
}

func serverMaterialProgressStoreFromStackV0(
	stack orquestaappcodexstack.StackV0,
) orquestaautoprogramming.MaterialProgressStateStorePortV0 {
	store, _ := stack.Stores.AppGoalStateStore.(orquestaautoprogramming.MaterialProgressStateStorePortV0)
	return store
}

type serverForceExitPortV0 struct{}

func (serverForceExitPortV0) ExitV0(code int) {
	os.Exit(code)
}

func buildServerAppHandlerV0(
	stack orquestaappcodexstack.StackV0,
	serverConfigs ...orquestaserver.ConfigV0,
) (http.Handler, error) {
	var serverConfig orquestaserver.ConfigV0
	if len(serverConfigs) > 0 {
		serverConfig = serverConfigs[0]
	}
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
	if stack.MCPTransportBindings.CodebaseStatus != nil {
		mux.Handle(orquestamcp.MCPCodebaseStatusHTTPPathV0, orquestamcp.NewMCPCodebaseStatusHTTPHandlerV0(
			stack.MCPTransportBindings.CodebaseStatus,
		))
	}
	if stack.MCPTransportBindings.CodebaseQuery != nil {
		mux.Handle(orquestamcp.MCPCodebaseQueryHTTPPathV0, orquestamcp.NewMCPCodebaseQueryHTTPHandlerV0(
			stack.MCPTransportBindings.CodebaseQuery,
		))
	}
	observedWeb := orquestaappgateway.ObserveWebHTMLRenderErrorsV0(
		stack.Handler,
		serverWebHTMLRenderObserverV0(),
	)
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	telegramOperator := telegramOperatorAdapterFromProjectConfigFileV0(
		projectConfig,
		telegramOperatorHTTPPortsV0{
			Handler:         observedWeb,
			DirectorMessage: stack.MCPTransportBindings.OperatorDirectorMessage,
		},
	)
	if telegramOperator.Enabled {
		mux.Handle(
			telegramOperatorUpdateHTTPPathV0,
			newTelegramOperatorUpdateHTTPHandlerV0(
				telegramOperator,
				telegramBotAPISenderFromProjectConfigFileV0(projectConfig),
			),
		)
	}
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
	projectConfig, err := projectConfigForBuildStackV0(serverConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	return buildStackFromProjectConfigWithGoalBackendsV0(
		serverConfig,
		goalBackend,
		goalBackend,
		projectConfig,
		supervisorWakeups...,
	)
}

func buildStackFromProjectConfigV0(
	serverConfig orquestaserver.ConfigV0,
	goalBackend serverCodexGoalBackendV0,
	projectConfig serverProjectConfigFileV0,
	supervisorWakeups ...*serverSupervisorWakeupRelayV0,
) (orquestaappcodexstack.StackV0, error) {
	return buildStackFromProjectConfigWithGoalBackendsV0(
		serverConfig,
		goalBackend,
		goalBackend,
		projectConfig,
		supervisorWakeups...,
	)
}

func buildStackFromProjectConfigWithGoalBackendsV0(
	serverConfig orquestaserver.ConfigV0,
	goalBackend serverCodexGoalBackendV0,
	autoprogrammingGoalBackend serverCodexGoalBackendV0,
	projectConfig serverProjectConfigFileV0,
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
	domainWorkExecutor, err := domainWorkExecutorFromEnvV0(serverConfig, projectConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	operatorConnector, err := hermesOperatorConnectorFromProjectConfigV0(serverConfig, projectConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	requiredTestRunner, err := requiredTestRunnerFromEnvV0(serverConfig, stateStore, projectConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	var goalRequiredTestAttestation *serverGoalRequiredTestAttestationWorkspaceSelectorV0
	var batchTestRunner *serverAutoprogrammingBatchTestRunnerV0
	resourcesTransferred := false
	defer func() {
		if !resourcesTransferred {
			hook := newServerRequiredTestResourceShutdownHookV0(
				requiredTestRunner,
				goalRequiredTestAttestation,
				batchTestRunner,
			)
			_ = hook.ShutdownV0(context.Background())
		}
	}()
	idleBudgetSource, err := newServerAutoprogrammingIdleBudgetSourceV0(serverConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	goalRequiredTestAttestation, err = goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		serverConfig,
		projectConfig,
		serverCodexGoalWorkspaceLookupFromBackendV0(autoprogrammingGoalBackend),
	)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	var goalRequiredTestSpecBinder orquestagoal.GoalRequiredTestSpecBinderPortV0
	var goalRequiredTestSnapshotObserver orquestagoal.GoalRequiredTestFinalSnapshotObserverPortV0
	var goalRequiredTestAttestor orquestagoal.GoalRequiredTestAttestorPortV0
	var goalRequiredTestIdentityVerifier orquestagoal.GoalRequiredTestIdentityVerifierPortV0
	if goalRequiredTestAttestation != nil {
		goalRequiredTestSpecBinder = goalRequiredTestAttestation
		goalRequiredTestSnapshotObserver = goalRequiredTestAttestation
		goalRequiredTestAttestor = goalRequiredTestAttestation
		goalRequiredTestIdentityVerifier = goalRequiredTestAttestation
	}
	egressSanitizer, err := egressSanitizerConfigWithSidecarPortFromProjectConfigFileV0(
		projectConfig,
	)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	runtimeModels, err := runtimeModelManagerFromConfigV0(serverConfig, projectConfig)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	var worktreeSnapshotStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0 = stateStore
	canonicalSelfProgrammingDir := firstNonEmptyServerStackV0(
		serverConfig.IdleSelfImprovementProjectWorkDir,
		serverConfig.ProjectWorkDir,
	)
	autoprogrammingPromotion := autoprogrammingPromotionConfigFromEnvV0(serverConfig)
	autoprogrammingPromotion.GoalFirstSnapshotStore = worktreeSnapshotStore
	autoprogrammingPromotion.GoalWorkspaceIntegration = orquestaruntimeworktree.GitGoalWorkspaceIntegrationConnectorV0{}
	batchTestRunner, err = autoprogrammingBatchTestRunnerFromConfigV0(serverConfig, canonicalSelfProgrammingDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	autoprogrammingPromotion.BatchTestRunner = batchTestRunner
	if promotionPort, ok := autoprogrammingPromotion.Port.(serverAutoprogrammingPromotionPortV0); ok {
		autoprogrammingPromotion.GoalWorkspaceIntegration = promotionPort.IntegrationConnector
		autoprogrammingPromotion.BatchIntegrationReceiptDir = promotionPort.IntegrationReceiptDir
	}
	if autoprogrammingPromotion.BatchIntegrationReceiptDir == "" && autoprogrammingPromotion.GoalWorkspaceRoot != "" {
		autoprogrammingPromotion.BatchIntegrationReceiptDir = filepath.Join(autoprogrammingPromotion.GoalWorkspaceRoot, "autoprogramming-integration-receipts-v0")
	}
	batchPromotionFinalizer := newServerAutoprogrammingBatchPromotionFinalizerV0(
		canonicalSelfProgrammingDir,
		autoprogrammingPromotion.BatchIntegrationReceiptDir,
	)
	autoprogrammingPromotion.BatchPromotionFinalizer = batchPromotionFinalizer
	autoprogrammingPromotion.BatchPromotionReconciler = batchPromotionFinalizer
	autoprogrammingPromotion.BatchPromotionReceiptDir = autoprogrammingPromotion.BatchIntegrationReceiptDir
	runStore := orquestacionnucleoapp.RunStorePortV0(stateStore)
	runQueue := orquestarunqueue.RunQueuePortV0(runFileStore)
	runControl := orquestaruncontrol.RunControlPortV0(runFileStore)
	receiptStorePort := orquestaappcodexstack.CodexReceiptStorePortV0(receiptStore)
	appChangeStore := orquestaappchange.AppChangeRecordStorePortV0(runFileStore)
	var appGoalStateStore orquestagoal.GoalWorkStateStorePortV0 = stateStore
	domainDeliveryLedger := domainDeliveryLedgerFromEnvV0(serverConfig, projectConfig)
	goalStateChange := &serverGoalStateChangeRelayV0{}
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
			inner:       stateStore,
			wakeup:      supervisorWakeup,
			stateChange: goalStateChange,
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
	codeContextWiring := codeContextBrokerWiringFromEnvV0(serverConfig, projectConfig)
	toolCapabilities, err := orquestatoolcapabilityfile.NewToolCapabilityFileCatalogV0(
		filepath.Join(serverConfig.StateDir, "tool-capabilities"),
	)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	// Raiz de ingesta documental: un inbox dedicado bajo el estado del servidor.
	// No se monta HOME ni se acepta ruta del host, y no gasta env nueva.
	documentInboxDir := filepath.Join(serverConfig.StateDir, "document-inbox")
	if err := os.MkdirAll(documentInboxDir, 0o700); err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	documentTextExtract, err := newDocumentTextExtractExecutorV0(documentInboxDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	dataInboxDir := filepath.Join(serverConfig.StateDir, "data-inbox")
	if err := os.MkdirAll(dataInboxDir, 0o700); err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	dataProfile, err := newDataProfileExecutorV0(dataInboxDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	councilExecutor, err := newCouncilExecutorV0(serverConfig.StateDir)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	councilExecutor = councilExecutor.withMemberSourceV0(newCouncilMemberSourceV0(
		projectConfig.Council.Members,
		codexUsageMetricsFromProjectConfigFileV0(projectConfig, receiptStore),
	))
	councilConvener, err := newCouncilConvenerV0(serverConfig.StateDir, councilExecutor)
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	stack, err := orquestaappcodexstack.BuildStackV0(orquestaappcodexstack.ConfigV0{
		Enabled:        true,
		Timeout:        30 * time.Second,
		DirectorLimits: directorLimitsV0(),
		Stores: orquestaappcodexstack.StoresV0{
			RunStore:                         runStore,
			EventSink:                        stateStore,
			OutboxLedger:                     outboxLedgerPort,
			TaskStore:                        stateStore,
			WaitStateStore:                   stateStore,
			OperationalPlanStateWriter:       stateStore,
			OperationalPlanStateStore:        stateStore,
			RequiredTestEvidenceStore:        stateStore,
			AppChangeStore:                   appChangeStore,
			ReceiptStore:                     receiptStorePort,
			ProgressState:                    progressStore,
			ProcessRegistry:                  stateStore,
			RunControl:                       runControl,
			RunQueue:                         runQueue,
			AppGoalStateStore:                appGoalStateStore,
			GoalRequiredTestAttestationStore: stateStore,
			AutoprogrammingBatchStore:        stateStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{
			QueueRef:       "global",
			MaxRunsPerTick: serverConfig.SupervisorCommand.MaxRunsPerTick,
			QueueLimit:     serverRunQueueLimitFromProjectConfigFileV0(projectConfig),
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
			codexUsageMetricsFromProjectConfigFileV0(projectConfig, receiptStorePort),
		),
		Gemini:                   geminiRuntimeConfigV0(serverConfig),
		Claude:                   claudeRuntimeConfigV0(serverConfig),
		EgressSanitizer:          egressSanitizer,
		WizardBotAssistant:       serverWizardBotLLMAssistantFromConfigV0(serverConfig.ProjectWorkDir, projectConfig, goalBackend),
		Capacity:                 codexStackCapacityConfigFromProjectConfigFileV0(projectConfig),
		AutonomousDirectorPolicy: orquestacionnucleoapp.HeuristicAutonomousDirectorPolicyV0{},
		AppGoalLauncher: serverGoalWorkLauncherForAppAndAutoprogrammingV0(
			goalBackend,
			autoprogrammingGoalBackend,
		),
		AppGoalReworkLauncher: serverGoalWorkLauncherForAppAndAutoprogrammingV0(
			goalBackend,
			autoprogrammingGoalBackend,
		),
		GoalRequiredTestDependencies: serverGoalRequiredTestDependencyResolverFromConfigV0(serverConfig),
		AppGoalObserver: serverGoalWorkObserverForAppAndAutoprogrammingV0(
			goalBackend,
			autoprogrammingGoalBackend,
		),
		AppGoalRequiredTestSpecBinder:       goalRequiredTestSpecBinder,
		AppGoalRequiredTestSnapshotObserver: goalRequiredTestSnapshotObserver,
		AppGoalRequiredTestAttestor:         goalRequiredTestAttestor,
		AppGoalRequiredTestIdentityVerifier: goalRequiredTestIdentityVerifier,
		GoalRequiredTestClaimPolicy: orquestagoal.GoalRequiredTestAttestationClaimPolicyV0{
			OwnerRef:                serverGoalRequiredTestClaimOwnerRefV0,
			LeaseDurationSeconds:    orquestagoal.GoalRequiredTestAttestationClaimDefaultLeaseSecondsV0,
			ReclaimExpired:          true,
			ReclaimAuthorizationRef: "authorization-ref-orquesta-server-goal-observer-reclaim-v0",
		},
		AppGoalBackendControl: serverGoalBackendControlForAppAndAutoprogrammingV0(
			goalBackend,
			autoprogrammingGoalBackend,
		),
		RuntimeModels:       runtimeModels,
		ToolCapabilities:    toolCapabilities,
		DocumentTextExtract: documentTextExtract,
		DataProfile:         dataProfile,
		Council:             councilExecutor,
		CouncilDoubleReview: orquestaappcodexstack.CouncilDoubleReviewConfigV0{
			Required: projectConfig.Council.DoubleReviewRequired,
			Reviews:  councilExecutor.reviews,
		},
		CouncilGate: orquestaappcodexstack.CouncilGateConfigV0{
			Required: projectConfig.Council.GateRequired,
			Decision: councilExecutor.receipts,
			Convener: councilConvener,
		},
		CouncilPublicErrorClassifier: councilPublicErrorClassifierV0,
		ReviewGate: orquestaappcodexstack.ReviewGateConfigV0{
			FileEvidence:            orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
			StrictGoLineBudget:      boolEnvOrDefaultV0(envReviewGateStrictGoLineBudgetV0, false),
			LineBudgetSnapshotStore: worktreeSnapshotStore,
			SnapshotReadBudget:      codexServerWorktreeSnapshotReadBudgetFromConfigFileV0(projectConfig),
		},
		RequiredTests:                    requiredTestRunner,
		DomainTests:                      domainWorkRequiredTestConfigFromEnvV0(),
		AutoprogrammingPromotion:         autoprogrammingPromotion,
		AutoprogrammingStatusDiagnostics: serverAutoprogrammingStatusDiagnosticsFromEffectiveConfigV0(serverConfig.EffectiveConfig),
		AutoprogrammingIdleSelfImprovementBudgetSource: idleBudgetSource,
		AutoprogrammingGoalProgressPolicy:              serverAutoprogrammingGoalProgressPolicyFromConfigV0(serverConfig),
		ConfigProjectionSettings:                       serverConfigProjectionSettingsForMCPV0(serverConfig),
		DomainWork:                                     domainWorkExecutor,
		CodeContext:                                    codeContextWiring.Query,
		CodeContextToolLeases:                          codeContextWiring.ToolLeases,
		ExternalWorkRunGuard:                           externalWorkRunProjectWorkDirGuardConfigFromEnvV0(serverConfig),
		ExternalWorkRunRuntimeGuard:                    externalWorkRunRuntimeCompatibilityGuardConfigFromServerV0(serverConfig),
		GoalObserverResidentEnabled:                    serverConfig.GoalObserverEnabled,
		PromoteMaterializedArtifactWithoutAck:          codexPromoteMaterializedArtifactWithoutAckFromEnvV0(),
		AllowLegacyAutoprogrammingRun: boolEnvOrDefaultV0(
			envAutoprogrammingLegacyDirectorLoopV0,
			false,
		),
		AllowLegacyExternalWorkRun: boolEnvOrDefaultV0(
			envExternalWorkLegacyDirectorLoopV0,
			false,
		),
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: domainWorkDeliveryEnabledFromProjectConfigFileV0(projectConfig),
			Ledger:  domainDeliveryLedger,
		},
	})
	if err != nil {
		return orquestaappcodexstack.StackV0{}, err
	}
	stack.MCPTransportBindings.AutoprogrammingPrepareRun = guardPrepareRunExecutorWorkdirV0(
		stack.MCPTransportBindings.AutoprogrammingPrepareRun,
		serverConfig.ProjectWorkDir,
	)
	stack.MCPTransportBindings.DocumentPlanExpand = serverDocumentPlanExpandExecutorV0(stack.DomainWork)
	if watcher := serverGoalMaterializedResultWatcherFromStackV0(
		serverConfig,
		stack,
		goalBackend,
		autoprogrammingGoalBackend,
		supervisorWakeup,
	); watcher != nil {
		goalStateChange.bindV0(watcher.NotifyActiveGoalsChangedV0)
		stack.GoalMaterializedResultWatcher = watcher
	}
	if operatorConnector != nil {
		stack.MCPTransportBindings.OperatorConnector = operatorConnector
	}
	stack.MCPTransportBindings.OperatorDirectorMessage = newOperatorDirectorChannelServiceWithMailboxV0(
		operatorConnector,
		newOperatorDirectorChannelMemoryStoreV0(),
		serverConfig.StateDir,
		operatorDirectorMailboxEnabledFromProjectConfigV0(projectConfig),
	)
	stack.MCPTransportBindings.CodebaseStatus = serverCodebaseStatusExecutorWithOwnerMarkersV0(
		stack.MCPTransportBindings.CodebaseStatus,
		codeContextWiring.ToolLeases,
		codeContextWiring.ToolOwnerObserver,
		nil,
	)
	stack.Handler = withFunctionContractRoutesV0(stack.Handler, stateStore)
	resourcesTransferred = true
	return stack, nil
}

// ConfigV0 carries the immutable revision, not the composition-private typed
// document. Consume its retained snapshot and fail closed if it vanished or
// became invalid; providers receive this same value rather than rereading it.
func projectConfigForBuildStackV0(config orquestaserver.ConfigV0) (serverProjectConfigFileV0, error) {
	config = orquestaserver.NormalizeConfigV0(config)
	if loaded, ok := serverProjectConfigStartupLoadV0(config.ProjectWorkDir, config.ProjectConfigFilePath, config.ProjectConfigRevision); ok {
		if err := validateServerProjectConfigLoadV0(loaded); err != nil {
			return serverProjectConfigFileV0{}, err
		}
		return loaded.Config, nil
	}
	if strings.TrimSpace(config.ProjectConfigRevision) != "" {
		return serverProjectConfigFileV0{}, fmt.Errorf("server project config startup snapshot unavailable: %s", config.ProjectConfigRevision)
	}
	if path := strings.TrimSpace(config.ProjectConfigFilePath); path != "" {
		loaded, ok, err := loadServerProjectConfigProductPathV0(path)
		if err != nil || !ok {
			return serverProjectConfigFileV0{}, fmt.Errorf("%s: read", configFileInvalidPublicCodeV0)
		}
		if err := validateServerProjectConfigLoadV0(loaded); err != nil {
			return serverProjectConfigFileV0{}, err
		}
		return loaded.Config, nil
	}
	if strings.TrimSpace(config.ProjectWorkDir) == "" {
		return serverProjectConfigFileV0{}, nil
	}
	path := serverProjectConfigFilePathV0(config.ProjectWorkDir)
	loaded, ok, err := loadServerProjectConfigProductPathV0(path)
	if err != nil || !ok {
		if err != nil {
			return serverProjectConfigFileV0{}, err
		}
		return serverProjectConfigFileV0{}, nil
	}
	if err := validateServerProjectConfigLoadV0(loaded); err != nil {
		return serverProjectConfigFileV0{}, err
	}
	return loaded.Config, nil
}

func serverConfigProjectionSettingsForMCPV0(
	serverConfig orquestaserver.ConfigV0,
) []orquestamcp.MCPConfigProjectionSettingV0 {
	effective := orquestaserver.NormalizeServerEffectiveConfigV0(serverConfig.EffectiveConfig)
	if len(effective.Settings) == 0 {
		return nil
	}
	out := make([]orquestamcp.MCPConfigProjectionSettingV0, 0, len(effective.Settings))
	for _, setting := range effective.Settings {
		if strings.TrimSpace(setting.Key) == "" {
			continue
		}
		out = append(out, orquestamcp.MCPConfigProjectionSettingV0{
			Key:       strings.TrimSpace(setting.Key),
			Value:     strings.TrimSpace(setting.Value),
			Sensitive: setting.Sensitive,
		})
	}
	return out
}

func serverBackgroundWorkersFromStackV0(
	stack orquestaappcodexstack.StackV0,
) []orquestaserver.RuntimeBackgroundWorkerPortV0 {
	if stack.GoalMaterializedResultWatcher == nil {
		return nil
	}
	return []orquestaserver.RuntimeBackgroundWorkerPortV0{stack.GoalMaterializedResultWatcher}
}

func serverGoalMaterializedResultWatcherFromStackV0(
	serverConfig orquestaserver.ConfigV0,
	stack orquestaappcodexstack.StackV0,
	goalBackend serverCodexGoalBackendV0,
	autoprogrammingGoalBackend serverCodexGoalBackendV0,
	supervisorWakeup *serverSupervisorWakeupRelayV0,
) *orquestaappcodexstack.GoalMaterializedResultWatcherV0 {
	serverConfig = orquestaserver.NormalizeConfigV0(serverConfig)
	if supervisorWakeup == nil ||
		!serverConfig.GoalObserverEnabled ||
		stack.Stores.AppGoalStateStore == nil ||
		strings.TrimSpace(stack.Codex.ProjectWorkDir) == "" ||
		goalBackend.Observer == nil {
		return nil
	}
	return orquestaappcodexstack.NewGoalMaterializedResultWatcherV0(
		orquestaappcodexstack.GoalMaterializedResultWatcherConfigV0{
			ProjectWorkDir: stack.Codex.ProjectWorkDir,
			ProjectRootResolver: serverGoalMaterializedResultWorkspaceRootV0{
				CanonicalRoot:   stack.Codex.ProjectWorkDir,
				WorkspaceLookup: serverCodexGoalWorkspaceLookupFromBackendV0(autoprogrammingGoalBackend),
			},
			StateStore: stack.Stores.AppGoalStateStore,
			BeforeWakeup: func(_ context.Context, wakeup orquestaappcodexstack.GoalMaterializedResultWakeupV0) {
				supervisorWakeup.forgetGoalObservationFingerprintV0(wakeup.RunRef)
			},
			Wakeup: func(_ context.Context, wakeup orquestaappcodexstack.GoalMaterializedResultWakeupV0) bool {
				return supervisorWakeup.requestGoalObservationV0(wakeup.Cause)
			},
		},
	)
}

func codexPromoteMaterializedArtifactWithoutAckFromEnvV0() bool {
	return boolEnvOrDefaultV0(envCodexPromoteMaterializedArtifactWithoutAckV0, true)
}

func codexStackCapacityConfigFromEnvV0() orquestaappcodexstack.CapacityConfigV0 {
	return codexStackCapacityConfigFromProjectConfigV0("")
}

func codexStackCapacityConfigFromProjectConfigV0(projectDir string) orquestaappcodexstack.CapacityConfigV0 {
	return codexStackCapacityConfigFromProjectConfigFileV0(projectConfigFromProjectDirBestEffortV0(projectDir))
}

func codexStackCapacityConfigFromProjectConfigFileV0(projectConfig serverProjectConfigFileV0) orquestaappcodexstack.CapacityConfigV0 {
	envConfig := codexStackCapacityEnvConfigFromProjectConfigFileV0(projectConfig)
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
	return codexStackCapacityEnvConfigFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func (supervisor serverStackSupervisorV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request orquestaserver.IdleSelfImprovementRequestFilterRequestV0,
) (orquestaserver.IdleSelfImprovementRequestFilterResultV0, error) {
	out := make([]orquestaserver.IdleSelfImprovementRequestV0, 0, len(request.Requests))
	dropped := 0
	for _, candidate := range request.Requests {
		if serverStackIdleSelfImprovementBacklogRequestAllowedV0(candidate) {
			out = append(out, supervisor.withCuratedSkillRefsV0(candidate))
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
	projectConfigs ...serverProjectConfigFileV0,
) orquestaappcodexstack.DomainWorkArtifactSubmissionLedgerPortV0 {
	projectConfig := serverProjectConfigFileV0{}
	if len(projectConfigs) > 0 {
		projectConfig = projectConfigs[0]
	} else {
		projectConfig = projectConfigFromServerConfigBestEffortV0(serverConfig)
	}
	path := strings.TrimSpace(domainWorkDeliveryLedgerPathFromProjectConfigFileV0(projectConfig))
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

type codexDirectorWaveLimitsEnvConfigV0 struct {
	Agents               int
	MaxSubagentsPerAgent int
	RecursiveAgentBudget int
}

func codexDirectorWaveLimitsEnvConfigFromEnvV0() codexDirectorWaveLimitsEnvConfigV0 {
	return codexDirectorWaveLimitsFromProjectConfigFileV0(serverProjectConfigFileV0{})
}
