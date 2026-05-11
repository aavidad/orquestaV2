package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func BuildReplanFollowupsV0(input ReplanFollowupsInputV0) (ReplanFollowupsResultV0, error) {
	input = normalizeReplanFollowupsInputV0(input)
	if err := validateReplanFollowupsInputV0(input); err != nil {
		return ReplanFollowupsResultV0{}, err
	}

	decision, err := orquestacoreworkflow.NewRecordReplanDecisionCommandV0(
		input.DecisionCommandMeta,
		input.DecisionPayload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result := ReplanFollowupsResultV0{
		RecordReplanDecisionCommand: decision,
		FollowupStatus:              ReplanFollowupStatusNeedsDirectorUnsupportedV0,
	}
	result, err = buildOpenPhaseReplanFollowupV0(input, result)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}

	if input.SourceKind == ReplanFollowupSourceQualityGateBlockedV0 ||
		input.SourceKind == ReplanFollowupSourceReviewReworkV0 {
		return buildSourceScopedReplanFollowupsV0(input, result)
	}

	switch input.DecisionPayload.AcceptedAction {
	case orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0:
		return buildCapacityAgentReplanFollowupsV0(input, result)
	case orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0:
		return buildCapacityOnlyReplanFollowupV0(input, result)
	case orquestacoreworkflow.ReplanDecisionActionAskDirectorV0:
		return buildAskDirectorReplanFollowupV0(input, result)
	default:
		return result, nil
	}
}

func buildOpenPhaseReplanFollowupV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	if input.OpenPhaseCandidate == nil {
		return result, nil
	}
	openPhase, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		input.OpenPhaseCandidate.CommandMeta,
		input.OpenPhaseCandidate.Payload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result.OpenPhaseCommand = &openPhase
	return result, nil
}

func buildSourceScopedReplanFollowupsV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	if input.SourceKind == ReplanFollowupSourceReviewReworkV0 &&
		input.DecisionPayload.AcceptedAction == orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 &&
		len(input.MicrotaskCandidates) > 0 {
		return buildMicrotaskReplanFollowupsV0(input, result)
	}
	switch input.DecisionPayload.AcceptedAction {
	case orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0:
		return buildCapacityAgentReplanFollowupsV0(input, result)
	case orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0:
		return buildCapacityOnlyReplanFollowupV0(input, result)
	case orquestacoreworkflow.ReplanDecisionActionAskDirectorV0,
		orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
		orquestacoreworkflow.ReplanDecisionActionAbortTaskV0:
		return buildAskDirectorReplanFollowupV0(input, result)
	default:
		return result, nil
	}
}

func buildMicrotaskReplanFollowupsV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	for _, candidate := range input.MicrotaskCandidates {
		command, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
			candidate.CommandMeta,
			candidate.Payload,
		)
		if err != nil {
			return ReplanFollowupsResultV0{}, err
		}
		result.CreateMicrotaskCommands = append(result.CreateMicrotaskCommands, command)
	}
	result.FollowupStatus = ReplanFollowupStatusMicrotasksRequestedV0
	return result, nil
}

func buildCapacityAgentReplanFollowupsV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	if input.CapacityCandidate == nil {
		return result, nil
	}
	capacity, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		input.CapacityCandidate.CommandMeta,
		input.CapacityCandidate.Payload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result.RequestCapacityCommand = &capacity
	result.FollowupStatus = ReplanFollowupStatusCapacityRequestedV0

	if input.AgentCandidate == nil {
		return result, nil
	}
	agent, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		input.AgentCandidate.CommandMeta,
		input.AgentCandidate.Payload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result.RequestAgentCommand = &agent
	result.FollowupStatus = ReplanFollowupStatusCapacityAndAgentRequestedV0
	return result, nil
}

func buildCapacityOnlyReplanFollowupV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	if input.CapacityCandidate == nil {
		return result, nil
	}
	capacity, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		input.CapacityCandidate.CommandMeta,
		input.CapacityCandidate.Payload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result.RequestCapacityCommand = &capacity
	result.FollowupStatus = ReplanFollowupStatusCapacityRequestedV0
	return result, nil
}

func buildAskDirectorReplanFollowupV0(
	input ReplanFollowupsInputV0,
	result ReplanFollowupsResultV0,
) (ReplanFollowupsResultV0, error) {
	if input.AskDirectorCandidate == nil {
		return result, nil
	}
	ask, err := orquestacoreworkflow.NewAskDirectorCommandV0(
		input.AskDirectorCandidate.CommandMeta,
		input.AskDirectorCandidate.Payload,
	)
	if err != nil {
		return ReplanFollowupsResultV0{}, err
	}
	result.AskDirectorCommand = &ask
	result.FollowupStatus = ReplanFollowupStatusAskDirectorV0
	return result, nil
}
