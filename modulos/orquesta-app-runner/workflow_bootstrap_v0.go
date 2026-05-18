package orquestaapprunner

import (
	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func buildInitialAppRunV0(
	request PrepareAppOrchestrationRequestV0,
	plan orquestaappplanner.AppMicrotaskPlanV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	run := orquestacoreworkflow.OrchestrationRunV0{}
	commands, err := initialAppRunCommandsV0(request, plan)
	if err != nil {
		return run, err
	}
	for _, command := range commands {
		next, err := applyWorkflowCommandV0(run, command)
		if err != nil {
			return run, err
		}
		run = next
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		return run, AppRunnerIssueV0{Field: "run." + issues[0].Field}
	}
	return run, nil
}

func initialAppRunCommandsV0(
	request PrepareAppOrchestrationRequestV0,
	plan orquestaappplanner.AppMicrotaskPlanV0,
) ([]orquestacoreworkflow.OrchestrationCommandV0, error) {
	start, err := orquestacoreworkflow.NewStartRunCommandV0(
		appRunCommandMetaV0(request, "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: request.ProjectRef,
			AppSpecRef: request.AppSpec.SpecID,
		},
	)
	if err != nil {
		return nil, err
	}
	openVote, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		appRunCommandMetaV0(request, "open-vote"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Reason:  "Preparar decision compacta de app.",
		},
	)
	if err != nil {
		return nil, err
	}
	vote, err := orquestacoreworkflow.NewRequestVoteCommandV0(
		appRunCommandMetaV0(request, "vote"),
		appRunVotePayloadV0(request),
	)
	if err != nil {
		return nil, err
	}
	accept, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(
		appRunCommandMetaV0(request, "accept-decision"),
		appRunAcceptDecisionPayloadV0(request),
	)
	if err != nil {
		return nil, err
	}
	openPlan, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		appRunCommandMetaV0(request, "open-planificacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			Reason:  "Materializar microtareas del plan.",
		},
	)
	if err != nil {
		return nil, err
	}
	contract, err := orquestacoreworkflow.NewPublishFunctionContractCommandV0(
		appRunCommandMetaV0(request, "publish-contract"),
		appRunFunctionContractPayloadV0(request, plan),
	)
	if err != nil {
		return nil, err
	}
	tasks, err := appRunCreateMicrotaskCommandsV0(request, plan)
	if err != nil {
		return nil, err
	}
	openProgramacion, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		appRunCommandMetaV0(request, "open-programacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Plan de app preparado desde AppSpecV0 validada.",
		},
	)
	if err != nil {
		return nil, err
	}
	commands := []orquestacoreworkflow.OrchestrationCommandV0{
		start,
		openVote,
		vote,
		accept,
		openPlan,
		contract,
	}
	commands = append(commands, tasks...)
	commands = append(commands, openProgramacion)
	return commands, nil
}

func applyWorkflowCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	next, _, err := applyWorkflowCommandWithEventsV0(run, command)
	return next, err
}

func applyWorkflowCommandWithEventsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationRunV0, []orquestacoreworkflow.OrchestrationEventV0, error) {
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return run, nil, err
	}
	next := run
	for _, event := range result.Events {
		applied, err := orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return next, result.Events, err
		}
		next = applied
	}
	return next, result.Events, nil
}

func appRunCommandMetaV0(
	request PrepareAppOrchestrationRequestV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref := safeAppRunnerRefPartV0(kind + "-" + request.RunRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-run-" + ref,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-app-run-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}
