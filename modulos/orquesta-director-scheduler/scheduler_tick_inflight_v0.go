package orquestadirectorscheduler

func schedulerHasInFlightAgentsV0(snapshot RunSchedulingSnapshotV0) bool {
	activeStarted := schedulerActiveStartedAgentsV0(snapshot)
	if activeStarted == 0 {
		return false
	}
	completed := len(compactSchedulerStringsV0(snapshot.Deliveries)) +
		len(compactSchedulerStringsV0(snapshot.PhaseArtifacts))
	return activeStarted > completed
}

func schedulerActiveStartedAgentsV0(snapshot RunSchedulingSnapshotV0) int {
	failed := schedulerStringSetV0(snapshot.FailedAgents)
	stopped := schedulerStringSetV0(snapshot.StoppedAgents)
	active := 0
	for _, agentRef := range compactSchedulerStringsV0(snapshot.StartedAgents) {
		if failed[agentRef] || stopped[agentRef] {
			continue
		}
		active++
	}
	return active
}
