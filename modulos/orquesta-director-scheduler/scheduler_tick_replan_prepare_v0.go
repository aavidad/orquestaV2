package orquestadirectorscheduler

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

type schedulerReplanFollowupStateV0 struct {
	replanRef                 string
	openPhaseID               string
	capacityRef               string
	agentRef                  string
	questionRef               string
	microtaskRefs             []string
	requestOpenPhaseAllowed   bool
	requestMicrotasksAllowed  bool
	requestCapacityAllowed    bool
	requestAgentAllowed       bool
	requestAskDirectorAllowed bool
	waitingCapacity           bool
	waitingQuestion           bool
	followupSettled           bool
	needsDirector             bool
}

func (collector *schedulerTickCollectorV0) prepareReplanFollowupsInputV0(
	input orquestadirector.ReplanFollowupsInputV0,
	replanRecorded bool,
) (orquestadirector.ReplanFollowupsInputV0, schedulerReplanFollowupStateV0) {
	state := schedulerReplanFollowupStateV0{replanRef: input.DecisionPayload.ReplanRef}
	state.capacityRef = replanCapacityRefV0(input)
	state.agentRef = replanAgentRefV0(input)
	action := input.DecisionPayload.AcceptedAction

	collector.prepareReplanOpenPhaseCandidateV0(&input, &state)
	collector.prepareReplanMicrotaskCandidatesV0(&input, &state, action)
	collector.prepareReplanCapacityCandidateV0(&input, &state, action)
	collector.prepareReplanAgentCandidateV0(&input, &state, action)
	collector.prepareReplanAskDirectorCandidateV0(&input, &state, action)
	input.BlockedAgentRefs = collector.replanBlockedAgentRefsV0(input.BlockedAgentRefs)
	return input, state
}

func (collector *schedulerTickCollectorV0) prepareReplanOpenPhaseCandidateV0(
	input *orquestadirector.ReplanFollowupsInputV0,
	state *schedulerReplanFollowupStateV0,
) {
	if input.OpenPhaseCandidate == nil {
		return
	}
	phaseID := input.OpenPhaseCandidate.Payload.PhaseID
	state.openPhaseID = phaseID
	if phaseID == collector.input.Snapshot.CurrentPhaseID {
		input.OpenPhaseCandidate = nil
		return
	}
	state.requestOpenPhaseAllowed = true
}

func (collector *schedulerTickCollectorV0) prepareReplanMicrotaskCandidatesV0(
	input *orquestadirector.ReplanFollowupsInputV0,
	state *schedulerReplanFollowupStateV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) {
	if action != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
		len(input.MicrotaskCandidates) == 0 {
		input.MicrotaskCandidates = nil
		return
	}
	ready := make([]orquestadirector.ReplanMicrotaskCandidateV0, 0, len(input.MicrotaskCandidates))
	allSettled := true
	for _, candidate := range input.MicrotaskCandidates {
		task := candidate.Payload.Task
		taskRef := task.TaskID
		if collector.tasks[taskRef] || collector.plannedTasks[taskRef] {
			continue
		}
		allSettled = false
		if !collector.replanPhaseReadyForFollowupsV0(string(task.PhaseID), state) {
			state.needsDirector = state.openPhaseID == ""
			continue
		}
		ready = append(ready, candidate)
		state.microtaskRefs = append(state.microtaskRefs, taskRef)
	}
	input.MicrotaskCandidates = ready
	state.requestMicrotasksAllowed = len(ready) > 0
	state.followupSettled = allSettled
}

func (collector *schedulerTickCollectorV0) prepareReplanCapacityCandidateV0(
	input *orquestadirector.ReplanFollowupsInputV0,
	state *schedulerReplanFollowupStateV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) {
	if !replanActionRequiresCapacityV0(action) {
		input.CapacityCandidate = nil
		return
	}
	if input.CapacityCandidate == nil {
		state.needsDirector = true
		return
	}
	capacityRef := input.CapacityCandidate.Payload.CapacityRequestID
	state.capacityRef = capacityRef
	if !collector.replanPhaseReadyForFollowupsV0(input.CapacityCandidate.Payload.PhaseID, state) {
		input.CapacityCandidate = nil
		return
	}
	if collector.capacityDecisions[capacityRef] {
		return
	}
	if collector.capacityRequests[capacityRef] || collector.plannedCapacityRequests[capacityRef] {
		state.waitingCapacity = true
		return
	}
	state.requestCapacityAllowed = true
}

func (collector *schedulerTickCollectorV0) prepareReplanAgentCandidateV0(
	input *orquestadirector.ReplanFollowupsInputV0,
	state *schedulerReplanFollowupStateV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) {
	if !replanActionRequiresAgentV0(action) {
		input.AgentCandidate = nil
		return
	}
	if input.AgentCandidate == nil {
		state.needsDirector = true
		return
	}
	agentRef := input.AgentCandidate.Payload.AgentRequestID
	state.agentRef = agentRef
	if !collector.replanPhaseReadyForFollowupsV0(input.AgentCandidate.Payload.PhaseID, state) {
		input.AgentCandidate = nil
		return
	}
	if !collector.capacityDecisions[input.AgentCandidate.Payload.CapacityRequestRef] {
		input.AgentCandidate = nil
		return
	}
	if collector.agentCannotBeReplanFollowupV0(agentRef) {
		collector.markBlockedReplanAgentV0(agentRef)
		state.needsDirector = collector.failedAgents[agentRef] || collector.stoppedAgents[agentRef]
		input.AgentCandidate = nil
		return
	}
	state.requestAgentAllowed = true
}

func (collector *schedulerTickCollectorV0) prepareReplanAskDirectorCandidateV0(
	input *orquestadirector.ReplanFollowupsInputV0,
	state *schedulerReplanFollowupStateV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) {
	if action != orquestacoreworkflow.ReplanDecisionActionAskDirectorV0 {
		input.AskDirectorCandidate = nil
		return
	}
	if input.AskDirectorCandidate == nil {
		state.needsDirector = true
		return
	}
	questionRef := input.AskDirectorCandidate.Payload.QuestionID
	state.questionRef = questionRef
	if collector.directorQuestions[questionRef] && !collector.directorAnsweredQuestions[questionRef] {
		state.waitingQuestion = true
		input.AskDirectorCandidate = nil
		return
	}
	if collector.directorAnsweredQuestions[questionRef] || collector.plannedDirectorQuestions[questionRef] {
		state.followupSettled = true
		input.AskDirectorCandidate = nil
		return
	}
	state.requestAskDirectorAllowed = true
}

func (collector *schedulerTickCollectorV0) replanPhaseReadyForFollowupsV0(
	phaseID string,
	state *schedulerReplanFollowupStateV0,
) bool {
	if phaseID == collector.input.Snapshot.CurrentPhaseID {
		return true
	}
	return state.requestOpenPhaseAllowed && phaseID == state.openPhaseID
}

func (collector *schedulerTickCollectorV0) agentCannotBeReplanFollowupV0(agentRef string) bool {
	return collector.failedAgents[agentRef] ||
		collector.stoppedAgents[agentRef] ||
		collector.agents[agentRef] ||
		collector.startedAgents[agentRef] ||
		collector.plannedAgents[agentRef]
}

func (collector *schedulerTickCollectorV0) markBlockedReplanAgentV0(agentRef string) {
	if collector.failedAgents[agentRef] || collector.stoppedAgents[agentRef] {
		collector.addBlockedRefsV0([]string{agentRef})
		return
	}
	collector.addWaitingV0(SchedulerWaitingAgentLifecyclePendingV0)
}
