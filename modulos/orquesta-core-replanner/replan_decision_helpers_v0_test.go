package orquestacorereplanner

import (
	"errors"
	"testing"
)

func validReplanDecisionV0() ReplanDecisionV0 {
	return ReplanDecisionV0{
		ReplanRef:      "replan-001",
		RunRef:         "run-001",
		TaskRef:        "task-001",
		SourceRef:      "replan-proposal-001",
		AcceptedAction: ReplanActionRetryTaskV0,
		FollowupRefs:   []string{"rework-001"},
		Summary:        "Decision compacta para continuar el rework.",
		EvidenceRefs:   []string{"proposal-001"},
	}
}

func mustReplanDecisionV0(t *testing.T, decision ReplanDecisionV0) ReplanDecisionV0 {
	t.Helper()
	normalized, err := NewReplanDecisionV0(decision)
	if err != nil {
		t.Fatalf("NewReplanDecisionV0: %v", err)
	}
	return normalized
}

func assertReplanDecisionErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr ReplanDecisionErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReplanDecisionErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func assertReplanDecisionErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr ReplanDecisionErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReplanDecisionErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
