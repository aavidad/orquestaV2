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
	delivered := autonomousStringSetV0(run.DeliveredTasks)
	liveAgents := liveDirectorTaskAgentsV0(run)
	runTaskSet := autonomousStringSetV0(run.Tasks)
	reflectedAgents := reflectedDirectorAgentSetV0(run)
	byTask := latestDirectorProgressByTaskV0(observations)
	byAssessedTask := latestDirectorAssessmentByTaskV0(run.AgentAssessments)
	tasks := compactStringsV0(append(append(run.Tasks, observedTaskRefsV0(observations)...), assessedTaskRefsV0(run.AgentAssessments)...))
	out := make([]DirectorTaskProgressV0, 0, len(tasks))
	for _, taskRef := range tasks {
		observation, observed := byTask[taskRef]
		if observed && directorTaskProgressReflectedAgentOnlyV0(taskRef, runTaskSet, reflectedAgents, observation.Report.AgentRequestID) {
			continue
		}
		task := DirectorTaskProgressV0{TaskRef: taskRef, Status: DirectorTaskProgressPendingV0}
		if observed {
			task = directorTaskProgressFromObservationV0(taskRef, observation)
		} else if assessment, assessed := byAssessedTask[taskRef]; assessed {
			if directorTaskProgressReflectedAgentOnlyV0(taskRef, runTaskSet, reflectedAgents, assessment.AgentRequestID) {
				continue
			}
			task = directorTaskProgressFromAssessmentV0(assessment)
		} else if agentRef := liveAgents[taskRef]; agentRef != "" {
			task = liveDirectorTaskProgressV0(taskRef, agentRef)
		}
		if delivered[taskRef] {
			task.Status = DirectorTaskProgressDeliveredV0
			if task.AgentRequestID == "" {
				task.AgentRequestID = WorkflowTaskAgentRequestRefV0(taskRef)
			}
			task.LastReportRef = ""
			task.ProgressStatus = ""
			task.NoProgressTicks = 0
			task.RepeatedActionCount = 0
			task.DecisionRequired = false
		}
		if closed[taskRef] {
			task.Status = DirectorTaskProgressClosedV0
			task.DecisionRequired = false
		}
		out = append(out, task)
	}
	return out
}

func directorTaskProgressReflectedAgentOnlyV0(
	taskRef string,
	runTaskSet map[string]bool,
	reflectedAgents map[string]bool,
	agentRef string,
) bool {
	return !runTaskSet[strings.TrimSpace(taskRef)] &&
		reflectedAgents[strings.TrimSpace(agentRef)]
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
	byAssessmentAgent := latestDirectorAssessmentByAgentV0(run.AgentAssessments)
	reflected := reflectedDirectorAgentSetV0(run)
	progress.ObservedAgents = len(byAgent)
	for _, observation := range byAgent {
		if directorProgressObservationResolvedV0(run, observation, reflected) {
			progress.ObservedAgents--
			continue
		}
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
	for agentRef, assessment := range byAssessmentAgent {
		if _, observed := byAgent[agentRef]; observed {
			continue
		}
		if directorAssessmentProgressResolvedV0(run, assessment, reflected) {
			continue
		}
		progress.ObservedAgents++
		switch directorAssessmentProgressStatusV0(assessment) {
		case string(orquestaruntime.AgentLoopDetectedV0):
			progress.LoopDetectedAgents++
		case string(orquestaruntime.AgentStoppedV0):
			progress.StoppedAgents++
		case string(orquestaruntime.AgentStalledV0):
			progress.StalledAgents++
		default:
			progress.ProgressingAgents++
		}
	}
	if observations != nil {
		progress.NoSignalAgentRefs = noSignalDirectorAgentRefsV0(run, byAgent, byAssessmentAgent)
	}
	return progress
}

func directorProgressObservationResolvedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation AgentProgressObservationV0,
	reflectedAgents map[string]bool,
) bool {
	agentRef := strings.TrimSpace(observation.Report.AgentRequestID)
	if reflectedAgents[agentRef] {
		return true
	}
	return directorProgressTaskResolvedV0(
		run,
		observation.TaskRef,
	)
}

func directorAssessmentProgressResolvedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	reflectedAgents map[string]bool,
) bool {
	if reflectedAgents[strings.TrimSpace(assessment.AgentRequestID)] {
		return true
	}
	return directorProgressTaskResolvedV0(
		run,
		assessment.TaskRef,
	)
}

func directorProgressTaskResolvedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	refs := append(append([]string{}, run.DeliveredTasks...), run.ClosedTasks...)
	resolvedTasks := autonomousStringSetV0(refs)
	return resolvedTasks[strings.TrimSpace(taskRef)]
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

func assessedTaskRefsV0(assessments []string) []string {
	refs := make([]string, 0, len(assessments))
	for _, value := range assessments {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if ok {
			refs = append(refs, assessment.TaskRef)
		}
	}
	return compactStringsV0(refs)
}

func countObservedDirectorTasksV0(tasks []DirectorTaskProgressV0) int {
	count := 0
	for _, task := range tasks {
		if task.LastReportRef != "" ||
			task.Status == DirectorTaskProgressDeliveredV0 ||
			task.ProgressStatus == DirectorTaskProgressProcessRegisteredV0 {
			count++
		}
	}
	return count
}
