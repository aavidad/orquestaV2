package orquestacoreworkflow

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestHandleRequestAgentCommandV0ReturnsEventAndOutbox(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-001", "idem-agent-001", "agent-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestAgent: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventAgentRequestedV0)
	if len(result.Outbox) != 1 {
		t.Fatalf("outbox=%d, want 1", len(result.Outbox))
	}
	outbox := result.Outbox[0]
	if outbox.MessageType != OutboxMessageLaunchRuntimeAgentV0 || outbox.TargetPort != OutboxTargetAgentLauncherV0 {
		t.Fatalf("unexpected outbox envelope: %+v", outbox)
	}
	if outbox.CausationEventID != result.Events[0].EventID {
		t.Fatalf("causation_event_id=%q, want %q", outbox.CausationEventID, result.Events[0].EventID)
	}
	var payload LaunchRuntimeAgentRequestV0
	if err := json.Unmarshal(outbox.Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload.AgentRequestID != "agent-request-001" || payload.RunID != command.RunID {
		t.Fatalf("unexpected outbox payload: %+v", payload)
	}
	if payload.Role != "implementacion" || payload.CapacityRequestRef != defaultAgentCapacityRequestIDV0 {
		t.Fatalf("unexpected logical refs: %+v", payload)
	}
}

func TestApplyAgentRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustReducerRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	event := mustAgentRequestedEventV0(t, "evt-agent-reducer-001", run.LastSequence+1, "agent-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply AgentRequested: %v", err)
	}
	if !reflect.DeepEqual(got.Agents, []string{"agent-request-001"}) {
		t.Fatalf("agents=%v, want [agent-request-001]", got.Agents)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Agents, got.Agents) {
		t.Fatalf("agents duplicated: %v", again.Agents)
	}
}

func TestReplayDurableEventsV0AcceptsAgentRequested(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-agent", 1, "idem-start-agent"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-agent", 2, "idem-phase-agent", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-agent", 3, "idem-capacity-agent", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-agent", 4, "idem-capacity-decision-agent", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent", 5, "idem-agent", "agent-request-001"),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent", 5, "idem-agent", "agent-request-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable AgentRequested: %v", err)
	}
	if got.LastSequence != 5 {
		t.Fatalf("last_sequence=%d, want 5", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Agents, []string{"agent-request-001"}) {
		t.Fatalf("agents=%v, want [agent-request-001]", got.Agents)
	}
}

func TestHandleRequestAgentCommandV0RetriesPendingLaunchAfterPartialPersist(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-repeat", "idem-agent-repeat", "agent-request-001")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	if err != nil {
		t.Fatalf("retry RequestAgent after partial persist: %v", err)
	}
	if len(result.Events) != 0 || len(result.Outbox) != 1 {
		t.Fatalf("result=%+v, want no events and one launch outbox", result)
	}
	if result.Outbox[0].MessageType != OutboxMessageLaunchRuntimeAgentV0 ||
		result.Outbox[0].TargetPort != OutboxTargetAgentLauncherV0 {
		t.Fatalf("unexpected launch retry outbox: %+v", result.Outbox[0])
	}
}

func TestHandleRequestAgentCommandV0RepeatedAfterStartedDoesNotDuplicate(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-started-repeat-request", "idem-agent-started-repeat-request", "agent-request-started-repeat")
	run = mustApplySingleCommandEventV0(t, run, command)
	started := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started-repeat", "idem-agent-started-repeat", "agent-request-started-repeat")
	run = mustApplySingleCommandEventV0(t, run, started)

	result, err := HandleCommandV0(run, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func mustRequestAgentCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestAgentCommandV0(validCommandMetaV0(commandID, idempotencyKey), validRequestAgentPayloadV0(requestID))
	return mustCommandV0(t, command, err)
}

func mustAgentRequestedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustAgentRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustAgentRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentRequestedEventV0(meta, agentRequestedPayloadFromCommandV0(validRequestAgentPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validRequestAgentPayloadV0(requestID string) RequestAgentCommandPayloadV0 {
	return RequestAgentCommandPayloadV0{
		AgentRequestID:     requestID,
		PhaseID:            string(OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ncw-011",
		CapacityRequestRef: defaultAgentCapacityRequestIDV0,
		Role:               "implementacion",
		Summary:            "Ejecutar una microtarea acotada con evidencias compactas.",
		EvidenceRefs:       []string{"docs/contratos_agentes.md#RequestAgent"},
	}
}
