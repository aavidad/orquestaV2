package orquestacoreworkflow

import "testing"

func TestRecordReplanDecisionCommandV0AcceptsAgentFailedSourceInProgramming(t *testing.T) {
	run := mustReplanDecisionAgentFailedReadyRunV0(t)
	command := mustRecordReplanDecisionFromAgentFailedCommandV0(t, "cmd-replan-agent-failed", "idem-replan-agent-failed", "replan-decision-agent-failed")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordReplanDecision from AgentFailed: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventReplanDecisionRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
}

func TestReplanDecisionRecordedEventV0AcceptsAgentFailedSourceInProgramming(t *testing.T) {
	run := mustReplanDecisionAgentFailedReadyRunV0(t)
	payload := validReplanDecisionFromAgentFailedPayloadV0("replan-decision-agent-failed-event")
	event := mustReplanDecisionRecordedEventV0(t, "evt-replan-agent-failed", run.LastSequence+1, payload)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply ReplanDecisionRecorded from AgentFailed: %v", err)
	}
	if _, ok := replanDecisionProjectionForRefV0(got, payload.ReplanRef); !ok {
		t.Fatalf("replan decision %q no proyectada: %+v", payload.ReplanRef, got.ReplanDecisions)
	}
}

func TestRecordReplanDecisionCommandV0RejectsAgentFailedSourceOutsideProgramming(t *testing.T) {
	run := mustReplanDecisionAgentFailedReadyRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-revision-after-agent-failed", "idem-open-revision-after-agent-failed", OrchestrationPhaseRevisionV0)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRecordReplanDecisionFromAgentFailedCommandV0(t, "cmd-replan-agent-failed-phase", "idem-replan-agent-failed-phase", "replan-decision-agent-failed-phase")

	_, err := HandleCommandV0(run, command)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func TestValidateOrchestrationRunV0AcceptsReplanDecisionFromFailedAgent(t *testing.T) {
	run := mustReplanDecisionAgentFailedReadyRunV0(t)
	payload := validReplanDecisionFromAgentFailedPayloadV0("replan-decision-agent-failed-state")
	run.ReplanDecisions = []string{replanDecisionProjectionRefV0(payload)}

	if issues := ValidateOrchestrationRunV0(run); len(issues) != 0 {
		t.Fatalf("run con replan desde AgentFailed invalido: %+v", issues)
	}
}

func mustReplanDecisionAgentFailedReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryRunWithRequestedAgentV0(t)
	failed := mustRegisterAgentFailedCommandV0(t, "cmd-agent-failed-for-replan", "idem-agent-failed-for-replan", "agent-request-001")
	return mustApplySingleCommandEventV0(t, run, failed)
}

func mustRecordReplanDecisionFromAgentFailedCommandV0(t *testing.T, commandID string, idempotencyKey string, replanRef string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordReplanDecisionCommandV0(validCommandMetaV0(commandID, idempotencyKey), validReplanDecisionFromAgentFailedPayloadV0(replanRef))
	return mustCommandV0(t, command, err)
}

func validReplanDecisionFromAgentFailedPayloadV0(replanRef string) ReplanDecisionRecordedPayloadV0 {
	payload := validReplanDecisionPayloadV0(replanRef)
	payload.SourceRef = "agent-request-001"
	payload.AcceptedAction = ReplanDecisionActionReplaceAgentV0
	payload.FollowupRefs = []string{"capacity-replacement-001", "agent-replacement-001"}
	payload.Summary = "Decision compacta para reemplazar agente tras fallo de lanzamiento."
	return payload
}
