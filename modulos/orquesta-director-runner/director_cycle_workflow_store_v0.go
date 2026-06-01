package orquestadirectorrunner

import (
	"bytes"
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const ErrDirectorRunnerWorkflowStoreV0 = "director_runner_workflow_store"

type WorkflowEventStorePortV0 interface {
	LoadRunEventsV0(ctx context.Context, runRef string) ([]orquestacoreworkflow.OrchestrationEventV0, error)
	AppendRunEventsV0(ctx context.Context, runRef string, events []orquestacoreworkflow.OrchestrationEventV0) error
}

type StoredWorkflowCommandPortV0 struct {
	EventStore WorkflowEventStorePortV0
}

type WorkflowEventStoreErrorV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func (err WorkflowEventStoreErrorV0) Error() string {
	parts := []string{err.Code}
	if strings.TrimSpace(err.Field) != "" {
		parts = append(parts, strings.TrimSpace(err.Field))
	}
	if strings.TrimSpace(err.Message) != "" {
		parts = append(parts, strings.TrimSpace(err.Message))
	}
	return strings.Join(parts, ":")
}

func NewStoredWorkflowCommandPortV0(
	eventStore WorkflowEventStorePortV0,
) StoredWorkflowCommandPortV0 {
	return StoredWorkflowCommandPortV0{EventStore: eventStore}
}

func HandleStoredWorkflowCommandV0(
	ctx context.Context,
	eventStore WorkflowEventStorePortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	return NewStoredWorkflowCommandPortV0(eventStore).HandleWorkflowCommandV0(ctx, command)
}

func (port StoredWorkflowCommandPortV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	run, err := LoadWorkflowRunFromEventStoreV0(ctx, port.EventStore, command.RunID)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		return result, err
	}
	if len(result.Events) == 0 {
		return result, nil
	}
	if err := appendWorkflowCommandEventsV0(ctx, port.EventStore, command.RunID, result.Events); err != nil {
		return result, err
	}
	if _, err := LoadWorkflowRunFromEventStoreV0(ctx, port.EventStore, command.RunID); err != nil {
		return result, err
	}
	return result, nil
}

func LoadWorkflowRunFromEventStoreV0(
	ctx context.Context,
	eventStore WorkflowEventStorePortV0,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if err := validateWorkflowEventStoreInputV0(ctx, eventStore, runRef); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	events, err := eventStore.LoadRunEventsV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return orquestacoreworkflow.ReplayDurableEventsV0(cloneWorkflowEventStoreEventsV0(events))
}

func appendWorkflowCommandEventsV0(
	ctx context.Context,
	eventStore WorkflowEventStorePortV0,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if err := validateWorkflowEventStoreInputV0(ctx, eventStore, runRef); err != nil {
		return err
	}
	runRef = strings.TrimSpace(runRef)
	cloned := cloneWorkflowEventStoreEventsV0(events)
	for _, event := range cloned {
		if err := orquestacoreworkflow.ValidateOrchestrationEventV0(event); err != nil {
			return err
		}
		if strings.TrimSpace(event.RunID) != runRef {
			return workflowEventStoreErrorV0("events.run_id", "evento de otro run")
		}
	}
	return eventStore.AppendRunEventsV0(ctx, runRef, cloned)
}

func validateWorkflowEventStoreInputV0(
	ctx context.Context,
	eventStore WorkflowEventStorePortV0,
	runRef string,
) error {
	if ctx == nil {
		return workflowEventStoreErrorV0("context", "context requerido")
	}
	if isNilDirectorCyclePortV0(eventStore) {
		return workflowEventStoreErrorV0("event_store", "event_store requerido")
	}
	if strings.TrimSpace(runRef) == "" {
		return workflowEventStoreErrorV0("run_ref", "run_ref requerido")
	}
	return ctx.Err()
}

func workflowEventStoreErrorV0(field string, message string) WorkflowEventStoreErrorV0 {
	return WorkflowEventStoreErrorV0{
		Code:    ErrDirectorRunnerWorkflowStoreV0,
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}
}

func cloneWorkflowEventStoreEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(events))
	for _, event := range events {
		event.Payload = bytes.Clone(event.Payload)
		cloned = append(cloned, event)
	}
	return cloned
}
