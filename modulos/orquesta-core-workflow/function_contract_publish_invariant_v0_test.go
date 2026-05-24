package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestPublishFunctionContractCommandV0RejectsMissingDecisionRef(t *testing.T) {
	run := mustPlanificationActiveRunWithoutDecisionV0(t)
	command := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-no-decision", "idem-function-contract-no-decision", "contract:function:no-decision:v0")

	_, err := HandleCommandV0(run, command)
	assertPublishFunctionContractCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestFunctionContractPublishedEventV0RejectsMissingDecisionRef(t *testing.T) {
	run := mustPlanificationActiveRunWithoutDecisionV0(t)
	event := mustFunctionContractPublishedEventV0(t, "evt-function-contract-no-decision", run.LastSequence+1, "contract:function:no-decision:v0")

	_, err := ApplyEventV0(run, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrSecuenciaInvalidaV0)
	}
}

func TestPublishFunctionContractCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	command := mustPublishFunctionContractCommandV0(t, "cmd-function-contract-phase", "idem-function-contract-phase", "contract:function:wrong-phase:v0")

	_, err := HandleCommandV0(run, command)
	assertPublishFunctionContractCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestPublishFunctionContractCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validPublishFunctionContractPayloadV0("contract:function:forbidden:v0")
	payload.Summary = "usar api_key=valor"

	_, err := NewPublishFunctionContractCommandV0(validCommandMetaV0("cmd-function-contract-forbidden", "idem-function-contract-forbidden"), payload)
	assertPublishFunctionContractCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestFunctionContractPublishedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := functionContractPublishedPayloadFromCommandV0(validPublishFunctionContractPayloadV0("contract:function:event-forbidden:v0"))
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewFunctionContractPublishedEventV0(reducerEventMetaV0("evt-function-contract-forbidden", 5), payload)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func mustPlanificationActiveRunWithoutDecisionV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-plan-no-decision", "idem-open-plan-no-decision", OrchestrationPhasePlanificacionMicrotareasV0)
	return mustApplySingleCommandEventV0(t, run, open)
}
