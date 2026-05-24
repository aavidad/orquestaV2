package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRegisterFinalValidationCommandV0RejectsNonFinalValidationPhase(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	closeTask := mustCloseTaskCommandV0(t, "cmd-close-task-before-final-phase", "idem-close-task-before-final-phase", "task-ncw-009")
	run = mustApplySingleCommandEventV0(t, run, closeTask)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-phase", "idem-final-validation-phase", "validation-phase")

	_, err := HandleCommandV0(run, command)
	assertRegisterFinalValidationCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterFinalValidationCommandV0RejectsMissingClosedTask(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-final-validation-missing-task", "idem-open-final-validation-missing-task", OrchestrationPhaseValidacionFinalV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-missing-task", "idem-final-validation-missing-task", "validation-missing-task")

	_, err := HandleCommandV0(run, command)
	assertRegisterFinalValidationCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestFinalValidationRegisteredEventV0RejectsMissingClosedTask(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-final-validation-event-missing-task", "idem-open-final-validation-event-missing-task", OrchestrationPhaseValidacionFinalV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	event := mustFinalValidationRegisteredEventV0(t, "evt-final-validation-missing-task", run.LastSequence+1, "validation-missing-task")

	_, err := ApplyEventV0(run, event)
	assertFinalValidationRegisteredEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRegisterFinalValidationCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterFinalValidationPayloadV0("validation-forbidden")
	payload.Summary = "usar api_key=valor"

	_, err := NewRegisterFinalValidationCommandV0(validCommandMetaV0("cmd-final-validation-forbidden", "idem-final-validation-forbidden"), payload)
	assertRegisterFinalValidationCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestFinalValidationRegisteredEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := finalValidationRegisteredPayloadFromCommandV0(validRegisterFinalValidationPayloadV0("validation-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

	_, err := NewFinalValidationRegisteredEventV0(reducerEventMetaV0("evt-final-validation-forbidden", 16), payload)
	assertFinalValidationRegisteredEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func assertFinalValidationRegisteredEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
