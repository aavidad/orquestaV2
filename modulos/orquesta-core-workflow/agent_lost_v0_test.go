package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestRegisterAgentLostCommandV0ProjectsLostAgent(t *testing.T) {
	run := mustRunWithStartedAgentForLostTestV0(t, "agent-request-lost")
	command := mustRegisterAgentLostCommandV0(t, "cmd-agent-lost", "idem-agent-lost", "agent-request-lost")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterAgentLost: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventAgentLostV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	lost := mustApplyReducerEventV0(t, run, result.Events[0])
	if !reflect.DeepEqual(lost.LostAgents, []string{"agent-request-lost"}) {
		t.Fatalf("lost_agents=%v", lost.LostAgents)
	}

	result, err = HandleCommandV0(lost, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterAgentLostCommandV0RejectsUnstartedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lost-unstarted")
	command := mustRegisterAgentLostCommandV0(t, "cmd-agent-lost-unstarted", "idem-agent-lost-unstarted", "agent-request-lost-unstarted")

	_, err := HandleCommandV0(run, command)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.agent_request_id")
}

func TestRegisterAgentLostCommandV0RejectsConfirmedStoppedAgent(t *testing.T) {
	run := mustRunWithStartedAgentForLostTestV0(t, "agent-request-lost-confirmed")
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-lost", "idem-stop-before-lost", "agent-request-lost-confirmed"))
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirm-before-lost", "idem-stop-confirm-before-lost", "agent-request-lost-confirmed"))
	command := mustRegisterAgentLostCommandV0(t, "cmd-agent-lost-confirmed", "idem-agent-lost-confirmed", "agent-request-lost-confirmed")

	_, err := HandleCommandV0(run, command)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.agent_request_id")
}

func TestRegisterAgentLostCommandV0AllowsStopRequestedWithoutConfirmation(t *testing.T) {
	run := mustRunWithStartedAgentForLostTestV0(t, "agent-request-lost-stop-request")
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-lost-ok", "idem-stop-before-lost-ok", "agent-request-lost-stop-request"))
	command := mustRegisterAgentLostCommandV0(t, "cmd-agent-lost-stop-request", "idem-agent-lost-stop-request", "agent-request-lost-stop-request")

	got := mustApplySingleCommandEventV0(t, run, command)

	if !reflect.DeepEqual(got.LostAgents, []string{"agent-request-lost-stop-request"}) {
		t.Fatalf("lost_agents=%v", got.LostAgents)
	}
}

func TestValidateOrchestrationRunV0RejectsLostAgentWithoutStart(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-lost-invalid")
	run.LostAgents = []string{"agent-request-lost-invalid"}

	if issues := ValidateOrchestrationRunV0(run); len(issues) != 1 ||
		issues[0].Field != "lost_agents" {
		t.Fatalf("issues=%+v, want lost_agents invalido", issues)
	}
}

func mustRunWithStartedAgentForLostTestV0(t *testing.T, agentID string) OrchestrationRunV0 {
	t.Helper()
	run := mustRunWithRequestedAgentV0(t, agentID)
	return mustApplySingleCommandEventV0(
		t,
		run,
		mustRegisterAgentStartedCommandV0(t, "cmd-start-"+agentID, "idem-start-"+agentID, agentID),
	)
}

func mustRegisterAgentLostCommandV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	agentID string,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterAgentLostCommandV0(
		validCommandMetaV0(commandID, idempotencyKey),
		validRegisterAgentLostPayloadV0(agentID),
	)
	return mustCommandV0(t, command, err)
}

func validRegisterAgentLostPayloadV0(agentID string) RegisterAgentLostCommandPayloadV0 {
	return RegisterAgentLostCommandPayloadV0{
		AgentRequestID: agentID,
		LossRef:        "agent-loss-ref-001",
		ReasonCode:     "agent_stop_process_lost",
		ObservedAt:     "2026-05-15T09:00:00Z",
		Retryable:      false,
		EvidenceRefs:   []string{"evidence-ref-agent-lost-001"},
	}
}
