package orquestaappcodexstack

import (
	"net/http"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type StackV0 struct {
	Handler                  http.Handler
	MCPTransportBindings     orquestamcp.MCPTransportBindingsV0
	Ports                    orquestaappdirectorservice.StartAppDirectorPortsV0
	Stores                   StoresV0
	RunQueue                 RunQueueConfigV0
	RunSupervisor            RunSupervisorConfigV0
	DirectorLimits           orquestaweb.WebArrancarDirectorAppLimitsV0
	Clock                    orquestafactoryhttp.AppSpecHTTPClockV0
	AutoprogrammingPromotion AutoprogrammingPromotionConfigV0
	DomainWork               orquestamcp.MCPDomainWorkExecutorPortV0
	DomainDelivery           DomainWorkDeliveryBridgeConfigV0
	CodexRuntimeWorkDir      string
}

func BuildStackV0(config ConfigV0) (StackV0, error) {
	if err := validateConfigV0(config); err != nil {
		return StackV0{}, err
	}
	config.DomainDelivery = normalizeDomainWorkDeliveryBridgeConfigV0(config.DomainDelivery)
	ports := buildDirectorPortsV0(config)
	queueConfig := normalizeRunQueueConfigV0(config.RunQueue)
	supervisorConfig := normalizeRunSupervisorConfigV0(config.RunSupervisor)
	stack := StackV0{
		Ports:                    ports,
		Stores:                   config.Stores,
		RunQueue:                 queueConfig,
		RunSupervisor:            supervisorConfig,
		DirectorLimits:           config.DirectorLimits,
		Clock:                    config.Clock,
		AutoprogrammingPromotion: config.AutoprogrammingPromotion,
		DomainWork:               config.DomainWork,
		DomainDelivery:           config.DomainDelivery,
		CodexRuntimeWorkDir:      config.Codex.RuntimeWorkDir,
	}
	stack.MCPTransportBindings = buildStackMCPTransportBindingsV0(config, ports, queueConfig, &stack)
	stack.Handler = buildStackHTTPHandlerV0(config, stack.MCPTransportBindings)
	return stack, nil
}

func buildStackMCPTransportBindingsV0(
	config ConfigV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	queueConfig RunQueueConfigV0,
	stack *StackV0,
) orquestamcp.MCPTransportBindingsV0 {
	arrancar := NewQueuedArrancarDirectorExecutorV0(QueuedArrancarDirectorConfigV0{
		Inner:  orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(startOnlyPortsV0(ports)),
		Writer: config.Stores.RunQueue,
		Queue:  queueConfig,
		Clock:  config.Clock,
		Source: "orquesta-app-codex-stack",
		Reason: "run creado desde director app",
	})
	return orquestamcp.MCPTransportBindingsV0{
		ArrancarDirector: arrancar,
		RequestAppChange: orquestamcp.NewMCPRequestAppChangeToolExecutorV0(appChangePortsV0(config)),
		DirectorStats: orquestamcp.MCPDirectorStatsToolExecutorV0{
			RunStore:          config.Stores.RunStore,
			ProcessRegistry:   config.Stores.ProcessRegistry,
			ProgressSource:    statsProgressSourceV0(config),
			AgentUsageSource:  agentUsageSourceV0(config),
			ExternalJobSource: externalJobStatsSourceV0(config),
		},
		RunControl: orquestamcp.MCPRunControlToolExecutorV0{
			Port:              config.Stores.RunControl,
			ExternalJobSource: externalJobStatsSourceV0(config),
		},
		RunQueuePriority: orquestamcp.MCPRunQueuePriorityToolExecutorV0{
			Reader: config.Stores.RunQueue,
			Writer: config.Stores.RunQueue,
		},
		RunSupervisor: NewCodexStackRunSupervisorExecutorV0(stack),
		AppVCS:        NewCodexStackAppVCSExecutorV0(config.Codex.ProjectWorkDir),
		AutoprogrammingPrepareRun: NewCodexStackAutoprogrammingPrepareRunExecutorV0(
			stack,
			config.Capacity.OccurredAt,
			config.Capacity.RequestedBy,
			config.Stores.RunQueue,
			queueConfig,
			config.Clock,
			config.Codex.RuntimeWorkDir,
		),
		ServerShutdown: serverShutdownExecutorV0(config, stack),
		DomainWork:     config.DomainWork,
		ExternalWorkRun: externalWorkRunExecutorV0(
			config,
			queueConfig,
		),
	}
}

func buildStackHTTPHandlerV0(
	config ConfigV0,
	bindings orquestamcp.MCPTransportBindingsV0,
) http.Handler {
	handler := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Clock:                     config.Clock,
		ArrancarDirector:          bindings.ArrancarDirector,
		RequestAppChange:          bindings.RequestAppChange,
		DirectorStats:             bindings.DirectorStats,
		RunControl:                bindings.RunControl,
		RunQueuePriority:          bindings.RunQueuePriority,
		RunSupervisor:             bindings.RunSupervisor,
		AppVCS:                    bindings.AppVCS,
		AutoprogrammingPrepareRun: bindings.AutoprogrammingPrepareRun,
		ServerShutdown:            bindings.ServerShutdown,
		DomainWork:                bindings.DomainWork,
		ExternalWorkRun:           bindings.ExternalWorkRun,
		OpsAgentRuntimeDetail: NewCodexStackAgentRuntimeDetailHTTPHandlerV0(CodexStackAgentRuntimeDetailConfigV0{
			ReceiptStore:   config.Stores.ReceiptStore,
			RuntimeWorkDir: config.Codex.RuntimeWorkDir,
		}),
		Timeout:        config.Timeout,
		DirectorLimits: config.DirectorLimits,
	})
	return handler
}

