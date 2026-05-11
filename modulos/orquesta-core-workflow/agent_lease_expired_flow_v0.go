package orquestacoreworkflow

import "strings"

func ensureRegisterAgentLeaseExpiredCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RegisterAgentLeaseExpiredCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if !agentLeaseExpiredRunMatchesV0(current, payload.RunRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.run_ref")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	return nil
}

func ensureAgentLeaseExpiredEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload AgentLeaseExpiredPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if !agentLeaseExpiredRunMatchesV0(current, payload.RunRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.run_ref")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	return nil
}

func agentLeaseExpiredRunMatchesV0(current OrchestrationRunV0, runRef string) bool {
	return strings.TrimSpace(current.RunID) == strings.TrimSpace(runRef)
}
