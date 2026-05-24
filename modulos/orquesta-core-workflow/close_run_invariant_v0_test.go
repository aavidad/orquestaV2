package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestCloseRunCommandV0RejectsNonClosurePhase(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	validation := mustRegisterFinalValidationCommandV0(t, "cmd-validation-before-close-run-phase", "idem-validation-before-close-run-phase", "validation-001")
	run = mustApplySingleCommandEventV0(t, run, validation)
	command := mustCloseRunCommandV0(t, "cmd-close-run-phase", "idem-close-run-phase", "closure-phase")

	_, err := HandleCommandV0(run, command)
	assertCloseRunCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestCloseRunCommandV0RejectsMissingValidation(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-closure-missing-validation", "idem-open-closure-missing-validation", OrchestrationPhaseCierreV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustCloseRunCommandV0(t, "cmd-close-run-missing-validation", "idem-close-run-missing-validation", "closure-missing-validation")

	_, err := HandleCommandV0(run, command)
	assertCloseRunCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRunClosedEventV0RejectsMissingValidation(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-closure-event-missing-validation", "idem-open-closure-event-missing-validation", OrchestrationPhaseCierreV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	event := mustRunClosedEventV0(t, "evt-close-run-missing-validation", run.LastSequence+1, "closure-missing-validation")

	_, err := ApplyEventV0(run, event)
	assertRunClosedEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestCloseRunCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validCloseRunPayloadV0("closure-forbidden")
	payload.Summary = "usar api_key=valor"

	_, err := NewCloseRunCommandV0(validCommandMetaV0("cmd-close-run-forbidden", "idem-close-run-forbidden"), payload)
	assertCloseRunCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestRunClosedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := runClosedPayloadFromCommandV0(validCloseRunPayloadV0("closure-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewRunClosedEventV0(reducerEventMetaV0("evt-close-run-forbidden", 18), payload)
	assertRunClosedEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func assertRunClosedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