func buildDirectorPortsV0(
	config ConfigV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	return orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                    config.Stores.RunStore,
		EventSink:                   ackRuntimeCleanupSinkV0(config),
		EventReader:                 eventReaderV0(config),
		OutboxLedger:                config.Stores.OutboxLedger,
		DeliverySource:              deliverySourceV0(config),
		ReviewGateSource:            reviewGateSourceV0(config),
		ReviewReworkReplanSource:    reviewReworkReplanSourceV0(config),
		AssessmentReplanSource:      assessmentReplanSourceV0(config),
		ProgressSource:              progressSourceV0(config),
		RunControl:                  config.Stores.RunControl,
		RunControlTerminal:          config.Stores.RunControl,
		DirectorDecisionSource:      directorDecisionSourceV0(config),
		DirectorTaskStore:           config.Stores.TaskStore,
		WorkflowTaskDefaultCapacity: config.Capacity.Tier,
		WaitStateWriter:             waitStateWriterV0(config),
		WaitStateStore:              waitStateStoreV0(config),
		OperationalPlanStateWriter:  operationalPlanStateWriterV0(config),
		OperationalPlanStateStore:   operationalPlanStateStoreV0(config),
		RequiredTestEvidenceStore:   requiredTestEvidenceStoreV0(config),
		RequiredTestRunner:          requiredTestRunnerV0(config),
		ExternalWaiter:              ackWaiterV0(config),
		OperationalClosureSource:    operationalClosureSourceV0(config),
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			capacityDispatcherV0(config),
			stopperDispatcherV0(config),
			directorQuestionDispatcherV0(config),
		},
		BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{
			agentBatchDispatcherV0(config),
		},
	}
}

func requiredTestEvidenceStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.RequiredTestEvidenceStorePortV0 {
	if config.Stores.RequiredTestEvidenceStore != nil {
		return config.Stores.RequiredTestEvidenceStore
	}
	store, _ := config.Stores.TaskStore.(orquestacionnucleoapp.RequiredTestEvidenceStorePortV0)
	return store
}

func waitStateWriterV0(
	config ConfigV0,
) orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0 {
	writer, _ := config.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0)
	return writer
}

func waitStateStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0 {
	if config.Stores.WaitStateStore != nil {
		return config.Stores.WaitStateStore
	}
	store, _ := config.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0)
	return store
}

func operationalPlanStateWriterV0(
	config ConfigV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0 {
	if config.Stores.OperationalPlanStateWriter != nil {
		return config.Stores.OperationalPlanStateWriter
	}
	writer, _ := config.Stores.TaskStore.(orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0)
	return writer
}

func operationalPlanStateStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0 {
	if config.Stores.OperationalPlanStateStore != nil {
		return config.Stores.OperationalPlanStateStore
	}
	store, _ := config.Stores.OperationalPlanStateWriter.(orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0)
	if store != nil {
		return store
	}
	store, _ = config.Stores.TaskStore.(orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0)
	return store
}

func ackRuntimeCleanupSinkV0(config ConfigV0) orquestacionnucleoapp.EventSinkPortV0 {
	return ackRuntimeCleanupEventSinkV0{
		Inner: config.Stores.EventSink,
		Stopper: orquestacionnucleoapp.ProcessAgentStopperV0{
			Registry: config.Stores.ProcessRegistry,
			Runtime:  config.Codex.ProcessStopper,
		},
	}
}

func startOnlyPortsV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	ports.DirectorDecisionSource = nil
	ports.ExternalWaiter = nil
	return ports
}
