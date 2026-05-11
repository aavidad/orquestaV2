package orquestadirectorsupervisor

import (
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func DecideDirectorSupervisorNextActionV0(
	input DirectorSupervisorDecisionInputV0,
) (DirectorSupervisorDecisionV0, error) {
	input = normalizeDirectorSupervisorInputV0(input)
	decision := newDirectorSupervisorDecisionV0(input)
	if err := validateDirectorSupervisorInputV0(input); err != nil {
		return resultWithSupervisorErrorV0(decision, err)
	}
	if input.LastErrorCode != "" {
		return supervisorDecisionWithActionV0(
			decision,
			DirectorSupervisorActionStopErrorV0,
			DirectorSupervisorReasonLastErrorV0,
			false,
		), nil
	}
	if len(decision.PendingOutboxRefs) > 0 || input.LastStepResult.OutboxPendingAfterCount > 0 {
		return supervisorWaitOutboxDecisionV0(decision), nil
	}
	return decideSupervisorFromCycleStatusV0(input, decision)
}

func decideSupervisorFromCycleStatusV0(
	input DirectorSupervisorDecisionInputV0,
	decision DirectorSupervisorDecisionV0,
) (DirectorSupervisorDecisionV0, error) {
	switch input.LastStepResult.Status {
	case orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0:
		return supervisorWaitOutboxDecisionV0(decision), nil
	case orquestadirectorrunner.DirectorCycleStatusWaitingV0:
		return supervisorWaitingDecisionV0(input, decision), nil
	case orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0:
		return supervisorCommandsAppliedDecisionV0(input, decision), nil
	case orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0:
		if supervisorHasWaitingReasonV0(input, orquestadirectorscheduler.SchedulerWaitingCandidateMissingV0) {
			return supervisorCandidateWaitDecisionV0(decision), nil
		}
		return supervisorDecisionWithActionV0(
			decision,
			DirectorSupervisorActionNeedsDirectorV0,
			DirectorSupervisorReasonNeedsDirectorV0,
			false,
		), nil
	case orquestadirectorrunner.DirectorCycleStatusBlockedV0:
		return supervisorDecisionWithActionV0(
			decision,
			DirectorSupervisorActionBlockedV0,
			DirectorSupervisorReasonBlockedV0,
			false,
		), nil
	case orquestadirectorrunner.DirectorCycleStatusQuiescentV0:
		return supervisorDecisionWithActionV0(
			decision,
			DirectorSupervisorActionStopQuiescentV0,
			DirectorSupervisorReasonQuiescentV0,
			false,
		), nil
	default:
		return resultWithSupervisorErrorV0(decision, supervisorErrorV0(
			input,
			"last_step_result.status no soportado",
			"last_step_result.status",
		))
	}
}

func supervisorWaitingDecisionV0(
	input DirectorSupervisorDecisionInputV0,
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorDecisionV0 {
	if supervisorHasWaitingReasonV0(input, orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0) {
		return supervisorWaitOutboxDecisionV0(decision)
	}
	if supervisorHasWaitingReasonV0(input, orquestadirectorscheduler.SchedulerWaitingCandidateMissingV0) {
		return supervisorCandidateWaitDecisionV0(decision)
	}
	return supervisorDecisionWithActionV0(
		decision,
		DirectorSupervisorActionWaitExternalV0,
		DirectorSupervisorReasonExternalWaitV0,
		false,
	)
}

func supervisorCandidateWaitDecisionV0(
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorDecisionV0 {
	return supervisorDecisionWithActionV0(
		decision,
		DirectorSupervisorActionWaitExternalV0,
		DirectorSupervisorReasonCandidateWaitV0,
		false,
	)
}

func supervisorCommandsAppliedDecisionV0(
	input DirectorSupervisorDecisionInputV0,
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorDecisionV0 {
	if input.StepNumber >= input.MaxSteps {
		return supervisorDecisionWithActionV0(
			decision,
			DirectorSupervisorActionStopMaxStepsV0,
			DirectorSupervisorReasonMaxStepsV0,
			false,
		)
	}
	return supervisorDecisionWithActionV0(
		decision,
		DirectorSupervisorActionContinueV0,
		DirectorSupervisorReasonContinueV0,
		true,
	)
}
