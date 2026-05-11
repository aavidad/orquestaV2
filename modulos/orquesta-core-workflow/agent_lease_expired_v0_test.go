package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestRegisterAgentLeaseExpiredCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lease")
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-expired", "idem-lease-expired", "agent-request-lease", "lease-ref-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterAgentLeaseExpired: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventAgentLeaseExpiredV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyAgentLeaseExpiredV0ProjectsCompactLeaseOnce(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lease")
	payload := validAgentLeaseExpiredPayloadV0("agent-request-lease", "lease-ref-001")
	event := mustAgentLeaseExpiredEventV0(t, "evt-lease-expired-reducer", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply AgentLeaseExpired: %v", err)
	}
	want := []string{agentLeaseExpiredProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.AgentLeaseExpirations, want) {
		t.Fatalf("agent_lease_expirations=%v, want %v", got.AgentLeaseExpirations, want)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.AgentLeaseExpirations, got.AgentLeaseExpirations) {
		t.Fatalf("agent_lease_expirations duplicated: %v", again.AgentLeaseExpirations)
	}
}

func TestReplayDurableEventsV0AcceptsAgentLeaseExpiredAndExactDuplicate(t *testing.T) {
	payload := validAgentLeaseExpiredPayloadV0("agent-request-lease", "lease-ref-replay")
	event := mustAgentLeaseExpiredEventWithKeyV0(t, "evt-durable-lease-expired", 6, "idem-lease-expired", payload)
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-lease", 1, "idem-start-lease"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-lease", 2, "idem-phase-lease", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-lease", 3, "idem-capacity-lease", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-lease", 4, "idem-capacity-decision-lease", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent-lease", 5, "idem-agent-lease", "agent-request-lease"),
		event,
		event,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable AgentLeaseExpired: %v", err)
	}
	if got.LastSequence != 6 {
		t.Fatalf("last_sequence=%d, want 6", got.LastSequence)
	}
	want := []string{agentLeaseExpiredProjectionRefV0(payload)}
	if !reflect.DeepEqual(got.AgentLeaseExpirations, want) {
		t.Fatalf("agent_lease_expirations=%v, want %v", got.AgentLeaseExpirations, want)
	}
}

func TestRegisterAgentLeaseExpiredCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-repeat")
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-repeat", "idem-lease-repeat", "agent-request-repeat", "lease-ref-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lease-effect")
	payload := validRegisterAgentLeaseExpiredPayloadV0("agent-request-lease-effect", "lease-ref-effect")
	command := mustRegisterAgentLeaseExpiredCommandWithPayloadV0(t, "cmd-lease-effect", "idem-lease-effect", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.EvidenceRefs = []string{"evidence-ref-lease-changed"}
	conflicting := mustRegisterAgentLeaseExpiredCommandWithPayloadV0(t, "cmd-lease-effect", "idem-lease-effect", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lease-key")
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-key", "idem-lease-key", "agent-request-lease-key", "lease-ref-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-key-2", "idem-lease-key-2", "agent-request-lease-key", "lease-ref-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyAgentLeaseExpiredV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lease-event-effect")
	payload := validAgentLeaseExpiredPayloadV0("agent-request-lease-event-effect", "lease-ref-event-effect")
	first := mustAgentLeaseExpiredEventWithKeyV0(t, "evt-lease-effect", run.LastSequence+1, "idem-lease-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustAgentLeaseExpiredEventWithKeyV0(t, "evt-lease-effect-conflict", applied.LastSequence+1, "idem-lease-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterAgentLeaseExpiredV0DoesNotStopOrFailAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-observed")
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-observed", "idem-lease-observed", "agent-request-observed", "lease-ref-observed")

	got := mustApplySingleCommandEventV0(t, run, command)
	if !reflect.DeepEqual(got.StoppedAgents, run.StoppedAgents) {
		t.Fatalf("stopped_agents changed: before=%v after=%v", run.StoppedAgents, got.StoppedAgents)
	}
	if !reflect.DeepEqual(got.FailedAgents, run.FailedAgents) {
		t.Fatalf("failed_agents changed: before=%v after=%v", run.FailedAgents, got.FailedAgents)
	}
	if !reflect.DeepEqual(got.Agents, run.Agents) {
		t.Fatalf("agents changed: before=%v after=%v", run.Agents, got.Agents)
	}
}

func mustRegisterAgentLeaseExpiredCommandV0(t *testing.T, commandID string, idempotencyKey string, agentID string, leaseRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterAgentLeaseExpiredCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterAgentLeaseExpiredPayloadV0(agentID, leaseRef))
}

func mustRegisterAgentLeaseExpiredCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterAgentLeaseExpiredCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustAgentLeaseExpiredEventV0(t *testing.T, eventID string, sequence int64, payload AgentLeaseExpiredPayloadV0) OrchestrationEventV0 {
	t.Helper()
	return mustAgentLeaseExpiredEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, payload)
}

func mustAgentLeaseExpiredEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload AgentLeaseExpiredPayloadV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentLeaseExpiredEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}

func validRegisterAgentLeaseExpiredPayloadV0(agentID string, leaseRef string) RegisterAgentLeaseExpiredCommandPayloadV0 {
	return RegisterAgentLeaseExpiredCommandPayloadV0{
		RunRef:            "run-001",
		AgentRequestID:    agentID,
		LeaseRef:          leaseRef,
		ReasonCode:        "heartbeat_timeout",
		ObservedAt:        "2026-05-04T10:00:00Z",
		RecommendedAction: AgentLeaseActionStopAgentV0,
		EvidenceRefs:      []string{"evidence-ref-lease-001"},
	}
}

func validAgentLeaseExpiredPayloadV0(agentID string, leaseRef string) AgentLeaseExpiredPayloadV0 {
	return agentLeaseExpiredPayloadFromCommandV0(validRegisterAgentLeaseExpiredPayloadV0(agentID, leaseRef))
}
