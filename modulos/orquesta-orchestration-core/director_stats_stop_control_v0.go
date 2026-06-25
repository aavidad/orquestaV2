package orquestacionnucleoapp

import "strings"

const (
	directorRunControlStatusStopRequestedV0 = "stop_requested"
	directorRunControlStatusStoppedV0       = "stopped"
)

func ApplyDirectorRunControlStateV0(
	stats *DirectorRunStatsV0,
	status string,
	checkpointRecorded bool,
	forced bool,
	evidenceRefs []string,
) {
	if stats == nil {
		return
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return
	}
	stats.StopControl.RunControlStatus = status
	stats.StopControl.CheckpointRecorded = checkpointRecorded
	stats.StopControl.Forced = forced
	stats.StopControl.EvidenceRefs = compactStringsV0(append(
		stats.StopControl.EvidenceRefs,
		evidenceRefs...,
	))
	switch status {
	case directorRunControlStatusStopRequestedV0:
		stats.StopControl.Requested = true
		if !stats.StopControl.Propagated {
			stats.StopControl.Status = DirectorRunStopStatusRequestedV0
			stats.StopControl.Pending = stats.StopControl.StartedAgents > 0
			stats.StopControl.StopPendingAgents = stats.StopControl.StartedAgents
			stats.StopControl.PendingAgentRefs = pendingDirectorRunControlStartedRefsV0(*stats)
			return
		}
		if stats.StopControl.Pending {
			stats.StopControl.Status = DirectorRunStopStatusPendingV0
			return
		}
		if stats.StopControl.Confirmed {
			stats.StopControl.Status = DirectorRunStopStatusConfirmedV0
			return
		}
		stats.StopControl.Status = DirectorRunStopStatusPropagatedV0
	case directorRunControlStatusStoppedV0:
		stats.StopControl.Requested = true
		stats.StopControl.Confirmed = true
		stats.StopControl.Pending = false
		stats.StopControl.StopPendingAgents = 0
		stats.StopControl.PendingAgentRefs = nil
		stats.StopControl.Status = DirectorRunStopStatusConfirmedV0
	}
}

func pendingDirectorRunControlStartedRefsV0(
	stats DirectorRunStatsV0,
) []string {
	refs := make([]string, 0, len(stats.Agents))
	for _, agent := range stats.Agents {
		if !agent.Started || agent.Failed || agent.Lost || agent.Completed || agent.StopConfirmed {
			continue
		}
		refs = append(refs, strings.TrimSpace(agent.AgentRequestID))
	}
	return compactStringsV0(refs)
}
