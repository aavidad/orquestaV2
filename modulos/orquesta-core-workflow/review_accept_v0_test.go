package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleAcceptReviewCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-001", "idem-accept-review-001", "accepted-review-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AcceptReview: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventReviewAcceptedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyReviewAcceptedV0ProjectsRefOnce(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	event := mustReviewAcceptedEventV0(t, "evt-accept-review-reducer-001", run.LastSequence+1, "accepted-review-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReviewAccepted: %v", err)
	}
	if !reflect.DeepEqual(got.AcceptedReviews, []string{"accepted-review-001"}) {
		t.Fatalf("accepted_reviews=%v, want [accepted-review-001]", got.AcceptedReviews)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.AcceptedReviews, got.AcceptedReviews) {
		t.Fatalf("accepted_reviews duplicated: %v", again.AcceptedReviews)
	}
}

func TestReplayDurableEventsV0AcceptsReviewAccepted(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-accept", 15, "idem-review-for-accept", "review-request-001")
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-accept", 16, "idem-record-result-for-accept", validRecordReviewResultPayloadV0("review-result-for-accept", ReviewResultStatusAcceptedV0))
	accepted := mustReviewAcceptedEventWithKeyV0(t, "evt-durable-accept-review", 17, "idem-accept-review", "accepted-review-001")
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-accept-review", 14, "idem-open-revision-accept-review", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		accepted,
		accepted,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ReviewAccepted: %v", err)
	}
	if got.LastSequence != 17 {
		t.Fatalf("last_sequence=%d, want 17", got.LastSequence)
	}
	if !reflect.DeepEqual(got.AcceptedReviews, []string{"accepted-review-001"}) {
		t.Fatalf("accepted_reviews=%v", got.AcceptedReviews)
	}
}

func TestAcceptReviewCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-repeat", "idem-accept-review-repeat", "accepted-review-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestAcceptReviewCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	payload := validAcceptReviewPayloadV0("accepted-review-conflict")
	command := mustAcceptReviewCommandWithPayloadV0(t, "cmd-accept-review-conflict", "idem-accept-review-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Aceptar revision compacta con otro resumen."
	conflicting := mustAcceptReviewCommandWithPayloadV0(t, "cmd-accept-review-conflict", "idem-accept-review-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestAcceptReviewCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	command := mustAcceptReviewCommandV0(t, "cmd-accept-review-key", "idem-accept-review-key", "accepted-review-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustAcceptReviewCommandV0(t, "cmd-accept-review-key-2", "idem-accept-review-key-2", "accepted-review-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyReviewAcceptedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustAcceptReviewReadyRunV0(t)
	first := mustReviewAcceptedEventWithKeyV0(t, "evt-accept-review-effect", run.LastSequence+1, "idem-accept-review-effect", "accepted-review-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustReviewAcceptedEventWithKeyV0(t, "evt-accept-review-effect-conflict", applied.LastSequence+1, "idem-accept-review-effect-conflict", "accepted-review-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestAcceptReviewCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-accept-review", "idem-start-after-accept-review")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding AcceptReview: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustAcceptReviewReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustReviewReadyRunV0(t)
	review := mustRequestReviewCommandV0(t, "cmd-request-review-for-accept", "idem-request-review-for-accept", "review-request-001")
	run = mustApplySingleCommandEventV0(t, run, review)
	result := mustRecordReviewResultCommandV0(t, "cmd-record-result-for-accept", "idem-record-result-for-accept", "review-result-for-accept", ReviewResultStatusAcceptedV0)
	return mustApplySingleCommandEventV0(t, run, result)
}

func mustAcceptReviewCommandV0(t *testing.T, commandID string, idempotencyKey string, acceptedReviewRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustAcceptReviewCommandWithPayloadV0(t, commandID, idempotencyKey, validAcceptReviewPayloadV0(acceptedReviewRef))
}

func mustAcceptReviewCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload AcceptReviewCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAcceptReviewCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustReviewAcceptedEventV0(t *testing.T, eventID string, sequence int64, acceptedReviewRef string) OrchestrationEventV0 {
	t.Helper()
	return mustReviewAcceptedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, acceptedReviewRef)
}

func mustReviewAcceptedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, acceptedReviewRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewReviewAcceptedEventV0(meta, reviewAcceptedPayloadFromCommandV0(validAcceptReviewPayloadV0(acceptedReviewRef)))
	return mustReducerEventV0(t, event, err)
}

func validAcceptReviewPayloadV0(acceptedReviewRef string) AcceptReviewCommandPayloadV0 {
	return AcceptReviewCommandPayloadV0{
		AcceptedReviewRef: acceptedReviewRef,
		PhaseID:           string(OrchestrationPhaseRevisionV0),
		ReviewRequestID:   "review-request-001",
		DeliveryRef:       "delivery-001",
		Summary:           "Aceptar revision compacta de una entrega validada.",
		EvidenceRefs:      []string{"docs/contratos_revisiones.md#ReviewRequested"},
	}
}

func assertAcceptReviewCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
