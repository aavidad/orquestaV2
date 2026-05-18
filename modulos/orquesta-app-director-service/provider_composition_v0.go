package orquestaappdirectorservice

import orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"

func composeStartAppDirectorProviderV0(
	base orquestacionnucleoapp.CandidateProviderPortV0,
	ports StartAppDirectorPortsV0,
	requestedBy string,
) orquestacionnucleoapp.CandidateProviderPortV0 {
	provider := base
	if ports.DeliverySource != nil {
		provider = orquestacionnucleoapp.DeliveryCandidateProviderV0{
			Base:           provider,
			DeliverySource: ports.DeliverySource,
			RequestedBy:    requestedBy,
		}
	}
	if ports.ReviewGateSource != nil {
		provider = orquestacionnucleoapp.ReviewGateCandidateProviderV0{
			Base:        provider,
			GateSource:  ports.ReviewGateSource,
			RequestedBy: requestedBy,
		}
	}
	if ports.ReviewReworkReplanSource != nil {
		provider = orquestacionnucleoapp.ReviewReworkReplanCandidateProviderV0{
			Base:        provider,
			PlanSource:  ports.ReviewReworkReplanSource,
			TaskWriter:  ports.DirectorTaskStore,
			RequestedBy: requestedBy,
		}
	}
	if ports.ProgressSource != nil {
		provider = orquestacionnucleoapp.ProgressSupervisionCandidateProviderV0{
			Base:           provider,
			ProgressSource: ports.ProgressSource,
			RequestedBy:    requestedBy,
		}
	}
	if ports.LeaseSource != nil {
		provider = orquestacionnucleoapp.AgentLeaseActionCandidateProviderV0{
			Base:        provider,
			LeaseSource: ports.LeaseSource,
			RequestedBy: requestedBy,
		}
	}
	if ports.AssessmentReplanSource != nil {
		provider = orquestacionnucleoapp.AgentAssessmentReplanCandidateProviderV0{
			Base:        provider,
			PlanSource:  ports.AssessmentReplanSource,
			RequestedBy: requestedBy,
		}
	}
	if ports.DirectorTaskStore != nil {
		provider = orquestacionnucleoapp.WorkflowTaskCandidateProviderV0{
			Base:            provider,
			TaskStore:       ports.DirectorTaskStore,
			RequestedBy:     requestedBy,
			DefaultCapacity: ports.WorkflowTaskDefaultCapacity,
		}
	}
	return provider
}
