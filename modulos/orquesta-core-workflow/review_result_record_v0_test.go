package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRecordReviewResultCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	for _, status := range []ReviewResultStatusV0{
		ReviewResultStatusAcceptedV0,
		ReviewResultStatusChangesRequestedV0,
		ReviewResultStatusRejectedV0,
	} {
		command := mustRecordReviewResultCommandV0(t, "cmd-record-result-"+string(status), "idem-record-result-"+string(status), "review-result-"+string(status), status)

		result, err := HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("handle RecordReviewResult %s: %v", status, err)
		}
		assertSingleEventTypeV0(t, result, OrchestrationEventReviewResultRecordedV0)
		if len(result.Outbox) != 0 {
			t.Fatalf("outbox=%d, want empty", len(result.Outbox))
		}
		if result.Events[0].Sequence != run.LastSequence+1 {
			t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
		}
	}
}

func TestApplyReviewResultRecordedV0ProjectsCompactResultOnce(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	payload := validRecordReviewResultPayloadV0("review-result-001", ReviewResultStatusChangesRequestedV0)
	event := mustReviewResultRecordedEventV0(t, "evt-record-result-reducer-001", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReviewResultRecorded: %v", err)
	}
	want := []string{reviewResultProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ReviewResults, want) {
		t.Fatalf("review_results=%v, want %v", got.ReviewResults, want)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.ReviewResults, got.ReviewResults) {
		t.Fatalf("review_results duplicated: %v", again.ReviewResults)
	}
}

func TestReplayDurableEventsV0AcceptsReviewResultRecorded(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-result", 15, "idem-review-for-result", "review-request-001")
	payload := validRecordReviewResultPayloadV0("review-result-replay", ReviewResultStatusRejectedV0)
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result", 16, "idem-record-result", payload)
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-record-result", 14, "idem-open-revision-record-result", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		recorded,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ReviewResultRecorded: %v", err)
	}
	if got.LastSequence != 16 {
		t.Fatalf("last_sequence=%d, want 16", got.LastSequence)
	}
	want := []string{reviewResultProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ReviewResults, want) {
		t.Fatalf("review_results=%v, want %v", got.ReviewResults, want)
	}
}

func TestRecordReviewResultCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-repeat", "idem-record-result-repeat", "review-result-repeat", ReviewResultStatusAcceptedV0)
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRecordReviewResultCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	payload := validRecordReviewResultPayloadV0("review-result-effect-conflict", ReviewResultStatusAcceptedV0)
	command := mustRecordReviewResultCommandWithPayloadV0(t, "cmd-record-result-effect-conflict", "idem-record-result-effect-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Resultado compacto de revision con otro resumen."
	conflicting := mustRecordReviewResultCommandWithPayloadV0(t, "cmd-record-result-effect-conflict", "idem-record-result-effect-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRecordReviewResultCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-key", "idem-record-result-key", "review-result-key", ReviewResultStatusAcceptedV0)
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRecordReviewResultCommandV0(t, "cmd-record-result-key-2", "idem-record-result-key-2", "review-result-key", ReviewResultStatusAcceptedV0)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyReviewResultRecordedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	payload := validRecordReviewResultPayloadV0("review-result-effect", ReviewResultStatusAcceptedV0)
	first := mustReviewResultRecordedEventWithKeyV0(t, "evt-record-result-effect", run.LastSequence+1, "idem-record-result-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustReviewResultRecordedEventWithKeyV0(t, "evt-record-result-effect-conflict", applied.LastSequence+1, "idem-record-result-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRecordReviewResultV0DoesNotAcceptReviewOrCloseTask(t *testing.T) {
	run := mustRecordReviewResultReadyRunV0(t)
	command := mustRecordReviewResultCommandV0(t, "cmd-record-result-accepted", "idem-record-result-accepted", "review-result-accepted", ReviewResultStatusAcceptedV0)

	got := mustApplySingleCommandEventV0(t, run, command)
	if len(got.AcceptedReviews) != 0 {
		t.Fatalf("accepted_reviews=%v, want empty", got.AcceptedReviews)
	}
	if len(got.ClosedTasks) != 0 {
		t.Fatalf("closed_tasks=%v, want empty", got.ClosedTasks)
	}
	if len(got.ReviewResults) != 1 {
		t.Fatalf("review_results=%v, want one compact projection", got.ReviewResults)
	}
}

func TestRecordReviewResultCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-record-result", "idem-start-after-record-result")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding RecordReviewResult: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustRecordReviewResultReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustReviewReadyRunV0(t)
	review := mustRequestReviewCommandV0(t, "cmd-request-review-for-record-result", "idem-request-review-for-record-result", "review-request-001")
	run = mustApplySingleCommandEventV0(t, run, review)
	if len(run.AcceptedReviews) != 0 {
		t.Fatalf("test seed accepted_reviews=%v, want empty before result", run.AcceptedReviews)
	}
	return run
}

func mustRecordReviewResultCommandV0(t *testing.T, commandID string, idempotencyKey string, reviewResultRef string, status ReviewResultStatusV0) OrchestrationCommandV0 {
	t.Helper()
	return mustRecordReviewResultCommandWithPayloadV0(t, commandID, idempotencyKey, validRecordReviewResultPayloadV0(reviewResultRef, status))
}

func mustRecordReviewResultCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload ReviewResultV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordReviewResultCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustReviewResultRecordedEventV0(t *testing.T, eventID string, sequence int64, payload ReviewResultV0) OrchestrationEventV0 {
	t.Helper()
	return mustReviewResultRecordedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, payload)
}

func mustReviewResultRecordedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload ReviewResultV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewReviewResultRecordedEventV0(meta, reviewResultRecordedPayloadFromCommandV0(payload))
	return mustReducerEventV0(t, event, err)
}

func validRecordReviewResultPayloadV0(reviewResultRef string, status ReviewResultStatusV0) ReviewResultV0 {
	return ReviewResultV0{
		ReviewResultRef: reviewResultRef,
		ReviewRequestID: "review-request-001",
		DeliveryRef:     "delivery-001",
		Status:          status,
		Summary:         "Resultado compacto de revision registrado para la entrega.",
		EvidenceRefs:    []string{"docs/contratos_revisiones.md#ReviewResultRecorded"},
		QualityGateRef:  "quality-gate-001",
	}
}

func assertRecordReviewResultCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}

func assertReviewResultRecordedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
