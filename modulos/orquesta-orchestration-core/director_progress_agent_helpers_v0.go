package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

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
		if !ok || directorProgressObservationResolvedV0(run, observation, reflected) {
			continue
		}
		applyDirectorAgentProgressObservationV0(stats, index, observation, run)
	}
}

func applyDirectorAgentProgressObservationV0(
	stats *DirectorRunStatsV0,
	index int,
	observation AgentProgressObservationV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	progress := directorAgentProgressFromObservationV0(observation)
	stats.Agents[index].LastProgress = &progress
	if progress.Status == string(orquestaruntime.AgentStoppedV0) {
		markDirectorAgentStoppedFromProgressV0(stats, index)
	}
	if progress.Status == string(orquestaruntime.AgentLoopDetectedV0) || progress.DecisionRequired {
		stats.Agents[index].NeedsAttention = true
	}
	if protectedDirectionAgentObservationV0(run, observation) &&
		progress.Status != string(orquestaruntime.AgentLoopDetectedV0) {
		stats.Agents[index].CanStop = false
	}
}

func markDirectorAgentStoppedFromProgressV0(stats *DirectorRunStatsV0, index int) {
	wasInFlight := stats.Agents[index].InFlight
	wasControlRegistered := stats.Agents[index].ControlRegistered
	wasControlMissing := stats.Agents[index].ControlState == DirectorAgentControlStateMissingV0
	stats.Agents[index].InFlight = false
	stats.Agents[index].StopConfirmed = true
	stats.Agents[index].Status = DirectorAgentStatusStoppedV0
	stats.Agents[index].ControlState = DirectorAgentControlStateNotNeededV0
	stats.Agents[index].CanStop = false
	decrementDirectorStoppedAgentCountsV0(stats, wasInFlight, wasControlRegistered, wasControlMissing)
}

func decrementDirectorStoppedAgentCountsV0(
	stats *DirectorRunStatsV0,
	wasInFlight bool,
	wasControlRegistered bool,
	wasControlMissing bool,
) {
	if wasInFlight && stats.Counts.AgentsInFlight > 0 {
		stats.Counts.AgentsInFlight--
	}
	if wasControlRegistered && stats.Counts.AgentsControlRegistered > 0 {
		stats.Counts.AgentsControlRegistered--
	}
	if wasControlMissing && stats.Counts.AgentsControlMissing > 0 {
		stats.Counts.AgentsControlMissing--
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
		if !ok || directorAssessmentProgressResolvedV0(run, assessment, reflected) {
			continue
		}
		progress := directorAgentProgressFromAssessmentV0(assessment)
		stats.Agents[index].LastProgress = &progress
		if progress.Status == string(orquestaruntime.AgentLoopDetectedV0) || progress.DecisionRequired {
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
