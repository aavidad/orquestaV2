package orquestadirectortickinput

import (
	"strings"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func compactTickInputSnapshotForPhaseArtifactsV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	artifacts := map[string]bool{}
	agents := map[string]bool{}
	for _, candidate := range input.PhaseArtifactCandidates {
		artifacts[strings.TrimSpace(candidate.Payload.ArtifactRef)] = true
		agents[strings.TrimSpace(candidate.Payload.AgentRef)] = true
	}
	input.Snapshot = compactTickInputSnapshotForAgentsV0(input.Snapshot, agents)
	input.Snapshot.PhaseArtifacts = filterTickInputRefsByExactV0(input.Snapshot.PhaseArtifacts, artifacts)
	return input
}

func compactTickInputSnapshotForDeliveriesV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	deliveries := map[string]bool{}
	tasks := map[string]bool{}
	agents := map[string]bool{}
	for _, candidate := range input.DeliveryCandidates {
		payload := candidate.Payload
		deliveries[strings.TrimSpace(payload.DeliveryRef)] = true
		tasks[strings.TrimSpace(payload.TaskID)] = true
		agents[strings.TrimSpace(payload.AgentRef)] = true
	}
	input.Snapshot = compactTickInputSnapshotForAgentsV0(input.Snapshot, agents)
	input.Snapshot.Tasks = filterTickInputRefsByExactV0(input.Snapshot.Tasks, tasks)
	input.Snapshot.Deliveries = filterTickInputRefsByExactV0(input.Snapshot.Deliveries, deliveries)
	return input
}

func compactTickInputSnapshotForAgentsV0(
	snapshot orquestadirectorscheduler.RunSchedulingSnapshotV0,
	agents map[string]bool,
) orquestadirectorscheduler.RunSchedulingSnapshotV0 {
	snapshot.Agents = filterTickInputRefsByExactV0(snapshot.Agents, agents)
	snapshot.StartedAgents = filterTickInputRefsByExactV0(snapshot.StartedAgents, agents)
	snapshot.FailedAgents = filterTickInputRefsByExactV0(snapshot.FailedAgents, agents)
	snapshot.LostAgents = filterTickInputRefsByExactV0(snapshot.LostAgents, agents)
	snapshot.StoppedAgents = filterTickInputRefsByExactV0(snapshot.StoppedAgents, agents)
	return snapshot
}

func filterTickInputRefsByExactV0(values []string, keep map[string]bool) []string {
	if len(values) == 0 || len(keep) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if keep[trimmed] {
			out = append(out, trimmed)
		}
	}
	return compactTickInputRefsV0(out)
}
