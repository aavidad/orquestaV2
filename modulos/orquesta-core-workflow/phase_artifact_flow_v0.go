package orquestacoreworkflow

import "strings"

func ensureRegisterPhaseArtifactCommandAllowedV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload RegisterPhaseArtifactCommandPayloadV0,
) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if err := ensurePhaseArtifactCommandAgentPhaseV0(current, payload.AgentRef, payload.PhaseID); err != nil {
		return err
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	if !agentStartedAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRef) ||
		agentLostAlreadyReflectedV0(current, payload.AgentRef) ||
		agentStopAlreadyReflectedV0(current, payload.AgentRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_ref")
	}
	return nil
}

func ensurePhaseArtifactRegisteredEventAllowedV0(
	current OrchestrationRunV0,
	event OrchestrationEventV0,
	payload PhaseArtifactRegisteredPayloadV0,
) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if err := ensurePhaseArtifactEventAgentPhaseV0(current, payload.AgentRef, payload.PhaseID); err != nil {
		return err
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	if !agentStartedAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRef) ||
		agentLostAlreadyReflectedV0(current, payload.AgentRef) ||
		agentStopAlreadyReflectedV0(current, payload.AgentRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_ref")
	}
	return nil
}

func ensurePhaseArtifactCommandPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensurePhaseArtifactCommandAgentPhaseV0(run OrchestrationRunV0, agentRef string, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if len(compactStringsV0(run.AgentPhaseRefs)) == 0 {
		return ensurePhaseArtifactCommandPhaseCurrentV0(run, phaseID)
	}
	if agentRequestedForPhaseAlreadyReflectedV0(run, agentRef, phase) {
		return nil
	}
	return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
}

func ensurePhaseArtifactEventPhaseCurrentV0(run OrchestrationRunV0, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if normalizePhaseIDV0(run.CurrentPhase) != phase {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(run, phase) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func ensurePhaseArtifactEventAgentPhaseV0(run OrchestrationRunV0, agentRef string, phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if len(compactStringsV0(run.AgentPhaseRefs)) == 0 {
		return ensurePhaseArtifactEventPhaseCurrentV0(run, phaseID)
	}
	if agentRequestedForPhaseAlreadyReflectedV0(run, agentRef, phase) {
		return nil
	}
	return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
}
