package orquestaappcodexstack

import (
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
)

func legacyAppRunnerPortsFromDirectorPortsV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) orquestaapprunner.RunPreparedAppOrchestrationPortsV0 {
	return orquestaapprunner.RunPreparedAppOrchestrationPortsV0{
		RunStore: ports.RunStore, EventSink: ports.EventSink,
		OutboxLedger: ports.OutboxLedger, Dispatchers: ports.Dispatchers, BatchDispatchers: ports.BatchDispatchers,
		DeliverySource: ports.DeliverySource, ReviewGateSource: ports.ReviewGateSource,
		ProgressSource: ports.ProgressSource, LeaseSource: ports.LeaseSource,
		AssessmentReplanSource: ports.AssessmentReplanSource, ExternalWaiter: ports.ExternalWaiter,
		AutonomousDirectorPolicy: ports.AutonomousDirectorPolicy,
	}
}
