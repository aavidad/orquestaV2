package orquestadirectorsupervisor

import (
	"strings"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func validateDirectorSupervisorInputV0(input DirectorSupervisorDecisionInputV0) error {
	if strings.TrimSpace(input.RunRef) == "" {
		return supervisorErrorV0(input, "run_ref requerido", "run_ref")
	}
	if input.StepNumber < 1 {
		return supervisorErrorV0(input, "step_number debe ser mayor que cero", "step_number")
	}
	if input.MaxSteps < 1 {
		return supervisorErrorV0(input, "max_steps debe ser mayor que cero", "max_steps")
	}
	if input.MaxSteps > DirectorSupervisorMaxStepsLimitV0 {
		return supervisorErrorV0(input, "max_steps supera el limite", "max_steps")
	}
	if strings.TrimSpace(input.LastStepResult.RunRef) != input.RunRef {
		return supervisorErrorV0(input, "last_step_result.run_ref no coincide", "last_step_result.run_ref")
	}
	if input.LastErrorCode != "" {
		return nil
	}
	if input.LastStepResult.Status == "" {
		return supervisorErrorV0(input, "last_step_result.status requerido", "last_step_result.status")
	}
	if !isKnownSupervisorCycleStatusV0(input.LastStepResult.Status) {
		return supervisorErrorV0(input, "last_step_result.status no soportado", "last_step_result.status")
	}
	if input.LastStepResult.SchedulerStatus != "" && !isKnownSupervisorSchedulerStatusV0(input.LastStepResult.SchedulerStatus) {
		return supervisorErrorV0(input, "last_step_result.scheduler_status no soportado", "last_step_result.scheduler_status")
	}
	for _, reason := range input.LastStepResult.WaitingReasons {
		if !isKnownSupervisorWaitingReasonV0(reason) {
			return supervisorErrorV0(input, "last_step_result.waiting_reasons no soportado", "last_step_result.waiting_reasons")
		}
	}
	return nil
}

func isKnownSupervisorCycleStatusV0(status orquestadirectorrunner.DirectorCycleStatusV0) bool {
	switch status {
	case orquestadirectorrunner.DirectorCycleStatusQuiescentV0,
		orquestadirectorrunner.DirectorCycleStatusWaitingV0,
		orquestadirectorrunner.DirectorCycleStatusBlockedV0,
		orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0,
		orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0,
		orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0:
		return true
	default:
		return false
	}
}

func isKnownSupervisorSchedulerStatusV0(status orquestadirectorscheduler.DirectorSchedulerTickStatusV0) bool {
	switch status {
	case orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0,
		orquestadirectorscheduler.SchedulerTickStatusWaitingV0,
		orquestadirectorscheduler.SchedulerTickStatusBlockedV0,
		orquestadirectorscheduler.SchedulerTickStatusNeedsDirectorV0,
		orquestadirectorscheduler.SchedulerTickStatusQuiescentV0:
		return true
	default:
		return false
	}
}

func isKnownSupervisorWaitingReasonV0(reason orquestadirectorscheduler.SchedulerWaitingReasonV0) bool {
	switch reason {
	case orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0,
		orquestadirectorscheduler.SchedulerWaitingCapacityPendingV0,
		orquestadirectorscheduler.SchedulerWaitingAgentLifecyclePendingV0,
		orquestadirectorscheduler.SchedulerWaitingAgentDeliveryPendingV0,
		orquestadirectorscheduler.SchedulerWaitingCandidateMissingV0,
		orquestadirectorscheduler.SchedulerWaitingDirectorQuestionPendingV0,
		orquestadirectorscheduler.SchedulerWaitingQualityGateFollowupV0:
		return true
	default:
		return false
	}
}
