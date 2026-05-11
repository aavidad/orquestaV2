package orquestadirectoragentworkflow

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func buildDirectorAgentAskDirectorCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	return buildDirectorAgentAskQuestionCommandV0(request, request.Decision.AskDirector)
}

func buildDirectorAgentAskUserCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	return buildDirectorAgentAskQuestionCommandV0(request, request.Decision.AskUser)
}

func buildDirectorAgentAskQuestionCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
	payload *orquestadirectoragent.DirectorAgentAskQuestionCommandV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	command, err := orquestacoreworkflow.NewAskDirectorCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.AskDirectorCommandPayloadV0{
			QuestionID:   payload.QuestionID,
			SourceGroup:  payload.SourceGroup,
			TargetGroup:  payload.TargetGroup,
			Summary:      payload.Summary,
			Options:      payload.Options,
			EvidenceRefs: payload.EvidenceRefs,
			Blocking:     payload.Blocking,
		},
	)
	if err != nil {
		return invalidDirectorAgentWorkflowCommandV0()
	}
	return command, nil
}

func buildDirectorAgentRequestCapacityCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestCapacity
	command, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          payload.CapacityRequestID,
			PhaseID:                    payload.PhaseID,
			TaskRef:                    payload.TaskRef,
			ReasonCode:                 payload.ReasonCode,
			Summary:                    payload.Summary,
			MinimumRecommendedCapacity: directorAgentCapacityV0(payload.MinimumRecommendedCapacity),
			EvidenceRefs:               payload.EvidenceRefs,
		},
	)
	if err != nil {
		return invalidDirectorAgentWorkflowCommandV0()
	}
	return command, nil
}

func buildDirectorAgentRequestAgentCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestAgent
	command, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     payload.AgentRequestID,
			PhaseID:            payload.PhaseID,
			TaskRef:            payload.TaskRef,
			CapacityRequestRef: payload.CapacityRequestRef,
			Role:               payload.Role,
			Summary:            payload.Summary,
			EvidenceRefs:       payload.EvidenceRefs,
		},
	)
	if err != nil {
		return invalidDirectorAgentWorkflowCommandV0()
	}
	return command, nil
}

func buildDirectorAgentRequestReworkCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestRework
	command, err := orquestacoreworkflow.NewRequestReworkCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestReworkCommandPayloadV0{
			ReworkRequestRef: payload.ReworkRequestRef,
			PhaseID:          payload.PhaseID,
			ReviewResultRef:  payload.ReviewResultRef,
			ReviewRequestID:  payload.ReviewRequestID,
			DeliveryRef:      payload.DeliveryRef,
			Summary:          payload.Summary,
			EvidenceRefs:     payload.EvidenceRefs,
		},
	)
	if err != nil {
		return invalidDirectorAgentWorkflowCommandV0()
	}
	return command, nil
}

func buildDirectorAgentRecordReplanDecisionCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RecordReplanDecision
	command, err := orquestacoreworkflow.NewRecordReplanDecisionCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      payload.ReplanRef,
			RunRef:         payload.RunRef,
			TaskRef:        payload.TaskRef,
			SourceRef:      payload.SourceRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionV0(payload.AcceptedAction),
			FollowupRefs:   payload.FollowupRefs,
			Summary:        payload.Summary,
			EvidenceRefs:   payload.EvidenceRefs,
		},
	)
	if err != nil {
		return invalidDirectorAgentWorkflowCommandV0()
	}
	return command, nil
}

func invalidDirectorAgentWorkflowCommandV0() (
	orquestacoreworkflow.OrchestrationCommandV0,
	[]DirectorAgentWorkflowIssueV0,
) {
	return orquestacoreworkflow.OrchestrationCommandV0{},
		[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
}
