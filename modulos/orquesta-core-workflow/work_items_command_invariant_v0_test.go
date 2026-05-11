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

func TestCreateMicrotaskCommandV0RejectsFunctionNameOnlyRef(t *testing.T) {
	task := validMicrotaskWorkflowTaskV0("task-function-name-only")
	task.FunctionContractRefs = []WorkflowFunctionContractRefV0{
		{FunctionName: "NewWorkflowTaskV0"},
	}

	_, err := NewCreateMicrotaskCommandV0(validCommandMetaV0("cmd-microtask-function-name-only", "idem-microtask-function-name-only"), CreateMicrotaskCommandPayloadV0{
		Task: task,
	})
	assertCreateMicrotaskCommandErrorV0(t, err, ErrPayloadInvalidoV0)
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
