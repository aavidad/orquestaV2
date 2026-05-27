package orquestafactory

import (
	"reflect"
	"testing"
	"time"
)

func TestSolicitarNuevaAppV0RequiresInjectedReceptionTime(t *testing.T) {
	_, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), time.Time{})
	if !hasIssueFieldV0(issues, "received_at") {
		t.Fatalf("expected received_at clock issue, got %+v", issues)
	}
	if !hasIssueMessageV0(issues, AppSpecReceptionTimeRequiredReasonV0) {
		t.Fatalf("expected stable clock reason, got %+v", issues)
	}
}

func TestSolicitarNuevaAppV0DeterministicWithFixedReceptionTime(t *testing.T) {
	now := time.Date(2026, 5, 24, 16, 10, 0, 0, time.UTC)
	first, firstIssues := SolicitarNuevaAppV0(validMinimalRequestV0(), now)
	second, secondIssues := SolicitarNuevaAppV0(validMinimalRequestV0(), now)
	if len(firstIssues) > 0 || len(secondIssues) > 0 {
		t.Fatalf("unexpected issues: first=%+v second=%+v", firstIssues, secondIssues)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("fixed clock should produce same AppSpec")
	}
	if first.CreatedAt != "2026-05-24T16:10:00Z" {
		t.Fatalf("created_at=%q", first.CreatedAt)
	}
}

func hasIssueMessageV0(issues []ValidationIssue, message string) bool {
	for _, issue := range issues {
		if issue.Message == message {
			return true
		}
	}
	return false
}
