package main

import (
	"context"
	"errors"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func (check serverStartupCheckV0) startupCandidateCanBeCompletedByStartupV0(
	ctx context.Context,
	runRef string,
) (bool, error) {
	if strings.TrimSpace(check.Mode) == startupCleanupModeForcedStopV0 {
		return true, nil
	}
	if check.Stack.Stores.RunStore == nil {
		return true, nil
	}
	run, err := check.Stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		var notFound orquestacionnucleoapp.RunNotFoundErrorV0
		if errors.As(err, &notFound) {
			return true, nil
		}
		return false, err
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(run)
	return stats.Counts.AgentsInFlight == 0 || startupRunHasNoLivePendingAgentsV0(run), nil
}

func (check serverStartupCheckV0) startupCleanupCandidatesV0(
	ctx context.Context,
) ([]startupCandidateCleanupV0, error) {
	candidates, err := check.Stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef: check.Stack.RunQueue.QueueRef,
			Limit:    check.QueueLimit,
		},
	)
	if err != nil {
		return nil, err
	}
	cleanup := make([]startupCandidateCleanupV0, 0, len(candidates))
	for _, candidate := range candidates {
		if startupQueueCandidateTerminalV0(candidate) {
			continue
		}
		state, err := check.readStartupRunControlV0(ctx, candidate.RunRef)
		if err != nil {
			return nil, err
		}
		if orquestaruncontrol.IsTerminalRunControlStatusV0(state.Status) {
			cleanup = append(cleanup, startupCandidateCleanupV0{
				Candidate:   candidate,
				QueueDirty:  true,
				QueueStatus: startupQueueStatusFromControlV0(state.Status),
			})
			continue
		}
		queueStatus, ok, err := check.startupQueueStatusFromRunStoreV0(ctx, candidate.RunRef)
		if err != nil {
			return nil, err
		}
		if ok {
			cleanup = append(cleanup, startupCandidateCleanupV0{
				Candidate:   candidate,
				QueueDirty:  true,
				QueueStatus: queueStatus,
			})
			continue
		}
		cleanup = append(cleanup, startupCandidateCleanupV0{
			Candidate: candidate,
			Active:    true,
		})
	}
	return cleanup, nil
}

func (check serverStartupCheckV0) readStartupRunControlV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := check.Stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}

func (check serverStartupCheckV0) startupQueueStatusFromRunStoreV0(
	ctx context.Context,
	runRef string,
) (string, bool, error) {
	if check.Stack.Stores.RunStore == nil {
		return "", false, nil
	}
	run, err := check.Stack.Stores.RunStore.LoadRunV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		var notFound orquestacionnucleoapp.RunNotFoundErrorV0
		if errors.As(err, &notFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0, true, nil
	}
	if startupRunHasAllTasksDeliveredOrClosedV0(run) &&
		orquestacionnucleoapp.BuildDirectorRunStatsV0(run).Counts.AgentsInFlight == 0 {
		hold, err := orquestaappcodexstack.RunHasOpenOperationalDirectorTasksV0(
			ctx,
			check.Stack.Stores.TaskStore,
			run,
		)
		if err != nil || hold {
			return "", false, err
		}
		return orquestarunqueue.RunStatusDeliveredV0, true, nil
	}
	return "", false, nil
}

func startupRunHasAllTasksDeliveredOrClosedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	tasks := compactStartupStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return false
	}
	delivered := startupStringSetV0(run.DeliveredTasks)
	closed := startupStringSetV0(run.ClosedTasks)
	for _, taskRef := range tasks {
		if !delivered[taskRef] && !closed[taskRef] {
			return false
		}
	}
	return true
}

func startupRunHasNoLivePendingAgentsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	started := compactStartupStringsV0(run.StartedAgents)
	if len(started) == 0 {
		return true
	}
	delivered := startupStringSetV0(run.DeliveredAgents)
	failed := startupStringSetV0(run.FailedAgents)
	lost := startupStringSetV0(run.LostAgents)
	stopped := startupStringSetV0(run.StoppedAgents)
	confirmedStopped := startupStringSetV0(run.ConfirmedStoppedAgents)
	for _, agentRef := range started {
		if delivered[agentRef] ||
			failed[agentRef] ||
			lost[agentRef] ||
			stopped[agentRef] ||
			confirmedStopped[agentRef] {
			continue
		}
		return false
	}
	return true
}

func compactStartupStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func startupStringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range compactStartupStringsV0(values) {
		out[value] = true
	}
	return out
}

func (check serverStartupCheckV0) reconcileStartupQueueCandidatesV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidates []startupCandidateCleanupV0,
) (int, error) {
	synced := 0
	for _, entry := range candidates {
		if !entry.QueueDirty || entry.Active {
			continue
		}
		if err := check.markStartupQueueCandidateTerminalV0(ctx, command, entry.Candidate, entry.QueueStatus); err != nil {
			return synced, err
		}
		synced++
	}
	return synced, nil
}

func startupQueueCandidateTerminalV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) bool {
	return !orquestarunqueue.IsExecutableRunStatusV0(candidate.Status)
}

func startupQueueStatusFromControlV0(status orquestaruncontrol.RunControlStatusV0) string {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(status) {
	case orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestarunqueue.RunStatusCanceledV0
	case orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestarunqueue.RunStatusStoppedV0
	default:
		return orquestarunqueue.RunStatusStoppedV0
	}
}
