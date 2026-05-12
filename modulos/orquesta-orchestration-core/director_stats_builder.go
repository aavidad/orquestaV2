package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func BuildDirectorRunStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) DirectorRunStatsV0 {
	stats := DirectorRunStatsV0{
		SchemaVersion: DirectorRunStatsSchemaVersionV0,
		RunRef:        strings.TrimSpace(run.RunID),
		ProjectRef:    strings.TrimSpace(run.ProjectRef),
		AppSpecRef:    strings.TrimSpace(run.AppSpecRef),
		Status:        strings.TrimSpace(string(run.Status)),
		CurrentPhase:  strings.TrimSpace(string(run.CurrentPhase)),
		Counts:        buildDirectorRunCountsV0(run),
		Refs:          buildDirectorRunRefsV0(run),
		Progress:      BuildDirectorProgressStatsV0(run, nil, nil),
		Closure:       buildDirectorClosureStatsV0(run),
		Phases:        buildDirectorPhaseStatsV0(run.Phases),
		Agents:        buildDirectorAgentStatsV0(run),
	}
	applyDirectorAgentProgressV0(&stats, run, nil)
	refreshDirectorControlCountsV0(&stats, false)
	return stats
}

func BuildDirectorRunStatsWithProcessRegistryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	registry AgentProcessRegistryPortV0,
) DirectorRunStatsV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	stats := BuildDirectorRunStatsV0(run)
	if registry == nil {
		return stats
	}
	for index := range stats.Agents {
		if err := ctx.Err(); err != nil {
			break
		}
		record, err := registry.ResolveAgentProcessV0(
			ctx,
			stats.RunRef,
			stats.Agents[index].AgentRequestID,
		)
		if err != nil {
			continue
		}
		process := directorProcessStatsFromRecordV0(record)
		stats.Agents[index].Process = &process
		stats.Agents[index].ControlRegistered = true
	}
	refreshDirectorControlCountsV0(&stats, true)
	return stats
}

func BuildDirectorRunStatsWithObservationsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []AgentProgressObservationV0,
	issues []DirectorProgressIssueV0,
) DirectorRunStatsV0 {
	stats := BuildDirectorRunStatsV0(run)
	ApplyDirectorProgressObservationsV0(&stats, observations, issues)
	return stats
}

func BuildDirectorRunStatsWithPortsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	registry AgentProcessRegistryPortV0,
	progressSource AgentProgressObservationProviderPortV0,
	request DirectorProgressSourceRequestV0,
) DirectorRunStatsV0 {
	stats := BuildDirectorRunStatsWithProcessRegistryV0(ctx, run, registry)
	observations, issues := loadDirectorProgressObservationsV0(
		ctx,
		run,
		progressSource,
		request,
	)
	ApplyDirectorProgressObservationsV0(&stats, observations, issues)
	return stats
}

func BuildDirectorRunStatsWithTelemetryPortsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	registry AgentProcessRegistryPortV0,
	progressSource AgentProgressObservationProviderPortV0,
	usageSource AgentUsageStatsProviderPortV0,
	request DirectorProgressSourceRequestV0,
) DirectorRunStatsV0 {
	stats := BuildDirectorRunStatsWithPortsV0(ctx, run, registry, progressSource, request)
	if usageSource == nil {
		return stats
	}
	usage, err := usageSource.BuildAgentUsageStatsV0(ctx, AgentUsageStatsRequestV0{
		Run:           run,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  request.EvidenceRefs,
	})
	if err != nil {
		stats.Progress.Issues = append(stats.Progress.Issues, DirectorProgressIssueV0{
			Code:    "agent_usage_source_error",
			Field:   "agent_usage_source",
			Message: err.Error(),
		})
		return stats
	}
	ApplyDirectorAgentUsageStatsV0(&stats, usage)
	return stats
}

func BuildAutonomousDirectorLoopStatsV0(
	decision AutonomousDirectorDecisionV0,
	loop ProgressiveLoopResultV0,
) AutonomousDirectorLoopStatsV0 {
	return AutonomousDirectorLoopStatsV0{
		SchemaVersion: DirectorLoopStatsSchemaVersionV0,
		Run:           BuildDirectorRunStatsV0(loop.Run),
		Decision:      decision,
		Loop: DirectorProgressiveLoopStatsV0{
			Status:              strings.TrimSpace(string(loop.Status)),
			Bursts:              len(loop.Bursts),
			Dispatches:          len(loop.Dispatches),
			BatchDispatches:     len(loop.BatchDispatches),
			TotalExecutedSteps:  loop.TotalExecutedSteps,
			FirstPendingCount:   loop.FirstPendingCount,
			PendingOutboxCount:  loop.PendingOutboxCount,
			RecommendedCapacity: decision.RecommendedCapacity,
		},
	}
}

func buildDirectorRunCountsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) DirectorRunStatsCountsV0 {
	tasks := compactStringsV0(run.Tasks)
	closed := compactStringsV0(run.ClosedTasks)
	return DirectorRunStatsCountsV0{
		TasksTotal:            len(tasks),
		TasksClosed:           len(closed),
		TasksOpen:             countOpenDirectorTasksV0(tasks, closed),
		Brainstorms:           len(compactStringsV0(run.Brainstorms)),
		Votes:                 len(compactStringsV0(run.Votes)),
		FunctionContracts:     len(compactStringsV0(run.FunctionContracts)),
		Decisions:             len(compactStringsV0(run.Decisions)),
		CapacityRequests:      len(compactStringsV0(run.CapacityRequests)),
		CapacityDecisions:     len(compactStringsV0(run.CapacityDecisions)),
		AgentsRequested:       len(compactStringsV0(run.Agents)),
		AgentsStarted:         len(compactStringsV0(run.StartedAgents)),
		AgentsFailed:          len(compactStringsV0(run.FailedAgents)),
		AgentsStopRequested:   len(compactStringsV0(run.StoppedAgents)),
		AgentsStopConfirmed:   len(compactStringsV0(run.ConfirmedStoppedAgents)),
		AgentAssessments:      len(compactStringsV0(run.AgentAssessments)),
		AgentLeaseExpirations: len(compactStringsV0(run.AgentLeaseExpirations)),
		ConcurrencyGates:      len(compactStringsV0(run.ConcurrencyGates)),
		QualityGates:          len(compactStringsV0(run.QualityGates)),
		PhaseArtifacts:        len(compactStringsV0(run.PhaseArtifacts)),
		Deliveries:            len(compactStringsV0(run.Deliveries)),
		Reviews:               len(compactStringsV0(run.Reviews)),
		ReviewResults:         len(compactStringsV0(run.ReviewResults)),
		ReworkRequests:        len(compactStringsV0(run.ReworkRequests)),
		ReplanDecisions:       len(compactStringsV0(run.ReplanDecisions)),
		AcceptedReviews:       len(compactStringsV0(run.AcceptedReviews)),
		Validations:           len(compactStringsV0(run.Validations)),
		Closures:              len(compactStringsV0(run.Closures)),
		DirectorQuestions:     len(compactStringsV0(run.DirectorQuestions)),
		DirectorAnswers:       len(compactStringsV0(run.DirectorAnswers)),
		Blockers:              len(compactStringsV0(run.Blockers)),
		CommandEffects:        len(compactStringsV0(run.CommandEffects)),
	}
}

func buildDirectorRunRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) DirectorRunStatsRefsV0 {
	return DirectorRunStatsRefsV0{
		OpenTasks:             openDirectorTaskRefsV0(run.Tasks, run.ClosedTasks),
		ClosedTasks:           compactStringsV0(run.ClosedTasks),
		Brainstorms:           compactStringsV0(run.Brainstorms),
		Votes:                 compactStringsV0(run.Votes),
		FunctionContracts:     compactStringsV0(run.FunctionContracts),
		Decisions:             compactStringsV0(run.Decisions),
		CapacityRequests:      compactStringsV0(run.CapacityRequests),
		CapacityDecisions:     compactStringsV0(run.CapacityDecisions),
		AgentsRequested:       compactStringsV0(run.Agents),
		AgentsStarted:         compactStringsV0(run.StartedAgents),
		AgentsFailed:          compactStringsV0(run.FailedAgents),
		AgentsStopRequested:   compactStringsV0(run.StoppedAgents),
		AgentsStopConfirmed:   compactStringsV0(run.ConfirmedStoppedAgents),
		AgentAssessments:      compactStringsV0(run.AgentAssessments),
		AgentLeaseExpirations: compactStringsV0(run.AgentLeaseExpirations),
		ConcurrencyGates:      compactStringsV0(run.ConcurrencyGates),
		QualityGates:          compactStringsV0(run.QualityGates),
		PhaseArtifacts:        compactStringsV0(run.PhaseArtifacts),
		Deliveries:            compactStringsV0(run.Deliveries),
		Reviews:               compactStringsV0(run.Reviews),
		ReviewResults:         compactStringsV0(run.ReviewResults),
		ReworkRequests:        compactStringsV0(run.ReworkRequests),
		ReplanDecisions:       compactStringsV0(run.ReplanDecisions),
		AcceptedReviews:       compactStringsV0(run.AcceptedReviews),
		Validations:           compactStringsV0(run.Validations),
		Closures:              compactStringsV0(run.Closures),
		DirectorQuestions:     compactStringsV0(run.DirectorQuestions),
		DirectorAnswers:       compactStringsV0(run.DirectorAnswers),
		DirectorAnswered:      compactStringsV0(run.DirectorAnsweredQuestions),
		Blockers:              compactStringsV0(run.Blockers),
	}
}
