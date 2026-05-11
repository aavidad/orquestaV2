package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandlePublishFunctionContractCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	command := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-001", "idem-function-contract-001", "contract:function:workflow-task:v0")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle PublishFunctionContract: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventFunctionContractPublishedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyFunctionContractPublishedV0ProjectsRefOnce(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	event := mustFunctionContractPublishedEventV0(t, "evt-function-contract-reducer-001", run.LastSequence+1, "contract:function:workflow-task:v0")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply FunctionContractPublished: %v", err)
	}
	if !reflect.DeepEqual(got.FunctionContracts, []string{"contract:function:workflow-task:v0"}) {
		t.Fatalf("function_contracts=%v", got.FunctionContracts)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.FunctionContracts, got.FunctionContracts) {
		t.Fatalf("function contracts duplicated: %v", again.FunctionContracts)
	}
}

func TestReplayDurableEventsV0AcceptsFunctionContractPublished(t *testing.T) {
	contract := mustFunctionContractPublishedEventWithKeyV0(t, "evt-durable-function-contract", 6, "idem-function-contract", "contract:function:workflow-task:v0")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-function-contract", 1, "idem-start-function-contract"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-vote-function-contract", 2, "idem-open-vote-function-contract", OrchestrationPhaseVotacionYDecisionV0),
		mustVoteRequestedEventWithKeyV0(t, "evt-durable-vote-function-contract", 3, "idem-vote-function-contract", "vote-request-001"),
		mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-durable-decision-function-contract", 4, "idem-decision-function-contract", "decision-001"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-plan-function-contract", 5, "idem-open-plan-function-contract", OrchestrationPhasePlanificacionMicrotareasV0),
		contract,
		contract,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable FunctionContractPublished: %v", err)
	}
	if got.LastSequence != 6 {
		t.Fatalf("last_sequence=%d, want 6", got.LastSequence)
	}
	if !reflect.DeepEqual(got.FunctionContracts, []string{"contract:function:workflow-task:v0"}) {
		t.Fatalf("function_contracts=%v", got.FunctionContracts)
	}
}

func TestPublishFunctionContractCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	command := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-repeat", "idem-function-contract-repeat", "contract:function:repeat:v0")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestPublishFunctionContractCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	payload := validPublishFunctionContractPayloadV0("contract:function:conflict:v0")
	command := mustPublishFunctionContractCommandWithPayloadV0(t, "cmd-function-contract-conflict", "idem-function-contract-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.FunctionNames = []string{"NewWorkflowTaskV0"}
	conflicting := mustPublishFunctionContractCommandWithPayloadV0(t, "cmd-function-contract-conflict", "idem-function-contract-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestPublishFunctionContractCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	command := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-key", "idem-function-contract-key", "contract:function:key:v0")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-key-2", "idem-function-contract-key-2", "contract:function:key:v0")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyFunctionContractPublishedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustFunctionContractPlanRunV0(t)
	first := mustFunctionContractPublishedEventWithKeyV0(t, "evt-function-contract-effect", run.LastSequence+1, "idem-function-contract-effect", "contract:function:effect:v0")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustFunctionContractPublishedEventWithKeyV0(t, "evt-function-contract-effect-conflict", applied.LastSequence+1, "idem-function-contract-effect-conflict", "contract:function:effect:v0")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestPublishFunctionContractCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-function-contract", "idem-start-after-function-contract")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding PublishFunctionContract: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustFunctionContractPlanRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDecisionReadyRunV0(t)
	decision := mustAcceptDecisionCommandV0(t, "cmd-accept-decision-for-function-contract", "idem-accept-decision-for-function-contract", "decision-001")
	run = mustApplySingleCommandEventV0(t, run, decision)
	open := mustOpenPhaseCommandV0(t, "cmd-open-plan-function-contract", "idem-open-plan-function-contract", OrchestrationPhasePlanificacionMicrotareasV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustPublishFunctionContractCommandV0(t *testing.T, commandID string, idempotencyKey string, contractRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustPublishFunctionContractCommandWithPayloadV0(t, commandID, idempotencyKey, validPublishFunctionContractPayloadV0(contractRef))
}

func mustPublishFunctionContractCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload PublishFunctionContractCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewPublishFunctionContractCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustFunctionContractPublishedEventV0(t *testing.T, eventID string, sequence int64, contractRef string) OrchestrationEventV0 {
	t.Helper()
	return mustFunctionContractPublishedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, contractRef)
}

func mustFunctionContractPublishedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, contractRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewFunctionContractPublishedEventV0(meta, functionContractPublishedPayloadFromCommandV0(validPublishFunctionContractPayloadV0(contractRef)))
	return mustReducerEventV0(t, event, err)
}

func validPublishFunctionContractPayloadV0(contractRef string) PublishFunctionContractCommandPayloadV0 {
	return PublishFunctionContractCommandPayloadV0{
		ContractRef:   contractRef,
		PhaseID:       string(OrchestrationPhasePlanificacionMicrotareasV0),
		DecisionRef:   "decision-001",
		Summary:       "Publicar contrato compacto de funcion para microtareas.",
		FunctionNames: []string{"NewWorkflowTaskV0", "ValidateWorkflowTaskV0"},
		EvidenceRefs:  []string{"docs/contratos_decisiones.md#ArchitectureDecisionAccepted"},
	}
}

func assertPublishFunctionContractCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
