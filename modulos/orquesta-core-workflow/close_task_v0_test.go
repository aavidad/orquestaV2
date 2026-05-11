package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleCloseTaskCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-001", "idem-close-task-001", "task-ncw-009")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle CloseTask: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventTaskClosedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyTaskClosedV0ProjectsRefOnce(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	event := mustTaskClosedEventV0(t, "evt-close-task-reducer-001", run.LastSequence+1, "task-ncw-009")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply TaskClosed: %v", err)
	}
	if !reflect.DeepEqual(got.ClosedTasks, []string{"task-ncw-009"}) {
		t.Fatalf("closed_tasks=%v, want [task-ncw-009]", got.ClosedTasks)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.ClosedTasks, got.ClosedTasks) {
		t.Fatalf("closed_tasks duplicated: %v", again.ClosedTasks)
	}
}

func TestReplayDurableEventsV0AcceptsTaskClosed(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-close-task", 15, "idem-review-for-close-task", "review-request-001")
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-close-task", 16, "idem-record-result-for-close-task", validRecordReviewResultPayloadV0("review-result-for-close-task", ReviewResultStatusAcceptedV0))
	accepted := mustReviewAcceptedEventWithKeyV0(t, "evt-durable-accept-review-for-close-task", 17, "idem-accept-review-for-close-task", "accepted-review-001")
	closed := mustTaskClosedEventWithKeyV0(t, "evt-durable-close-task", 18, "idem-close-task", "task-ncw-009")
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-close-task", 14, "idem-open-revision-close-task", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		accepted,
		closed,
		closed,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable TaskClosed: %v", err)
	}
	if got.LastSequence != 18 {
		t.Fatalf("last_sequence=%d, want 18", got.LastSequence)
	}
	if !reflect.DeepEqual(got.ClosedTasks, []string{"task-ncw-009"}) {
		t.Fatalf("closed_tasks=%v", got.ClosedTasks)
	}
}

func TestCloseTaskCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-repeat", "idem-close-task-repeat", "task-ncw-009")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestCloseTaskCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	payload := validCloseTaskPayloadV0("task-ncw-009")
	command := mustCloseTaskCommandWithPayloadV0(t, "cmd-close-task-effect-conflict", "idem-close-task-effect-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Cerrar microtarea revisada con otro resumen."
	conflicting := mustCloseTaskCommandWithPayloadV0(t, "cmd-close-task-effect-conflict", "idem-close-task-effect-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestCloseTaskCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-key", "idem-close-task-key", "task-ncw-009")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustCloseTaskCommandV0(t, "cmd-close-task-key-2", "idem-close-task-key-2", "task-ncw-009")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyTaskClosedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	first := mustTaskClosedEventWithKeyV0(t, "evt-close-task-effect", run.LastSequence+1, "idem-close-task-effect", "task-ncw-009")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustTaskClosedEventWithKeyV0(t, "evt-close-task-effect-conflict", applied.LastSequence+1, "idem-close-task-effect-conflict", "task-ncw-009")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestCloseTaskCommandV0DoesNotClosePhaseOrRun(t *testing.T) {
	run := mustCloseTaskReadyRunV0(t)
	command := mustCloseTaskCommandV0(t, "cmd-close-task-state", "idem-close-task-state", "task-ncw-009")

	got := mustApplySingleCommandEventV0(t, run, command)
	if got.Status != OrchestrationRunStatusActiveV0 {
		t.Fatalf("status=%q, want active", got.Status)
	}
	if got.CurrentPhase != OrchestrationPhaseRevisionV0 {
		t.Fatalf("current_phase=%q, want revision", got.CurrentPhase)
	}
	if reducerPhaseByIDV0(t, got, OrchestrationPhaseRevisionV0).Status != OrchestrationPhaseStatusActiveV0 {
		t.Fatalf("revision phase not active after TaskClosed")
	}
}

func mustCloseTaskReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustAcceptReviewReadyRunV0(t)
	accept := mustAcceptReviewCommandV0(t, "cmd-accept-review-for-close-task", "idem-accept-review-for-close-task", "accepted-review-001")
	return mustApplySingleCommandEventV0(t, run, accept)
}

func mustCloseTaskCommandV0(t *testing.T, commandID string, idempotencyKey string, taskID string) OrchestrationCommandV0 {
	t.Helper()
	return mustCloseTaskCommandWithPayloadV0(t, commandID, idempotencyKey, validCloseTaskPayloadV0(taskID))
}

func mustCloseTaskCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload CloseTaskCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewCloseTaskCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustTaskClosedEventV0(t *testing.T, eventID string, sequence int64, taskID string) OrchestrationEventV0 {
	t.Helper()
	return mustTaskClosedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, taskID)
}

func mustTaskClosedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, taskID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewTaskClosedEventV0(meta, taskClosedPayloadFromCommandV0(validCloseTaskPayloadV0(taskID)))
	return mustReducerEventV0(t, event, err)
}

func validCloseTaskPayloadV0(taskID string) CloseTaskCommandPayloadV0 {
	return CloseTaskCommandPayloadV0{
		TaskID:            taskID,
		PhaseID:           string(OrchestrationPhaseRevisionV0),
		DeliveryRef:       "delivery-001",
		AcceptedReviewRef: "accepted-review-001",
		Summary:           "Cerrar microtarea revisada con evidencia compacta.",
		EvidenceRefs:      []string{"docs/contratos_cierre_tareas.md#TaskClosed"},
	}
}

func assertCloseTaskCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
