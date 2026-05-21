package orquestadirectoragentworkflow

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func BuildDirectorAgentWorkflowCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	request = normalizeDirectorAgentWorkflowRequestV0(request)
	if issues := validateDirectorAgentWorkflowRequestV0(request); len(issues) > 0 {
		return orquestacoreworkflow.OrchestrationCommandV0{}, issues
	}
	switch request.Decision.CommandType {
	case orquestadirectoragent.DirectorAgentCommandRequestBrainstormV0:
		return buildDirectorAgentBrainstormCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandOpenPhaseV0:
		return buildDirectorAgentOpenPhaseCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRequestVoteV0:
		return buildDirectorAgentVoteCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0:
		return buildDirectorAgentAcceptDecisionCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandPublishContractV0:
		return buildDirectorAgentPublishContractCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0:
		return buildDirectorAgentCreateMicrotaskCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandAskDirectorV0:
		return buildDirectorAgentAskDirectorCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandAskUserV0:
		return buildDirectorAgentAskUserCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRequestCapacityV0:
		return buildDirectorAgentRequestCapacityCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRequestAgentV0:
		return buildDirectorAgentRequestAgentCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRequestReviewV0:
		return buildDirectorAgentRequestReviewCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRecordReviewResultV0:
		return buildDirectorAgentRecordReviewResultCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandAcceptReviewV0:
		return buildDirectorAgentAcceptReviewCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRequestReworkV0:
		return buildDirectorAgentRequestReworkCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRecordReplanDecisionV0:
		return buildDirectorAgentRecordReplanDecisionCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandCloseTaskV0:
		return buildDirectorAgentCloseTaskCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandRegisterFinalValidationV0:
		return buildDirectorAgentFinalValidationCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandCloseRunV0:
		return buildDirectorAgentCloseRunCommandV0(request)
	case orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0:
		return buildDirectorAgentAnswerQuestionCommandV0(request)
	default:
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_command_no_soportado", "command_type")}
	}
}

func buildDirectorAgentBrainstormCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestBrainstorm
	command, err := orquestacoreworkflow.NewRequestBrainstormCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestBrainstormCommandPayloadV0{
			BrainstormRequestID:        payload.BrainstormRequestID,
			PhaseID:                    payload.PhaseID,
			TopicRef:                   payload.TopicRef,
			Summary:                    payload.Summary,
			MinimumRecommendedCapacity: directorAgentCapacityV0(payload.MinimumRecommendedCapacity),
			EvidenceRefs:               payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentOpenPhaseCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.OpenPhase
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: payload.PhaseID,
			Reason:  payload.Reason,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentVoteCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestVote
	command, err := orquestacoreworkflow.NewRequestVoteCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestVoteCommandPayloadV0{
			VoteRequestID:              payload.VoteRequestID,
			PhaseID:                    payload.PhaseID,
			DecisionTopicRef:           payload.DecisionTopicRef,
			BrainstormRef:              payload.BrainstormRef,
			Summary:                    payload.Summary,
			MinimumRecommendedCapacity: directorAgentCapacityV0(payload.MinimumRecommendedCapacity),
			EvidenceRefs:               payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentAcceptDecisionCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.AcceptDecision
	command, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
			DecisionRef:       payload.DecisionRef,
			PhaseID:           payload.PhaseID,
			VoteRef:           payload.VoteRef,
			AcceptedOptionRef: payload.AcceptedOptionRef,
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

func buildDirectorAgentPublishContractCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.PublishContract
	command, err := orquestacoreworkflow.NewPublishFunctionContractCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.PublishFunctionContractCommandPayloadV0{
			ContractRef:   payload.ContractRef,
			PhaseID:       payload.PhaseID,
			DecisionRef:   payload.DecisionRef,
			Summary:       payload.Summary,
			FunctionNames: payload.FunctionNames,
			EvidenceRefs:  payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentCreateMicrotaskCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	command, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
			Task: directorAgentWorkflowTaskV0(request.Decision.CreateMicrotask.Task),
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentRequestReviewCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RequestReview
	command, err := orquestacoreworkflow.NewRequestReviewCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RequestReviewCommandPayloadV0{
			ReviewRequestID: payload.ReviewRequestID,
			PhaseID:         payload.PhaseID,
			DeliveryRef:     payload.DeliveryRef,
			Summary:         payload.Summary,
			EvidenceRefs:    payload.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentRecordReviewResultCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.RecordReviewResult
	command, err := orquestacoreworkflow.NewRecordReviewResultCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.RecordReviewResultCommandPayloadV0{
			ReviewResultRef: payload.ReviewResultRef,
			ReviewRequestID: payload.ReviewRequestID,
			DeliveryRef:     payload.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusV0(payload.Status),
			Summary:         payload.Summary,
			EvidenceRefs:    payload.EvidenceRefs,
			QualityGateRef:  payload.QualityGateRef,
		},
	)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{},
			[]DirectorAgentWorkflowIssueV0{directorAgentWorkflowIssueV0("director_agent_workflow_command_invalido", "command")}
	}
	return command, nil
}

func buildDirectorAgentAcceptReviewCommandV0(
	request DirectorAgentWorkflowCommandRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, []DirectorAgentWorkflowIssueV0) {
	payload := request.Decision.AcceptReview
	command, err := orquestacoreworkflow.NewAcceptReviewCommandV0(
		directorAgentCommandMetaV0(request),
		orquestacoreworkflow.AcceptReviewCommandPayloadV0{
			AcceptedReviewRef: payload.AcceptedReviewRef,
			PhaseID:           payload.PhaseID,
			ReviewRequestID:   payload.ReviewRequestID,
			DeliveryRef:       payload.DeliveryRef,
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

func directorAgentCommandMetaV0(
	request DirectorAgentWorkflowCommandRequestV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      request.Decision.CommandRef,
		RunID:          request.Decision.RunID,
		IdempotencyKey: "idem-" + request.Decision.CommandRef,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}

func directorAgentCapacityV0(value string) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	return orquestacoreworkflow.OrchestrationCapacityRecommendationV0(value)
}

func directorAgentWorkflowTaskV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        task.SchemaVersion,
		TaskID:               task.TaskID,
		RunID:                task.RunID,
		PhaseID:              orquestacoreworkflow.OrchestrationPhaseIDV0(task.PhaseID),
		Title:                task.Title,
		Summary:              task.Summary,
		WriteSet:             task.WriteSet,
		AcceptanceCriteria:   directorAgentWorkflowTaskAcceptanceCriteriaV0(task),
		RequiredTests:        task.RequiredTests,
		DependsOn:            task.DependsOn,
		ParentTaskRef:        task.ParentTaskRef,
		CohortRef:            task.CohortRef,
		WaveRef:              task.WaveRef,
		DelegationDepth:      task.DelegationDepth,
		MaxChildAgents:       task.MaxChildAgents,
		ChildTaskRefs:        task.ChildTaskRefs,
		FunctionContractRefs: directorAgentWorkflowFunctionRefsV0(task.FunctionContractRefs),
	}
}

func directorAgentWorkflowTaskAcceptanceCriteriaV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) []string {
	return directorAgentWorkflowOperationalAcceptanceCriteriaV0(task.PhaseID, task.AcceptanceCriteria)
}

func directorAgentWorkflowFunctionRefsV0(
	refs []orquestadirectoragent.DirectorAgentFunctionContractRefV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	result := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		result = append(result, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  ref.ContractRef,
			FunctionName: ref.FunctionName,
		})
	}
	return result
}
