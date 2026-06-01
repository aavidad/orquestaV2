package orquestadirectorrunner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestStoredWorkflowCommandPortV0PersisteEventosYRecargaRun(t *testing.T) {
	ctx := context.Background()
	store := &runnerMemoryWorkflowEventStoreV0{}
	workflow := NewStoredWorkflowCommandPortV0(store)

	start := mustRunnerStartCommandV0(t, "event-store-start")
	startResult, err := workflow.HandleWorkflowCommandV0(ctx, start)
	if err != nil {
		t.Fatalf("start stored workflow: %v", err)
	}
	if len(startResult.Events) != 1 || startResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("unexpected start result: %+v", startResult)
	}

	open := mustRunnerOpenProgramacionCommandV0(t, "event-store-open")
	openResult, err := workflow.HandleWorkflowCommandV0(ctx, open)
	if err != nil {
		t.Fatalf("open stored workflow: %v", err)
	}
	if len(openResult.Events) != 1 || openResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventPhaseOpenedV0 {
		t.Fatalf("unexpected open result: %+v", openResult)
	}

	run, err := LoadWorkflowRunFromEventStoreV0(ctx, store, runnerRunRefV0)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run.RunID != runnerRunRefV0 || run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("unexpected reloaded run: %+v", run)
	}

	repeated, err := workflow.HandleWorkflowCommandV0(ctx, open)
	if err != nil {
		t.Fatalf("repeat open stored workflow: %v", err)
	}
	if !repeated.Idempotent || len(repeated.Events) != 0 || len(store.EventsV0()) != 2 {
		t.Fatalf("repeat should be idempotent without append: result=%+v events=%+v", repeated, store.EventsV0())
	}
}

func TestStoredWorkflowCommandPortV0RejectsNilEventStore(t *testing.T) {
	_, err := HandleStoredWorkflowCommandV0(context.Background(), nil, mustRunnerStartCommandV0(t, "nil-store"))
	var storeErr WorkflowEventStoreErrorV0
	if !errors.As(err, &storeErr) {
		t.Fatalf("expected store error, got %T %v", err, err)
	}
	if storeErr.Code != ErrDirectorRunnerWorkflowStoreV0 || storeErr.Field != "event_store" {
		t.Fatalf("unexpected store error: %+v", storeErr)
	}
}

func newRunnerWorkflowEventStoreProgramacionV0(
	t *testing.T,
) (*runnerMemoryWorkflowEventStoreV0, StoredWorkflowCommandPortV0, orquestacoreworkflow.OrchestrationRunV0) {
	t.Helper()
	ctx := context.Background()
	store := &runnerMemoryWorkflowEventStoreV0{}
	workflow := NewStoredWorkflowCommandPortV0(store)
	mustRunnerApplyStoredWorkflowCommandV0(t, workflow, mustRunnerStartCommandV0(t, "memory-start"))
	mustRunnerApplyStoredWorkflowCommandV0(t, workflow, mustRunnerOpenProgramacionCommandV0(t, "memory-open"))
	run, err := LoadWorkflowRunFromEventStoreV0(ctx, store, runnerRunRefV0)
	if err != nil {
		t.Fatalf("reload workflow run: %v", err)
	}
	return store, workflow, run
}

func mustRunnerApplyStoredWorkflowCommandV0(
	t *testing.T,
	workflow StoredWorkflowCommandPortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	result, err := workflow.HandleWorkflowCommandV0(context.Background(), command)
	if err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
	if len(result.Events) == 0 {
		t.Fatalf("command %s produced no events", command.CommandType)
	}
}

type runnerMemoryWorkflowEventStoreV0 struct {
	events []orquestacoreworkflow.OrchestrationEventV0
}

func (store *runnerMemoryWorkflowEventStoreV0) LoadRunEventsV0(
	ctx context.Context,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = strings.TrimSpace(runRef)
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(store.events))
	for _, event := range store.events {
		if strings.TrimSpace(event.RunID) == runRef {
			events = append(events, event)
		}
	}
	return cloneWorkflowEventStoreEventsV0(events), nil
}

func (store *runnerMemoryWorkflowEventStoreV0) AppendRunEventsV0(
	ctx context.Context,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return fmt.Errorf("run_ref requerido")
	}
	for _, event := range cloneWorkflowEventStoreEventsV0(events) {
		if strings.TrimSpace(event.RunID) != runRef {
			return fmt.Errorf("evento de otro run: %s", event.RunID)
		}
		duplicate, err := store.eventAlreadyAppendedV0(event)
		if err != nil {
			return err
		}
		if duplicate {
			continue
		}
		store.events = append(store.events, event)
	}
	return nil
}

func (store *runnerMemoryWorkflowEventStoreV0) EventsV0() []orquestacoreworkflow.OrchestrationEventV0 {
	return cloneWorkflowEventStoreEventsV0(store.events)
}

func (store *runnerMemoryWorkflowEventStoreV0) eventAlreadyAppendedV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (bool, error) {
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" {
		return false, nil
	}
	for _, existing := range store.events {
		if strings.TrimSpace(existing.RunID) != strings.TrimSpace(event.RunID) ||
			strings.TrimSpace(existing.EventID) != eventID {
			continue
		}
		if reflect.DeepEqual(existing, event) {
			return true, nil
		}
		return false, fmt.Errorf("evento conflictivo: %s", eventID)
	}
	return false, nil
}
