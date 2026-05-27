package orquestaservershutdown

import (
	"context"
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
	if !serverShutdownRequesterAuthorizedV0(command.RequestedBy) {
		result := missingServerShutdownDepV0(ServerShutdownStatusRequesterDeniedV0)
		result.EvidenceRefs = compactServerShutdownStringsV0(append(
			command.EvidenceRefs,
			"evidence-ref-shutdown-requester-not-director",
		))
		return result, nil
	}
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
