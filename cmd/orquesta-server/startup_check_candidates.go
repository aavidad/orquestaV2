package main

import (
	"context"
	"errors"
	"os"
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
		reconciled, ok, err := check.startupPendingStopCandidateV0(ctx, candidate, state)
		if err != nil {
			return nil, err
		}
		if ok {
			cleanup = append(cleanup, reconciled)
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
		if suppressed, ok := check.suppressStartupAutoprogrammingForDomainSessionV0(candidate); ok {
			cleanup = append(cleanup, suppressed)
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
		if entry.CompleteControlPending {
			if err := check.completeStartupRunControlV0(ctx, entry); err != nil {
				return synced, err
			}
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

const (
	startupIdleSelfImprovementDomainSessionSuppressedReasonV0 = "idle_self_improvement_suppressed_by_domain_session"
	startupIdleSelfImprovementDomainSessionEvidenceV0         = "evidence-ref-idle-self-improvement-domain-session"
)

func (check serverStartupCheckV0) suppressStartupAutoprogrammingForDomainSessionV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (startupCandidateCleanupV0, bool) {
	if !check.startupDomainSessionSuppressesIdleSelfImprovementV0() ||
		!startupAutoprogrammingRunRefV0(candidate.RunRef) {
		return startupCandidateCleanupV0{}, false
	}
	candidate.RescueReason = startupIdleSelfImprovementDomainSessionSuppressedReasonV0
	candidate.EvidenceRefs = appendStartupEvidenceRefV0(
		candidate.EvidenceRefs,
		startupIdleSelfImprovementDomainSessionEvidenceV0,
	)
	return startupCandidateCleanupV0{
		Candidate:               candidate,
		QueueDirty:              true,
		QueueStatus:             orquestarunqueue.RunStatusStoppedV0,
		DomainSessionSuppressed: true,
	}, true
}

func (check serverStartupCheckV0) startupDomainSessionSuppressesIdleSelfImprovementV0() bool {
	if !check.ServerConfig.IdleSelfImprovementDisabled {
		return false
	}
	if serverProjectWorkDirLooksLikeOPESV0(check.ServerConfig.ProjectWorkDir) ||
		serverProjectWorkDirLooksLikeOPESV0(os.Getenv(envCodexProjectWorkDirV0)) ||
		strings.TrimSpace(os.Getenv(envOPESProjectWorkDirV0)) != "" {
		return true
	}
	for _, setting := range check.ServerConfig.EffectiveConfig.Settings {
		key := strings.ToUpper(strings.TrimSpace(setting.Key))
		scope := strings.ToLower(strings.TrimSpace(setting.Scope))
		value := strings.ToLower(strings.TrimSpace(setting.Value))
		source := strings.ToLower(strings.TrimSpace(setting.Source))
		if value == "" || value == "false" || value == "0" || value == "absent" || source == "defaulted" {
			continue
		}
		if strings.HasPrefix(key, "ORQUESTA_OPES_") ||
			strings.Contains(scope, "domain_work") ||
			strings.Contains(scope, "external_bridge") ||
			strings.Contains(scope, "opes") {
			return true
		}
	}
	return false
}

func startupAutoprogrammingRunRefV0(runRef string) bool {
	return strings.HasPrefix(strings.TrimSpace(runRef), "request-ref-autoprogramming-")
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

func (check serverStartupCheckV0) startupPendingStopCandidateV0(
	ctx context.Context,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) (startupCandidateCleanupV0, bool, error) {
	target, ok := startupCompleteStatusFromPendingControlV0(state.Status)
	if !ok {
		return startupCandidateCleanupV0{}, false, nil
	}
	ready, err := check.startupCandidateCanBeCompletedByStartupV0(ctx, candidate.RunRef)
	if err != nil {
		return startupCandidateCleanupV0{}, false, err
	}
	if !ready {
		return startupCandidateCleanupV0{Candidate: candidate, Active: true}, true, nil
	}
	return startupCandidateCleanupV0{
		Candidate:              candidate,
		QueueDirty:             true,
		QueueStatus:            startupQueueStatusFromControlV0(target),
		CompleteControlStatus:  target,
		CompleteControlReason:  "shutdown/reload: stop/cancel pendiente sin agentes en vuelo",
		CompleteControlPending: true,
	}, true, nil
}

func startupCompleteStatusFromPendingControlV0(
	status orquestaruncontrol.RunControlStatusV0,
) (orquestaruncontrol.RunControlStatusV0, bool) {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(status) {
	case orquestaruncontrol.RunControlStatusStopRequestedV0:
		return orquestaruncontrol.RunControlStatusStoppedV0, true
	case orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return orquestaruncontrol.RunControlStatusCanceledV0, true
	default:
		return "", false
	}
}
