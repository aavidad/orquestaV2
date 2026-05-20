package orquestadirectorscheduler

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func (collector *schedulerTickCollectorV0) collectReplanFollowupCandidateV0(
	candidate SchedulableReplanFollowupCandidateV0,
) error {
	input := candidate.ReplanFollowupsInput
	replanRecorded := collector.replanAlreadyRecordedV0(input.DecisionPayload.ReplanRef)
	prepared, state := collector.prepareReplanFollowupsInputV0(input, replanRecorded)
	if state.waitingCapacity {
		collector.addWaitingV0(SchedulerWaitingCapacityPendingV0)
		return nil
	}
	if state.waitingQuestion {
		collector.addWaitingV0(SchedulerWaitingDirectorQuestionPendingV0)
	}

	result, err := orquestadirector.BuildReplanFollowupsV0(prepared)
	if err != nil {
		return schedulerTickWrappedExternalErrorV0("replan_followup_candidates", err)
	}
	return collector.collectReplanFollowupsResultV0(result, replanRecorded, state)
}

func (collector *schedulerTickCollectorV0) replanAlreadyRecordedV0(replanRef string) bool {
	return collector.replanRefs[replanRef] || collector.plannedReplanRefs[replanRef]
}

func (collector *schedulerTickCollectorV0) collectReplanFollowupsResultV0(
	result orquestadirector.ReplanFollowupsResultV0,
	replanRecorded bool,
	state schedulerReplanFollowupStateV0,
) error {
	followups := collector.readyReplanFollowupCommandsV0(result, state)
	if !replanRecorded {
		if len(followups) > 0 {
			collector.addReadyCommandV0(result.RecordReplanDecisionCommand)
		} else {
			collector.addCommandV0(result.RecordReplanDecisionCommand)
		}
		collector.plannedReplanRefs[state.replanRef] = true
	}
	for _, command := range followups {
		collector.addReadyCommandV0(command)
	}
	if state.needsDirector || (result.FollowupStatus == orquestadirector.ReplanFollowupStatusNeedsDirectorUnsupportedV0 &&
		!state.waitingQuestion && !state.followupSettled) {
		collector.needsDirector = true
	}
	return nil
}

func (collector *schedulerTickCollectorV0) readyReplanFollowupCommandsV0(
	result orquestadirector.ReplanFollowupsResultV0,
	state schedulerReplanFollowupStateV0,
) []orquestacoreworkflow.OrchestrationCommandV0 {
	var commands []orquestacoreworkflow.OrchestrationCommandV0
	if result.OpenPhaseCommand != nil && state.requestOpenPhaseAllowed {
		commands = append(commands, *result.OpenPhaseCommand)
	}
	if len(result.CreateMicrotaskCommands) > 0 && state.requestMicrotasksAllowed {
		commands = append(commands, result.CreateMicrotaskCommands...)
		for _, ref := range state.microtaskRefs {
			collector.plannedTasks[ref] = true
		}
	}
	if result.RequestCapacityCommand != nil && state.requestCapacityAllowed {
		commands = append(commands, *result.RequestCapacityCommand)
		collector.plannedCapacityRequests[state.capacityRef] = true
	}
	if result.RequestAgentCommand != nil && state.requestAgentAllowed {
		commands = append(commands, *result.RequestAgentCommand)
		collector.plannedAgents[state.agentRef] = true
	}
	if result.AskDirectorCommand != nil && state.requestAskDirectorAllowed {
		commands = append(commands, *result.AskDirectorCommand)
		collector.plannedDirectorQuestions[state.questionRef] = true
	}
	return commands
}
