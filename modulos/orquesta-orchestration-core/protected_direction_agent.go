package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
	if !protectedDirectionPrimaryAgentRefV0(agentRef) {
		return false
	}
	return stringInSetV0(agentRef, run.Agents) || stringInSetV0(agentRef, run.StartedAgents)
}

func protectedDirectionPrimaryAgentRefV0(agentRef string) bool {
	agentRef = strings.ToLower(strings.TrimSpace(agentRef))
	return strings.HasSuffix(agentRef, "-director")
}

func protectedDirectionStopAllowedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation AgentProgressObservationV0,
) *bool {
	if observation.Report.Status == orquestaruntime.AgentStoppedV0 ||
		observation.Report.Status == orquestaruntime.AgentLoopDetectedV0 {
		return nil
	}
	if !protectedDirectionAgentObservationV0(run, observation) {
		return nil
	}
	stopAllowed := false
	return &stopAllowed
}
