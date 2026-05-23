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
	result.Runs, err = refreshShutdownRunsV0(ctx, deps, command, result.Runs)
	if err != nil {
		return ServerShutdownResultV0{}, err
	}
	if shouldRunShutdownSupervisorV0(result.Runs, deps.StatsReader != nil) && deps.Supervisor != nil {
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
		checkpointRef := ""
		if !command.Forced && !state.CheckpointRecorded {
			prepared, checkpoint, ok, err := prepareShutdownCheckpointV0(
				ctx,
				deps,
				command,
				candidate,
				state,
			)
			if err != nil {
				return nil, err
			}
			if !ok {
				if checkpointDeadlineExpiredV0(command) {
					forcedCommand := forcedShutdownCommandAfterCheckpointDeadlineV0(command)
					state, err = requestShutdownStopV0(ctx, deps, forcedCommand, runRef)
					if err != nil {
						return nil, err
					}
					stopped := shutdownRunFromStateV0(candidate, state)
					stopped.CheckpointDeadlineExpired = true
					stopped.ForcedAfterCheckpointDeadline = true
					stopped.PendingCheckpointAgentRefs = compactServerShutdownStringsV0(checkpoint.PendingAgentRefs)
					stopped.CheckpointEvidenceRefs = compactServerShutdownStringsV0(checkpoint.EvidenceRefs)
					targets = append(targets, stopped)
					continue
				}
				pending := shutdownRunFromStateV0(candidate, state)
				pending.CheckpointRequired = true
				pending.Ready = false
				pending.CheckpointRef = checkpoint.CheckpointRef
				pending.PendingCheckpointAgentRefs = compactServerShutdownStringsV0(checkpoint.PendingAgentRefs)
				pending.CheckpointEvidenceRefs = compactServerShutdownStringsV0(checkpoint.EvidenceRefs)
				targets = append(targets, pending)
				continue
			}
			state = prepared
			checkpointRef = checkpoint.CheckpointRef
			state, err = requestShutdownStopV0(ctx, deps, command, runRef)
			if err != nil {
				return nil, err
			}
		} else {
			state, err = requestShutdownStopV0(ctx, deps, command, runRef)
			if err != nil {
				return nil, err
			}
		}
		stopped := shutdownRunFromStateV0(candidate, state)
		stopped.CheckpointRef = checkpointRef
		targets = append(targets, stopped)
	}
	return targets, nil
}

func requestShutdownStopV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	return deps.RunControlWriter.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:         runRef,
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		Forced:         command.Forced,
		IdempotencyKey: command.IdempotencyKey,
		EvidenceRefs:   command.EvidenceRefs,
	})
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
		StopRequested:      shutdownStopRequestedV0(state.Status),
		Ready:              evaluation.Terminal,
	}
}

func shutdownStopRequestedV0(status orquestaruncontrol.RunControlStatusV0) bool {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(status) {
	case orquestaruncontrol.RunControlStatusStopRequestedV0,
		orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return true
	default:
		return false
	}
}

func shouldRunShutdownSupervisorV0(
	runs []ServerShutdownRunResultV0,
	statsAvailable bool,
) bool {
	for _, run := range runs {
		if run.CheckpointRequired || run.Terminal {
			continue
		}
		if !statsAvailable || run.AgentsInFlight > 0 {
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
