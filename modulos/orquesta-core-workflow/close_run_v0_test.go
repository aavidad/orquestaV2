package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleCloseRunCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	command := mustCloseRunCommandV0(t, "cmd-close-run-001", "idem-close-run-001", "closure-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle CloseRun: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventRunClosedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyRunClosedV0ClosesRunAndProjectsRefOnce(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	event := mustRunClosedEventV0(t, "evt-close-run-reducer-001", run.LastSequence+1, "closure-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply RunClosed: %v", err)
	}
	if got.Status != OrchestrationRunStatusClosedV0 {
		t.Fatalf("status=%q, want closed", got.Status)
	}
	if !reflect.DeepEqual(got.Closures, []string{"closure-001"}) {
		t.Fatalf("closures=%v, want [closure-001]", got.Closures)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Closures, got.Closures) {
		t.Fatalf("closures duplicated: %v", again.Closures)
	}
}

func TestReplayDurableEventsV0AcceptsRunClosed(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-close-run", 15, "idem-review-for-close-run", "review-request-001")
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-close-run", 16, "idem-record-result-for-close-run", validRecordReviewResultPayloadV0("review-result-for-close-run", ReviewResultStatusAcceptedV0))
	accepted := mustReviewAcceptedEventWithKeyV0(t, "evt-durable-accept-review-for-close-run", 17, "idem-accept-review-for-close-run", "accepted-review-001")
	closedTask := mustTaskClosedEventWithKeyV0(t, "evt-durable-close-task-for-close-run", 18, "idem-close-task-for-close-run", "task-ncw-009")
	validation := mustFinalValidationRegisteredEventWithKeyV0(t, "evt-durable-validation-for-close-run", 20, "idem-validation-for-close-run", "validation-001")
	closedRun := mustRunClosedEventWithKeyV0(t, "evt-durable-close-run", 22, "idem-close-run", "closure-001")
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-close-run", 14, "idem-open-revision-close-run", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		accepted,
		closedTask,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-final-validation-close-run", 19, "idem-open-final-validation-close-run", OrchestrationPhaseValidacionFinalV0),
		validation,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-closure-close-run", 21, "idem-open-closure-close-run", OrchestrationPhaseCierreV0),
		closedRun,
		closedRun,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable RunClosed: %v", err)
	}
	if got.LastSequence != 22 {
		t.Fatalf("last_sequence=%d, want 22", got.LastSequence)
	}
	if got.Status != OrchestrationRunStatusClosedV0 {
		t.Fatalf("status=%q, want closed", got.Status)
	}
	if !reflect.DeepEqual(got.Closures, []string{"closure-001"}) {
		t.Fatalf("closures=%v", got.Closures)
	}
}

func TestCloseRunCommandV0RepeatedDoesNotDuplicateAfterClosed(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	command := mustCloseRunCommandV0(t, "cmd-close-run-repeat", "idem-close-run-repeat", "closure-repeat")
	closed := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(closed, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestCloseRunCommandV0RejectsReflectedPayloadConflictAfterClosed(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	payload := validCloseRunPayloadV0("closure-effect-conflict")
	command := mustCloseRunCommandWithPayloadV0(t, "cmd-close-run-effect-conflict", "idem-close-run-effect-conflict", payload)
	closed := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Cerrar run con otro resumen compacto."
	conflicting := mustCloseRunCommandWithPayloadV0(t, "cmd-close-run-effect-conflict", "idem-close-run-effect-conflict", payload)
	_, err := HandleCommandV0(closed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestCloseRunCommandV0RejectsReflectedIdempotencyConflictAfterClosed(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	command := mustCloseRunCommandV0(t, "cmd-close-run-key", "idem-close-run-key", "closure-key")
	closed := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustCloseRunCommandV0(t, "cmd-close-run-key-2", "idem-close-run-key-2", "closure-key")
	_, err := HandleCommandV0(closed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyRunClosedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustCloseRunReadyRunV0(t)
	first := mustRunClosedEventWithKeyV0(t, "evt-close-run-effect", run.LastSequence+1, "idem-close-run-effect", "closure-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustRunClosedEventWithKeyV0(t, "evt-close-run-effect-conflict", applied.LastSequence+1, "idem-close-run-effect-conflict", "closure-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func mustCloseRunReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustFinalValidationReadyRunV0(t)
	validation := mustRegisterFinalValidationCommandV0(t, "cmd-final-validation-for-close-run", "idem-final-validation-for-close-run", "validation-001")
	run = mustApplySingleCommandEventV0(t, run, validation)
	open := mustOpenPhaseCommandV0(t, "cmd-open-closure", "idem-open-closure", OrchestrationPhaseCierreV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustCloseRunCommandV0(t *testing.T, commandID string, idempotencyKey string, closureRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustCloseRunCommandWithPayloadV0(t, commandID, idempotencyKey, validCloseRunPayloadV0(closureRef))
}

func mustCloseRunCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload CloseRunCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewCloseRunCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustRunClosedEventV0(t *testing.T, eventID string, sequence int64, closureRef string) OrchestrationEventV0 {
	t.Helper()
	return mustRunClosedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, closureRef)
}

func mustRunClosedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, closureRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewRunClosedEventV0(meta, runClosedPayloadFromCommandV0(validCloseRunPayloadV0(closureRef)))
	return mustReducerEventV0(t, event, err)
}

func validCloseRunPayloadV0(closureRef string) CloseRunCommandPayloadV0 {
	return CloseRunCommandPayloadV0{
		ClosureRef:    closureRef,
		PhaseID:       string(OrchestrationPhaseCierreV0),
		ValidationRef: "validation-001",
		Summary:       "Cerrar run con evidencia compacta de validacion final.",
		EvidenceRefs:  []string{"docs/contratos_cierre_run.md#RunClosed"},
	}
}

func assertCloseRunCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
