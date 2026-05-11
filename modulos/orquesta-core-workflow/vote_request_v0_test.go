package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRequestVoteCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	command := mustRequestVoteCommandV0(t, "cmd-vote-001", "idem-vote-001", "vote-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventVoteRequestedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyVoteRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	event := mustVoteRequestedEventV0(t, "evt-vote-reducer-001", run.LastSequence+1, "vote-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply VoteRequested: %v", err)
	}
	if !reflect.DeepEqual(got.Votes, []string{"vote-request-001"}) {
		t.Fatalf("votes=%v, want [vote-request-001]", got.Votes)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Votes, got.Votes) {
		t.Fatalf("votes duplicated: %v", again.Votes)
	}
}

func TestReplayDurableEventsV0AcceptsVoteRequested(t *testing.T) {
	vote := mustVoteRequestedEventWithKeyV0(t, "evt-durable-vote", 3, "idem-vote", "vote-request-001")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-vote", 1, "idem-start-vote"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-vote", 2, "idem-open-vote", OrchestrationPhaseVotacionYDecisionV0),
		vote,
		vote,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable VoteRequested: %v", err)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence=%d, want 3", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Votes, []string{"vote-request-001"}) {
		t.Fatalf("votes=%v, want [vote-request-001]", got.Votes)
	}
}

func TestHandleRequestVoteCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	command := mustRequestVoteCommandV0(t, "cmd-vote-repeat", "idem-vote-repeat", "vote-request-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleRequestVoteCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	payload := validRequestVotePayloadV0("vote-request-conflict")
	command := mustRequestVoteCommandWithPayloadV0(t, "cmd-vote-conflict", "idem-vote-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.DecisionTopicRef = "decision_topic:seguridad_inicial"
	conflicting := mustRequestVoteCommandWithPayloadV0(t, "cmd-vote-conflict", "idem-vote-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleRequestVoteCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	command := mustRequestVoteCommandV0(t, "cmd-vote-key", "idem-vote-key", "vote-request-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRequestVoteCommandV0(t, "cmd-vote-key-2", "idem-vote-key-2", "vote-request-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyVoteRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	first := mustVoteRequestedEventWithKeyV0(t, "evt-vote-effect", run.LastSequence+1, "idem-vote-effect", "vote-request-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustVoteRequestedEventWithKeyV0(t, "evt-vote-effect-conflict", applied.LastSequence+1, "idem-vote-effect-conflict", "vote-request-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRequestVoteCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustRequestVoteCommandV0(t, "cmd-vote-phase", "idem-vote-phase", "vote-request-phase")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestRequestVoteCommandV0RejectsNonVotePhase(t *testing.T) {
	payload := validRequestVotePayloadV0("vote-request-wrong-phase")
	payload.PhaseID = string(OrchestrationPhaseProgramacionV0)

	_, err := NewRequestVoteCommandV0(validCommandMetaV0("cmd-vote-wrong-phase", "idem-vote-wrong-phase"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestRequestVoteCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRequestVotePayloadV0("vote-request-forbidden")
	payload.Summary = "votar provider con Codex"

	_, err := NewRequestVoteCommandV0(validCommandMetaV0("cmd-vote-forbidden", "idem-vote-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestVoteRequestedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := voteRequestedPayloadFromCommandV0(validRequestVotePayloadV0("vote-request-event-forbidden"))
	payload.Summary = "usar Claude"

	_, err := NewVoteRequestedEventV0(reducerEventMetaV0("evt-vote-forbidden", 3), payload)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestHandleRequestVoteCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-vote", "idem-start-after-vote")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding vote command: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustVoteActiveRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-vote-base", "idem-open-vote-base", OrchestrationPhaseVotacionYDecisionV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustRequestVoteCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRequestVoteCommandWithPayloadV0(t, commandID, idempotencyKey, validRequestVotePayloadV0(requestID))
}

func mustRequestVoteCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RequestVoteCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestVoteCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustVoteRequestedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustVoteRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustVoteRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewVoteRequestedEventV0(meta, voteRequestedPayloadFromCommandV0(validRequestVotePayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validRequestVotePayloadV0(requestID string) RequestVoteCommandPayloadV0 {
	return RequestVoteCommandPayloadV0{
		VoteRequestID:              requestID,
		PhaseID:                    string(OrchestrationPhaseVotacionYDecisionV0),
		DecisionTopicRef:           "decision_topic:arquitectura_inicial",
		BrainstormRef:              "brainstorm-request-001",
		Summary:                    "Solicitar votos compactos sobre opciones de arquitectura hexagonal e i18n.",
		MinimumRecommendedCapacity: OrchestrationCapacityXHighV0,
		EvidenceRefs:               []string{"docs/contratos_brainstorm.md#BrainstormRequested"},
	}
}
