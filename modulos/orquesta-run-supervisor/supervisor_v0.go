package orquestarunsupervisor

import (
	"context"
	"errors"
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	stopreason "orquesta/modulos/orquesta-run-supervisor/stopreason"
)

var ErrRunSupervisorTickerRequiredV0 = errors.New("run_supervisor: ticker required")

func SuperviseRunsV0(
	ctx context.Context,
	deps RunSupervisorDepsV0,
	command RunSupervisorCommandV0,
) (RunSupervisorResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeRunSupervisorCommandV0(command)
	result := RunSupervisorResultV0{}
	if deps.Ticker == nil {
		result.StopReason = RunSupervisorStopNoTickerV0
		result.StopProjection = runSupervisorStopProjectionV0(result)
		return result, ErrRunSupervisorTickerRequiredV0
	}
	excluded := compactRunSupervisorStringsV0(nil)
	for tickNumber := 1; tickNumber <= command.MaxTicks; tickNumber++ {
		if err := ctx.Err(); err != nil {
			result.StopReason = RunSupervisorStopContextDoneV0
			result.StopProjection = runSupervisorStopProjectionV0(result)
			return result, err
		}
		if command.MaxExecutions > 0 && result.TotalExecutions >= command.MaxExecutions {
			result.StopReason = RunSupervisorStopMaxExecutionsV0
			result.StopProjection = runSupervisorStopProjectionV0(result)
			return result, nil
		}
		tickCommand := coordinatorCommandV0(command, excluded, result.TotalExecutions)
		tickResult, err := deps.Ticker.RunGlobalTickV0(ctx, tickCommand)
		summary := RunSupervisorTickSummaryV0{
			TickNumber: tickNumber,
			Excluded:   append([]string(nil), excluded...),
			Result:     tickResult,
		}
		result.Ticks = append(result.Ticks, summary)
		result.TotalExecutions += len(tickResult.Executions)
		result.TotalSkips += len(tickResult.Skips)
		if err != nil {
			result.StopReason = RunSupervisorStopTickErrorV0
			result.Error = strings.TrimSpace(err.Error())
			result.ErrorTickNumber = tickNumber
			result.ErrorRunRefs = runSupervisorTickRunRefsV0(tickResult)
			result.Diagnostics = runSupervisorTickErrorDiagnosticsV0(tickNumber, tickResult, err)
			result.StopProjection = runSupervisorStopProjectionV0(result)
			return result, err
		}
		if !command.AllowRepeatedRuns {
			excluded = appendExecutedRunRefsV0(excluded, tickResult.Executions)
		}
		if command.MaxExecutions > 0 && result.TotalExecutions >= command.MaxExecutions {
			result.StopReason = RunSupervisorStopMaxExecutionsV0
			result.StopProjection = runSupervisorStopProjectionV0(result)
			return result, nil
		}
		if len(tickResult.Executions) == 0 && command.StopOnNoExecution {
			result.StopReason = RunSupervisorStopNoExecutionV0
			result.StopProjection = runSupervisorStopProjectionV0(result)
			return result, nil
		}
	}
	result.StopReason = RunSupervisorStopMaxTicksV0
	result.StopProjection = runSupervisorStopProjectionV0(result)
	return result, nil
}

func runSupervisorStopProjectionV0(result RunSupervisorResultV0) stopreason.ProjectionV0 {
	return stopreason.ProjectV0(stopreason.ProjectionInputV0{
		Source:     stopreason.SourceRunSupervisorV0,
		StopReason: result.StopReason,
		Executions: result.TotalExecutions,
		Skips:      result.TotalSkips,
		Ticks:      len(result.Ticks),
		ErrorRefs:  result.ErrorRunRefs,
	})
}

func runSupervisorTickErrorDiagnosticsV0(
	tickNumber int,
	tickResult orquestaruncoordinator.RunCoordinatorTickResultV0,
	err error,
) []orquestaruncoordinator.RunDrainDiagnosticV0 {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	diagnostics := []orquestaruncoordinator.RunDrainDiagnosticV0{}
	for _, execution := range tickResult.Executions {
		diagnostics = append(diagnostics, execution.Diagnostics...)
	}
	if len(diagnostics) == 0 || message != "" {
		for _, runRef := range runSupervisorTickRunRefsV0(tickResult) {
			diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
				Kind:          "supervisor_tick_error",
				Status:        "error",
				RunRef:        runRef,
				Error:         message,
				AttemptNumber: tickNumber,
			})
		}
	}
	if len(diagnostics) == 0 && message != "" {
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "supervisor_tick_error",
			Status:        "error",
			Error:         message,
			AttemptNumber: tickNumber,
		})
	}
	return diagnostics
}

func runSupervisorTickRunRefsV0(
	tickResult orquestaruncoordinator.RunCoordinatorTickResultV0,
) []string {
	refs := []string{}
	for _, execution := range tickResult.Executions {
		refs = append(refs, execution.RunRef)
	}
	for _, skip := range tickResult.Skips {
		refs = append(refs, skip.RunRef)
	}
	return compactRunSupervisorStringsV0(refs)
}

func normalizeRunSupervisorCommandV0(command RunSupervisorCommandV0) RunSupervisorCommandV0 {
	command.QueueRef = strings.TrimSpace(command.QueueRef)
	command.AppRefs = compactRunSupervisorStringsV0(command.AppRefs)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	if command.MaxTicks <= 0 {
		command.MaxTicks = 1
	}
	if command.MaxRunsPerTick <= 0 {
		command.MaxRunsPerTick = 1
	}
	if command.QueueLimit < 0 {
		command.QueueLimit = 0
	}
	if command.MaxExecutions < 0 {
		command.MaxExecutions = 0
	}
	return command
}

func coordinatorCommandV0(
	command RunSupervisorCommandV0,
	excluded []string,
	totalExecutions int,
) orquestaruncoordinator.RunCoordinatorTickCommandV0 {
	maxRuns := command.MaxRunsPerTick
	if command.MaxExecutions > 0 {
		remaining := command.MaxExecutions - totalExecutions
		if remaining < maxRuns {
			maxRuns = remaining
		}
	}
	return orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef:       command.QueueRef,
		AppRefs:        append([]string(nil), command.AppRefs...),
		ExcludeRunRefs: append([]string(nil), excluded...),
		QueueLimit:     command.QueueLimit,
		MaxRuns:        maxRuns,
		OccurredAt:     command.OccurredAt,
		CorrelationID:  command.CorrelationID,
		DrainLimits:    command.DrainLimits,
		RankingPolicy:  command.RankingPolicy,
	}
}

func appendExecutedRunRefsV0(
	excluded []string,
	executions []orquestaruncoordinator.RunExecutionSummaryV0,
) []string {
	for _, execution := range executions {
		excluded = append(excluded, execution.RunRef)
	}
	return compactRunSupervisorStringsV0(excluded)
}
