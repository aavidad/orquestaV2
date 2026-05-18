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

func applyDirectorAgentProgressV0(
	stats *DirectorRunStatsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
) {
	if stats == nil {
		return
	}
	applyDirectorAgentAssessmentProgressV0(stats, run)
	if len(observations) == 0 {
		return
	}
	byAgent := latestDirectorProgressByAgentV0(observations)
	reflected := reflectedDirectorAgentSetV0(run)
	for index := range stats.Agents {
		observation, ok := byAgent[stats.Agents[index].AgentRequestID]
		if !ok {
			continue
		}
		if directorProgressObservationResolvedV0(run, observation, reflected) {
			continue
		}
		progress := directorAgentProgressFromObservationV0(observation)
		stats.Agents[index].LastProgress = &progress
		if progress.Status == string(orquestaruntime.AgentLoopDetectedV0) ||
			progress.DecisionRequired {
			stats.Agents[index].NeedsAttention = true
		}
		if protectedDirectionAgentObservationV0(run, observation) &&
			progress.Status != string(orquestaruntime.AgentLoopDetectedV0) {
			stats.Agents[index].CanStop = false
		}
	}
}

func applyDirectorAgentAssessmentProgressV0(
	stats *DirectorRunStatsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	byAgent := latestDirectorAssessmentByAgentV0(run.AgentAssessments)
	if len(byAgent) == 0 {
		return
	}
	reflected := reflectedDirectorAgentSetV0(run)
	for index := range stats.Agents {
		assessment, ok := byAgent[stats.Agents[index].AgentRequestID]
		if !ok {
			continue
		}
		if directorAssessmentProgressResolvedV0(run, assessment, reflected) {
			continue
		}
		progress := directorAgentProgressFromAssessmentV0(assessment)
		stats.Agents[index].LastProgress = &progress
		if progress.Status == string(orquestaruntime.AgentLoopDetectedV0) ||
			progress.DecisionRequired {
			stats.Agents[index].NeedsAttention = true
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

func directorAgentProgressFromAssessmentV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) DirectorAgentProgressV0 {
	return DirectorAgentProgressV0{
		TaskRef:     strings.TrimSpace(assessment.TaskRef),
		DeliveryRef: strings.TrimSpace(assessment.DeliveryRef),
		ReportRef:   strings.TrimSpace(assessment.AssessmentRef),
		Status:      directorAssessmentProgressStatusV0(assessment),
		DirectorProgressTemporalV0: DirectorProgressTemporalV0{
			DecisionRequired: directorAssessmentDecisionRequiredV0(assessment),
		},
	}
}

func directorTaskProgressFromAssessmentV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) DirectorTaskProgressV0 {
	return DirectorTaskProgressV0{
		TaskRef:        strings.TrimSpace(assessment.TaskRef),
		Status:         directorTaskStatusFromAssessmentV0(assessment),
		AgentRequestID: strings.TrimSpace(assessment.AgentRequestID),
		DeliveryRef:    strings.TrimSpace(assessment.DeliveryRef),
		LastReportRef:  strings.TrimSpace(assessment.AssessmentRef),
		ProgressStatus: directorAssessmentProgressStatusV0(assessment),
		DirectorProgressTemporalV0: DirectorProgressTemporalV0{
			DecisionRequired: directorAssessmentDecisionRequiredV0(assessment),
		},
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
			task.Status == DirectorTaskProgressDeliveredV0 {
			count++
		}
	}
	return count
}

func noSignalDirectorAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	byAgent map[string]AgentProgressObservationV0,
	byAssessmentAgent map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) []string {
	missing := make([]string, 0)
	failed := autonomousStringSetV0(run.FailedAgents)
	lost := autonomousStringSetV0(run.LostAgents)
	stopRequested := autonomousStringSetV0(run.StoppedAgents)
	stopConfirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	reflected := reflectedDirectorAgentSetV0(run)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if failed[agentRef] || lost[agentRef] || stopRequested[agentRef] || stopConfirmed[agentRef] || reflected[agentRef] {
			continue
		}
		if _, ok := byAssessmentAgent[agentRef]; ok {
			continue
		}
		if _, ok := byAgent[agentRef]; !ok {
			missing = append(missing, agentRef)
		}
	}
	return missing
}

func latestDirectorAssessmentByAgentV0(
	assessments []string,
) map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	latest := map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}
	for _, value := range assessments {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if !ok {
			continue
		}
		agentRef := strings.TrimSpace(assessment.AgentRequestID)
		if agentRef != "" {
			latest[agentRef] = assessment
		}
	}
	return latest
}

func latestDirectorAssessmentByTaskV0(
	assessments []string,
) map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	latest := map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}
	for _, value := range assessments {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if !ok {
			continue
		}
		taskRef := strings.TrimSpace(assessment.TaskRef)
		if taskRef != "" {
			latest[taskRef] = assessment
		}
	}
	return latest
}

func directorAssessmentProgressStatusV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) string {
	switch strings.TrimSpace(assessment.Verdict) {
	case orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0:
		return string(orquestaruntime.AgentLoopDetectedV0)
	case orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0:
		return string(orquestaruntime.AgentStalledV0)
	}
	if strings.TrimSpace(assessment.Action) == orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
		return string(orquestaruntime.AgentStoppedV0)
	}
	return string(orquestaruntime.AgentProgressingV0)
}

func directorTaskStatusFromAssessmentV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) string {
	switch directorAssessmentProgressStatusV0(assessment) {
	case string(orquestaruntime.AgentLoopDetectedV0):
		return DirectorTaskProgressLoopDetectedV0
	case string(orquestaruntime.AgentStoppedV0):
		return DirectorTaskProgressStoppedV0
	case string(orquestaruntime.AgentStalledV0):
		return DirectorTaskProgressStalledV0
	default:
		return DirectorTaskProgressInProgressV0
	}
}

func directorAssessmentDecisionRequiredV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) bool {
	switch strings.TrimSpace(assessment.Action) {
	case orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
		orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0:
		return true
	default:
		return false
	}
}
