package orquestaoperatormcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOperatorCapabilitiesAreCompactAndPublicV0(t *testing.T) {
	capabilities := NewOperatorMCPCapabilitiesV0()
	data, err := json.Marshal(capabilities)
	if err != nil {
		t.Fatalf("marshal capabilities: %v", err)
	}
	payload := strings.ToLower(string(data))
	for _, forbidden := range []string{"postgres", "sqlite", "/home/", "oauth", "provider=", "model=", "password", "token"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("capabilities leaked forbidden marker %q in %s", forbidden, payload)
		}
	}
	if len(capabilities.Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(capabilities.Tools))
	}
}

func TestValidateOperatorStatusQueryRequiresOpaqueRefsV0(t *testing.T) {
	issues := ValidateOperatorStatusQueryV0(OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		StatusConnectorRef: "status-connector-1",
		IncludeSections:    []string{"estado", "bloqueos"},
	})
	if len(issues) != 0 {
		t.Fatalf("expected valid query, got %#v", issues)
	}

	issues = ValidateOperatorStatusQueryV0(OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "subject..ref",
		StatusConnectorRef: "status-connector-1",
	})
	if !hasIssueCodeV0(issues, ErrOperatorMCPOpaqueRefV0) {
		t.Fatalf("expected opaque ref issue, got %#v", issues)
	}
}

func TestValidateOperatorSupervisedBurstBudgetV0(t *testing.T) {
	issues := ValidateOperatorSupervisedBurstV0(OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-1",
		RunRef:            "run-1",
		BurstConnectorRef: "burst-connector-1",
		SupervisionRef:    "supervision-1",
		MaxSteps:          2,
	})
	if len(issues) != 0 {
		t.Fatalf("expected valid burst request, got %#v", issues)
	}

	issues = ValidateOperatorSupervisedBurstV0(OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-1",
		RunRef:            "run-1",
		BurstConnectorRef: "burst-connector-1",
		SupervisionRef:    "supervision-1",
		MaxSteps:          OperatorMCPMaxBurstStepsLimitV0 + 1,
	})
	if !hasIssueCodeV0(issues, ErrOperatorMCPBudgetInvalidV0) {
		t.Fatalf("expected budget issue, got %#v", issues)
	}
}

func TestValidateOperatorDirectedQueryRejectsSensitiveRefsV0(t *testing.T) {
	issues := ValidateOperatorDirectedQueryV0(OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "query-connector-1",
		Question:          "Que accion publica corresponde ahora?",
	})
	if len(issues) != 0 {
		t.Fatalf("expected valid directed query, got %#v", issues)
	}

	issues = ValidateOperatorDirectedQueryV0(OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "query..connector",
		Question:          "Que accion publica corresponde ahora?",
	})
	if !hasIssueCodeV0(issues, ErrOperatorMCPOpaqueRefV0) {
		t.Fatalf("expected sensitive ref issue, got %#v", issues)
	}

	issues = ValidateOperatorDirectedQueryV0(OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "query-connector-1",
		Question:          "revisa token de operador",
	})
	if !hasIssueCodeV0(issues, ErrOperatorMCPQuestionInvalidV0) {
		t.Fatalf("expected sensitive question issue, got %#v", issues)
	}
}

func TestValidateOperatorPendingOutboxV0(t *testing.T) {
	issues := ValidateOperatorPendingOutboxV0(OperatorPendingOutboxQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-connector-1",
		Limit:              10,
		IncludeKinds:       []string{"director_question"},
	})
	if len(issues) != 0 {
		t.Fatalf("expected valid outbox query, got %#v", issues)
	}

	issues = ValidateOperatorPendingOutboxV0(OperatorPendingOutboxQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-connector-1",
		Limit:              OperatorMCPMaxOutboxLimitV0 + 1,
	})
	if !hasIssueCodeV0(issues, ErrOperatorMCPLimitInvalidV0) {
		t.Fatalf("expected limit issue, got %#v", issues)
	}
}

func hasIssueCodeV0(issues []OperatorMCPIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
