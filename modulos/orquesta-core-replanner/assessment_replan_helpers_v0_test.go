package orquestacorereplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validAgentWorkAssessmentReplanInputV0(verdict string, assessmentAction string, requestedAction ReplanRecommendedActionV0) AgentWorkAssessmentReplanInputV0 {
	return AgentWorkAssessmentReplanInputV0{
		ReplanRef:       "replan-agent-001",
		SignalRef:       "agent-signal-001",
		RunRef:          "run-001",
		TaskRef:         "task-001",
		RequestedAction: requestedAction,
		ReplacementRole: "implementer",
		ReasonRef:       "agent_rework_requested",
		Assessment: orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: "agent-request-001",
			TaskRef:        "task-001",
			Verdict:        verdict,
			Action:         assessmentAction,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
			Summary:        "Evaluacion compacta requiere replanificacion.",
			EvidenceRefs:   []string{"assessment-evidence-001"},
		},
	}
}

func validAgentReworkSignalV0() AgentReworkSignalV0 {
	input := validAgentWorkAssessmentReplanInputV0(
		orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
		ReplanActionReplaceAgentV0,
	)
	return AgentReworkSignalV0{
		SignalRef:        input.SignalRef,
		RunRef:           input.RunRef,
		TaskRef:          input.Assessment.TaskRef,
		AgentRequestID:   input.Assessment.AgentRequestID,
		SourceRef:        input.Assessment.AssessmentRef,
		AssessmentStatus: input.Assessment.Verdict,
		AssessmentAction: input.Assessment.Action,
		RequestedAction:  input.RequestedAction,
		ReplacementRole:  input.ReplacementRole,
		ReasonRef:        input.ReasonRef,
		Summary:          input.Assessment.Summary,
		EvidenceRefs:     input.Assessment.EvidenceRefs,
	}
}

func mustAgentWorkAssessmentReplanProposalV0(t *testing.T, input AgentWorkAssessmentReplanInputV0) ReplanProposalV0 {
	t.Helper()
	proposal, err := AgentWorkAssessmentToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("AgentWorkAssessmentToReplanProposalV0: %v", err)
	}
	if proposal == nil {
		t.Fatalf("expected agent replan proposal")
	}
	return *proposal
}

func assertAgentAssessmentReplanProposalTraceV0(t *testing.T, proposal ReplanProposalV0, input AgentWorkAssessmentReplanInputV0) {
	t.Helper()
	if proposal.ReplanRef != input.ReplanRef ||
		proposal.RunRef != input.RunRef ||
		proposal.TaskRef != input.Assessment.TaskRef ||
		proposal.SourceRef != input.Assessment.AssessmentRef ||
		proposal.ReasonCode != input.ReasonRef {
		t.Fatalf("proposal trace mismatch: %#v input=%#v", proposal, input)
	}
	if proposal.CapacityRequestRef != "" {
		t.Fatalf("capacity_request_ref must stay empty: %#v", proposal)
	}
	if proposal.RecommendedAction == ReplanActionReplaceAgentV0 && proposal.ReplacementRole == "" {
		t.Fatalf("replacement_role required for replace_agent: %#v", proposal)
	}
	if proposal.RecommendedAction != ReplanActionReplaceAgentV0 && proposal.ReplacementRole != "" {
		t.Fatalf("replacement_role must stay empty for %q: %#v", proposal.RecommendedAction, proposal)
	}
}
