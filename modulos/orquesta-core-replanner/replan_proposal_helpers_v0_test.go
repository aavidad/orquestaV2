package orquestacorereplanner

import (
	"errors"
	"testing"
)

func validReplanProposalV0() ReplanProposalV0 {
	return ReplanProposalV0{
		ReplanRef:         "replan-001",
		RunRef:            "run-001",
		TaskRef:           "task-001",
		SourceRef:         "review-result-001",
		ReasonCode:        "changes_requested",
		RecommendedAction: ReplanActionRetryTaskV0,
		Summary:           "Rehacer la entrega con evidencia compacta.",
		EvidenceRefs:      []string{"review-result-001"},
	}
}

func mustReplanProposalV0(t *testing.T, proposal ReplanProposalV0) ReplanProposalV0 {
	t.Helper()
	normalized, err := NewReplanProposalV0(proposal)
	if err != nil {
		t.Fatalf("NewReplanProposalV0: %v", err)
	}
	return normalized
}

func assertReplanProposalErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr ReplanProposalErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReplanProposalErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func assertReplanProposalErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr ReplanProposalErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReplanProposalErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
