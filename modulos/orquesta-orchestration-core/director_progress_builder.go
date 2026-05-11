package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func BuildDirectorProgressStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
	issues []DirectorProgressIssueV0,
) DirectorProgressStatsV0 {
	tasks := compactStringsV0(run.Tasks)
	closed := autonomousStringSetV0(run.ClosedTasks)
	progress := DirectorProgressStatsV0{
		SourceStatus:    progressSourceStatusV0(observations, issues),
		PercentComplete: directorProgressPercentV0(len(tasks), len(closed)),
		TasksTotal:      len(tasks),
		TasksClosed:     len(closed),
		Issues:          issues,
	}
	progress.Tasks = buildDirectorTaskProgressV0(run, observations)
	progress.TasksObserved = countObservedDirectorTasksV0(progress.Tasks)
	return summarizeDirectorProgressV0(run, observations, progress)
}

func ApplyDirectorProgressObservationsV0(
	stats *DirectorRunStatsV0,
	observations []AgentProgressObservationV0,
	issues []DirectorProgressIssueV0,
) {
	if stats == nil {
		return
	}
	run := runFromDirectorStatsV0(*stats)
	stats.Progress = BuildDirectorProgressStatsV0(run, observations, issues)
	applyDirectorAgentProgressV0(stats, run, observations)
}

func loadDirectorProgressObservationsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	source AgentProgressObservationProviderPortV0,
	request DirectorProgressSourceRequestV0,
) ([]AgentProgressObservationV0, []DirectorProgressIssueV0) {
	if ctx == nil {
		ctx = context.Background()
	}
	if source == nil {
		return nil, nil
	}
	observations, err := source.BuildAgentProgressObservationsV0(ctx, AgentProgressObservationRequestV0{
		Run:           run,
		StepNumber:    request.StepNumber,
		MaxSteps:      request.MaxSteps,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  request.EvidenceRefs,
	})
	if err != nil {
		return nil, []DirectorProgressIssueV0{{
			Code:    "progress_source_error",
			Field:   "progress_source",
			Message: err.Error(),
		}}
	}
	return observations, nil
}

func progressSourceStatusV0(
	observations []AgentProgressObservationV0,
	issues []DirectorProgressIssueV0,
) string {
	if len(issues) > 0 {
		return DirectorProgressSourceErrorV0
	}
	if observations == nil {
		return DirectorProgressSourceNotConfiguredV0
	}
	return DirectorProgressSourceLoadedV0
}

func directorProgressPercentV0(total int, closed int) int {
	if total <= 0 {
		return 100
	}
	if closed < 0 {
		closed = 0
	}
	if closed > total {
		closed = total
	}
	return (closed * 100) / total
}

func runFromDirectorStatsV0(
	stats DirectorRunStatsV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:                  stats.RunRef,
		ProjectRef:             stats.ProjectRef,
		AppSpecRef:             stats.AppSpecRef,
		Status:                 orquestacoreworkflow.OrchestrationRunStatusV0(stats.Status),
		CurrentPhase:           orquestacoreworkflow.OrchestrationPhaseIDV0(stats.CurrentPhase),
		Tasks:                  append(append([]string{}, stats.Refs.OpenTasks...), stats.Refs.ClosedTasks...),
		ClosedTasks:            append([]string{}, stats.Refs.ClosedTasks...),
		Agents:                 append([]string{}, stats.Refs.AgentsRequested...),
		StartedAgents:          append([]string{}, stats.Refs.AgentsStarted...),
		FailedAgents:           append([]string{}, stats.Refs.AgentsFailed...),
		StoppedAgents:          append([]string{}, stats.Refs.AgentsStopRequested...),
		ConfirmedStoppedAgents: append([]string{}, stats.Refs.AgentsStopConfirmed...),
		Deliveries:             append([]string{}, stats.Refs.Deliveries...),
	}
}

func latestDirectorProgressByAgentV0(
	observations []AgentProgressObservationV0,
) map[string]AgentProgressObservationV0 {
	latest := map[string]AgentProgressObservationV0{}
	for _, observation := range observations {
		agentRef := strings.TrimSpace(observation.Report.AgentRequestID)
		if agentRef != "" {
			latest[agentRef] = observation
		}
	}
	return latest
}

func directorTaskStatusFromReportV0(
	report orquestaruntime.AgentProgressReportV0,
) string {
	switch report.Status {
	case orquestaruntime.AgentStalledV0:
		return DirectorTaskProgressStalledV0
	case orquestaruntime.AgentLoopDetectedV0:
		return DirectorTaskProgressLoopDetectedV0
	case orquestaruntime.AgentStoppedV0:
		return DirectorTaskProgressStoppedV0
	default:
		return DirectorTaskProgressInProgressV0
	}
}
