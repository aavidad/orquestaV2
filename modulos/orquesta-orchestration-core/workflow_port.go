package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type storedWorkflowPortV0 struct {
	RunStore  RunStorePortV0
	EventSink EventSinkPortV0
}

func HandleStoredWorkflowCommandV0(
	ctx context.Context,
	runStore RunStorePortV0,
	eventSink EventSinkPortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	return storedWorkflowPortV0{
		RunStore:  runStore,
		EventSink: eventSink,
	}.HandleWorkflowCommandV0(ctx, command)
}

func (port storedWorkflowPortV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	run, err := port.RunStore.LoadRunV0(ctx, command.RunID)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return result, err
	}
	next, err := applyCommandEventsV0(run, result.Events)
	if err != nil {
		return result, err
	}
	if port.EventSink != nil && len(result.Events) > 0 {
		if err := port.EventSink.AppendRunEventsV0(ctx, command.RunID, result.Events); err != nil {
			return result, err
		}
	}
	if err := port.RunStore.SaveRunV0(ctx, next); err != nil {
		return result, err
	}
	return result, nil
}

func applyCommandEventsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	next := run
	var err error
	for _, event := range events {
		next, err = orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, err
		}
	}
	return next, nil
}
