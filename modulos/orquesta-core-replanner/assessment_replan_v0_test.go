package orquestacorereplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAgentWorkAssessmentToReplanProposalV0AcceptableReturnsNil(t *testing.T) {
	input := validAgentWorkAssessmentReplanInputV0(
		orquestacoreworkflow.AgentAssessmentVerdictAcceptableV0,
		orquestacoreworkflow.AgentAssessmentActionContinueV0,
		ReplanActionRetryTaskV0,
	)

	proposal, err := AgentWorkAssessmentToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("AgentWorkAssessmentToReplanProposalV0: %v", err)
	}
	if proposal != nil {
		t.Fatalf("acceptable assessment must not propose replan: %#v", proposal)
	}
}

func TestAgentWorkAssessmentToReplanProposalV0NeedsRevisionCanRetryTask(t *testing.T) {
	input := validAgentWorkAssessmentReplanInputV0(
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0,
		ReplanActionRetryTaskV0,
	)

	proposal := mustAgentWorkAssessmentReplanProposalV0(t, input)
	if proposal.RecommendedAction != ReplanActionRetryTaskV0 {
		t.Fatalf("recommended_action=%q, want %q", proposal.RecommendedAction, ReplanActionRetryTaskV0)
	}
	assertAgentAssessmentReplanProposalTraceV0(t, proposal, input)
}

func TestAgentWorkAssessmentToReplanProposalV0StopAgentCanReplaceOrAbort(t *testing.T) {
	cases := map[string]struct {
		verdict string
		action  ReplanRecommendedActionV0
	}{
		"garbage_replace": {
			verdict: orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			action:  ReplanActionReplaceAgentV0,
		},
		"loop_abort": {
			verdict: orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			action:  ReplanActionAbortTaskV0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			input := validAgentWorkAssessmentReplanInputV0(
				tc.verdict,
				orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
				tc.action,
			)

			proposal := mustAgentWorkAssessmentReplanProposalV0(t, input)
			if proposal.RecommendedAction != tc.action {
				t.Fatalf("recommended_action=%q, want %q", proposal.RecommendedAction, tc.action)
			}
			assertAgentAssessmentReplanProposalTraceV0(t, proposal, input)
		})
	}
}

func TestAgentWorkAssessmentToReplanProposalV0AskDirectorActionAsksDirector(t *testing.T) {
	for _, verdict := range []string{
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
		orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
	} {
		t.Run(verdict, func(t *testing.T) {
			input := validAgentWorkAssessmentReplanInputV0(
				verdict,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				ReplanActionAskDirectorV0,
			)

			proposal := mustAgentWorkAssessmentReplanProposalV0(t, input)
			if proposal.RecommendedAction != ReplanActionAskDirectorV0 {
				t.Fatalf("recommended_action=%q, want %q", proposal.RecommendedAction, ReplanActionAskDirectorV0)
			}
			assertAgentAssessmentReplanProposalTraceV0(t, proposal, input)
		})
	}
}

func TestAgentWorkAssessmentToReplanProposalV0JSONHasNoForbiddenDetails(t *testing.T) {
	input := validAgentWorkAssessmentReplanInputV0(
		orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
		ReplanActionReplaceAgentV0,
	)
	proposal := mustAgentWorkAssessmentReplanProposalV0(t, input)

	assertReplanJSONHasNoForbiddenDetailsV0(t, "agent replan proposal", proposal)
}
