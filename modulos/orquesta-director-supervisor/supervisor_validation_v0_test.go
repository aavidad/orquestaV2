package orquestadirectorsupervisor

import (
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestDecideDirectorSupervisorNextActionV0RejectsInvalidInput(t *testing.T) {
	valid := supervisorTestInputV0(orquestadirectorrunner.DirectorCycleStatusQuiescentV0)

	missingRun := valid
	missingRun.RunRef = ""
	assertSupervisorErrorV0(t, missingRun, "run_ref")

	mismatchedRun := valid
	mismatchedRun.LastStepResult.RunRef = "run-other"
	assertSupervisorErrorV0(t, mismatchedRun, "last_step_result.run_ref")

	missingStep := valid
	missingStep.StepNumber = 0
	assertSupervisorErrorV0(t, missingStep, "step_number")

	missingBudget := valid
	missingBudget.MaxSteps = 0
	assertSupervisorErrorV0(t, missingBudget, "max_steps")

	excessiveBudget := valid
	excessiveBudget.MaxSteps = DirectorSupervisorMaxStepsLimitV0 + 1
	assertSupervisorErrorV0(t, excessiveBudget, "max_steps")

	unknownStatus := valid
	unknownStatus.LastStepResult.Status = "unknown"
	assertSupervisorErrorV0(t, unknownStatus, "last_step_result.status")

	unknownSchedulerStatus := valid
	unknownSchedulerStatus.LastStepResult.SchedulerStatus = "unknown"
	assertSupervisorErrorV0(t, unknownSchedulerStatus, "last_step_result.scheduler_status")

	unknownWaitingReason := valid
	unknownWaitingReason.LastStepResult.WaitingReasons = []orquestadirectorscheduler.SchedulerWaitingReasonV0{"unknown"}
	assertSupervisorErrorV0(t, unknownWaitingReason, "last_step_result.waiting_reasons")
}
