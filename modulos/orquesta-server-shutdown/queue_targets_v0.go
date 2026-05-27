package orquestaservershutdown

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

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
		target, ok, err := stopShutdownTargetV0(ctx, deps, command, candidate, state)
		if err != nil {
			return nil, err
		}
		if ok {
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func stopShutdownTargetV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) (ServerShutdownRunResultV0, bool, error) {
	checkpointRef := ""
	runRef := strings.TrimSpace(candidate.RunRef)
	if !command.Forced && !state.CheckpointRecorded {
		prepared, checkpoint, ok, err := prepareShutdownCheckpointV0(
			ctx,
			deps,
			command,
			candidate,
			state,
		)
		if err != nil {
			return ServerShutdownRunResultV0{}, false, err
		}
		if !ok {
			return unresolvedShutdownCheckpointTargetV0(ctx, deps, command, candidate, state, checkpoint)
		}
		state = prepared
		checkpointRef = checkpoint.CheckpointRef
	}
	state, err := requestShutdownStopV0(ctx, deps, command, runRef)
	if err != nil {
		return ServerShutdownRunResultV0{}, false, err
	}
	stopped := shutdownRunFromStateV0(candidate, state)
	stopped.CheckpointRef = checkpointRef
	return stopped, true, nil
}

func unresolvedShutdownCheckpointTargetV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
	checkpoint PrepareAgentShutdownResultV0,
) (ServerShutdownRunResultV0, bool, error) {
	if checkpointDeadlineExpiredV0(command) {
		forcedCommand := forcedShutdownCommandAfterCheckpointDeadlineV0(command)
		state, err := requestShutdownStopV0(
			ctx,
			deps,
			forcedCommand,
			strings.TrimSpace(candidate.RunRef),
		)
		if err != nil {
			return ServerShutdownRunResultV0{}, false, err
		}
		stopped := shutdownRunFromStateV0(candidate, state)
		stopped.CheckpointDeadlineExpired = true
		stopped.ForcedAfterCheckpointDeadline = true
		stopped.PendingCheckpointAgentRefs = compactServerShutdownStringsV0(checkpoint.PendingAgentRefs)
		stopped.CheckpointEvidenceRefs = compactServerShutdownStringsV0(checkpoint.EvidenceRefs)
		return stopped, true, nil
	}
	pending := shutdownRunFromStateV0(candidate, state)
	pending.CheckpointRequired = true
	pending.Ready = false
	pending.CheckpointRef = checkpoint.CheckpointRef
	pending.PendingCheckpointAgentRefs = compactServerShutdownStringsV0(checkpoint.PendingAgentRefs)
	pending.CheckpointEvidenceRefs = compactServerShutdownStringsV0(checkpoint.EvidenceRefs)
	return pending, true, nil
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
