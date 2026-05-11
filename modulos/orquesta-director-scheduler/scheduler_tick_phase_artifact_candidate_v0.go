package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func (collector *schedulerTickCollectorV0) collectPhaseArtifactCandidateV0(
	candidate SchedulablePhaseArtifactCandidateV0,
) error {
	payload := candidate.Payload
	artifactRef := payload.ArtifactRef
	if collector.phaseArtifacts[artifactRef] || collector.plannedPhaseArtifacts[artifactRef] {
		return nil
	}
	if strings.TrimSpace(payload.PhaseID) != collector.input.Snapshot.CurrentPhaseID {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if !collector.agents[payload.AgentRef] {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	if collector.failedAgents[payload.AgentRef] || collector.stoppedAgents[payload.AgentRef] {
		collector.addBlockedRefsV0([]string{payload.AgentRef})
		return nil
	}
	if !collector.startedAgents[payload.AgentRef] {
		collector.addWaitingV0(SchedulerWaitingAgentLifecyclePendingV0)
		return nil
	}
	command, err := orquestacoreworkflow.NewRegisterPhaseArtifactCommandV0(
		candidate.CommandMeta,
		payload,
	)
	if err != nil {
		return err
	}
	collector.addReadyCommandV0(command)
	collector.plannedPhaseArtifacts[artifactRef] = true
	return nil
}
