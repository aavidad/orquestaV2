package orquestadirectorsupervisor

import (
	"errors"
	"testing"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
)

const supervisorTestRunRefV0 = "run-supervisor-001"

func supervisorTestInputV0(
	status orquestadirectorrunner.DirectorCycleStatusV0,
) DirectorSupervisorDecisionInputV0 {
	return DirectorSupervisorDecisionInputV0{
		RunRef:     supervisorTestRunRefV0,
		StepNumber: 1,
		MaxSteps:   3,
		LastStepResult: orquestadirectorcycle.DirectorCycleStepResultV0{
			RunRef: supervisorTestRunRefV0,
			Status: status,
		},
	}
}

func assertSupervisorActionV0(
	t *testing.T,
	input DirectorSupervisorDecisionInputV0,
	action DirectorSupervisorActionV0,
	shouldContinue bool,
) DirectorSupervisorDecisionV0 {
	t.Helper()
	decision, err := DecideDirectorSupervisorNextActionV0(input)
	if err != nil {
		t.Fatalf("decide supervisor: %v", err)
	}
	if decision.Action != action || decision.ShouldContinue != shouldContinue {
		t.Fatalf("decision=%+v, want action=%s continue=%v", decision, action, shouldContinue)
	}
	if !isKnownSupervisorAutonomousRecommendationV0(decision.AutonomousRecommendation) {
		t.Fatalf("recommendation=%q is not known", decision.AutonomousRecommendation)
	}
	return decision
}

func assertSupervisorErrorV0(t *testing.T, input DirectorSupervisorDecisionInputV0, field string) {
	t.Helper()
	_, err := DecideDirectorSupervisorNextActionV0(input)
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr DirectorSupervisorErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T %v", err, err)
	}
	if publicErr.Code != ErrDirectorSupervisorDecisionInvalidaV0 || publicErr.Field != field {
		t.Fatalf("error=%+v, want field=%s", publicErr, field)
	}
}
