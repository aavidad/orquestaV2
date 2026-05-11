package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestRecordConcurrencyGateCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-gate", "idem-concurrency-gate", validConcurrencyGatePayloadV0("gate-ref-001", ConcurrencyGateDecisionAllowRequestAgentV0))

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordConcurrencyGate: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventConcurrencyGateRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
}

func TestApplyConcurrencyGateRecordedV0ProjectsCompactGateOnce(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGateRecordedPayloadV0("gate-ref-001", ConcurrencyGateDecisionAllowRequestAgentV0)
	event := mustConcurrencyGateRecordedEventV0(t, "evt-concurrency-gate", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ConcurrencyGateRecorded: %v", err)
	}
	want := []string{concurrencyGateProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ConcurrencyGates, want) {
		t.Fatalf("concurrency_gates=%v, want %v", got.ConcurrencyGates, want)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.ConcurrencyGates, got.ConcurrencyGates) {
		t.Fatalf("concurrency_gates duplicated: %v", again.ConcurrencyGates)
	}
}

func TestReplayDurableEventsV0AcceptsConcurrencyGateRecordedAndExactDuplicate(t *testing.T) {
	payload := validConcurrencyGateRecordedPayloadV0("gate-ref-replay", ConcurrencyGateDecisionAllowRequestAgentV0)
	event := mustConcurrencyGateRecordedEventWithKeyV0(t, "evt-durable-concurrency-gate", 3, "idem-concurrency-gate", payload)
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-concurrency-gate", 1, "idem-start-concurrency-gate"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-concurrency-gate", 2, "idem-phase-concurrency-gate", OrchestrationPhaseProgramacionV0),
		event,
		event,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable ConcurrencyGateRecorded: %v", err)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence=%d, want 3", got.LastSequence)
	}
	want := []string{concurrencyGateProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.ConcurrencyGates, want) {
		t.Fatalf("concurrency_gates=%v, want %v", got.ConcurrencyGates, want)
	}
}

func TestRecordConcurrencyGateCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-repeat", "idem-concurrency-repeat", validConcurrencyGatePayloadV0("gate-ref-repeat", ConcurrencyGateDecisionAllowRequestAgentV0))
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRecordConcurrencyGateCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGatePayloadV0("gate-ref-effect", ConcurrencyGateDecisionAllowRequestAgentV0)
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-effect", "idem-concurrency-effect", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "concurrency_gate con otro resumen."
	conflicting := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-effect", "idem-concurrency-effect", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRecordConcurrencyGateCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGatePayloadV0("gate-ref-key", ConcurrencyGateDecisionAllowRequestAgentV0)
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-key", "idem-concurrency-key", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-key-2", "idem-concurrency-key-2", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyConcurrencyGateRecordedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGateRecordedPayloadV0("gate-ref-event-effect", ConcurrencyGateDecisionAllowRequestAgentV0)
	first := mustConcurrencyGateRecordedEventWithKeyV0(t, "evt-concurrency-effect", run.LastSequence+1, "idem-concurrency-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustConcurrencyGateRecordedEventWithKeyV0(t, "evt-concurrency-effect-conflict", applied.LastSequence+1, "idem-concurrency-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRecordConcurrencyGateV0DoesNotRequestOrStopAgents(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-before-gate")
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-no-effects", "idem-concurrency-no-effects", validConcurrencyGatePayloadV0("gate-ref-no-effects", ConcurrencyGateDecisionBlockRequestAgentV0))

	got := mustApplySingleCommandEventV0(t, run, command)
	if !reflect.DeepEqual(got.Agents, run.Agents) {
		t.Fatalf("agents changed: before=%v after=%v", run.Agents, got.Agents)
	}
	if !reflect.DeepEqual(got.StoppedAgents, run.StoppedAgents) {
		t.Fatalf("stopped_agents changed: before=%v after=%v", run.StoppedAgents, got.StoppedAgents)
	}
}

func mustRecordConcurrencyGateCommandV0(t *testing.T, commandID string, idempotencyKey string, payload RecordConcurrencyGateCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustConcurrencyGateRecordedEventV0(t *testing.T, eventID string, sequence int64, payload ConcurrencyGateRecordedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	return mustConcurrencyGateRecordedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, payload)
}

func mustConcurrencyGateRecordedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload ConcurrencyGateRecordedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewConcurrencyGateRecordedEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}

func validConcurrencyGatePayloadV0(gateRef string, decision ConcurrencyGateDecisionV0) RecordConcurrencyGateCommandPayloadV0 {
	payload := RecordConcurrencyGateCommandPayloadV0{
		RunRef:           "run-001",
		GateRef:          gateRef,
		PlanRef:          "plan-ref-" + gateRef,
		SubjectClaimRefs: []string{"claim-ref-ready-001"},
		ReadyClaimRefs:   []string{"claim-ref-ready-001"},
		Decision:         decision,
		Summary:          "concurrency_gate decision compacta.",
		EvidenceRefs:     []string{"evidence-ref-concurrency-gate-001"},
	}
	if decision == ConcurrencyGateDecisionBlockRequestAgentV0 {
		payload.ReadyClaimRefs = nil
		payload.BlockedClaimRefs = []string{"claim-ref-ready-001"}
		payload.ConflictRefs = []string{"conflict-ref-001"}
	}
	if decision == ConcurrencyGateDecisionAskDirectorV0 {
		payload.SubjectClaimRefs = nil
		payload.ReadyClaimRefs = nil
		payload.BlockedClaimRefs = nil
	}
	return payload
}

func validConcurrencyGateRecordedPayloadV0(gateRef string, decision ConcurrencyGateDecisionV0) ConcurrencyGateRecordedPayloadV0 {
	return concurrencyGateRecordedPayloadFromCommandV0(validConcurrencyGatePayloadV0(gateRef, decision))
}
