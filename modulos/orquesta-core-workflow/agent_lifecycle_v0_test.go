package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegisterAgentStartedCommandV0ProjectsStartedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-started")
	command := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started", "idem-agent-started", "agent-request-started")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterAgentStarted: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventAgentStartedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	started := mustApplyReducerEventV0(t, run, result.Events[0])
	if !reflect.DeepEqual(started.StartedAgents, []string{"agent-request-started"}) {
		t.Fatalf("started_agents=%v", started.StartedAgents)
	}

	result, err = HandleCommandV0(started, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterAgentStartedCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-started-effect")
	payload := validRegisterAgentStartedPayloadV0("agent-request-started-effect")
	command := mustRegisterAgentStartedCommandWithPayloadV0(t, "cmd-agent-started-effect", "idem-agent-started-effect", payload)
	started := mustApplySingleCommandEventV0(t, run, command)

	payload.AckRef = "ack-ref-changed"
	conflicting := mustRegisterAgentStartedCommandWithPayloadV0(t, "cmd-agent-started-effect", "idem-agent-started-effect", payload)
	_, err := HandleCommandV0(started, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterAgentStartedCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-started-key")
	command := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started-key", "idem-agent-started-key", "agent-request-started-key")
	started := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started-key-2", "idem-agent-started-key-2", "agent-request-started-key")
	_, err := HandleCommandV0(started, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyAgentStartedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-started-event-effect")
	first := mustAgentStartedEventWithKeyV0(t, "evt-agent-started-effect", run.LastSequence+1, "idem-agent-started-effect", "agent-request-started-event-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustAgentStartedEventWithKeyV0(t, "evt-agent-started-effect-conflict", applied.LastSequence+1, "idem-agent-started-effect-conflict", "agent-request-started-event-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterAgentFailedCommandV0ProjectsFailedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-failed")
	command := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed", "idem-agent-failed", "agent-request-failed")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterAgentFailed: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventAgentFailedV0)
	failed := mustApplyReducerEventV0(t, run, result.Events[0])
	if !reflect.DeepEqual(failed.FailedAgents, []string{"agent-request-failed"}) {
		t.Fatalf("failed_agents=%v", failed.FailedAgents)
	}
}

func TestRegisterAgentFailedCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-failed-effect")
	payload := validRegisterAgentFailedPayloadV0("agent-request-failed-effect")
	command := mustRegisterAgentFailedCommandWithPayloadV0(t, "cmd-agent-failed-effect", "idem-agent-failed-effect", payload)
	failed := mustApplySingleCommandEventV0(t, run, command)

	payload.ReasonCode = "launch_timeout"
	conflicting := mustRegisterAgentFailedCommandWithPayloadV0(t, "cmd-agent-failed-effect", "idem-agent-failed-effect", payload)
	_, err := HandleCommandV0(failed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterAgentFailedCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-failed-key")
	command := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-key", "idem-agent-failed-key", "agent-request-failed-key")
	failed := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-key-2", "idem-agent-failed-key-2", "agent-request-failed-key")
	_, err := HandleCommandV0(failed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyAgentFailedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-failed-event-effect")
	first := mustAgentFailedEventWithKeyV0(t, "evt-agent-failed-effect", run.LastSequence+1, "idem-agent-failed-effect", "agent-request-failed-event-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustAgentFailedEventWithKeyV0(t, "evt-agent-failed-effect-conflict", applied.LastSequence+1, "idem-agent-failed-effect-conflict", "agent-request-failed-event-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterAgentStartedCommandV0RequiresRequestedAgent(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started-missing", "idem-agent-started-missing", "agent-request-missing")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.agent_request_id" {
		t.Fatalf("error=%+v, want invalid agent_request_id", publicErr)
	}
}

func TestAgentLifecycleV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterAgentStartedPayloadV0("agent-request-forbidden")
	payload.LaunchRef = "runtime-launch-ref"

	_, err := NewRegisterAgentStartedCommandV0(validCommandMetaV0("cmd-agent-started-forbidden", "idem-agent-started-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func mustRegisterAgentStartedCommandV0(t *testing.T, commandID string, idempotencyKey string, agentID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterAgentStartedCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterAgentStartedPayloadV0(agentID))
}

func mustRegisterAgentStartedCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterAgentStartedCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterAgentStartedCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustRegisterAgentFailedCommandV0(t *testing.T, commandID string, idempotencyKey string, agentID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterAgentFailedCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterAgentFailedPayloadV0(agentID))
}

func mustRegisterAgentFailedCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterAgentFailedCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterAgentFailedCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func validRegisterAgentStartedPayloadV0(agentID string) RegisterAgentStartedCommandPayloadV0 {
	return RegisterAgentStartedCommandPayloadV0{
		AgentRequestID: agentID,
		LaunchRef:      "launch-ref-001",
		AckRef:         "ack-ref-001",
		ReadinessRef:   "readiness-ref-001",
		EvidenceRefs:   []string{"evidence-ref-agent-started-001"},
	}
}

func validRegisterAgentFailedPayloadV0(agentID string) RegisterAgentFailedCommandPayloadV0 {
	return RegisterAgentFailedCommandPayloadV0{
		AgentRequestID: agentID,
		LaunchRef:      "launch-ref-001",
		ReasonCode:     "launch_rejected",
		Retryable:      true,
		EvidenceRefs:   []string{"evidence-ref-agent-failed-001"},
	}
}
