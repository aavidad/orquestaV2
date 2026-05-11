package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func buildDirectorTaskProgressV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
) []DirectorTaskProgressV0 {
	closed := autonomousStringSetV0(run.ClosedTasks)
	byTask := latestDirectorProgressByTaskV0(observations)
	tasks := compactStringsV0(append(run.Tasks, observedTaskRefsV0(observations)...))
	out := make([]DirectorTaskProgressV0, 0, len(tasks))
	for _, taskRef := range tasks {
		observation, observed := byTask[taskRef]
		task := DirectorTaskProgressV0{TaskRef: taskRef, Status: DirectorTaskProgressPendingV0}
		if observed {
			task = directorTaskProgressFromObservationV0(taskRef, observation)
		}
		if closed[taskRef] {
			task.Status = DirectorTaskProgressClosedV0
		}
		out = append(out, task)
	}
	return out
}

func directorTaskProgressFromObservationV0(
	taskRef string,
	observation AgentProgressObservationV0,
) DirectorTaskProgressV0 {
	report := observation.Report
	return DirectorTaskProgressV0{
		TaskRef:                    strings.TrimSpace(taskRef),
		Status:                     directorTaskStatusFromReportV0(report),
		AgentRequestID:             strings.TrimSpace(report.AgentRequestID),
		DeliveryRef:                strings.TrimSpace(observation.DeliveryRef),
		LastReportRef:              strings.TrimSpace(report.ReportID),
		ProgressStatus:             strings.TrimSpace(string(report.Status)),
		NoProgressTicks:            report.NoProgressTicks,
		RepeatedActionCount:        report.RepeatedActionCount,
		DirectorProgressTemporalV0: directorProgressTemporalFromReportV0(report),
		Summary:                    strings.TrimSpace(report.Summary),
		EvidenceRefs:               compactStringsV0(append(observation.EvidenceRefs, report.EvidenceRefs...)),
	}
}

func summarizeDirectorProgressV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
	progress DirectorProgressStatsV0,
) DirectorProgressStatsV0 {
	byAgent := latestDirectorProgressByAgentV0(observations)
	progress.ObservedAgents = len(byAgent)
	for _, observation := range byAgent {
		progress = countDirectorBudgetClassificationV0(observation, progress)
		switch observation.Report.Status {
		case orquestaruntime.AgentStalledV0:
			progress.StalledAgents++
		case orquestaruntime.AgentLoopDetectedV0:
			progress.LoopDetectedAgents++
		case orquestaruntime.AgentStoppedV0:
			progress.StoppedAgents++
		default:
			progress.ProgressingAgents++
		}
	}
	if observations != nil {
		progress.NoSignalAgentRefs = noSignalDirectorAgentRefsV0(run, byAgent)
	}
	return progress
}

func countDirectorBudgetClassificationV0(
	observation AgentProgressObservationV0,
	progress DirectorProgressStatsV0,
) DirectorProgressStatsV0 {
	switch strings.TrimSpace(string(observation.Report.BudgetStatus)) {
	case DirectorProgressClassificationOverBudgetButActiveV0:
		progress.OverBudgetButActiveAgents++
	case DirectorProgressClassificationOverBudgetNoActivityV0:
		progress.OverBudgetNoActivityAgents++
	}
	return progress
}

func applyDirectorAgentProgressV0(
	stats *DirectorRunStatsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
) {
	if stats == nil || len(observations) == 0 {
		return
	}
	byAgent := latestDirectorProgressByAgentV0(observations)
	for index := range stats.Agents {
		observation, ok := byAgent[stats.Agents[index].AgentRequestID]
		if !ok {
			continue
		}
		progress := directorAgentProgressFromObservationV0(observation)
		stats.Agents[index].LastProgress = &progress
		if progress.Status == string(orquestaruntime.AgentStalledV0) ||
			progress.Status == string(orquestaruntime.AgentLoopDetectedV0) ||
			progress.DecisionRequired {
			stats.Agents[index].NeedsAttention = true
		}
		if protectedDirectionAgentObservationV0(run, observation) {
			stats.Agents[index].CanStop = false
		}
	}
}

func directorAgentProgressFromObservationV0(
	observation AgentProgressObservationV0,
) DirectorAgentProgressV0 {
	report := observation.Report
	return DirectorAgentProgressV0{
		TaskRef:                    strings.TrimSpace(observation.TaskRef),
		DeliveryRef:                strings.TrimSpace(observation.DeliveryRef),
		ReportRef:                  strings.TrimSpace(report.ReportID),
		Status:                     strings.TrimSpace(string(report.Status)),
		NoProgressTicks:            report.NoProgressTicks,
		RepeatedActionCount:        report.RepeatedActionCount,
		DirectorProgressTemporalV0: directorProgressTemporalFromReportV0(report),
		Summary:                    strings.TrimSpace(report.Summary),
		EvidenceRefs:               compactStringsV0(append(observation.EvidenceRefs, report.EvidenceRefs...)),
	}
}

func latestDirectorProgressByTaskV0(
	observations []AgentProgressObservationV0,
) map[string]AgentProgressObservationV0 {
	latest := map[string]AgentProgressObservationV0{}
	for _, observation := range observations {
		taskRef := strings.TrimSpace(observation.TaskRef)
		if taskRef != "" {
			latest[taskRef] = observation
		}
	}
	return latest
}

func observedTaskRefsV0(observations []AgentProgressObservationV0) []string {
	refs := make([]string, 0, len(observations))
	for _, observation := range observations {
		refs = append(refs, observation.TaskRef)
	}
	return compactStringsV0(refs)
}

func countObservedDirectorTasksV0(tasks []DirectorTaskProgressV0) int {
	count := 0
	for _, task := range tasks {
		if task.LastReportRef != "" {
			count++
		}
	}
	return count
}

func noSignalDirectorAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	byAgent map[string]AgentProgressObservationV0,
) []string {
	missing := make([]string, 0)
	failed := autonomousStringSetV0(run.FailedAgents)
	stopped := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if failed[agentRef] || stopped[agentRef] {
			continue
		}
		if _, ok := byAgent[agentRef]; !ok {
			missing = append(missing, agentRef)
		}
	}
	return missing
}
