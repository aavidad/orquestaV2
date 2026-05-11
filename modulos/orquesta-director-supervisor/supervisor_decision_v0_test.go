package orquestadirectorsupervisor

import (
	"reflect"
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestDecideDirectorSupervisorNextActionV0ContinuesUnderBudget(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0)
	assertSupervisorActionV0(t, input, DirectorSupervisorActionContinueV0, true)
}

func TestDecideDirectorSupervisorNextActionV0StopsAtMaxSteps(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0)
	input.StepNumber = input.MaxSteps
	assertSupervisorActionV0(t, input, DirectorSupervisorActionStopMaxStepsV0, false)
}

func TestDecideDirectorSupervisorNextActionV0WaitsForOutbox(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0)
	input.LastStepResult.PendingOutboxBeforeRefs = []string{"outbox-ref-001"}
	input.LastStepResult.PendingOutboxAfterRefs = []string{"outbox-ref-001", "outbox-ref-002"}

	decision := assertSupervisorActionV0(t, input, DirectorSupervisorActionWaitOutboxV0, false)
	expected := []string{"outbox-ref-001", "outbox-ref-002"}
	if !reflect.DeepEqual(decision.PendingOutboxRefs, expected) {
		t.Fatalf("pending refs=%v, want %v", decision.PendingOutboxRefs, expected)
	}
}

func TestDecideDirectorSupervisorNextActionV0OutboxRefsWinOverMaxSteps(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0)
	input.StepNumber = input.MaxSteps
	input.LastStepResult.PendingOutboxAfterRefs = []string{"outbox-ref-after-001"}
	assertSupervisorActionV0(t, input, DirectorSupervisorActionWaitOutboxV0, false)
}

func TestDecideDirectorSupervisorNextActionV0ClassifiesWaitingReasons(t *testing.T) {
	outbox := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusWaitingV0)
	outbox.LastStepResult.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{
		orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0,
	}
	outboxDecision := assertSupervisorActionV0(t, outbox, DirectorSupervisorActionWaitOutboxV0, false)
	if outboxDecision.AutonomousRecommendation != DirectorSupervisorAutonomousWaitV0 {
		t.Fatalf("recommendation=%s, want wait", outboxDecision.AutonomousRecommendation)
	}

	external := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusWaitingV0)
	external.LastStepResult.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{
		orquestadirectorscheduler.SchedulerWaitingAgentDeliveryPendingV0,
	}
	assertSupervisorActionV0(t, external, DirectorSupervisorActionWaitExternalV0, false)
}

func TestDecideDirectorSupervisorNextActionV0WaitsForPendingCandidates(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0)
	input.LastStepResult.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{
		orquestadirectorscheduler.SchedulerWaitingCandidateMissingV0,
	}

	decision := assertSupervisorActionV0(t, input, DirectorSupervisorActionWaitExternalV0, false)
	if decision.ReasonCode != DirectorSupervisorReasonCandidateWaitV0 {
		t.Fatalf("reason=%s, want %s", decision.ReasonCode, DirectorSupervisorReasonCandidateWaitV0)
	}
	if decision.AutonomousRecommendation != DirectorSupervisorAutonomousWaitV0 {
		t.Fatalf("recommendation=%s, want wait", decision.AutonomousRecommendation)
	}
}

func TestDecideDirectorSupervisorNextActionV0RecommendsAutonomousContinue(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0)
	decision := assertSupervisorActionV0(t, input, DirectorSupervisorActionContinueV0, true)
	if decision.AutonomousRecommendation != DirectorSupervisorAutonomousContinueV0 {
		t.Fatalf("recommendation=%s, want continue", decision.AutonomousRecommendation)
	}
}

func TestDecideDirectorSupervisorNextActionV0BlockedRefsDoNotOverrideCommandsApplied(t *testing.T) {
	input := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0)
	input.LastStepResult.BlockedRefs = []string{"claim-blocked-001"}
	assertSupervisorActionV0(t, input, DirectorSupervisorActionContinueV0, true)
}

func TestDecideDirectorSupervisorNextActionV0TerminalStatuses(t *testing.T) {
	cases := []struct {
		name   string
		status orquestadirectorrunner.DirectorCycleStatusV0
		action DirectorSupervisorActionV0
	}{
		{"needs_director", orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0, DirectorSupervisorActionNeedsDirectorV0},
		{"blocked", orquestadirectorrunner.DirectorCycleStatusBlockedV0, DirectorSupervisorActionBlockedV0},
		{"quiescent", orquestadirectorrunner.DirectorCycleStatusQuiescentV0, DirectorSupervisorActionStopQuiescentV0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := supervisorTestInputV0(tc.status)
			assertSupervisorActionV0(t, input, tc.action, false)
		})
	}
}

func TestDecideDirectorSupervisorNextActionV0StopsOnLastError(t *testing.T) {
	input := supervisorTestInputV0("")
	input.LastErrorCode = "director_cycle_step_runner"
	assertSupervisorActionV0(t, input, DirectorSupervisorActionStopErrorV0, false)
}
