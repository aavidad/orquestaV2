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
		return withServerShutdownRecommendedActionV0(result), nil
	}
	if result, ok := missingRequiredServerShutdownDepsV0(deps); ok {
		return result, nil
	}
	goalActions := []ServerShutdownGoalActionV0{}
	initialActiveWorks := []ActiveShutdownWorkV0{}
	if result, ok, evidenceRefs, actions, observedWorks, err := blockingActiveShutdownWorkV0(ctx, deps, command); err != nil || ok {
		return result, err
	} else if len(evidenceRefs) > 0 {
		command.EvidenceRefs = compactServerShutdownStringsV0(append(command.EvidenceRefs, evidenceRefs...))
		goalActions = mergeServerShutdownGoalActionsV0(goalActions, actions)
	} else if len(actions) > 0 {
		goalActions = mergeServerShutdownGoalActionsV0(goalActions, actions)
	} else if len(observedWorks) > 0 {
		initialActiveWorks = observedWorks
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
		GoalActions:   goalActions,
	}
	if len(targets) == 0 {
		result.Status = ServerShutdownStatusReadyV0
		result.ShutdownReady = true
		return finalizeShutdownActiveWorkV0(ctx, deps, command, withServerShutdownRecommendedActionV0(result), initialActiveWorks)
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
	return finalizeShutdownActiveWorkV0(ctx, deps, command, summarizeShutdownResultV0(result), initialActiveWorks)
}
