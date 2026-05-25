package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestCreateMicrotaskCommandV0RejectsNonPlanningPhase(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-wrong-current-phase", "idem-microtask-wrong-current-phase", "task-wrong-current-phase")

	_, err := HandleCommandV0(run, command)
	assertCreateMicrotaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestCreateMicrotaskCommandV0RejectsUnpublishedFunctionContract(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	command := mustCreateMicrotaskCommandV0(t, "cmd-microtask-unpublished-contract", "idem-microtask-unpublished-contract", "task-unpublished-contract")

	_, err := HandleCommandV0(run, command)
	assertCreateMicrotaskCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestMicrotaskCreatedEventV0RejectsUnpublishedFunctionContract(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	event := mustMicrotaskCreatedEventV0(t, "evt-microtask-unpublished-contract", run.LastSequence+1, "task-unpublished-contract")

	_, err := ApplyEventV0(run, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrSecuenciaInvalidaV0)
	}
}

func TestCreateMicrotaskCommandV0RejectsMissingFunctionContractRefs(t *testing.T) {
	task := validMicrotaskWorkflowTaskV0("task-missing-contract-refs")
	task.FunctionContractRefs = nil

	_, err := NewCreateMicrotaskCommandV0(validCommandMetaV0("cmd-microtask-missing-contract-refs", "idem-microtask-missing-contract-refs"), CreateMicrotaskCommandPayloadV0{
		Task: task,
	})
	assertCreateMicrotaskCommandErrorV0(t, err, ErrPayloadInvalidoV0)
}

func TestCreateMicrotaskCommandV0AcceptsFunctionNameOnlyRefWhenPublished(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	publish := mustPublishFunctionContractCommandV0(
		t,
		"cmd-function-name-contract",
		"idem-function-name-contract",
		"NewWorkflowTaskV0",
	)
	run = mustApplySingleCommandEventV0(t, run, publish)
	task := validMicrotaskWorkflowTaskV0("task-function-name-only")
	task.FunctionContractRefs = []WorkflowFunctionContractRefV0{
		{FunctionName: "NewWorkflowTaskV0"},
	}

	command, err := NewCreateMicrotaskCommandV0(validCommandMetaV0("cmd-microtask-function-name-only", "idem-microtask-function-name-only"), CreateMicrotaskCommandPayloadV0{
		Task: task,
	})
	if err != nil {
		t.Fatalf("NewCreateMicrotaskCommandV0 function_name-only: %v", err)
	}
	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("HandleCommandV0 function_name-only: %v", err)
	}
	if len(result.Events) != 1 || result.Events[0].EventType != OrchestrationEventMicrotaskCreatedV0 {
		t.Fatalf("events=%+v", result.Events)
	}
}

func assertCreateMicrotaskCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
