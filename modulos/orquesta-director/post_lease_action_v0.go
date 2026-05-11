package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func BuildPostLeaseActionV0(input PostLeaseActionInputV0) (PostLeaseActionResultV0, error) {
	normalized := normalizePostLeaseActionInputV0(input)
	if err := validatePostLeaseActionInputV0(normalized); err != nil {
		return PostLeaseActionResultV0{}, err
	}
	register, err := orquestacoreworkflow.NewRegisterAgentLeaseExpiredCommandV0(
		normalized.CommandMeta,
		registerLeaseExpiredPayloadFromPostLeaseV0(normalized),
	)
	if err != nil {
		return PostLeaseActionResultV0{}, err
	}
	result := PostLeaseActionResultV0{
		RegisterLeaseExpiredCommand: register,
		FollowupStatus:              PostLeaseFollowupStatusUnsupportedNeedsDirectorV0,
	}
	switch normalized.RecommendedAction {
	case orquestacoreworkflow.AgentLeaseActionStopAgentV0:
		stop, err := orquestacoreworkflow.NewStopAgentCommandV0(
			postLeaseFollowupMetaV0(normalized.CommandMeta, "-stop-agent"),
			stopAgentPayloadFromPostLeaseV0(normalized),
		)
		if err != nil {
			return PostLeaseActionResultV0{}, err
		}
		result.StopAgentCommand = &stop
		result.FollowupStatus = PostLeaseFollowupStatusStopAgentV0
	case orquestacoreworkflow.AgentLeaseActionAskDirectorV0:
		ask, err := orquestacoreworkflow.NewAskDirectorCommandV0(
			postLeaseFollowupMetaV0(normalized.CommandMeta, "-ask-director"),
			askDirectorPayloadFromPostLeaseV0(normalized),
		)
		if err != nil {
			return PostLeaseActionResultV0{}, err
		}
		result.AskDirectorCommand = &ask
		result.FollowupStatus = PostLeaseFollowupStatusAskDirectorV0
	}
	return result, nil
}
