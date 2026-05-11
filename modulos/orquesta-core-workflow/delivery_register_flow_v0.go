package orquestacoreworkflow

import "strings"

func ensureRegisterDeliveryCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RegisterDeliveryCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureRegisterDeliveryCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.task_id")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	if !agentStartedAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	if agentStopAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	return nil
}

func ensureDeliveryRegisteredEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload DeliveryRegisteredPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensureDeliveryRegisteredEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.task_id")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	if !agentStartedAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	if agentStopAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	return nil
}

func ensureRegisterDeliveryCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !deliveryProgrammingPhaseCurrentV0(run, phaseID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureDeliveryRegisteredEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	if !deliveryProgrammingPhaseCurrentV0(run, phaseID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func deliveryProgrammingPhaseCurrentV0(run OrchestrationRunV0, phaseID string) bool {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if phase != OrchestrationPhaseProgramacionV0 {
		return false
	}
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return false
	}
	return phaseIsCurrentAndActiveV0(run, phase)
}
