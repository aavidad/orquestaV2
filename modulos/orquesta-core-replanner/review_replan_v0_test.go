package orquestacorereplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestReviewResultToReplanProposalV0AcceptedReturnsNil(t *testing.T) {
	input := validReviewResultReplanInputV0(orquestacoreworkflow.ReviewResultStatusAcceptedV0, ReplanActionRetryTaskV0)

	proposal, err := ReviewResultToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("ReviewResultToReplanProposalV0: %v", err)
	}
	if proposal != nil {
		t.Fatalf("accepted review must not propose replan: %#v", proposal)
	}
}

func TestReviewResultToReplanProposalV0ChangesRequestedSupportsSplitRetryAndAskDirector(t *testing.T) {
	for _, action := range []ReplanRecommendedActionV0{
		ReplanActionSplitTaskV0,
		ReplanActionRetryTaskV0,
		ReplanActionAskDirectorV0,
	} {
		t.Run(string(action), func(t *testing.T) {
			input := validReviewResultReplanInputV0(orquestacoreworkflow.ReviewResultStatusChangesRequestedV0, action)

			proposal := mustReviewResultReplanProposalV0(t, input)
			if proposal.RecommendedAction != action {
				t.Fatalf("recommended_action=%q, want %q", proposal.RecommendedAction, action)
			}
			assertReviewReplanProposalTraceV0(t, proposal, input)
		})
	}
}

func TestReviewResultToReplanProposalV0RejectedSupportsSplitRetryAndAskDirector(t *testing.T) {
	for _, action := range []ReplanRecommendedActionV0{
		ReplanActionSplitTaskV0,
		ReplanActionRetryTaskV0,
		ReplanActionAskDirectorV0,
	} {
		t.Run(string(action), func(t *testing.T) {
			input := validReviewResultReplanInputV0(orquestacoreworkflow.ReviewResultStatusRejectedV0, action)

			proposal := mustReviewResultReplanProposalV0(t, input)
			if proposal.RecommendedAction != action {
				t.Fatalf("recommended_action=%q, want %q", proposal.RecommendedAction, action)
			}
			assertReviewReplanProposalTraceV0(t, proposal, input)
		})
	}
}

func TestValidateReviewReworkSignalV0RejectsAcceptedAndInvalidActions(t *testing.T) {
	cases := map[string]struct {
		status orquestacoreworkflow.ReviewResultStatusV0
		action ReplanRecommendedActionV0
		code   string
		field  string
	}{
		"accepted": {
			status: orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			action: ReplanActionRetryTaskV0,
			code:   ErrReviewReworkStatusNoSoportadoV0,
			field:  "review_status",
		},
		"rejected_replace_agent": {
			status: orquestacoreworkflow.ReviewResultStatusRejectedV0,
			action: ReplanActionReplaceAgentV0,
			code:   ErrReviewReworkActionNoSoportadaV0,
			field:  "requested_action",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			signal := validReviewReworkSignalV0()
			signal.ReviewStatus = tc.status
			signal.RequestedAction = tc.action

			err := ValidateReviewReworkSignalV0(NormalizeReviewReworkSignalV0(signal))
			assertReviewReworkSignalErrorV0(t, err, tc.code, tc.field)
		})
	}
}

func TestValidateReviewReworkSignalV0RejectsForbiddenDetails(t *testing.T) {
	signal := validReviewReworkSignalV0()
	signal.ReasonRef = "oauth-policy"

	err := ValidateReviewReworkSignalV0(NormalizeReviewReworkSignalV0(signal))
	assertReviewReworkSignalErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func TestReviewResultToReplanProposalV0JSONHasNoForbiddenDetails(t *testing.T) {
	input := validReviewResultReplanInputV0(orquestacoreworkflow.ReviewResultStatusRejectedV0, ReplanActionSplitTaskV0)
	proposal := mustReviewResultReplanProposalV0(t, input)

	assertReplanJSONHasNoForbiddenDetailsV0(t, "review replan proposal", proposal)
}
