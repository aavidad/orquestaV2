package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRegisterAgentStartedCommandV0RejectsFailedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-start-after-failed")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-before-start", "idem-agent-failed-before-start", "agent-request-start-after-failed"))
	command := mustRegisterAgentStartedCommandV0(t, "cmd-start-after-failed", "idem-start-after-failed", "agent-request-start-after-failed")

	_, err := HandleCommandV0(run, command)
	assertAgentLifecycleCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterAgentStartedCommandV0RejectsStoppedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-start-after-stop")
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-start", "idem-stop-before-start", "agent-request-start-after-stop"))
	command := mustRegisterAgentStartedCommandV0(t, "cmd-start-after-stop", "idem-start-after-stop", "agent-request-start-after-stop")

	_, err := HandleCommandV0(run, command)
	assertAgentLifecycleCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterAgentFailedCommandV0RejectsStartedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-fail-after-start")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentStartedCommandV0(t, "cmd-start-before-fail", "idem-start-before-fail", "agent-request-fail-after-start"))
	command := mustRegisterAgentFailedCommandV0(t, "cmd-fail-after-start", "idem-fail-after-start", "agent-request-fail-after-start")

	_, err := HandleCommandV0(run, command)
	assertAgentLifecycleCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterAgentFailedCommandV0RejectsStoppedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-fail-after-stop")
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-fail", "idem-stop-before-fail", "agent-request-fail-after-stop"))
	command := mustRegisterAgentFailedCommandV0(t, "cmd-fail-after-stop", "idem-fail-after-stop", "agent-request-fail-after-stop")

	_, err := HandleCommandV0(run, command)
	assertAgentLifecycleCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAgentStartedEventV0RejectsFailedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-event-start-after-failed")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentFailedCommandV0(t, "cmd-event-failed-before-start", "idem-event-failed-before-start", "agent-request-event-start-after-failed"))
	event := mustAgentStartedEventWithKeyV0(t, "evt-start-after-failed", run.LastSequence+1, "idem-start-after-failed", "agent-request-event-start-after-failed")

	_, err := ApplyEventV0(run, event)
	assertAgentLifecycleEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestAgentFailedEventV0RejectsStartedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-event-fail-after-start")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentStartedCommandV0(t, "cmd-event-start-before-fail", "idem-event-start-before-fail", "agent-request-event-fail-after-start"))
	event := mustAgentFailedEventWithKeyV0(t, "evt-fail-after-start", run.LastSequence+1, "idem-fail-after-start", "agent-request-event-fail-after-start")

	_, err := ApplyEventV0(run, event)
	assertAgentLifecycleEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func mustAgentFailedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, agentRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentFailedEventV0(meta, AgentFailedPayloadV0{
		AgentRequestID: agentRef,
		LaunchRef:      "launch-ref-" + agentRef,
		ReasonCode:     "launch_rejected",
		Retryable:      true,
		EvidenceRefs:   []string{"evidence-ref-agent-failed-001"},
	})
	return mustReducerEventV0(t, event, err)
}

func assertAgentLifecycleCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}

func assertAgentLifecycleEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
