package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestAssessAgentWorkAskDirectorBoundaryV0(t *testing.T) {
	run, replayEvents := mustRunWithRequestedAgentAndEventsV0(t, "agent-request-boundary")
	assess := mustAssessAgentWorkCommandV0(t, "cmd-assess-boundary", "idem-assess-boundary", assessmentNeedsDirectorPayloadV0())

	assessResult, err := HandleCommandV0(run, assess)
	if err != nil {
		t.Fatalf("handle AssessAgentWork ask_director: %v", err)
	}
	assertSingleEventTypeV0(t, assessResult, OrchestrationEventAgentWorkAssessedV0)
	if len(assessResult.Outbox) != 0 {
		t.Fatalf("AssessAgentWork outbox=%d, want empty", len(assessResult.Outbox))
	}

	assessed := mustApplyReducerEventV0(t, run, assessResult.Events[0])
	if len(assessed.AgentAssessments) != 1 ||
		AgentAssessmentProjectionIDV0(assessed.AgentAssessments[0]) != "assessment-boundary" {
		t.Fatalf("agent_assessments=%v", assessed.AgentAssessments)
	}
	if len(assessed.DirectorQuestions) != 0 {
		t.Fatalf("AssessAgentWork projected director_questions=%v, want empty", assessed.DirectorQuestions)
	}

	ask := mustAskDirectorCommandWithPayloadV0(t, "cmd-ask-boundary", "idem-ask-boundary", boundaryDirectorQuestionPayloadV0())
	askResult, err := HandleCommandV0(assessed, ask)
	if err != nil {
		t.Fatalf("handle separate AskDirector: %v", err)
	}
	if got := eventTypesV0(askResult.Events); !reflect.DeepEqual(got, []string{OrchestrationEventDirectorQuestionRaisedV0, OrchestrationEventRunBlockedV0}) {
		t.Fatalf("AskDirector event types=%v", got)
	}
	assertSingleDirectorOutboxV0(t, askResult, askResult.Events[0].EventID)
	for _, event := range askResult.Events {
		if event.CausationID != ask.CommandID {
			t.Fatalf("AskDirector event causation_id=%q, want %q", event.CausationID, ask.CommandID)
		}
	}

	questioned := mustApplyReducerEventV0(t, assessed, askResult.Events[0])
	blocked := mustApplyReducerEventV0(t, questioned, askResult.Events[1])
	assertAssessmentAndQuestionRefsV0(t, blocked)
	if blocked.Status != OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%q, want %q", blocked.Status, OrchestrationRunStatusBlockedV0)
	}
	if !blockerAlreadyReflectedV0(blocked, "director-question-question-boundary") {
		t.Fatalf("blocker not projected: %v", blocked.Blockers)
	}

	replayEvents = append(replayEvents, assessResult.Events[0])
	replayEvents = append(replayEvents, askResult.Events...)
	replayed, err := ReplayDurableEventsV0(replayEvents)
	if err != nil {
		t.Fatalf("replay boundary events: %v", err)
	}
	assertAssessmentAndQuestionRefsV0(t, replayed)
}

func mustRunWithRequestedAgentAndEventsV0(t *testing.T, agentRequestID string) (OrchestrationRunV0, []OrchestrationEventV0) {
	t.Helper()
	start := mustStartRunCommandV0(t, "cmd-start-boundary", "idem-start-boundary")
	startResult, err := HandleCommandV0(OrchestrationRunV0{}, start)
	if err != nil {
		t.Fatalf("handle StartRun boundary setup: %v", err)
	}
	assertSingleEventTypeV0(t, startResult, OrchestrationEventRunStartedV0)
	run := mustApplyReducerEventV0(t, OrchestrationRunV0{}, startResult.Events[0])

	open := mustOpenPhaseCommandV0(t, "cmd-open-boundary", "idem-open-boundary", OrchestrationPhaseProgramacionV0)
	openResult, err := HandleCommandV0(run, open)
	if err != nil {
		t.Fatalf("handle OpenPhase boundary setup: %v", err)
	}
	assertSingleEventTypeV0(t, openResult, OrchestrationEventPhaseOpenedV0)
	run = mustApplyReducerEventV0(t, run, openResult.Events[0])

	capacity := mustRequestCapacityCommandV0(t, "cmd-capacity-boundary", "idem-capacity-boundary", defaultAgentCapacityRequestIDV0)
	capacityResult, err := HandleCommandV0(run, capacity)
	if err != nil {
		t.Fatalf("handle RequestCapacity boundary setup: %v", err)
	}
	assertSingleEventTypeV0(t, capacityResult, OrchestrationEventCapacityRequestedV0)
	run = mustApplyReducerEventV0(t, run, capacityResult.Events[0])

	decision := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-boundary", "idem-capacity-decision-boundary", defaultAgentCapacityRequestIDV0)
	decisionResult, err := HandleCommandV0(run, decision)
	if err != nil {
		t.Fatalf("handle RegisterCapacityDecision boundary setup: %v", err)
	}
	assertSingleEventTypeV0(t, decisionResult, OrchestrationEventCapacityDecidedV0)
	run = mustApplyReducerEventV0(t, run, decisionResult.Events[0])

	agent := mustRequestAgentCommandV0(t, "cmd-agent-boundary", "idem-agent-boundary", agentRequestID)
	agentResult, err := HandleCommandV0(run, agent)
	if err != nil {
		t.Fatalf("handle RequestAgent boundary setup: %v", err)
	}
	assertSingleEventTypeV0(t, agentResult, OrchestrationEventAgentRequestedV0)
	run = mustApplyReducerEventV0(t, run, agentResult.Events[0])

	return run, []OrchestrationEventV0{startResult.Events[0], openResult.Events[0], capacityResult.Events[0], decisionResult.Events[0], agentResult.Events[0]}
}

func assessmentNeedsDirectorPayloadV0() AssessAgentWorkCommandPayloadV0 {
	payload := validAssessmentPayloadV0("assessment-boundary", "agent-request-boundary")
	payload.Verdict = AgentAssessmentVerdictNeedsRevisionV0
	payload.Action = AgentAssessmentActionAskDirectorV0
	payload.Severity = AgentAssessmentSeverityMediumV0
	payload.Summary = "La evaluacion necesita criterio de alcance antes de revisar."
	payload.EvidenceRefs = []string{"docs/contratos_agentes.md#AssessAgentWork", "docs/contratos.md#AskDirector"}
	return payload
}

func boundaryDirectorQuestionPayloadV0() AskDirectorCommandPayloadV0 {
	payload := validAskDirectorPayloadV0(true)
	payload.QuestionID = "question-boundary"
	payload.Summary = "Decidir si esta evaluacion debe bloquear la revision."
	payload.EvidenceRefs = []string{"docs/contratos.md#AskDirector"}
	return payload
}

func mustAskDirectorCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload AskDirectorCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAskDirectorCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func assertAssessmentAndQuestionRefsV0(t *testing.T, run OrchestrationRunV0) {
	t.Helper()
	if len(run.AgentAssessments) != 1 ||
		AgentAssessmentProjectionIDV0(run.AgentAssessments[0]) != "assessment-boundary" {
		t.Fatalf("agent_assessments=%v", run.AgentAssessments)
	}
	if !reflect.DeepEqual(run.DirectorQuestions, []string{"question-boundary"}) {
		t.Fatalf("director_questions=%v, want [question-boundary]", run.DirectorQuestions)
	}
}
