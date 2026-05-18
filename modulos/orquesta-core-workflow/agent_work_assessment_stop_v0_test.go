package orquestacoreworkflow

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestHandleAssessAgentWorkCommandV0StopAgentEmitsStopAndOutbox(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-loop")
	payload := validAssessmentPayloadV0("assessment-loop", "agent-request-loop")
	payload.Verdict = AgentAssessmentVerdictLoopDetectedV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityCriticalV0
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-loop", "idem-assess-loop", payload)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AssessAgentWork stop_agent: %v", err)
	}
	if got := eventTypesV0(result.Events); !reflect.DeepEqual(got, []string{OrchestrationEventAgentWorkAssessedV0, OrchestrationEventAgentStopRequestedV0}) {
		t.Fatalf("event types=%v", got)
	}
	if result.Events[1].Sequence != run.LastSequence+2 {
		t.Fatalf("stop sequence=%d, want %d", result.Events[1].Sequence, run.LastSequence+2)
	}
	if len(result.Outbox) != 1 || result.Outbox[0].MessageType != OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("unexpected outbox: %+v", result.Outbox)
	}
	if result.Outbox[0].CausationEventID != result.Events[1].EventID {
		t.Fatalf("outbox causation=%q, want stop event %q", result.Outbox[0].CausationEventID, result.Events[1].EventID)
	}
	if result.Outbox[0].IdempotencyKey != result.Events[1].IdempotencyKey {
		t.Fatalf("outbox idempotency_key=%q, want stop event key %q", result.Outbox[0].IdempotencyKey, result.Events[1].IdempotencyKey)
	}
	var outboxPayload StopRuntimeAgentRequestV0
	if err := json.Unmarshal(result.Outbox[0].Payload, &outboxPayload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if outboxPayload.AgentRequestID != "agent-request-loop" || outboxPayload.ReasonCode != AgentAssessmentVerdictLoopDetectedV0 {
		t.Fatalf("unexpected stop payload: %+v", outboxPayload)
	}
}

func TestHandleAssessAgentWorkCommandV0StopAgentAlreadyStoppedOnlyAssesses(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-stopped")
	run = mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-before-assess", "idem-stop-before-assess", "agent-request-stopped"))
	payload := validAssessmentPayloadV0("assessment-after-stop", "agent-request-stopped")
	payload.Verdict = AgentAssessmentVerdictGarbageV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityHighV0
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-after-stop", "idem-assess-after-stop", payload)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AssessAgentWork already stopped: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventAgentWorkAssessedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
}

func TestHandleAssessAgentWorkCommandV0TimeoutPuedePararAgente(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-timeout")
	payload := validAssessmentPayloadV0("assessment-timeout", "agent-request-timeout")
	payload.Verdict = AgentAssessmentVerdictTimeoutV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityHighV0
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-timeout", "idem-assess-timeout", payload)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AssessAgentWork timeout: %v", err)
	}
	if got := eventTypesV0(result.Events); !reflect.DeepEqual(got, []string{OrchestrationEventAgentWorkAssessedV0, OrchestrationEventAgentStopRequestedV0}) {
		t.Fatalf("event types=%v", got)
	}
	var outboxPayload StopRuntimeAgentRequestV0
	if err := json.Unmarshal(result.Outbox[0].Payload, &outboxPayload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if outboxPayload.AgentRequestID != "agent-request-timeout" || outboxPayload.ReasonCode != AgentAssessmentVerdictTimeoutV0 {
		t.Fatalf("unexpected stop payload: %+v", outboxPayload)
	}
}

func TestAssessAgentWorkCommandV0RetriesPendingStopAfterPartialPersist(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-partial")
	payload := validAssessmentPayloadV0("assessment-partial", "agent-request-partial")
	payload.Verdict = AgentAssessmentVerdictLoopDetectedV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityCriticalV0
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-partial", "idem-assess-partial", payload)
	first, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AssessAgentWork: %v", err)
	}
	partiallyPersisted := mustApplyReducerEventV0(t, run, first.Events[0])

	result, err := HandleCommandV0(partiallyPersisted, command)
	if err != nil {
		t.Fatalf("handle retry AssessAgentWork: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventAgentStopRequestedV0)
	if len(result.Outbox) != 1 || result.Outbox[0].MessageType != OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("unexpected retry outbox: %+v", result.Outbox)
	}
	if result.Outbox[0].CausationEventID != result.Events[0].EventID {
		t.Fatalf("retry causation=%q, want stop event %q", result.Outbox[0].CausationEventID, result.Events[0].EventID)
	}
	stopped := mustApplyReducerEventV0(t, partiallyPersisted, result.Events[0])
	if !agentStopAlreadyReflectedV0(stopped, "agent-request-partial") {
		t.Fatalf("stop not reflected after retry: %+v", stopped.StoppedAgents)
	}
}

func TestAssessAgentWorkCommandV0RetriesOutboxAfterStopPersisted(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-outbox-lost")
	payload := validAssessmentPayloadV0("assessment-outbox-lost", "agent-request-outbox-lost")
	payload.Verdict = AgentAssessmentVerdictLoopDetectedV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityCriticalV0
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-outbox-lost", "idem-assess-outbox-lost", payload)
	first, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AssessAgentWork: %v", err)
	}
	assessed := mustApplyReducerEventV0(t, run, first.Events[0])
	stopped := mustApplyReducerEventV0(t, assessed, first.Events[1])

	result, err := HandleCommandV0(stopped, command)
	if err != nil {
		t.Fatalf("retry AssessAgentWork after stop persisted: %v", err)
	}
	if len(result.Events) != 0 || len(result.Outbox) != 1 {
		t.Fatalf("result=%+v, want no events and one outbox", result)
	}
	if result.Outbox[0].CausationEventID != first.Events[1].EventID {
		t.Fatalf("outbox causation=%q, want persisted stop %q", result.Outbox[0].CausationEventID, first.Events[1].EventID)
	}
	if result.Outbox[0].IdempotencyKey != first.Events[1].IdempotencyKey {
		t.Fatalf("outbox idempotency_key=%q, want %q", result.Outbox[0].IdempotencyKey, first.Events[1].IdempotencyKey)
	}
}

func TestAssessAgentWorkCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-phase")
	run = mustApplySingleCommandEventV0(t, run, mustOpenPhaseCommandV0(t, "cmd-open-review-for-assess", "idem-open-review-for-assess", OrchestrationPhaseRevisionV0))
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-phase", "idem-assess-phase", validAssessmentPayloadV0("assessment-phase", "agent-request-phase"))

	_, err := HandleCommandV0(run, command)
	assertAssessmentCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.phase_id")
}
