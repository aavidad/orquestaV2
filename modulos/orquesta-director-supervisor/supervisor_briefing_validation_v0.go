package orquestadirectorsupervisor

import "strings"

func validateDirectorSupervisorBriefingInputV0(
	input DirectorSupervisorBriefingInputV0,
) error {
	decision := input.Decision
	if strings.TrimSpace(decision.RunRef) == "" {
		return supervisorErrorFromDecisionV0(decision, "run_ref requerido", "decision.run_ref")
	}
	if decision.Action == "" {
		return supervisorErrorFromDecisionV0(decision, "action requerido", "decision.action")
	}
	if !isKnownSupervisorActionV0(decision.Action) {
		return supervisorErrorFromDecisionV0(decision, "action no soportado", "decision.action")
	}
	if !isKnownSupervisorAutonomousRecommendationV0(decision.AutonomousRecommendation) {
		return supervisorErrorFromDecisionV0(decision, "autonomous_recommendation no soportada", "decision.autonomous_recommendation")
	}
	if strings.TrimSpace(decision.ReasonCode) == "" {
		return supervisorErrorFromDecisionV0(decision, "reason_code requerido", "decision.reason_code")
	}
	if decision.StepNumber < 1 {
		return supervisorErrorFromDecisionV0(decision, "step_number debe ser mayor que cero", "decision.step_number")
	}
	if decision.MaxSteps < 1 {
		return supervisorErrorFromDecisionV0(decision, "max_steps debe ser mayor que cero", "decision.max_steps")
	}
	if decision.MaxSteps > DirectorSupervisorMaxStepsLimitV0 {
		return supervisorErrorFromDecisionV0(decision, "max_steps supera el limite", "decision.max_steps")
	}
	return nil
}

func isKnownSupervisorActionV0(action DirectorSupervisorActionV0) bool {
	switch action {
	case DirectorSupervisorActionContinueV0,
		DirectorSupervisorActionWaitOutboxV0,
		DirectorSupervisorActionWaitExternalV0,
		DirectorSupervisorActionNeedsDirectorV0,
		DirectorSupervisorActionBlockedV0,
		DirectorSupervisorActionStopQuiescentV0,
		DirectorSupervisorActionStopMaxStepsV0,
		DirectorSupervisorActionStopErrorV0:
		return true
	default:
		return false
	}
}

func supervisorErrorFromDecisionV0(
	decision DirectorSupervisorDecisionV0,
	message string,
	field string,
) DirectorSupervisorErrorV0 {
	return DirectorSupervisorErrorV0{
		Code:      ErrDirectorSupervisorDecisionInvalidaV0,
		Message:   message,
		Field:     field,
		Retryable: true,
		Issues: []DirectorSupervisorIssueV0{{
			Code:    ErrDirectorSupervisorDecisionInvalidaV0,
			Field:   field,
			Message: message,
		}},
	}
}
