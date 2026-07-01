package orquestaserver

import (
	"strings"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	stopreason "orquesta/modulos/orquesta-run-supervisor/stopreason"
)

type supervisorResultMetricsV0 struct {
	QueueSize    int
	LastTick     int
	Executions   int
	Skips        int
	ResultTicks  int
	StopReason   string
	PublicStop   string
	StopCategory string
}

func collectSupervisorResultMetricsV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) supervisorResultMetricsV0 {
	metrics := supervisorResultMetricsV0{
		Executions:  result.TotalExecutions,
		Skips:       result.TotalSkips,
		ResultTicks: len(result.Ticks),
		StopReason:  result.StopReason,
	}
	if len(result.Ticks) == 0 {
		projection := supervisorResultStopProjectionV0(result, metrics.Executions, metrics.Skips)
		metrics.PublicStop = projection.PublicReason
		metrics.StopCategory = projection.Category
		return metrics
	}
	last := result.Ticks[len(result.Ticks)-1]
	metrics.QueueSize = len(last.Result.Ranked)
	metrics.LastTick = last.TickNumber
	if metrics.LastTick <= 0 {
		metrics.LastTick = len(result.Ticks)
	}
	if metrics.Executions == 0 {
		for _, tick := range result.Ticks {
			metrics.Executions += len(tick.Result.Executions)
		}
	}
	if metrics.Skips == 0 {
		for _, tick := range result.Ticks {
			metrics.Skips += len(tick.Result.Skips)
		}
	}
	projection := supervisorResultStopProjectionV0(result, metrics.Executions, metrics.Skips)
	metrics.PublicStop = projection.PublicReason
	metrics.StopCategory = projection.Category
	return metrics
}

func supervisorResultStopProjectionV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
	executions int,
	skips int,
) stopreason.ProjectionV0 {
	if result.StopProjection.SchemaVersion != "" {
		return result.StopProjection
	}
	return stopreason.ProjectV0(stopreason.ProjectionInputV0{
		Source:     stopreason.SourceRunSupervisorV0,
		StopReason: result.StopReason,
		Executions: executions,
		Skips:      skips,
		Ticks:      len(result.Ticks),
		ErrorRefs:  result.ErrorRunRefs,
	})
}

func supervisorCommandAuditSummaryV0(
	command orquestarunsupervisor.RunSupervisorCommandV0,
) map[string]interface{} {
	return map[string]interface{}{
		"queue_ref":            strings.TrimSpace(command.QueueRef),
		"app_refs_count":       len(command.AppRefs),
		"queue_limit":          command.QueueLimit,
		"max_runs_per_tick":    command.MaxRunsPerTick,
		"max_ticks":            command.MaxTicks,
		"max_executions":       command.MaxExecutions,
		"stop_on_no_execution": command.StopOnNoExecution,
		"allow_repeated_runs":  command.AllowRepeatedRuns,
	}
}

func supervisorResultAuditSummaryV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) map[string]interface{} {
	metrics := collectSupervisorResultMetricsV0(result)
	return map[string]interface{}{
		"stop_reason":          strings.TrimSpace(metrics.StopReason),
		"public_stop_reason":   strings.TrimSpace(metrics.PublicStop),
		"stop_category":        strings.TrimSpace(metrics.StopCategory),
		"total_executions":     metrics.Executions,
		"total_skips":          metrics.Skips,
		"result_ticks":         metrics.ResultTicks,
		"last_tick":            metrics.LastTick,
		"queue_size":           metrics.QueueSize,
		"error_tick_number":    result.ErrorTickNumber,
		"error_run_refs_count": len(result.ErrorRunRefs),
	}
}

func supervisorResultAuditSummaryWithStateV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
	state StateV0,
) map[string]interface{} {
	summary := supervisorResultAuditSummaryV0(result)
	metrics := collectSupervisorResultMetricsV0(result)
	projection := supervisorPublicProjectionV0(result, metrics)
	projection, snapshot := supervisorProjectionWithGoalBackendV0(projection, metrics, state)
	summary["public_stop_reason"] = strings.TrimSpace(projection.StopPublic)
	summary["stop_category"] = strings.TrimSpace(projection.StopCategory)
	if supervisorGoalBackendSnapshotActiveV0(snapshot) {
		summary["goal_backend_active"] = snapshot.Active
		summary["goal_backend_run_refs_count"] = len(snapshot.RunRefs)
		summary["goal_backend_goal_refs_count"] = len(snapshot.GoalRefs)
	}
	return summary
}
