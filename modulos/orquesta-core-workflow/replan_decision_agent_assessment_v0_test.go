package orquestacoreworkflow

import "testing"

func TestRecordReplanDecisionCommandV0AcceptsAgentAssessmentSourceInProgramming(t *testing.T) {
	run := mustReplanDecisionAgentAssessmentReadyRunV0(t)
	command := mustRecordReplanDecisionFromAgentAssessmentCommandV0(
		t,
		"cmd-replan-agent-assessment",
		"idem-replan-agent-assessment",
		"replan-decision-agent-assessment",
	)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RecordReplanDecision from AgentWorkAssessed: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventReplanDecisionRecordedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
}

func TestValidateOrchestrationRunV0AcceptsReplanDecisionFromAgentAssessment(t *testing.T) {
	run := mustReplanDecisionAgentAssessmentReadyRunV0(t)
	payload := validReplanDecisionFromAgentAssessmentPayloadV0(
		"replan-decision-agent-assessment-state",
	)
	run.ReplanDecisions = []string{replanDecisionProjectionRefV0(payload)}

	if issues := ValidateOrchestrationRunV0(run); len(issues) != 0 {
		t.Fatalf("run con replan desde AgentWorkAssessed invalido: %+v", issues)
	}
}

func TestRecordReplanDecisionCommandV0RejectsAgentAssessmentSourceOutsideProgramming(t *testing.T) {
	run := mustReplanDecisionAgentAssessmentReadyRunV0(t)
	open := mustOpenPhaseCommandV0(
		t,
		"cmd-open-revision-after-agent-assessment",
		"idem-open-revision-after-agent-assessment",
		OrchestrationPhaseRevisionV0,
	)
	run = mustApplySingleCommandEventV0(t, run, open)
	command := mustRecordReplanDecisionFromAgentAssessmentCommandV0(
		t,
		"cmd-replan-agent-assessment-phase",
		"idem-replan-agent-assessment-phase",
		"replan-decision-agent-assessment-phase",
	)

	_, err := HandleCommandV0(run, command)
	assertRecordReplanDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0)
}

func mustReplanDecisionAgentAssessmentReadyRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryRunWithRequestedAgentV0(t)
	payload := validAssessmentPayloadV0("assessment-loop-for-replan", "agent-request-001")
	payload.Verdict = AgentAssessmentVerdictLoopDetectedV0
	payload.Action = AgentAssessmentActionStopAgentV0
	payload.Severity = AgentAssessmentSeverityCriticalV0
	payload.Summary = "Bucle detectado; se solicita reemplazo de agente."
	command := mustAssessAgentWorkCommandV0(
		t,
		"cmd-assess-loop-for-replan",
		"idem-assess-loop-for-replan",
		payload,
	)
	return mustApplyCommandEventsV0(t, run, command)
}

func mustRecordReplanDecisionFromAgentAssessmentCommandV0(
	t *testing.T,
	commandID string,
	idempotencyKey string,
	replanRef string,
) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRecordReplanDecisionCommandV0(
		validCommandMetaV0(commandID, idempotencyKey),
		validReplanDecisionFromAgentAssessmentPayloadV0(replanRef),
	)
	return mustCommandV0(t, command, err)
}

func validReplanDecisionFromAgentAssessmentPayloadV0(
	replanRef string,
) ReplanDecisionRecordedPayloadV0 {
	payload := validReplanDecisionPayloadV0(replanRef)
	payload.SourceRef = "assessment-loop-for-replan"
	payload.AcceptedAction = ReplanDecisionActionReplaceAgentV0
	payload.FollowupRefs = []string{"capacity-replacement-001", "agent-replacement-001"}
	payload.Summary = "Decision compacta para reemplazar agente tras evaluacion de bucle."
	return payload
}

func mustApplyCommandEventsV0(
	t *testing.T,
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
) OrchestrationRunV0 {
	t.Helper()
	result, err := HandleCommandV0(current, command)
	if err != nil {
		t.Fatalf("handle %s: %v", command.CommandType, err)
	}
	if len(result.Events) == 0 {
		t.Fatalf("events=0, want at least 1")
	}
	next := current
	for _, event := range result.Events {
		next = mustApplyReducerEventV0(t, next, event)
	}
	return next
}
