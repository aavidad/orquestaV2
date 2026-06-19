package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRecordReplanDecisionCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-001", "idem-record-replan-001", "replan-decision-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordReplanDecision: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventReplanDecisionRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestRecordReplanDecisionCommandV0NormalizesReparableActionAlias(t *testing.T) {
	payload := validReplanDecisionPayloadV0("replan-decision-alias")
	payload.AcceptedAction = "retry"

	command := mustRecordReplanDecisionCommandWithPayloadV0(t, "cmd-record-replan-alias", "idem-record-replan-alias", payload)
	got, err := decodeRecordReplanDecisionCommandPayloadV0(command.Payload)
	if err != nil {
		t.Fatalf("decode RecordReplanDecision alias payload: %v", err)
	}
	if got.AcceptedAction != ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("accepted_action=%s want %s", got.AcceptedAction, ReplanDecisionActionRetryTaskV0)
	}
}

func TestApplyReplanDecisionRecordedV0ProjectsCompactDecisionOnce(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	payload := validReplanDecisionPayloadV0("replan-decision-001")
	event := mustReplanDecisionRecordedEventV0(t, "evt-record-replan-reducer-001", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReplanDecisionRecorded: %v", err)
	}
	want := []string{replanDecisionProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ReplanDecisions, want) {
		t.Fatalf("replan_decisions=%v, want %v", got.ReplanDecisions, want)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.ReplanDecisions, got.ReplanDecisions) {
		t.Fatalf("replan_decisions duplicated: %v", again.ReplanDecisions)
	}
}

func TestReplayDurableEventsV0AcceptsReplanDecisionRecorded(t *testing.T) {
	review := mustReviewRequestedEventWithKeyV0(t, "evt-durable-review-for-replan", 15, "idem-review-for-replan", "review-request-001")
	resultPayload := validRecordReviewResultPayloadV0("review-result-replan-replay", ReviewResultStatusChangesRequestedV0)
	recorded := mustReviewResultRecordedEventWithKeyV0(t, "evt-durable-record-result-for-replan", 16, "idem-record-result-for-replan", resultPayload)
	reworkPayload := validReworkRequestedPayloadV0("rework-request-replan-replay", "review-result-replan-replay")
	rework := mustReworkRequestedEventWithKeyV0(t, "evt-durable-request-rework-for-replan", 17, "idem-request-rework-for-replan", reworkPayload)
	replanPayload := validReplanDecisionPayloadV0("replan-decision-replay")
	replanPayload.SourceRef = "rework-request-replan-replay"
	replan := mustReplanDecisionRecordedEventWithKeyV0(t, "evt-durable-record-replan", 18, "idem-record-replan", replanPayload)
	events := mustDeliveryReplayEventsV0(t)
	events = append(events,
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-revision-replan", 14, "idem-open-revision-replan", OrchestrationPhaseRevisionV0),
		review,
		recorded,
		rework,
		replan,
		replan,
	)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ReplanDecisionRecorded: %v", err)
	}
	if got.LastSequence != 18 {
		t.Fatalf("last_sequence=%d, want 18", got.LastSequence)
	}
	want := []string{replanDecisionProjectionRefV0(replanPayload)}
	if !reflect.DeepEqual(got.ReplanDecisions, want) {
		t.Fatalf("replan_decisions=%v, want %v", got.ReplanDecisions, want)
	}
}

func TestRecordReplanDecisionCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-repeat", "idem-record-replan-repeat", "replan-decision-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRecordReplanDecisionCommandV0AcceptsDurableReworkAfterPhaseAdvanced(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	openProgramacion := mustOpenPhaseCommandV0(
		t,
		"cmd-open-programacion-after-rework",
		"idem-open-programacion-after-rework",
		OrchestrationPhaseProgramacionV0,
	)
	run = mustApplySingleCommandEventV0(t, run, openProgramacion)
	command := mustRecordReplanDecisionCommandV0(
		t,
		"cmd-record-replan-after-programacion",
		"idem-record-replan-after-programacion",
		"replan-decision-after-programacion",
	)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordReplanDecision tras fase avanzada: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventReplanDecisionRecordedV0)
}

func TestRecordReplanDecisionCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	payload := validReplanDecisionPayloadV0("replan-decision-effect-conflict")
	command := mustRecordReplanDecisionCommandWithPayloadV0(t, "cmd-record-replan-effect-conflict", "idem-record-replan-effect-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Decision compacta de replan con otro resumen."
	conflicting := mustRecordReplanDecisionCommandWithPayloadV0(t, "cmd-record-replan-effect-conflict", "idem-record-replan-effect-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRecordReplanDecisionCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-key", "idem-record-replan-key", "replan-decision-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-key-2", "idem-record-replan-key-2", "replan-decision-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyReplanDecisionRecordedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	payload := validReplanDecisionPayloadV0("replan-decision-effect")
	first := mustReplanDecisionRecordedEventWithKeyV0(t, "evt-record-replan-effect", run.LastSequence+1, "idem-record-replan-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustReplanDecisionRecordedEventWithKeyV0(t, "evt-record-replan-effect-conflict", applied.LastSequence+1, "idem-record-replan-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRecordReplanDecisionV0DoesNotCreateFollowupEffects(t *testing.T) {
	run := mustReplanDecisionReadyRunV0(t)
	command := mustRecordReplanDecisionCommandV0(t, "cmd-record-replan-no-effects", "idem-record-replan-no-effects", "replan-decision-no-effects")

	got := mustApplySingleCommandEventV0(t, run, command)
	if !reflect.DeepEqual(got.Agents, run.Agents) ||
		!reflect.DeepEqual(got.CapacityRequests, run.CapacityRequests) ||
		!reflect.DeepEqual(got.Tasks, run.Tasks) ||
		!reflect.DeepEqual(got.ReworkRequests, run.ReworkRequests) {
		t.Fatalf("replan decision changed effects: before=%+v after=%+v", run, got)
	}
}

func mustReplanDecisionReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustRequestReworkReadyRunV0(t, "review-result-replan", ReviewResultStatusChangesRequestedV0)
	rework := mustRequestReworkCommandV0(t, "cmd-request-rework-for-replan", "idem-request-rework-for-replan", "rework-request-001", "review-result-replan")
	return mustApplySingleCommandEventV0(t, run, rework)
}

func mustRecordReplanDecisionCommandV0(t *testing.T, commandID string, idempotencyKey string, replanRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustRecordReplanDecisionCommandWithPayloadV0(t, commandID, idempotencyKey, validReplanDecisionPayloadV0(replanRef))
}

func mustRecordReplanDecisionCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload ReplanDecisionRecordedPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordReplanDecisionCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustReplanDecisionRecordedEventV0(t *testing.T, eventID string, sequence int64, payload ReplanDecisionRecordedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	return mustReplanDecisionRecordedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, payload)
}

func mustReplanDecisionRecordedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload ReplanDecisionRecordedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewReplanDecisionRecordedEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}

func validReplanDecisionPayloadV0(replanRef string) ReplanDecisionRecordedPayloadV0 {
	return ReplanDecisionRecordedPayloadV0{
		ReplanRef:      replanRef,
		RunRef:         "run-001",
		TaskRef:        "task-ncw-009",
		SourceRef:      "rework-request-001",
		AcceptedAction: ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{"followup-retry-task-001"},
		Summary:        "Decision compacta para repetir la tarea tras retrabajo.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-001"},
	}
}

func assertRecordReplanDecisionCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}

func assertReplanDecisionRecordedEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
