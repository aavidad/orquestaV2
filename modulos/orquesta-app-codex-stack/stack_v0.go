package orquestaappcodexstack

import (
	"net/http"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type StackV0 struct {
	Handler        http.Handler
	Ports          orquestaappdirectorservice.StartAppDirectorPortsV0
	Stores         StoresV0
	RunQueue       RunQueueConfigV0
	RunSupervisor  RunSupervisorConfigV0
	DirectorLimits orquestaweb.WebArrancarDirectorAppLimitsV0
	Clock          orquestafactory.AppSpecHTTPClockV0
	DomainWork     orquestamcp.MCPDomainWorkExecutorPortV0
	DomainDelivery DomainWorkDeliveryBridgeConfigV0
}

func BuildStackV0(config ConfigV0) (StackV0, error) {
	if err := validateConfigV0(config); err != nil {
		return StackV0{}, err
	}
	ports := buildDirectorPortsV0(config)
	queueConfig := normalizeRunQueueConfigV0(config.RunQueue)
	supervisorConfig := normalizeRunSupervisorConfigV0(config.RunSupervisor)
	stack := StackV0{
		Ports:          ports,
		Stores:         config.Stores,
		RunQueue:       queueConfig,
		RunSupervisor:  supervisorConfig,
		DirectorLimits: config.DirectorLimits,
		Clock:          config.Clock,
		DomainWork:     config.DomainWork,
		DomainDelivery: normalizeDomainWorkDeliveryBridgeConfigV0(config.DomainDelivery),
	}
	stack.Handler = buildStackHTTPHandlerV0(config, ports, queueConfig, &stack)
	return stack, nil
}

func buildStackHTTPHandlerV0(
	config ConfigV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	queueConfig RunQueueConfigV0,
	stack *StackV0,
) http.Handler {
	arrancar := NewQueuedArrancarDirectorExecutorV0(QueuedArrancarDirectorConfigV0{
		Inner:  orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(startOnlyPortsV0(ports)),
		Writer: config.Stores.RunQueue,
		Queue:  queueConfig,
		Clock:  config.Clock,
		Source: "orquesta-app-codex-stack",
		Reason: "run creado desde director app",
	})
	handler := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Clock:            config.Clock,
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
		ServerShutdown: serverShutdownExecutorV0(config, stack),
		DomainWork:     config.DomainWork,
		Timeout:        config.Timeout,
		DirectorLimits: config.DirectorLimits,
	})
	return handler
}

func buildDirectorPortsV0(
	config ConfigV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	return orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                 config.Stores.RunStore,
		EventSink:                ackRuntimeCleanupSinkV0(config),
		OutboxLedger:             config.Stores.OutboxLedger,
		DeliverySource:           deliverySourceV0(config),
		ReviewGateSource:         reviewGateSourceV0(config),
		ReviewReworkReplanSource: reviewReworkReplanSourceV0(config),
		AssessmentReplanSource:   assessmentReplanSourceV0(config),
		ProgressSource:           progressSourceV0(config),
		RunControl:               config.Stores.RunControl,
		RunControlTerminal:       config.Stores.RunControl,
		DirectorDecisionSource:   directorDecisionSourceV0(config),
		DirectorTaskStore:        config.Stores.TaskStore,
		ExternalWaiter:           ackWaiterV0(config),
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
