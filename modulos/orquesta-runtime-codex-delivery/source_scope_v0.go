package orquestaruntimecodexdelivery

import orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"

func codexDeliveryObservationAgentEligibleV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
	agentRef string,
) bool {
	if !codexDeliveryObservationAgentMatchesWaitAgentRefsV0(request, agentRef) {
		return false
	}
	return (stringInCodexDeliverySetV0(request.Run.Agents, agentRef) ||
		stringInCodexDeliverySetV0(request.Run.StartedAgents, agentRef)) &&
		!stringInCodexDeliverySetV0(request.Run.FailedAgents, agentRef)
}

func codexDeliveryObservationDescriptorMatchesWaitAgentRefsV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
) bool {
	return codexDeliveryObservationAgentMatchesWaitAgentRefsV0(
		request,
		codexReceiptDescriptorAgentRefV0(descriptor),
	)
}

func codexDeliveryObservationAgentMatchesWaitAgentRefsV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
	agentRef string,
) bool {
	waitAgentRefs := compactCodexDeliveryRefsV0(request.WaitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return true
	}
	return stringInCodexDeliverySetV0(waitAgentRefs, agentRef)
}
