package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func protectedDirectionAgentObservationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation AgentProgressObservationV0,
) bool {
	phaseID := strings.TrimSpace(observation.PhaseID)
	if phaseID == "" {
		phaseID = strings.TrimSpace(string(run.CurrentPhase))
	}
	if phaseID != string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0) {
		return false
	}
	taskRef := strings.TrimSpace(observation.TaskRef)
	if taskRef == "" || stringInSetV0(taskRef, run.Tasks) {
		return false
	}
	agentRef := strings.TrimSpace(observation.Report.AgentRequestID)
	return stringInSetV0(agentRef, run.Agents) || stringInSetV0(agentRef, run.StartedAgents)
}

func protectedDirectionStopAllowedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation AgentProgressObservationV0,
) *bool {
	if !protectedDirectionAgentObservationV0(run, observation) {
		return nil
	}
	stopAllowed := false
	return &stopAllowed
}
