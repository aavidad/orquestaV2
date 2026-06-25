package orquestaservershutdown

import (
	"context"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func shouldRunShutdownSupervisorV0(
	runs []ServerShutdownRunResultV0,
	statsAvailable bool,
) bool {
	for _, run := range runs {
		if run.CheckpointRequired || run.Terminal {
			continue
		}
		if !statsAvailable || shutdownRunWaitingAgentsV0(run) > 0 {
			return true
		}
	}
	return false
}

func supervisorCommandV0(
	command ServerShutdownCommandV0,
) orquestarunsupervisor.RunSupervisorCommandV0 {
	return orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          command.QueueRef,
		AppRefs:           append([]string(nil), command.AppRefs...),
		QueueLimit:        command.QueueLimit,
		MaxRunsPerTick:    command.MaxRunsPerTick,
		MaxTicks:          command.MaxTicks,
		MaxExecutions:     command.MaxExecutions,
		StopOnNoExecution: true,
		AllowRepeatedRuns: true,
		OccurredAt:        command.OccurredAt,
		CorrelationID:     command.CorrelationID,
	}
}

func refreshShutdownRunsV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	runs []ServerShutdownRunResultV0,
) ([]ServerShutdownRunResultV0, error) {
	out := make([]ServerShutdownRunResultV0, 0, len(runs))
	for _, run := range runs {
		state, err := readShutdownControlStateV0(ctx, deps.RunControlReader, run.RunRef)
		if err != nil {
			return nil, err
		}
		run = applyShutdownControlStateV0(run, state)
		if deps.StatsReader != nil {
			stats, err := deps.StatsReader.ReadRunShutdownStatsV0(
				ctx,
				RunShutdownStatsRequestV0{
					RunRef:        run.RunRef,
					CorrelationID: command.CorrelationID,
					EvidenceRefs:  command.EvidenceRefs,
				},
			)
			if err != nil {
				return nil, err
			}
			run = applyShutdownStatsV0(run, stats)
		}
		out = append(out, run)
	}
	return out, nil
}
