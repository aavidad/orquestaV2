package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegisterDeliveryCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustFunctionContractReadyRunV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-phase", "idem-delivery-phase", "delivery-phase")

	_, err := HandleCommandV0(run, command)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestRegisterDeliveryCommandV0AcceptsDocumentationPhaseTask(t *testing.T) {
	run := mustDocumentationDeliveryReadyRunV0(t)
	command := mustRegisterDeliveryCommandForPhaseV0(
		t,
		"cmd-delivery-doc-phase",
		"idem-delivery-doc-phase",
		"delivery-doc-phase",
		OrchestrationPhaseDocumentacionV0,
		"task-doc-001",
		"agent-doc-001",
	)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("HandleCommandV0: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDeliveryRegisteredV0)
}

func mustDocumentationDeliveryReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	const taskRef = "task-doc-001"
	const agentRef = "agent-doc-001"
	const capacityRef = "capacity-doc-001"

	run := mustFunctionContractReadyRunV0(t)
	task := validMicrotaskWorkflowTaskV0(taskRef)
	task.PhaseID = OrchestrationPhaseDocumentacionV0
	task.WriteSet = []string{"docs/manual.md"}
	create, err := NewCreateMicrotaskCommandV0(
		validCommandMetaV0("cmd-create-doc-task", "idem-create-doc-task"),
		CreateMicrotaskCommandPayloadV0{Task: task},
	)
	run = mustApplySingleCommandEventV0(t, run, mustCommandV0(t, create, err))
	open := mustOpenPhaseCommandV0(t, "cmd-open-doc-delivery", "idem-open-doc-delivery", OrchestrationPhaseDocumentacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)

	capacityPayload := validRequestCapacityPayloadV0(capacityRef)
	capacityPayload.PhaseID = string(OrchestrationPhaseDocumentacionV0)
	capacityPayload.TaskRef = taskRef
	capacity := mustRequestCapacityCommandWithPayloadV0(
		t,
		"cmd-capacity-doc",
		"idem-capacity-doc",
		capacityPayload,
	)
	run = mustApplySingleCommandEventV0(t, run, capacity)
	decision := mustRegisterCapacityDecisionCommandV0(
		t,
		"cmd-capacity-decision-doc",
		"idem-capacity-decision-doc",
		capacityRef,
	)
	run = mustApplySingleCommandEventV0(t, run, decision)

	agentPayload := validRequestAgentPayloadV0(agentRef)
	agentPayload.PhaseID = string(OrchestrationPhaseDocumentacionV0)
	agentPayload.TaskRef = taskRef
	agentPayload.CapacityRequestRef = capacityRef
	agent := mustRequestAgentCommandWithPayloadV0(
		t,
		"cmd-agent-doc",
		"idem-agent-doc",
		agentPayload,
	)
	run = mustApplySingleCommandEventV0(t, run, agent)
	started := mustRegisterAgentStartedCommandV0(t, "cmd-agent-doc-started", "idem-agent-doc-started", agentRef)
	return mustApplySingleCommandEventV0(t, run, started)
}

func mustRequestCapacityCommandWithPayloadV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	payload RequestCapacityCommandPayloadV0,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestCapacityCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustRequestAgentCommandWithPayloadV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	payload RequestAgentCommandPayloadV0,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestAgentCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
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

func TestRegisterDeliveryCommandV0AcceptsStoppedAgentWithLateAck(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-delivery", "idem-stop-before-delivery", "agent-request-001"))
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-stopped-agent", "idem-delivery-stopped-agent", "delivery-stopped-agent")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("late stopped delivery rejected: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDeliveryRegisteredV0)
}

func TestRegisterDeliveryCommandV0AcceptsLostAgentWithLateAckAndClearsLost(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	lost := mustRegisterAgentLostCommandV0(t, "cmd-agent-lost-before-delivery", "idem-agent-lost-before-delivery", "agent-request-001")
	run = mustApplySingleCommandEventV0(t, run, lost)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-lost-agent", "idem-delivery-lost-agent", "delivery-lost-agent")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("late lost delivery rejected: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDeliveryRegisteredV0)
	got, err := ApplyEventV0(run, result.Events[0])
	if err != nil {
		t.Fatalf("ApplyEventV0: %v", err)
	}
	if compactRefInListV0(got.LostAgents, "agent-request-001") {
		t.Fatalf("late ACK debe retirar lost_agents: %+v", got.LostAgents)
	}
	if !compactRefInListV0(got.DeliveredAgents, "agent-request-001") {
		t.Fatalf("late ACK no marco delivered_agents: %+v", got.DeliveredAgents)
	}
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

func TestDeliveryRegisteredEventV0AcceptsStoppedAgentWithLateAck(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-delivery-event", "idem-stop-before-delivery-event", "agent-request-001"))
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-stopped-agent", run.LastSequence+1, "delivery-stopped-agent")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("late stopped delivery event rejected: %v", err)
	}
	if !reflect.DeepEqual(got.DeliveredAgents, []string{"agent-request-001"}) {
		t.Fatalf("delivered_agents=%v", got.DeliveredAgents)
	}
}

func TestRegisterDeliveryCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterDeliveryPayloadV0("delivery-forbidden")
	payload.Summary = "usar api_key=valor"

	_, err := NewRegisterDeliveryCommandV0(validCommandMetaV0("cmd-delivery-forbidden", "idem-delivery-forbidden"), payload)
	assertRegisterDeliveryCommandErrorV0(t, err, ErrDetalleProhibidoV0)
}

func TestDeliveryRegisteredEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := deliveryRegisteredPayloadFromCommandV0(validRegisterDeliveryPayloadV0("delivery-event-forbidden"))
	payload.Summary = "usar authorization: bearer valor"

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
