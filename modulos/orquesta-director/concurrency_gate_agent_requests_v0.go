package orquestadirector

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func BuildConcurrencyGateAgentRequestsV0(
	input ConcurrencyGateAgentRequestsInputV0,
) (ConcurrencyGateAgentRequestsResultV0, error) {
	input = normalizeConcurrencyGateAgentRequestsInputV0(input)
	result := ConcurrencyGateAgentRequestsResultV0{}
	if err := validateConcurrencyGateAgentRequestsInputV0(input); err != nil {
		result.Issues = appendConcurrencyGateErrorIssuesV0(result.Issues, err)
		return result, err
	}

	evaluation := orquestacoreconcurrency.EvaluateConcurrencyGateV0(input.Claims, input.SubjectClaimRefs)
	result.Evaluation = evaluation
	result.BlockedClaimRefs = cloneConcurrencyGateStringsV0(evaluation.BlockedClaimRefs)
	result.ConflictRefs = cloneConcurrencyGateStringsV0(evaluation.ConflictRefs)

	gateCommand, err := orquestacoreworkflow.NewRecordConcurrencyGateCommandV0(
		input.GateCommandMeta,
		recordConcurrencyGatePayloadV0(evaluation, input.EvidenceRefs),
	)
	if err != nil {
		return result, err
	}
	result.GateCommand = gateCommand

	if evaluation.Decision != orquestacoreconcurrency.ConcurrencyGateDecisionAllowRequestAgentV0 {
		return result, nil
	}
	if err := validateAllowConcurrencyGateCandidatesV0(input, evaluation); err != nil {
		result.Issues = appendConcurrencyGateErrorIssuesV0(result.Issues, err)
		return result, err
	}
	commands, err := requestAgentCommandsForReadySubjectsV0(input, evaluation)
	if err != nil {
		return result, err
	}
	result.AgentCommands = commands
	return result, nil
}

func requestAgentCommandsForReadySubjectsV0(
	input ConcurrencyGateAgentRequestsInputV0,
	evaluation orquestacoreconcurrency.ConcurrencyGateEvaluationV0,
) ([]orquestacoreworkflow.OrchestrationCommandV0, error) {
	candidates := candidatesByClaimRefV0(input.CandidateRequests)
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, len(evaluation.SubjectClaimRefs))
	for _, claimRef := range evaluation.SubjectClaimRefs {
		candidate := candidates[claimRef]
		command, err := orquestacoreworkflow.NewRequestAgentCommandV0(
			candidate.CommandMeta,
			candidate.Payload,
		)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, nil
}

func appendConcurrencyGateErrorIssuesV0(
	issues []ConcurrencyGateAgentRequestsIssueV0,
	err error,
) []ConcurrencyGateAgentRequestsIssueV0 {
	gateErr, ok := err.(ConcurrencyGateAgentRequestsErrorV0)
	if !ok {
		return issues
	}
	return append(issues, gateErr.Issues...)
}
