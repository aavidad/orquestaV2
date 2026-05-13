package orquestaservershutdown

import (
	"context"
	"errors"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func ShutdownServerV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
) (ServerShutdownResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = NormalizeServerShutdownCommandV0(command)
	if deps.QueueReader == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoQueueReaderV0), nil
	}
	if deps.RunControlReader == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoRunControlReaderV0), nil
	}
	if deps.RunControlWriter == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoRunControlWriterV0), nil
	}
	candidates, err := deps.QueueReader.ListRunSchedulingCandidatesV0(
		ctx,
		queueReadRequestV0(command),
	)
	if err != nil {
		return ServerShutdownResultV0{}, err
	}
	targets, err := stopShutdownTargetsV0(ctx, deps, command, candidates)
	if err != nil {
		return ServerShutdownResultV0{}, err
	}
	result := ServerShutdownResultV0{
		SchemaVersion: ServerShutdownSchemaVersionV0,
		Runs:          targets,
		RunsRequested: len(targets),
		EvidenceRefs:  compactServerShutdownStringsV0(command.EvidenceRefs),
	}
	if len(targets) == 0 {
		result.Status = ServerShutdownStatusReadyV0
		result.ShutdownReady = true
		return result, nil
	}
	if shouldRunShutdownSupervisorV0(targets) && deps.Supervisor != nil {
		supervisor, err := deps.Supervisor.RunGlobalSupervisorV0(
			ctx,
			supervisorCommandV0(command),
		)
		if err != nil {
			return ServerShutdownResultV0{}, err
		}
		result.Supervisor = &supervisor
	}
	result.Runs, err = refreshShutdownRunsV0(ctx, deps, command, result.Runs)
	if err != nil {
		return ServerShutdownResultV0{}, err
	}
	return summarizeShutdownResultV0(result), nil
}

func missingServerShutdownDepV0(status string) ServerShutdownResultV0 {
	return ServerShutdownResultV0{
		SchemaVersion: ServerShutdownSchemaVersionV0,
		Status:        status,
	}
}

func queueReadRequestV0(
	command ServerShutdownCommandV0,
) orquestarunqueue.RunQueueReadRequestV0 {
	return orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: command.QueueRef,
		AppRefs:  append([]string(nil), command.AppRefs...),
		Limit:    command.QueueLimit,
	}
}

func stopShutdownTargetsV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) ([]ServerShutdownRunResultV0, error) {
	targets := make([]ServerShutdownRunResultV0, 0, len(candidates))
	for _, candidate := range candidates {
		runRef := strings.TrimSpace(candidate.RunRef)
		if runRef == "" || queueCandidateTerminalV0(candidate) {
			continue
		}
		state, err := readShutdownControlStateV0(ctx, deps.RunControlReader, runRef)
		if err != nil {
			return nil, err
		}
		if orquestaruncontrol.IsTerminalRunControlStatusV0(state.Status) {
			continue
		}
		state, err = deps.RunControlWriter.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
			RunRef:         runRef,
			RequestedBy:    command.RequestedBy,
			Reason:         command.Reason,
			Forced:         command.Forced,
			IdempotencyKey: command.IdempotencyKey,
			EvidenceRefs:   command.EvidenceRefs,
		})
		if err != nil {
			return nil, err
		}
		targets = append(targets, shutdownRunFromStateV0(candidate, state))
	}
	return targets, nil
}

func queueCandidateTerminalV0(candidate orquestarunqueue.RunSchedulingCandidateV0) bool {
	switch strings.TrimSpace(candidate.Status) {
	case orquestarunqueue.RunStatusCanceledV0,
		orquestarunqueue.RunStatusStoppedV0,
		orquestarunqueue.RunStatusClosedV0:
		return true
	default:
		return false
	}
}

func readShutdownControlStateV0(
	ctx context.Context,
	reader orquestaruncontrol.RunControlReaderPortV0,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := reader.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: runRef,
	})
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}

func shutdownRunFromStateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) ServerShutdownRunResultV0 {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return ServerShutdownRunResultV0{
		RunRef:             strings.TrimSpace(candidate.RunRef),
		AppRef:             strings.TrimSpace(candidate.AppRef),
		ControlStatus:      string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status)),
		CheckpointRequired: evaluation.CheckpointRequired,
		Terminal:           evaluation.Terminal,
		StopRequested:      true,
		Ready:              evaluation.Terminal,
	}
}

func shouldRunShutdownSupervisorV0(runs []ServerShutdownRunResultV0) bool {
	for _, run := range runs {
		if !run.CheckpointRequired && !run.Terminal {
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

func applyShutdownControlStateV0(
	run ServerShutdownRunResultV0,
	state orquestaruncontrol.RunControlStateV0,
) ServerShutdownRunResultV0 {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	run.ControlStatus = string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status))
	run.CheckpointRequired = evaluation.CheckpointRequired
	run.Terminal = evaluation.Terminal
	run.Ready = evaluation.Terminal
	return run
}

func applyShutdownStatsV0(
	run ServerShutdownRunResultV0,
	stats RunShutdownStatsV0,
) ServerShutdownRunResultV0 {
	run.AgentsInFlight = stats.AgentsInFlight
	run.AgentsStopRequested = stats.AgentsStopRequested
	run.AgentsStopConfirmed = stats.AgentsStopConfirmed
	if !run.Ready && !run.CheckpointRequired {
		run.Ready = stats.AgentsInFlight == 0 &&
			stats.AgentsStopRequested == stats.AgentsStopConfirmed
	}
	return run
}

func summarizeShutdownResultV0(
	result ServerShutdownResultV0,
) ServerShutdownResultV0 {
	result.Status = ServerShutdownStatusReadyV0
	result.ShutdownReady = true
	for _, run := range result.Runs {
		result.AgentsInFlight += run.AgentsInFlight
		if run.Ready {
			result.RunsStopped++
		}
		if run.CheckpointRequired {
			result.CheckpointsPending++
		}
		if !run.Ready {
			result.ShutdownReady = false
			if run.CheckpointRequired {
				result.Status = ServerShutdownStatusWaitingCheckpointV0
			} else if result.Status == ServerShutdownStatusReadyV0 {
				result.Status = ServerShutdownStatusWaitingDrainV0
			}
		}
	}
	return result
}
