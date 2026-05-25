package orquestadirector

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func BuildAgentProgressSupervisionV0(
	input AgentProgressSupervisionInputV0,
) (AgentProgressSupervisionResultV0, error) {
	normalized := normalizeAgentProgressSupervisionInputV0(input)
	if err := validateAgentProgressSupervisionInputV0(normalized); err != nil {
		return AgentProgressSupervisionResultV0{}, err
	}
	assessment := assessmentFromAgentProgressReportV0(normalized)
	assessCommand, err := orquestacoreworkflow.NewAssessAgentWorkCommandV0(
		normalized.CommandMeta,
		assessment,
	)
	if err != nil {
		return AgentProgressSupervisionResultV0{}, err
	}
	result := AgentProgressSupervisionResultV0{AssessCommand: assessCommand}
	if agentProgressShouldRegisterLostV0(normalized.Report, assessment) {
		lost, err := orquestacoreworkflow.NewRegisterAgentLostCommandV0(
			registerLostMetaFromSupervisionV0(normalized.CommandMeta),
			registerLostPayloadFromProgressReportV0(
				normalized.Report,
				normalized.CommandMeta.OccurredAt,
			),
		)
		if err != nil {
			return AgentProgressSupervisionResultV0{}, err
		}
		result.RegisterLostCommand = &lost
	}
	if assessment.Action != orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 {
		return result, nil
	}
	ask, err := orquestacoreworkflow.NewAskDirectorCommandV0(
		askDirectorMetaFromSupervisionV0(normalized.CommandMeta),
		askDirectorPayloadFromStalledReportV0(normalized, assessment),
	)
	if err != nil {
		return AgentProgressSupervisionResultV0{}, err
	}
	result.AskDirectorCommand = &ask
	return result, nil
}
