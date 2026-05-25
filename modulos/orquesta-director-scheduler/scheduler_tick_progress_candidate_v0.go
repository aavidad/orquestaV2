package orquestadirectorscheduler

import (
	"encoding/json"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func (collector *schedulerTickCollectorV0) collectProgressSupervisionCandidateV0(
	candidate SchedulableProgressSupervisionCandidateV0,
) error {
	input := schedulerProgressBudgetPreparedInputV0(candidate.SupervisionInput)
	agentRef := input.Report.AgentRequestID
	if !collector.progressAgentKnownV0(agentRef) {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if collector.failedAgents[agentRef] {
		collector.addBlockedRefsV0([]string{agentRef})
		return nil
	}
	if collector.lostAgents[agentRef] {
		return nil
	}
	if collector.stoppedAgents[agentRef] {
		collector.addInFlightAgentWaitIfAnyV0()
		return nil
	}
	if collector.progressQuestionPendingV0(input) {
		collector.addWaitingV0(SchedulerWaitingDirectorQuestionPendingV0)
		return nil
	}

	result, err := orquestadirector.BuildAgentProgressSupervisionV0(
		schedulerProgressSupervisionCommandInputV0(input),
	)
	if err != nil {
		return schedulerTickWrappedExternalErrorV0("progress_supervision_candidates", err)
	}
	collector.collectProgressSupervisionResultV0(input, result)
	return nil
}

func schedulerProgressSupervisionCommandInputV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	return input
}

func (collector *schedulerTickCollectorV0) collectProgressSupervisionResultV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
	result orquestadirector.AgentProgressSupervisionResultV0,
) {
	assessmentRef := input.AssessmentRef
	if collector.progressAssessmentCommandAllowedV0(input, result.AssessCommand) {
		collector.addReadyCommandV0(result.AssessCommand)
		collector.plannedAgentAssessments[assessmentRef] = true
	}
	if result.RegisterLostCommand != nil && collector.progressLostCommandAllowedV0(input) {
		collector.addReadyCommandV0(*result.RegisterLostCommand)
		collector.plannedLostAgents[input.Report.AgentRequestID] = true
	}
	if result.AskDirectorCommand != nil && collector.progressQuestionCommandAllowedV0(input.QuestionID) {
		collector.addReadyCommandV0(*result.AskDirectorCommand)
		collector.plannedDirectorQuestions[input.QuestionID] = true
	}
}

func (collector *schedulerTickCollectorV0) progressAgentKnownV0(agentRef string) bool {
	return collector.agents[agentRef] || collector.startedAgents[agentRef]
}

func (collector *schedulerTickCollectorV0) progressLostCommandAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	agentRef := input.Report.AgentRequestID
	return agentRef != "" &&
		!collector.lostAgents[agentRef] &&
		!collector.plannedLostAgents[agentRef] &&
		!collector.stoppedAgents[agentRef]
}

func schedulerProgressCandidateCanYieldToReplanV0(
	input DirectorSchedulerTickInputV0,
	candidate SchedulableProgressSupervisionCandidateV0,
) bool {
	if len(input.ReplanFollowupCandidates) == 0 {
		return false
	}
	agentRef := candidate.SupervisionInput.Report.AgentRequestID
	if agentRef == "" {
		return false
	}
	agents := schedulerStringSetV0(input.Snapshot.Agents)
	started := schedulerStringSetV0(input.Snapshot.StartedAgents)
	failed := schedulerStringSetV0(input.Snapshot.FailedAgents)
	stopped := schedulerStringSetV0(input.Snapshot.StoppedAgents)
	return (agents[agentRef] || started[agentRef]) &&
		!failed[agentRef] &&
		stopped[agentRef]
}

func (collector *schedulerTickCollectorV0) progressAssessmentCommandAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) bool {
	if !collector.progressAssessmentDoneV0(input.AssessmentRef) {
		return true
	}
	return progressSupervisionAssessmentStopsAgentV0(command) &&
		!collector.stoppedAgents[input.Report.AgentRequestID]
}

func progressSupervisionStopAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return input.StopAllowed == nil || *input.StopAllowed
}

func progressSupervisionAssessmentStopsAgentV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
) bool {
	if command.CommandType != orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0 {
		return false
	}
	var payload orquestacoreworkflow.AssessAgentWorkCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return false
	}
	return payload.Action == orquestacoreworkflow.AgentAssessmentActionStopAgentV0
}

func (collector *schedulerTickCollectorV0) progressAssessmentDoneV0(assessmentRef string) bool {
	return collector.agentAssessments[assessmentRef] || collector.plannedAgentAssessments[assessmentRef]
}

func (collector *schedulerTickCollectorV0) progressQuestionCommandAllowedV0(questionRef string) bool {
	return questionRef != "" &&
		!collector.directorQuestions[questionRef] &&
		!collector.directorAnsweredQuestions[questionRef] &&
		!collector.plannedDirectorQuestions[questionRef]
}

func (collector *schedulerTickCollectorV0) progressQuestionPendingV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	// Las preguntas de supervision de progreso se emiten como advisory
	// blocking=false. No deben parar el loop: si el director humano responde,
	// esa respuesta se incorporara; si no, la orquestacion debe poder seguir
	// reintentando, replanificando o cerrando segun la evidencia siguiente.
	return false
}
