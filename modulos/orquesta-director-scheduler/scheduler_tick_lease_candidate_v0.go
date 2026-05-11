package orquestadirectorscheduler

import orquestadirector "orquesta/modulos/orquesta-director"

func (collector *schedulerTickCollectorV0) collectLeaseActionCandidateV0(
	candidate SchedulableLeaseActionCandidateV0,
) error {
	actionInput := candidate.PostLeaseActionInput
	if !collector.leaseAgentKnownV0(actionInput.AgentRequestID) {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if collector.expiredLeaseRefs[actionInput.LeaseRef] {
		return nil
	}
	result, err := orquestadirector.BuildPostLeaseActionV0(actionInput)
	if err != nil {
		return err
	}
	return collector.collectPostLeaseActionResultV0(actionInput, result)
}

func (collector *schedulerTickCollectorV0) leaseAgentKnownV0(agentRef string) bool {
	return collector.agents[agentRef] || collector.startedAgents[agentRef]
}

func (collector *schedulerTickCollectorV0) collectPostLeaseActionResultV0(
	input orquestadirector.PostLeaseActionInputV0,
	result orquestadirector.PostLeaseActionResultV0,
) error {
	switch result.FollowupStatus {
	case orquestadirector.PostLeaseFollowupStatusStopAgentV0:
		collector.addReadyCommandV0(result.RegisterLeaseExpiredCommand)
		if result.StopAgentCommand != nil && collector.agentCanReceiveLeaseStopV0(input.AgentRequestID) {
			collector.addReadyCommandV0(*result.StopAgentCommand)
		}
	case orquestadirector.PostLeaseFollowupStatusAskDirectorV0:
		collector.addReadyCommandV0(result.RegisterLeaseExpiredCommand)
		if result.AskDirectorCommand != nil {
			collector.addReadyCommandV0(*result.AskDirectorCommand)
		}
	default:
		collector.addCommandV0(result.RegisterLeaseExpiredCommand)
		collector.needsDirector = true
	}
	return nil
}

func (collector *schedulerTickCollectorV0) agentCanReceiveLeaseStopV0(agentRef string) bool {
	return !collector.failedAgents[agentRef] && !collector.stoppedAgents[agentRef]
}
