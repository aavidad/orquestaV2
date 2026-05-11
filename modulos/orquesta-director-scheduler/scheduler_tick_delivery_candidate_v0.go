package orquestadirectorscheduler

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func (collector *schedulerTickCollectorV0) collectDeliveryCandidateV0(
	candidate SchedulableDeliveryCandidateV0,
) error {
	payload := candidate.Payload
	deliveryRef := payload.DeliveryRef
	if collector.deliveries[deliveryRef] || collector.plannedDeliveries[deliveryRef] {
		return nil
	}
	if !collector.tasks[payload.TaskID] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if !collector.agents[payload.AgentRef] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if collector.failedAgents[payload.AgentRef] || collector.stoppedAgents[payload.AgentRef] {
		collector.addBlockedRefsV0([]string{payload.AgentRef})
		return nil
	}
	if !collector.startedAgents[payload.AgentRef] {
		collector.addWaitingV0(SchedulerWaitingAgentLifecyclePendingV0)
		return nil
	}
	command, err := orquestacoreworkflow.NewRegisterDeliveryCommandV0(
		candidate.CommandMeta,
		payload,
	)
	if err != nil {
		return err
	}
	collector.addReadyCommandV0(command)
	collector.plannedDeliveries[deliveryRef] = true
	return nil
}
