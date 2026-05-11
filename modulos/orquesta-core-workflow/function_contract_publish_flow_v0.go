package orquestacoreworkflow

import "strings"

func ensureFunctionContractPublishCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload PublishFunctionContractCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensureFunctionContractPublishCommandPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !decisionAlreadyReflectedV0(current, payload.DecisionRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.decision_ref")
	}
	return nil
}

func ensureFunctionContractPublishedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload FunctionContractPublishedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	payload = normalizeFunctionContractPublishedPayloadV0(payload)
	if err := ensureFunctionContractPublishedEventPhaseCurrentV0(current, payload.PhaseID); err != nil {
		return err
	}
	if !decisionAlreadyReflectedV0(current, payload.DecisionRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.decision_ref")
	}
	return nil
}

func ensureFunctionContractPublishCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureFunctionContractPublishedEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
