package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRequestReworkCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	for _, status := range []ReviewResultStatusV0{
		ReviewResultStatusChangesRequestedV0,
		ReviewResultStatusRejectedV0,
	} {
		run := mustRequestReworkReadyRunV0(t, "review-result-"+string(status), status)
		command := mustRequestReworkCommandV0(t, "cmd-request-rework-"+string(status), "idem-request-rework-"+string(status), "rework-request-"+string(status), "review-result-"+string(status))

		result, err := HandleCommandV0(run, command)
		if err != nil {
			t.Fatalf("handle RequestRework %s: %v", status, err)
		}
		assertSingleEventTypeV0(t, result, OrchestrationEventReworkRequestedV0)
		if len(result.Outbox) != 0 {
			t.Fatalf("outbox=%d, want empty", len(result.Outbox))
		}
		if result.Events[0].Sequence != run.LastSequence+1 {
			t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
		}
	}
}

func TestApplyReworkRequestedV0ProjectsCompactReworkOnce(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-001", ReviewResultStatusChangesRequestedV0)
	payload := validReworkRequestedPayloadV0("rework-request-001", "review-result-001")
	event := mustReworkRequestedEventV0(t, "evt-request-rework-reducer-001", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReworkRequested: %v", err)
	}
	want := []string{reworkRequestProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ReworkRequests, want) {
		t.Fatalf("rework_requests=%v, want %v", got.ReworkRequests, want)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.ReworkRequests, got.ReworkRequests) {
		t.Fatalf("rework_requests duplicated: %v", again.ReworkRequests)
	}
}

func TestReplayDurableEventsV0AcceptsReworkRequested(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-rework", 15, "idem-review-for-rework", "review-request-001")
	resultPayload := validRecordReviewResultPayloadV0("review-result-rework-replay", ReviewResultStatusRejectedV0)
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-rework", 16, "idem-record-result-for-rework", resultPayload)
	reworkPayload := validReworkRequestedPayloadV0("rework-request-replay", "review-result-rework-replay")
	rework := mustReworkRequestedEventWithKeyV0(t, "evt-durable-request-rework", 17, "idem-request-rework", reworkPayload)
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-rework", 14, "idem-open-revision-rework", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		rework,
		rework,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ReworkRequested: %v", err)
	}
	if got.LastSequence != 17 {
		t.Fatalf("last_sequence=%d, want 17", got.LastSequence)
	}
	want := []string{reworkRequestProjectionRefV0(reworkPayload)}
	if !reflect.DeepEqual(got.ReworkRequests, want) {
		t.Fatalf("rework_requests=%v, want %v", got.ReworkRequests, want)
	}
}

func TestRequestReworkCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-repeat", ReviewResultStatusChangesRequestedV0)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-repeat", "idem-request-rework-repeat", "rework-request-repeat", "review-result-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRequestReworkCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-effect-conflict", ReviewResultStatusChangesRequestedV0)
	payload := validRequestReworkPayloadV0("rework-request-effect-conflict", "review-result-effect-conflict")
	command := mustRequestReworkCommandWithPayloadV0(t, "cmd-request-rework-effect-conflict", "idem-request-rework-effect-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Registrar retrabajo compacto con otro resumen."
	conflicting := mustRequestReworkCommandWithPayloadV0(t, "cmd-request-rework-effect-conflict", "idem-request-rework-effect-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRequestReworkCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-effect-key", ReviewResultStatusChangesRequestedV0)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-key", "idem-request-rework-key", "rework-request-key", "review-result-effect-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRequestReworkCommandV0(t, "cmd-request-rework-key-2", "idem-request-rework-key-2", "rework-request-key", "review-result-effect-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyReworkRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-effect", ReviewResultStatusChangesRequestedV0)
	payload := validReworkRequestedPayloadV0("rework-request-effect", "review-result-effect")
	first := mustReworkRequestedEventWithKeyV0(t, "evt-request-rework-effect", run.LastSequence+1, "idem-request-rework-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustReworkRequestedEventWithKeyV0(t, "evt-request-rework-effect-conflict", applied.LastSequence+1, "idem-request-rework-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRequestReworkV0DoesNotAcceptReviewCloseTaskOrLaunchAgents(t *testing.T) {
	run := mustRequestReworkReadyRunV0(t, "review-result-no-effects", ReviewResultStatusRejectedV0)
	command := mustRequestReworkCommandV0(t, "cmd-request-rework-no-effects", "idem-request-rework-no-effects", "rework-request-no-effects", "review-result-no-effects")

	got := mustApplySingleCommandEventV0(t, run, command)
	if len(got.AcceptedReviews) != 0 {
		t.Fatalf("accepted_reviews=%v, want empty", got.AcceptedReviews)
	}
	if len(got.ClosedTasks) != 0 {
		t.Fatalf("closed_tasks=%v, want empty", got.ClosedTasks)
	}
	if !reflect.DeepEqual(got.Agents, run.Agents) {
		t.Fatalf("agents changed: before=%v after=%v", run.Agents, got.Agents)
	}
}

func mustRequestReworkReadyRunV0(t *testing.T, reviewResultRef string, status ReviewResultStatusV0) OrchestrationRunV0 {
	t.Helper()
	run := mustRecordReviewResultReadyRunV0(t)
	result := mustRecordReviewResultCommandV0(t, "cmd-record-result-for-"+reviewResultRef, "idem-record-result-for-"+reviewResultRef, reviewResultRef, status)
	return mustApplySingleCommandEventV0(t, run, result)
}

func mustRequestReworkCommandV0(t *testing.T, commandID string, idempotencyKey string, reworkRequestRef string, reviewResultRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustRequestReworkCommandWithPayloadV0(t, commandID, idempotencyKey, validRequestReworkPayloadV0(reworkRequestRef, reviewResultRef))
}

func mustRequestReworkCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RequestReworkCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestReworkCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustReworkRequestedEventV0(t *testing.T, eventID string, sequence int64, payload ReworkRequestedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	return mustReworkRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, payload)
}

func mustReworkRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload ReworkRequestedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewReworkRequestedEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}

func validRequestReworkPayloadV0(reworkRequestRef string, reviewResultRef string) RequestReworkCommandPayloadV0 {
	return RequestReworkCommandPayloadV0{
		ReworkRequestRef: reworkRequestRef,
		PhaseID:          string(OrchestrationPhaseRevisionV0),
		ReviewResultRef:  reviewResultRef,
		ReviewRequestID:  "review-request-001",
		DeliveryRef:      "delivery-001",
		Summary:          "Registrar retrabajo compacto solicitado por revision.",
		EvidenceRefs:     []string{"docs/contratos_revisiones.md#ReworkRequested"},
	}
}

func validReworkRequestedPayloadV0(reworkRequestRef string, reviewResultRef string) ReworkRequestedPayloadV0 {
	return reworkRequestedPayloadFromCommandV0(validRequestReworkPayloadV0(reworkRequestRef, reviewResultRef))
}

func assertRequestReworkCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}

func assertReworkRequestedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
