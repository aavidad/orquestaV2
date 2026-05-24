package orquestaserver

import orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"

type supervisorResultMetricsV0 struct {
	QueueSize   int
	LastTick    int
	Executions  int
	Skips       int
	ResultTicks int
	StopReason  string
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
	return metrics
}
