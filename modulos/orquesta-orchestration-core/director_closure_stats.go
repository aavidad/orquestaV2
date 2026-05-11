package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func buildDirectorClosureStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) DirectorClosureStatsV0 {
	closure := DirectorClosureStatsV0{
		Closed:      directorRunClosureClosedV0(run),
		BlockerRefs: compactStringsV0(run.Blockers),
	}
	if closure.Closed {
		closure.Status = DirectorClosureStatusClosedV0
		closure.BlockerRefs = nil
		return closure
	}
	closure.BlockedBy = directorClosureBlockedByV0(run)
	if len(closure.BlockedBy) > 0 {
		closure.Status = DirectorClosureStatusBlockedV0
		closure.Blocked = true
		return closure
	}
	closure.Status = DirectorClosureStatusReadyV0
	closure.Ready = true
	return closure
}

func directorClosureBlockedByV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	blockedBy := make([]string, 0, 6)
	if len(compactStringsV0(run.FunctionContracts)) == 0 {
		blockedBy = append(blockedBy, DirectorClosureBlockedByContratosV0)
	}
	if !directorProgrammingDeliveriesSatisfiedV0(run) {
		blockedBy = append(blockedBy, DirectorClosureBlockedByProgramacionEntregasV0)
	}
	if len(compactStringsV0(run.AcceptedReviews)) == 0 {
		blockedBy = append(blockedBy, DirectorClosureBlockedByRevisionFinalV0)
	}
	if len(compactStringsV0(run.Validations)) == 0 {
		blockedBy = append(blockedBy, DirectorClosureBlockedByValidacionFinalV0)
	}
	if !directorClosurePhaseActiveV0(run) {
		blockedBy = append(blockedBy, DirectorClosureBlockedByFaseCierreV0)
	}
	if len(compactStringsV0(run.Blockers)) > 0 {
		blockedBy = append(blockedBy, DirectorClosureBlockedByRunBlockersV0)
	}
	return blockedBy
}

func directorRunClosureClosedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if strings.TrimSpace(string(run.Status)) == string(orquestacoreworkflow.OrchestrationRunStatusClosedV0) {
		return true
	}
	return len(compactStringsV0(run.Closures)) > 0
}

func directorProgrammingDeliveriesSatisfiedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if len(compactStringsV0(run.Deliveries)) == 0 {
		return false
	}
	return countOpenDirectorTasksV0(run.Tasks, run.ClosedTasks) == 0
}

func directorClosurePhaseActiveV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if strings.TrimSpace(string(run.CurrentPhase)) != string(orquestacoreworkflow.OrchestrationPhaseCierreV0) {
		return false
	}
	for _, phase := range run.Phases {
		if strings.TrimSpace(string(phase.ID)) != string(orquestacoreworkflow.OrchestrationPhaseCierreV0) {
			continue
		}
		return strings.TrimSpace(string(phase.Status)) == string(orquestacoreworkflow.OrchestrationPhaseStatusActiveV0)
	}
	return false
}
