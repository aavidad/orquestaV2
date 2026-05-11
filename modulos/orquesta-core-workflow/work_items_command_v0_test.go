package orquestacoreworkflow

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestHandleCreateMicrotaskCommandV0ReturnsMicrotaskCreated(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-001", "idem-microtask-001", "task-ncw-009")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle CreateMicrotask: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventMicrotaskCreatedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyMicrotaskCreatedV0ProjectsTaskAndFunctionRefsOnce(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	event := mustMicrotaskCreatedEventV0(t, "evt-microtask-reducer-001", run.LastSequence+1, "task-ncw-009")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply MicrotaskCreated: %v", err)
	}

	if !reflect.DeepEqual(got.Tasks, []string{"task-ncw-009"}) {
		t.Fatalf("tasks=%v, want [task-ncw-009]", got.Tasks)
	}
	wantContracts := []string{"contract:function:workflow-task:v0"}
	if !reflect.DeepEqual(got.FunctionContracts, wantContracts) {
		t.Fatalf("function_contracts=%v, want %v", got.FunctionContracts, wantContracts)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Tasks, got.Tasks) {
		t.Fatalf("tasks duplicated: %v", again.Tasks)
	}
	if !reflect.DeepEqual(again.FunctionContracts, got.FunctionContracts) {
		t.Fatalf("function contracts duplicated: %v", again.FunctionContracts)
	}
}

func TestReplayDurableEventsV0AcceptsMicrotaskCreated(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-microtask", 1, "idem-start-microtask"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-vote-microtask", 2, "idem-open-vote-microtask", OrchestrationPhaseVotacionYDecisionV0),
		mustVoteRequestedEventWithKeyV0(t, "evt-durable-vote-microtask", 3, "idem-vote-microtask", "vote-request-001"),
		mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-durable-decision-microtask", 4, "idem-decision-microtask", "decision-001"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-plan-microtask", 5, "idem-open-plan-microtask", OrchestrationPhasePlanificacionMicrotareasV0),
		mustFunctionContractPublishedEventWithKeyV0(t, "evt-durable-contract-microtask", 6, "idem-contract-microtask", "contract:function:workflow-task:v0"),
		mustMicrotaskCreatedEventWithKeyV0(t, "evt-durable-microtask", 7, "idem-microtask", "task-ncw-009"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable MicrotaskCreated: %v", err)
	}

	if got.LastSequence != 7 {
		t.Fatalf("last_sequence=%d, want 7", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Tasks, []string{"task-ncw-009"}) {
		t.Fatalf("tasks=%v, want [task-ncw-009]", got.Tasks)
	}
}

func TestHandleCreateMicrotaskCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-repeat", "idem-microtask-repeat", "task-ncw-009")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleCreateMicrotaskCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	task := validMicrotaskWorkflowTaskV0("task-microtask-conflict")
	command := mustCreateMicrotaskCommandWithTaskV0(t, "cmd-microtask-conflict", "idem-microtask-conflict", task)
	created := mustApplySingleCommandEventV0(t, run, command)

	task.PhaseID = OrchestrationPhaseDocumentacionV0
	conflicting := mustCreateMicrotaskCommandWithTaskV0(t, "cmd-microtask-conflict", "idem-microtask-conflict", task)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleCreateMicrotaskCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-key", "idem-microtask-key", "task-microtask-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustCreateMicrotaskCommandV0(t, "cmd-microtask-key-2", "idem-microtask-key-2", "task-microtask-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyMicrotaskCreatedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	first := mustMicrotaskCreatedEventWithKeyV0(t, "evt-microtask-effect", run.LastSequence+1, "idem-microtask-effect", "task-microtask-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustMicrotaskCreatedEventWithKeyV0(t, "evt-microtask-effect-conflict", applied.LastSequence+1, "idem-microtask-effect-conflict", "task-microtask-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestHandleCreateMicrotaskCommandV0InvalidPayloadReturnsPublicError(t *testing.T) {
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-invalid", "idem-microtask-invalid", "task-ncw-009")
	command.Payload = []byte(`{"task":{"schema_version":"workflow_task.v0","task_id":"task-ncw-009","run_id":"run-001","phase_id":"programacion","title":"Crear microtarea","write_set":[],"acceptance_criteria":["criterio compacto"]}}`)

	_, err := HandleCommandV0(mustHandlerStartedRunV0(t), command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrPayloadInvalidoV0)
	}
}

