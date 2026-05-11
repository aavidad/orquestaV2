package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRegisterDeliveryCommandV0RejectsNonProgrammingPhase(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-phase", "idem-delivery-phase", "delivery-phase")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsMissingTask(t *testing.T) {
	run := mustProgrammingActiveRunWithoutTaskOrAgentV0(t)
	run = mustApplyCapacityDecisionToRunV0(t, run, defaultAgentCapacityRequestIDV0)
	agent := mustRequestAgentCommandV0(t, "cmd-agent-delivery-missing-task", "idem-agent-delivery-missing-task", "agent-request-001")
	run = mustApplySingleCommandEventV0(t, run, agent)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-missing-task", "idem-delivery-missing-task", "delivery-missing-task")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsMissingAgent(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	task := mustCreateMicrotaskCommandV0(t, "cmd-create-task-no-agent", "idem-create-task-no-agent", "task-ncw-009")
	run = mustApplySingleCommandEventV0(t, run, task)
	open := mustOpenPhaseCommandV0(t, "cmd-open-programacion-no-agent", "idem-open-programacion-no-agent", OrchestrationPhaseProgramacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-missing-agent", "idem-delivery-missing-agent", "delivery-missing-agent")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsAgentNotStarted(t *testing.T) {
	run := mustDeliveryRunWithRequestedAgentV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-agent-not-started", "idem-delivery-agent-not-started", "delivery-agent-not-started")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsFailedAgent(t *testing.T) {
	run := mustDeliveryRunWithRequestedAgentV0(t)
	failed := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-before-delivery", "idem-agent-failed-before-delivery", "agent-request-001")
	run = mustApplySingleCommandEventV0(t, run, failed)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-failed-agent", "idem-delivery-failed-agent", "delivery-failed-agent")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsStoppedAgent(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-delivery", "idem-stop-before-delivery", "agent-request-001"))
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-stopped-agent", "idem-delivery-stopped-agent", "delivery-stopped-agent")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestDeliveryRegisteredEventV0RejectsMissingTask(t *testing.T) {
	run := mustProgrammingActiveRunWithoutTaskOrAgentV0(t)
	run = mustApplyCapacityDecisionToRunV0(t, run, defaultAgentCapacityRequestIDV0)
	agent := mustRequestAgentCommandV0(t, "cmd-agent-event-missing-task", "idem-agent-event-missing-task", "agent-request-001")
	run = mustApplySingleCommandEventV0(t, run, agent)
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-missing-task", run.LastSequence+1, "delivery-missing-task")

	_, err := ApplyEventV0(run, event)
	assertDeliveryRegisteredEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestDeliveryRegisteredEventV0RejectsAgentNotStarted(t *testing.T) {
	run := mustDeliveryRunWithRequestedAgentV0(t)
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-agent-not-started", run.LastSequence+1, "delivery-agent-not-started")

	_, err := ApplyEventV0(run, event)
	assertDeliveryRegisteredEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestDeliveryRegisteredEventV0RejectsFailedAgent(t *testing.T) {
	run := mustDeliveryRunWithRequestedAgentV0(t)
	failed := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-before-delivery-event", "idem-agent-failed-before-delivery-event", "agent-request-001")
	run = mustApplySingleCommandEventV0(t, run, failed)
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-failed-agent", run.LastSequence+1, "delivery-failed-agent")

	_, err := ApplyEventV0(run, event)
	assertDeliveryRegisteredEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestDeliveryRegisteredEventV0RejectsStoppedAgent(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-delivery-event", "idem-stop-before-delivery-event", "agent-request-001"))
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-stopped-agent", run.LastSequence+1, "delivery-stopped-agent")

	_, err := ApplyEventV0(run, event)
	assertDeliveryRegisteredEventErrorV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestRegisterDeliveryCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterDeliveryPayloadV0("delivery-forbidden")
	payload.Summary = "usar provider externo"

	_, err := NewRegisterDeliveryCommandV0(validCommandMetaV0("cmd-delivery-forbidden", "idem-delivery-forbidden"), payload)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestDeliveryRegisteredEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := deliveryRegisteredPayloadFromCommandV0(validRegisterDeliveryPayloadV0("delivery-event-forbidden"))
	payload.Summary = "usar Claude"

	_, err := NewDeliveryRegisteredEventV0(reducerEventMetaV0("evt-delivery-forbidden", 10), payload)
	assertDeliveryRegisteredEventErrorV0(t, err, ErrDetalleProhibidoV0)
}

func mustProgrammingActiveRunWithoutTaskOrAgentV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustFunctionContractReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-programacion-empty", "idem-open-programacion-empty", OrchestrationPhaseProgramacionV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func assertDeliveryRegisteredEventErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
