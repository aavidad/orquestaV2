package orquestacorereplanner

import (
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validReviewResultReplanInputV0(status orquestacoreworkflow.ReviewResultStatusV0, action ReplanRecommendedActionV0) ReviewResultReplanInputV0 {
	return ReviewResultReplanInputV0{
		ReplanRef:       "replan-review-001",
		SignalRef:       "review-signal-001",
		RunRef:          "run-001",
		TaskRef:         "task-001",
		RequestedAction: action,
		ReasonRef:       "review_rework_requested",
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-001",
			ReviewRequestID: "review-request-001",
			DeliveryRef:     "delivery-001",
			Status:          status,
			Summary:         "Revision compacta pide rehacer la entrega.",
			EvidenceRefs:    []string{"review-evidence-001"},
			QualityGateRef:  "quality-gate-001",
		},
	}
}

func validReviewReworkSignalV0() ReviewReworkSignalV0 {
	input := validReviewResultReplanInputV0(orquestacoreworkflow.ReviewResultStatusRejectedV0, ReplanActionSplitTaskV0)
	return ReviewReworkSignalV0{
		SignalRef:        input.SignalRef,
		RunRef:           input.RunRef,
		TaskRef:          input.TaskRef,
		ReviewRequestRef: input.ReviewResult.ReviewRequestID,
		DeliveryRef:      input.ReviewResult.DeliveryRef,
		ReviewResultRef:  input.ReviewResult.ReviewResultRef,
		ReviewStatus:     input.ReviewResult.Status,
		RequestedAction:  input.RequestedAction,
		ReasonRef:        input.ReasonRef,
		Summary:          input.ReviewResult.Summary,
		EvidenceRefs:     input.ReviewResult.EvidenceRefs,
	}
}

func mustReviewResultReplanProposalV0(t *testing.T, input ReviewResultReplanInputV0) ReplanProposalV0 {
	t.Helper()
	proposal, err := ReviewResultToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("ReviewResultToReplanProposalV0: %v", err)
	}
	if proposal == nil {
		t.Fatalf("expected replan proposal")
	}
	return *proposal
}

func assertReviewReplanProposalTraceV0(t *testing.T, proposal ReplanProposalV0, input ReviewResultReplanInputV0) {
	t.Helper()
	if proposal.ReplanRef != input.ReplanRef ||
		proposal.RunRef != input.RunRef ||
		proposal.TaskRef != input.TaskRef ||
		proposal.SourceRef != input.ReviewResult.ReviewResultRef ||
		proposal.ReasonCode != input.ReasonRef {
		t.Fatalf("proposal trace mismatch: %#v input=%#v", proposal, input)
	}
}

func assertReviewReworkSignalErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr ReviewReworkSignalErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReviewReworkSignalErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}
