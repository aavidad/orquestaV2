package orquestadirectorscheduler

import (
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
		return err
	}
	collector.collectProgressSupervisionResultV0(input, result)
	return nil
}

func schedulerProgressSupervisionCommandInputV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	input.Report.EvidenceRefs = nil
	return input
}

func (collector *schedulerTickCollectorV0) collectProgressSupervisionResultV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
	result orquestadirector.AgentProgressSupervisionResultV0,
) {
	assessmentRef := input.AssessmentRef
	if collector.progressAssessmentCommandAllowedV0(input) {
		collector.addReadyCommandV0(result.AssessCommand)
		collector.plannedAgentAssessments[assessmentRef] = true
	}
	if result.AskDirectorCommand != nil && collector.progressQuestionCommandAllowedV0(input.QuestionID) {
		collector.addReadyCommandV0(*result.AskDirectorCommand)
		collector.plannedDirectorQuestions[input.QuestionID] = true
	}
}

func (collector *schedulerTickCollectorV0) progressAgentKnownV0(agentRef string) bool {
	return collector.agents[agentRef] || collector.startedAgents[agentRef]
}

func (collector *schedulerTickCollectorV0) progressAssessmentCommandAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	if !collector.progressAssessmentDoneV0(input.AssessmentRef) {
		return true
	}
	return input.Report.Status == orquestaruntime.AgentLoopDetectedV0 &&
		progressSupervisionStopAllowedV0(input) &&
		!collector.stoppedAgents[input.Report.AgentRequestID]
}

func progressSupervisionStopAllowedV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) bool {
	return input.StopAllowed == nil || *input.StopAllowed
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
	questionRef := input.QuestionID
	return schedulerProgressBudgetNeedsDirectorQuestionV0(input) &&
		collector.progressAssessmentDoneV0(input.AssessmentRef) &&
		collector.directorQuestions[questionRef] &&
		!collector.directorAnsweredQuestions[questionRef]
}
