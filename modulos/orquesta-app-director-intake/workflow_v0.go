package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func buildInitialDirectorIntakeRunV0(
	request PrepareAppDirectorInputRequestV0,
	tasks []AppDirectorTaskV0,
) (
	orquestacoreworkflow.OrchestrationRunV0,
	[]orquestacoreworkflow.OrchestrationEventV0,
	error,
) {
	run := orquestacoreworkflow.OrchestrationRunV0{}
	events := []orquestacoreworkflow.OrchestrationEventV0{}
	commands, err := initialDirectorIntakeCommandsV0(request, tasks)
	if err != nil {
		return run, nil, err
	}
	for _, command := range commands {
		next, appliedEvents, err := applyDirectorIntakeCommandV0(run, command)
		if err != nil {
			return run, nil, err
		}
		run = next
		events = append(events, appliedEvents...)
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		return run, nil, AppDirectorIntakeIssueV0{Field: "run." + issues[0].Field}
	}
	return run, events, nil
}

func initialDirectorIntakeCommandsV0(
	request PrepareAppDirectorInputRequestV0,
	tasks []AppDirectorTaskV0,
) ([]orquestacoreworkflow.OrchestrationCommandV0, error) {
	start, err := orquestacoreworkflow.NewStartRunCommandV0(
		directorIntakeCommandMetaV0(request, "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: request.ProjectRef,
			AppSpecRef: request.AppSpec.SpecID,
		},
	)
	if err != nil {
		return nil, err
	}
	open, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		directorIntakeCommandMetaV0(request, "open-brainstorm"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			Reason:  "Arranque de director para definir enfoque de app.",
		},
	)
	if err != nil {
		return nil, err
	}
	commands := []orquestacoreworkflow.OrchestrationCommandV0{start, open}
	for _, task := range tasks {
		brainstorm, err := orquestacoreworkflow.NewRequestBrainstormCommandV0(
			directorIntakeTaskCommandMetaV0(request, task, "brainstorm"),
			orquestacoreworkflow.RequestBrainstormCommandPayloadV0{
				BrainstormRequestID:        task.BrainstormRef,
				PhaseID:                    string(task.PhaseID),
				TopicRef:                   task.TopicRef,
				Summary:                    task.Summary,
				MinimumRecommendedCapacity: task.Capacity,
				EvidenceRefs:               task.EvidenceRefs,
			},
		)
		if err != nil {
			return nil, err
		}
		commands = append(commands, brainstorm)
	}
	return commands, nil
}

func applyDirectorIntakeCommandV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (
	orquestacoreworkflow.OrchestrationRunV0,
	[]orquestacoreworkflow.OrchestrationEventV0,
	error,
) {
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return run, nil, err
	}
	next := run
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(result.Events))
	for _, event := range result.Events {
		applied, err := orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return next, nil, err
		}
		next = applied
		events = append(events, event)
	}
	return next, events, nil
}

func directorIntakeCommandMetaV0(
	request PrepareAppDirectorInputRequestV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref := safeDirectorIntakeRefPartV0(kind + "-" + request.RunRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-director-" + ref,
		RunID:          request.RunRef,
		IdempotencyKey: "idem-app-director-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}

func directorIntakeTaskCommandMetaV0(
	request PrepareAppDirectorInputRequestV0,
	task AppDirectorTaskV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return directorIntakeCommandMetaV0(request, kind+"-"+task.TaskRef)
}
