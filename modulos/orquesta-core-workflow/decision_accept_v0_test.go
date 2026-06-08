package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleAcceptDecisionCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	command := mustAcceptDecisionCommandV0(t, "cmd-decision-001", "idem-decision-001", "decision-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AcceptDecision: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventArchitectureDecisionAcceptedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyArchitectureDecisionAcceptedV0ProjectsRefOnce(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	event := mustArchitectureDecisionAcceptedEventV0(t, "evt-decision-reducer-001", run.LastSequence+1, "decision-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ArchitectureDecisionAccepted: %v", err)
	}
	if !reflect.DeepEqual(got.Decisions, []string{"decision-001"}) {
		t.Fatalf("decisions=%v, want [decision-001]", got.Decisions)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Decisions, got.Decisions) {
		t.Fatalf("decisions duplicated: %v", again.Decisions)
	}
}

func TestReplayDurableEventsV0AcceptsArchitectureDecisionAccepted(t *testing.T) {
	vote := mustVoteRequestedEventWithKeyV0(t, "evt-durable-vote-decision", 3, "idem-vote-decision", "vote-request-001")
	decision := mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-durable-decision", 4, "idem-decision", "decision-001")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-decision", 1, "idem-start-decision"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-decision", 2, "idem-open-decision", OrchestrationPhaseVotacionYDecisionV0),
		vote,
		decision,
		decision,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ArchitectureDecisionAccepted: %v", err)
	}
	if got.LastSequence != 4 {
		t.Fatalf("last_sequence=%d, want 4", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Decisions, []string{"decision-001"}) {
		t.Fatalf("decisions=%v, want [decision-001]", got.Decisions)
	}
}

func TestHandleAcceptDecisionCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	command := mustAcceptDecisionCommandV0(t, "cmd-decision-repeat", "idem-decision-repeat", "decision-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleAcceptDecisionCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	payload := validAcceptDecisionPayloadV0("decision-conflict")
	command := mustAcceptDecisionCommandWithPayloadV0(t, "cmd-decision-conflict", "idem-decision-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.AcceptedOptionRef = "option:arquitectura_modular_hexagonal"
	conflicting := mustAcceptDecisionCommandWithPayloadV0(t, "cmd-decision-conflict", "idem-decision-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleAcceptDecisionCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	command := mustAcceptDecisionCommandV0(t, "cmd-decision-key-conflict", "idem-decision-key-conflict", "decision-key-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustAcceptDecisionCommandV0(t, "cmd-decision-key-conflict-2", "idem-decision-key-conflict-2", "decision-key-conflict")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyArchitectureDecisionAcceptedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustDecisionReadyRunV0(t)
	first := mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-decision-effect", run.LastSequence+1, "idem-decision-effect", "decision-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-decision-effect-conflict", applied.LastSequence+1, "idem-decision-effect-conflict", "decision-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestAcceptDecisionCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAcceptDecisionCommandV0(t, "cmd-decision-phase", "idem-decision-phase", "decision-phase")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestAcceptDecisionCommandV0NoBloqueaDetalleOperativoBlando(t *testing.T) {
	payload := validAcceptDecisionPayloadV0("decision-forbidden")
	payload.Summary = "aceptar api_key=valor"

	if _, err := NewAcceptDecisionCommandV0(validCommandMetaV0("cmd-decision-forbidden", "idem-decision-forbidden"), payload); err != nil {
		t.Fatalf("NewAcceptDecisionCommandV0: %v", err)
	}
}

func TestArchitectureDecisionAcceptedEventV0NoBloqueaDetalleOperativoBlando(t *testing.T) {
	payload := architectureDecisionAcceptedPayloadFromCommandV0(validAcceptDecisionPayloadV0("decision-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

	if _, err := NewArchitectureDecisionAcceptedEventV0(reducerEventMetaV0("evt-decision-forbidden", 3), payload); err != nil {
		t.Fatalf("NewArchitectureDecisionAcceptedEventV0: %v", err)
	}
}

func TestHandleAcceptDecisionCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-decision", "idem-start-after-decision")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding AcceptDecision: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustAcceptDecisionCommandV0(t *testing.T, commandID string, idempotencyKey string, decisionRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustAcceptDecisionCommandWithPayloadV0(t, commandID, idempotencyKey, validAcceptDecisionPayloadV0(decisionRef))
}

func mustAcceptDecisionCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload AcceptDecisionCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAcceptDecisionCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustDecisionReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustVoteActiveRunV0(t)
	vote := mustRequestVoteCommandV0(t, "cmd-vote-for-decision", "idem-vote-for-decision", "vote-request-001")
	return mustApplySingleCommandEventV0(t, run, vote)
}

func mustArchitectureDecisionAcceptedEventV0(t *testing.T, eventID string, sequence int64, decisionRef string) OrchestrationEventV0 {
	t.Helper()
	return mustArchitectureDecisionAcceptedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, decisionRef)
}

func mustArchitectureDecisionAcceptedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, decisionRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewArchitectureDecisionAcceptedEventV0(meta, architectureDecisionAcceptedPayloadFromCommandV0(validAcceptDecisionPayloadV0(decisionRef)))
	return mustReducerEventV0(t, event, err)
}

func validAcceptDecisionPayloadV0(decisionRef string) AcceptDecisionCommandPayloadV0 {
	return AcceptDecisionCommandPayloadV0{
		DecisionRef:       decisionRef,
		PhaseID:           string(OrchestrationPhaseVotacionYDecisionV0),
		VoteRef:           "vote-request-001",
		AcceptedOptionRef: "option:arquitectura_hexagonal_i18n",
		Summary:           "Aceptar arquitectura hexagonal con i18n obligatorio y conectores en periferia.",
		EvidenceRefs:      []string{"docs/contratos_votaciones.md#VoteRequested"},
	}
}
