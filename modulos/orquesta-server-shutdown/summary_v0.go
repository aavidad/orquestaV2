package orquestaservershutdown

import orquestaruncontrol "orquesta/modulos/orquesta-run-control"

func applyShutdownControlStateV0(
	run ServerShutdownRunResultV0,
	state orquestaruncontrol.RunControlStateV0,
) ServerShutdownRunResultV0 {
	pendingCheckpoint := run.CheckpointRequired && !run.StopRequested
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	run.ControlStatus = string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status))
	run.CheckpointRequired = evaluation.CheckpointRequired
	run.Terminal = evaluation.Terminal
	run.Ready = evaluation.Terminal
	if pendingCheckpoint {
		run.CheckpointRequired = true
		run.Ready = false
	}
	return run
}

func applyShutdownStatsV0(
	run ServerShutdownRunResultV0,
	stats RunShutdownStatsV0,
) ServerShutdownRunResultV0 {
	run.AgentsInFlight = stats.AgentsInFlight
	run.AgentsStopRequested = stats.AgentsStopRequested
	run.AgentsStopConfirmed = stats.AgentsStopConfirmed
	if !run.Ready && !run.CheckpointRequired {
		run.Ready = stats.AgentsInFlight == 0
	}
	return run
}

func summarizeShutdownResultV0(
	result ServerShutdownResultV0,
) ServerShutdownResultV0 {
	result.Status = ServerShutdownStatusReadyV0
	result.ShutdownReady = true
	for _, run := range result.Runs {
		result.AgentsInFlight += run.AgentsInFlight
		result.CheckpointAgentsPending += len(compactServerShutdownStringsV0(run.PendingCheckpointAgentRefs))
		if run.Ready {
			result.RunsStopped++
		}
		if run.CheckpointRequired {
			result.CheckpointsPending++
		}
		if run.ForcedAfterCheckpointDeadline {
			result.CheckpointDeadlinesExpired++
		}
		if !run.Ready {
			result.ShutdownReady = false
			if run.CheckpointRequired {
				result.Status = ServerShutdownStatusWaitingCheckpointV0
			} else if result.Status == ServerShutdownStatusReadyV0 {
				result.Status = ServerShutdownStatusWaitingDrainV0
			}
		}
	}
	return result
}
