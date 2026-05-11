package orquestaapprunner

import (
	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func basePreparedAppProviderV0(
	request RunPreparedAppOrchestrationRequestV0,
) orquestacionnucleoapp.CandidateProviderPortV0 {
	return orquestaappplanner.AppPlanCandidateProviderV0{
		Plan:        request.Prepared.Plan,
		RequestedBy: request.RequestedBy,
	}
}

func composePreparedAppProviderV0(
	base orquestacionnucleoapp.CandidateProviderPortV0,
	ports RunPreparedAppOrchestrationPortsV0,
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
	return provider
}
