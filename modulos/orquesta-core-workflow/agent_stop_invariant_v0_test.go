package orquestacoreworkflow

import "testing"

func TestStopAgentCommandV0RejectsFailedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-stop-after-failed")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentFailedCommandV0(t, "cmd-failed-before-stop", "idem-failed-before-stop", "agent-request-stop-after-failed"))
	command := mustStopAgentCommandV0(t, "cmd-stop-after-failed", "idem-stop-after-failed", "agent-request-stop-after-failed")

	_, err := HandleCommandV0(run, command)
	assertAgentLifecycleCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestAgentStopRequestedEventV0RejectsFailedAgent(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-stop-event-after-failed")
	run = mustApplySingleCommandEventV0(t, run, mustRegisterAgentFailedCommandV0(t, "cmd-failed-before-stop-event", "idem-failed-before-stop-event", "agent-request-stop-event-after-failed"))
	event := mustAgentStopRequestedEventWithKeyV0(t, "evt-stop-after-failed", run.LastSequence+1, "idem-stop-after-failed", "agent-request-stop-event-after-failed")

	_, err := ApplyEventV0(run, event)
	assertAgentLifecycleEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}
