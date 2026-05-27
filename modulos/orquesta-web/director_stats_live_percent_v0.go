package orquestaweb

import orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"

func webAutoprogrammingLivePercentV0(stats *orquestacionnucleoapp.DirectorRunStatsV0) int {
	if stats == nil {
		return 0
	}
	if stats.Progress.PercentComplete != 0 || !webAutoprogrammingStatsHasLiveWorkV0(stats) {
		return stats.Progress.PercentComplete
	}
	return 1
}

func webAutoprogrammingStatsHasLiveWorkV0(stats *orquestacionnucleoapp.DirectorRunStatsV0) bool {
	return stats.Counts.AgentsInFlight > 0 ||
		stats.Counts.AgentsStarted > stats.Counts.AgentsDelivered ||
		stats.Progress.ProgressingAgents > 0 ||
		stats.Progress.StalledAgents > 0 ||
		stats.Progress.LoopDetectedAgents > 0
}

func webDirectorStatsLivePercentV0(progress WebDirectorProgressStatsContractV0) int {
	if progress.PercentComplete != 0 || !webDirectorProgressHasLiveWorkV0(progress) {
		return progress.PercentComplete
	}
	return 1
}

func applyDirectorStatsLiveCountsPercentV0(vm *WebDirectorStatsViewModelV0) {
	if vm == nil || vm.Progress.PercentComplete != 0 || !webDirectorStatsCountsHaveLiveWorkV0(vm.Counts) {
		return
	}
	vm.Progress.PercentComplete = 1
	vm.Resumen.PercentComplete = 1
}

func webDirectorStatsCountsHaveLiveWorkV0(counts WebDirectorStatsCountsV0) bool {
	return counts.AgentsInFlight > 0 ||
		counts.AgentsStarted > counts.AgentsDelivered
}

func webDirectorProgressHasLiveWorkV0(progress WebDirectorProgressStatsContractV0) bool {
	return progress.ProgressingAgents > 0 ||
		progress.StalledAgents > 0 ||
		progress.LoopDetectedAgents > 0 ||
		progress.ObservedAgents > 0 ||
		hasWebDirectorLiveTaskV0(progress.Tasks)
}

func hasWebDirectorLiveTaskV0(tasks []WebDirectorTaskProgressV0) bool {
	for _, task := range tasks {
		switch trimDirectorStatsV0(task.Status) {
		case orquestacionnucleoapp.DirectorTaskProgressInProgressV0,
			orquestacionnucleoapp.DirectorTaskProgressStalledV0,
			orquestacionnucleoapp.DirectorTaskProgressLoopDetectedV0:
			return true
		}
	}
	return false
}