func TestHandleCreateMicrotaskCommandV0RejectsLargePayload(t *testing.T) {
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-large", "idem-microtask-large", "task-ncw-009")
	command.Payload = []byte(`{"task":` + strings.Repeat(`{"x":`, 1000) + `{}}`)

	_, err := HandleCommandV0(mustHandlerStartedRunV0(t), command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrPayloadInvalidoV0)
	}
}

func TestNewCreateMicrotaskCommandV0RejectsOversizedEventProjection(t *testing.T) {
	task := validMicrotaskWorkflowTaskV0("task-ncw-009-large-projection")
	task.FunctionContractRefs = make([]WorkflowFunctionContractRefV0, 0, maxWorkflowTaskCollectionV0)
	for index := 0; index < maxWorkflowTaskCollectionV0; index++ {
		task.FunctionContractRefs = append(task.FunctionContractRefs, WorkflowFunctionContractRefV0{
			ContractRef: fmt.Sprintf("contract:workflow-task:%02d:%s", index, strings.Repeat("x", 24)),
		})
	}

	_, err := NewCreateMicrotaskCommandV0(validCommandMetaV0("cmd-microtask-projection", "idem-microtask-projection"), CreateMicrotaskCommandPayloadV0{
		Task: task,
	})
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload.task.function_contract_refs" {
		t.Fatalf("error=%+v, want payload.task.function_contract_refs", publicErr)
	}
}

func TestHandleCreateMicrotaskCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-microtask", "idem-start-after-microtask")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding microtask command: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustCreateMicrotaskCommandV0(t *testing.T, commandID string, idempotencyKey string, taskID string) OrchestrationCommandV0 {
	t.Helper()
	return mustCreateMicrotaskCommandWithTaskV0(t, commandID, idempotencyKey, validMicrotaskWorkflowTaskV0(taskID))
}

func mustCreateMicrotaskCommandWithTaskV0(t *testing.T, commandID string, idempotencyKey string, task WorkflowTaskV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewCreateMicrotaskCommandV0(validCommandMetaV0(commandID, idempotencyKey), CreateMicrotaskCommandPayloadV0{Task: task})
	return mustCommandV0(t, command, err)
}

func mustFunctionContractReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustFunctionContractPlanRunV0(t)
	contract := mustPublishFunctionContractCommandV0(t, "cmd-publish-contract-for-microtask", "idem-publish-contract-for-microtask", "contract:function:workflow-task:v0")
	return mustApplySingleCommandEventV0(t, run, contract)
}

func mustMicrotaskCreatedEventV0(t *testing.T, eventID string, sequence int64, taskID string) OrchestrationEventV0 {
	t.Helper()
	return mustMicrotaskCreatedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, taskID)
}

func mustMicrotaskCreatedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, taskID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewMicrotaskCreatedEventV0(meta, microtaskCreatedPayloadFromTaskV0(validMicrotaskWorkflowTaskV0(taskID)))
	return mustReducerEventV0(t, event, err)
}

func validMicrotaskWorkflowTaskV0(taskID string) WorkflowTaskV0 {
	return WorkflowTaskV0{
		TaskID:  taskID,
		RunID:   "run-001",
		PhaseID: OrchestrationPhaseProgramacionV0,
		Title:   "Crear microtarea durable",
		Summary: "Transicion compacta para registrar una unidad pequena",
		WriteSet: []string{
			"commands_v0.go",
			"work_items_command_v0.go",
		},
		AcceptanceCriteria: []string{
			"handler emite evento compacto",
			"reducer proyecta refs sin duplicar",
		},
		FunctionContractRefs: []WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:workflow-task:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}
