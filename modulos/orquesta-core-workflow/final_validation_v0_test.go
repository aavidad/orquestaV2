package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRegisterFinalValidationCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-001", "idem-final-validation-001", "validation-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterFinalValidation: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventFinalValidationRegisteredV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestHandleRegisterFinalValidationCommandV0AceptaRunSinMicrotareas(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-final-validation-run-level", "idem-open-final-validation-run-level", OrchestrationPhaseValidacionFinalV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	payload := validRegisterFinalValidationPayloadV0("validation-run-level")
	payload.ClosedTaskRef = ""
	payload.Summary = "Registrar validacion final compacta de una run goal-first."
	command := mustRegisterFinalValidationCommandWithPayloadV0(t, "cmd-final-validation-run-level", "idem-final-validation-run-level", payload)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle run-level RegisterFinalValidation: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventFinalValidationRegisteredV0)
	next := mustApplySingleCommandEventV0(t, run, command)
	if !finalValidationAlreadyReflectedV0(next, "validation-run-level") {
		t.Fatalf("validacion run-level no proyectada: %+v", next.Validations)
	}
}

func TestApplyFinalValidationRegisteredV0ProjectsRefOnce(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	event := mustFinalValidationRegisteredEventV0(t, "evt-final-validation-reducer-001", run.LastSequence+1, "validation-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply FinalValidationRegistered: %v", err)
	}
	if !reflect.DeepEqual(got.Validations, []string{"validation-001"}) {
		t.Fatalf("validations=%v, want [validation-001]", got.Validations)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Validations, got.Validations) {
		t.Fatalf("validations duplicated: %v", again.Validations)
	}
}

func TestReplayDurableEventsV0AcceptsFinalValidationRegistered(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-final-validation", 15, "idem-review-for-final-validation", "review-request-001")
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-final-validation", 16, "idem-record-result-for-final-validation", validRecordReviewResultPayloadV0("review-result-for-final-validation", ReviewResultStatusAcceptedV0))
	accepted := mustReviewAcceptedEventWithKeyV0(t, "evt-durable-accept-review-for-final-validation", 17, "idem-accept-review-for-final-validation", "accepted-review-001")
	closed := mustTaskClosedEventWithKeyV0(t, "evt-durable-close-task-for-final-validation", 18, "idem-close-task-for-final-validation", "task-ncw-009")
	validation := mustFinalValidationRegisteredEventWithKeyV0(t, "evt-durable-final-validation", 20, "idem-final-validation", "validation-001")
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-final-validation", 14, "idem-open-revision-final-validation", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		accepted,
		closed,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-final-validation", 19, "idem-open-final-validation", OrchestrationPhaseValidacionFinalV0),
		validation,
		validation,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable FinalValidationRegistered: %v", err)
	}
	if got.LastSequence != 20 {
		t.Fatalf("last_sequence=%d, want 20", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Validations, []string{"validation-001"}) {
		t.Fatalf("validations=%v", got.Validations)
	}
}

func TestRegisterFinalValidationCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-repeat", "idem-final-validation-repeat", "validation-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterFinalValidationCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	payload := validRegisterFinalValidationPayloadV0("validation-effect-conflict")
	command := mustRegisterFinalValidationCommandWithPayloadV0(t, "cmd-final-validation-effect-conflict", "idem-final-validation-effect-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Registrar validacion final compacta con otro resumen."
	conflicting := mustRegisterFinalValidationCommandWithPayloadV0(t, "cmd-final-validation-effect-conflict", "idem-final-validation-effect-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterFinalValidationCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-key", "idem-final-validation-key", "validation-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-key-2", "idem-final-validation-key-2", "validation-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyFinalValidationRegisteredV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	first := mustFinalValidationRegisteredEventWithKeyV0(t, "evt-final-validation-effect", run.LastSequence+1, "idem-final-validation-effect", "validation-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustFinalValidationRegisteredEventWithKeyV0(t, "evt-final-validation-effect-conflict", applied.LastSequence+1, "idem-final-validation-effect-conflict", "validation-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterFinalValidationCommandV0DoesNotClosePhaseOrRun(t *testing.T) {
	run := mustFinalValidationReadyRunV0(t)
	command := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-state", "idem-final-validation-state", "validation-state")

	got := mustApplySingleCommandEventV0(t, run, command)
	if got.Status != OrchestrationRunStatusActiveV0 {
		t.Fatalf("status=%q, want active", got.Status)
	}
	if got.CurrentPhase != OrchestrationPhaseValidacionFinalV0 {
		t.Fatalf("current_phase=%q, want validacion_final", got.CurrentPhase)
	}
	if reducerPhaseByIDV0(t, got, OrchestrationPhaseValidacionFinalV0).Status != OrchestrationPhaseStatusActiveV0 {
		t.Fatalf("validacion_final phase not active after FinalValidationRegistered")
	}
}

func mustFinalValidationReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustCloseTaskReadyRunV0(t)
	closeTask := mustCloseTaskCommandV0(t, "cmd-close-task-for-final-validation", "idem-close-task-for-final-validation", "task-ncw-009")
	run = mustApplySingleCommandEventV0(t, run, closeTask)
	open := mustOpenPhaseCommandV0(t, "cmd-open-final-validation", "idem-open-final-validation", OrchestrationPhaseValidacionFinalV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustRegisterFinalValidationCommandV0(t *testing.T, commandID string, idempotencyKey string, validationRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterFinalValidationCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterFinalValidationPayloadV0(validationRef))
}

func mustRegisterFinalValidationCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterFinalValidationCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterFinalValidationCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustFinalValidationRegisteredEventV0(t *testing.T, eventID string, sequence int64, validationRef string) OrchestrationEventV0 {
	t.Helper()
	return mustFinalValidationRegisteredEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, validationRef)
}

func mustFinalValidationRegisteredEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, validationRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewFinalValidationRegisteredEventV0(meta, finalValidationRegisteredPayloadFromCommandV0(validRegisterFinalValidationPayloadV0(validationRef)))
	return mustReducerEventV0(t, event, err)
}

func validRegisterFinalValidationPayloadV0(validationRef string) RegisterFinalValidationCommandPayloadV0 {
	return RegisterFinalValidationCommandPayloadV0{
		ValidationRef: validationRef,
		PhaseID:       string(OrchestrationPhaseValidacionFinalV0),
		ClosedTaskRef: "task-ncw-009",
		Summary:       "Registrar validacion final compacta de una tarea cerrada.",
		EvidenceRefs:  []string{"docs/contratos_validacion_final.md#FinalValidationRegistered"},
	}
}

func assertRegisterFinalValidationCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
