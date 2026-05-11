package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRequestReviewCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustRequestReviewCommandV0(t, "cmd-review-001", "idem-review-001", "review-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestReview: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventReviewRequestedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyReviewRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	event := mustReviewRequestedEventV0(t, "evt-review-reducer-001", run.LastSequence+1, "review-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReviewRequested: %v", err)
	}
	if !reflect.DeepEqual(got.Reviews, []string{"review-request-001"}) {
		t.Fatalf("reviews=%v, want [review-request-001]", got.Reviews)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Reviews, got.Reviews) {
		t.Fatalf("reviews duplicated: %v", again.Reviews)
	}
}

func TestReplayDurableEventsV0AcceptsReviewRequested(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review", 15, "idem-review", "review-request-001")
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-review", 14, "idem-open-revision-review", OrchestrationPhaseRevisionV0),
		review,
		review,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ReviewRequested: %v", err)
	}
	if got.LastSequence != 15 {
		t.Fatalf("last_sequence=%d, want 15", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Reviews, []string{"review-request-001"}) {
		t.Fatalf("reviews=%v", got.Reviews)
	}
}

func TestRequestReviewCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustRequestReviewCommandV0(t, "cmd-review-repeat", "idem-review-repeat", "review-request-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRequestReviewCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	payload := validRequestReviewPayloadV0("review-request-conflict")
	command := mustRequestReviewCommandWithPayloadV0(t, "cmd-review-conflict", "idem-review-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Solicitar revision compacta con otro resumen."
	conflicting := mustRequestReviewCommandWithPayloadV0(t, "cmd-review-conflict", "idem-review-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRequestReviewCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	command := mustRequestReviewCommandV0(t, "cmd-review-key", "idem-review-key", "review-request-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRequestReviewCommandV0(t, "cmd-review-key-2", "idem-review-key-2", "review-request-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyReviewRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReviewReadyRunV0(t)
	first := mustReviewRequestedEventWithKeyV0(t, "evt-review-effect", run.LastSequence+1, "idem-review-effect", "review-request-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustReviewRequestedEventWithKeyV0(t, "evt-review-effect-conflict", applied.LastSequence+1, "idem-review-effect-conflict", "review-request-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRequestReviewCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-review", "idem-start-after-review")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding RequestReview: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustReviewReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryReadyRunV0(t)
	delivery := mustRegisterDeliveryCommandV0(t, "cmd-register-delivery-for-review", "idem-register-delivery-for-review", "delivery-001")
	run = mustApplySingleCommandEventV0(t, run, delivery)
	open := mustOpenPhaseCommandV0(t, "cmd-open-revision-review", "idem-open-revision-review", OrchestrationPhaseRevisionV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustRequestReviewCommandV0(t *testing.T, commandID string, idempotencyKey string, reviewRequestID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRequestReviewCommandWithPayloadV0(t, commandID, idempotencyKey, validRequestReviewPayloadV0(reviewRequestID))
}

func mustRequestReviewCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RequestReviewCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestReviewCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustReviewRequestedEventV0(t *testing.T, eventID string, sequence int64, reviewRequestID string) OrchestrationEventV0 {
	t.Helper()
	return mustReviewRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, reviewRequestID)
}

func mustReviewRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, reviewRequestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewReviewRequestedEventV0(meta, reviewRequestedPayloadFromCommandV0(validRequestReviewPayloadV0(reviewRequestID)))
	return mustReducerEventV0(t, event, err)
}

func validRequestReviewPayloadV0(reviewRequestID string) RequestReviewCommandPayloadV0 {
	return RequestReviewCommandPayloadV0{
		ReviewRequestID: reviewRequestID,
		PhaseID:         string(OrchestrationPhaseRevisionV0),
		DeliveryRef:     "delivery-001",
		Summary:         "Solicitar revision compacta de una entrega registrada.",
		EvidenceRefs:    []string{"docs/contratos_entregas.md#DeliveryRegistered"},
	}
}

func assertRequestReviewCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
