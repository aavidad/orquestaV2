package orquestadirectoragentworkflow

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func buildDirectorAgentCloseTaskCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.CloseTask
	command, err := orquestacoreworkflow.NewCloseTaskCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.CloseTaskCommandPayloadV0{
			TaskID:            payload.TaskID,
			PhaseID:           payload.PhaseID,
			DeliveryRef:       payload.DeliveryRef,
			AcceptedReviewRef: payload.AcceptedReviewRef,
			Summary:           payload.Summary,
			EvidenceRefs:      payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentFinalValidationCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RegisterFinalValidation
	command, err := orquestacoreworkflow.NewRegisterFinalValidationCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RegisterFinalValidationCommandPayloadV0{
			ValidationRef: payload.ValidationRef,
			PhaseID:       payload.PhaseID,
			ClosedTaskRef: payload.ClosedTaskRef,
			Summary:       payload.Summary,
			EvidenceRefs:  payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentCloseRunCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.CloseRun
	command, err := orquestacoreworkflow.NewCloseRunCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.CloseRunCommandPayloadV0{
			ClosureRef:    payload.ClosureRef,
			PhaseID:       payload.PhaseID,
			ValidationRef: payload.ValidationRef,
			Summary:       payload.Summary,
			EvidenceRefs:  payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentAnswerQuestionCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.AnswerQuestion
	command, err := orquestacoreworkflow.NewAnswerDirectorQuestionCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.AnswerDirectorQuestionCommandPayloadV0{
			AnswerID:     payload.AnswerID,
			QuestionID:   payload.QuestionID,
			Decision:     payload.Decision,
			Summary:      payload.Summary,
			EvidenceRefs: payload.EvidenceRefs,
			Unblocks:     payload.Unblocks,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}
