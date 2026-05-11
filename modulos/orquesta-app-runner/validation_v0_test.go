package orquestaapprunner

import (
	"errors"
	"testing"
)

func TestPrepareAppOrchestrationV0ConservaCampoPlanner(t *testing.T) {
	spec := validFactoryAppSpecForRunnerTestV0(t)
	spec.Validation.Estado = "provisional"

	_, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		RunRef:     "run-app-runner-field-001",
		OccurredAt: "2026-05-09T23:40:00Z",
		AppSpec:    spec,
	})
	if err == nil {
		t.Fatalf("esperaba error")
	}
	var issue AppRunnerIssueV0
	if !errors.As(err, &issue) || issue.Field != "app_spec.validation.estado" {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
}
