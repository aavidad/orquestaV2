package orquestaappcodexstack

import (
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func externalWorkRunExecutorV0(
	config ConfigV0,
	queueConfig RunQueueConfigV0,
) orquestamcp.MCPTransportExternalWorkRunExecutorV0 {
	queueConfig = normalizeRunQueueConfigV0(queueConfig)
	legacy := orquestamcp.NewMCPExternalWorkRunToolExecutorV0(
		orquestaexternalworkrun.StartExternalWorkRunPortsV0{
			RunStore:  config.Stores.RunStore,
			EventSink: config.Stores.EventSink,
			RunQueue:  config.Stores.RunQueue,
			AppChange: appChangePortsV0(config),
		},
		externalWorkRunStartConfigV0(config, queueConfig),
	)
	ports := buildDirectorPortsV0(config)
	return NewCodexStackExternalWorkGoalFirstExecutorV0(
		legacy,
		ports,
		externalWorkRunStartConfigV0(config, queueConfig),
		config.Stores.AppChangeStore,
		config.GoalObserverResidentEnabled,
		config.AllowLegacyExternalWorkRun,
	)
}

func externalWorkRunStartConfigV0(
	config ConfigV0,
	queueConfig RunQueueConfigV0,
) orquestaexternalworkrun.StartExternalWorkRunConfigV0 {
	queueConfig = normalizeRunQueueConfigV0(queueConfig)
	return orquestaexternalworkrun.StartExternalWorkRunConfigV0{
		QueueRef:             queueConfig.QueueRef,
		DefaultPriorityScore: queueConfig.DefaultPriorityScore,
		OccurredAt:           config.Capacity.OccurredAt,
		RequestedBy:          config.Capacity.RequestedBy,
	}
}
