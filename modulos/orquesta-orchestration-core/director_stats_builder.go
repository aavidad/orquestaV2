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
		StopControl:   buildDirectorRunStopControlV0(run),
		Phases:        buildDirectorPhaseStatsV0(run.Phases),
		Agents:        buildDirectorAgentStatsV0(run),
	}
	applyDirectorAgentStopReasonsV0(&stats, run)
	applyDirectorAgentProgressV0(&stats, run, nil)
	refreshDirectorControlCountsV0(&stats, false)
	applyDirectorRegisteredProcessProgressV0(&stats)
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
	appendDirectorUnreflectedProcessAgentsV0(ctx, &stats, registry)
	refreshDirectorControlCountsV0(&stats, true)
	applyDirectorRegisteredProcessProgressV0(&stats)
	return stats
}

func appendDirectorUnreflectedProcessAgentsV0(
	ctx context.Context,
	stats *DirectorRunStatsV0,
	registry AgentProcessRegistryPortV0,
) {
	if stats == nil || registry == nil {
		return
	}
	lister, ok := registry.(AgentProcessRegistryListPortV0)
	if !ok {
		return
	}
	records, err := lister.ListAgentProcessesV0(ctx, AgentProcessRegistryListFilterV0{
		RunID: stats.RunRef,
	})
	if err != nil {
		return
	}
	known := map[string]bool{}
	for _, agent := range stats.Agents {
		agentRef := strings.TrimSpace(agent.AgentRequestID)
		if agentRef != "" {
			known[agentRef] = true
		}
	}
	for _, record := range records {
		agentRef := strings.TrimSpace(record.AgentRequestID)
		if agentRef == "" || known[agentRef] {
			continue
		}
		process := directorProcessStatsFromRecordV0(record)
		stats.Agents = append(stats.Agents, DirectorAgentStatsV0{
			AgentRequestID:    agentRef,
			Status:            DirectorAgentStatusRunningV0,
			Requested:         true,
			Started:           true,
			InFlight:          true,
			ControlRegistered: true,
			ControlState:      DirectorAgentControlStateRegisteredV0,
			CanStop:           true,
			Process:           &process,
		})
		known[agentRef] = true
	}
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
	if !request.IncludeAgentUsage {
		return stats
	}
	if usageSource == nil {
		stats.Progress.Issues = append(stats.Progress.Issues, DirectorProgressIssueV0{
			Code:    "agent_usage_source_not_configured",
			Field:   "agent_usage_source",
			Message: "agent_usage_source_not_configured",
		})
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
			Message: "agent_usage_source_error",
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
		TasksDelivered:        len(compactStringsV0(run.DeliveredTasks)),
		Brainstorms:           len(compactStringsV0(run.Brainstorms)),
		Votes:                 len(compactStringsV0(run.Votes)),
		FunctionContracts:     len(compactStringsV0(run.FunctionContracts)),
		Decisions:             len(compactStringsV0(run.Decisions)),
		CapacityRequests:      len(compactStringsV0(run.CapacityRequests)),
		CapacityDecisions:     len(compactStringsV0(run.CapacityDecisions)),
		AgentsRequested:       len(compactStringsV0(run.Agents)),
		AgentsStarted:         len(compactStringsV0(run.StartedAgents)),
		AgentsFailed:          len(compactStringsV0(run.FailedAgents)),
		AgentsLost:            len(compactStringsV0(run.LostAgents)),
		AgentsStopRequested:   len(compactStringsV0(run.StoppedAgents)),
		AgentsStopConfirmed:   len(compactStringsV0(run.ConfirmedStoppedAgents)),
		AgentsDelivered:       len(compactStringsV0(run.DeliveredAgents)),
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
		DeliveredTasks:        compactStringsV0(run.DeliveredTasks),
		Brainstorms:           compactStringsV0(run.Brainstorms),
		Votes:                 compactStringsV0(run.Votes),
		FunctionContracts:     compactStringsV0(run.FunctionContracts),
		Decisions:             compactStringsV0(run.Decisions),
		CapacityRequests:      compactStringsV0(run.CapacityRequests),
		CapacityDecisions:     compactStringsV0(run.CapacityDecisions),
		AgentsRequested:       compactStringsV0(run.Agents),
		AgentsStarted:         compactStringsV0(run.StartedAgents),
		AgentsFailed:          compactStringsV0(run.FailedAgents),
		AgentsLost:            compactStringsV0(run.LostAgents),
		AgentsStopRequested:   compactStringsV0(run.StoppedAgents),
		AgentStopRequests:     compactStringsV0(run.AgentStopRequests),
		AgentsStopConfirmed:   compactStringsV0(run.ConfirmedStoppedAgents),
		AgentsDelivered:       compactStringsV0(run.DeliveredAgents),
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

func buildDirectorRunStopControlV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) DirectorRunStopControlV0 {
	started := compactStringsV0(run.StartedAgents)
	stopRequested := compactStringsV0(run.StoppedAgents)
	stopConfirmed := compactStringsV0(run.ConfirmedStoppedAgents)
	pending := pendingDirectorRunStopAgentRefsV0(run)
	allRequestedConfirmed := directorStopRequestedAllConfirmedV0(stopRequested, stopConfirmed)
	stats := DirectorRunStopControlV0{
		Status:                 DirectorRunStopStatusNoneV0,
		Propagated:             len(stopRequested) > 0,
		Pending:                len(pending) > 0,
		Confirmed:              allRequestedConfirmed,
		StartedAgents:          len(started),
		StopRequestedAgents:    len(stopRequested),
		StopConfirmedAgents:    len(stopConfirmed),
		StopPendingAgents:      len(pending),
		PendingAgentRefs:       pending,
		StopRequestedAgentRefs: stopRequested,
		StopConfirmedAgentRefs: stopConfirmed,
	}
	stats.Requested = stats.Propagated || stats.Confirmed
	switch {
	case stats.Pending:
		stats.Status = DirectorRunStopStatusPendingV0
	case stats.Confirmed:
		stats.Status = DirectorRunStopStatusConfirmedV0
	case stats.Propagated:
		stats.Status = DirectorRunStopStatusPropagatedV0
	}
	return stats
}

func pendingDirectorRunStopAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	stopRequested := autonomousStringSetV0(run.StoppedAgents)
	terminal := autonomousStringSetV0(compactStringsV0(append(
		append(append([]string{}, run.ConfirmedStoppedAgents...), run.DeliveredAgents...),
		append(run.FailedAgents, run.LostAgents...)...,
	)))
	pending := make([]string, 0, len(run.StartedAgents))
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if !stopRequested[agentRef] || terminal[agentRef] {
			continue
		}
		pending = append(pending, agentRef)
	}
	return pending
}

func directorStopRequestedAllConfirmedV0(
	stopRequested []string,
	stopConfirmed []string,
) bool {
	stopRequested = compactStringsV0(stopRequested)
	if len(stopRequested) == 0 {
		return false
	}
	confirmed := autonomousStringSetV0(stopConfirmed)
	for _, agentRef := range stopRequested {
		if !confirmed[strings.TrimSpace(agentRef)] {
			return false
		}
	}
	return true
}
