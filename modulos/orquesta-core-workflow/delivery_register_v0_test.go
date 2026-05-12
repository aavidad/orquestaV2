package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRegisterDeliveryCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-001", "idem-delivery-001", "delivery-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterDelivery: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventDeliveryRegisteredV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyDeliveryRegisteredV0ProjectsRefOnce(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	event := mustDeliveryRegisteredEventV0(t, "evt-delivery-reducer-001", run.LastSequence+1, "delivery-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply DeliveryRegistered: %v", err)
	}
	if !reflect.DeepEqual(got.Deliveries, []string{"delivery-001"}) ||
		!reflect.DeepEqual(got.DeliveredTasks, []string{"task-ncw-009"}) ||
		!reflect.DeepEqual(got.DeliveredAgents, []string{"agent-request-001"}) {
		t.Fatalf(
			"deliveries=%v delivered_tasks=%v delivered_agents=%v",
			got.Deliveries,
			got.DeliveredTasks,
			got.DeliveredAgents,
		)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Deliveries, got.Deliveries) ||
		!reflect.DeepEqual(again.DeliveredTasks, got.DeliveredTasks) ||
		!reflect.DeepEqual(again.DeliveredAgents, got.DeliveredAgents) {
		t.Fatalf("delivery projection duplicated: %+v", again)
	}
}

func TestReplayDurableEventsV0AcceptsDeliveryRegistered(t *testing.T) {
	delivery := mustDeliveryRegisteredEventWithKeyV0(t, "evt-durable-delivery", 13, "idem-delivery", "delivery-001")
	events := append(mustDeliveryReplayEventsV0(t), delivery)

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable DeliveryRegistered: %v", err)
	}
	if got.LastSequence != 13 {
		t.Fatalf("last_sequence=%d, want 13", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Deliveries, []string{"delivery-001"}) {
		t.Fatalf("deliveries=%v", got.Deliveries)
	}
	if !reflect.DeepEqual(got.DeliveredTasks, []string{"task-ncw-009"}) {
		t.Fatalf("delivered_tasks=%v", got.DeliveredTasks)
	}
	if !reflect.DeepEqual(got.DeliveredAgents, []string{"agent-request-001"}) {
		t.Fatalf("delivered_agents=%v", got.DeliveredAgents)
	}
}

func TestRegisterDeliveryCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-repeat", "idem-delivery-repeat", "delivery-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterDeliveryCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	payload := validRegisterDeliveryPayloadV0("delivery-effect")
	command := mustRegisterDeliveryCommandWithPayloadV0(t, "cmd-delivery-effect", "idem-delivery-effect", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Registrar entrega compacta con otro resumen."
	conflicting := mustRegisterDeliveryCommandWithPayloadV0(t, "cmd-delivery-effect", "idem-delivery-effect", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterDeliveryCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	command := mustRegisterDeliveryCommandV0(t, "cmd-delivery-key", "idem-delivery-key", "delivery-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterDeliveryCommandV0(t, "cmd-delivery-key-2", "idem-delivery-key-2", "delivery-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyDeliveryRegisteredV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustDeliveryReadyRunV0(t)
	first := mustDeliveryRegisteredEventWithKeyV0(t, "evt-delivery-effect", run.LastSequence+1, "idem-delivery-effect", "delivery-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustDeliveryRegisteredEventWithKeyV0(t, "evt-delivery-effect-conflict", applied.LastSequence+1, "idem-delivery-effect-conflict", "delivery-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterDeliveryCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-delivery", "idem-start-after-delivery")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding RegisterDelivery: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustDeliveryReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryRunWithRequestedAgentV0(t)
	started := mustRegisterAgentStartedCommandV0(t, "cmd-agent-started-for-delivery", "idem-agent-started-for-delivery", "agent-request-001")
	return mustApplySingleCommandEventV0(t, run, started)
}

func mustDeliveryRunWithRequestedAgentV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustFunctionContractReadyRunV0(t)
	task := mustCreateMicrotaskCommandV0(t, "cmd-create-task-for-delivery", "idem-create-task-for-delivery", "task-ncw-009")
	run = mustApplySingleCommandEventV0(t, run, task)
	open := mustOpenPhaseCommandV0(t, "cmd-open-programacion-delivery", "idem-open-programacion-delivery", OrchestrationPhaseProgramacionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	run = mustApplyCapacityDecisionToRunV0(t, run, defaultAgentCapacityRequestIDV0)
	agent := mustRequestAgentCommandV0(t, "cmd-agent-for-delivery", "idem-agent-for-delivery", "agent-request-001")
	return mustApplySingleCommandEventV0(t, run, agent)
}

func mustRegisterDeliveryCommandV0(t *testing.T, commandID string, idempotencyKey string, deliveryRef string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterDeliveryCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterDeliveryPayloadV0(deliveryRef))
}

func mustRegisterDeliveryCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterDeliveryCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterDeliveryCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustDeliveryRegisteredEventV0(t *testing.T, eventID string, sequence int64, deliveryRef string) OrchestrationEventV0 {
	t.Helper()
	return mustDeliveryRegisteredEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, deliveryRef)
}

func mustDeliveryRegisteredEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, deliveryRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewDeliveryRegisteredEventV0(meta, deliveryRegisteredPayloadFromCommandV0(validRegisterDeliveryPayloadV0(deliveryRef)))
	return mustReducerEventV0(t, event, err)
}

func validRegisterDeliveryPayloadV0(deliveryRef string) RegisterDeliveryCommandPayloadV0 {
	return RegisterDeliveryCommandPayloadV0{
		DeliveryRef:  deliveryRef,
		PhaseID:      string(OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ncw-009",
		AgentRef:     "agent-request-001",
		Summary:      "Registrar entrega compacta de una microtarea programada.",
		EvidenceRefs: []string{"docs/contratos_microtareas.md#MicrotaskCreated"},
	}
}

func assertRegisterDeliveryCommandErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
