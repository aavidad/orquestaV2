package orquestaruncoordinator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func CoordinateRunsTickV0(
	ctx context.Context,
	deps RunCoordinatorDepsV0,
	command RunCoordinatorTickCommandV0,
) (RunCoordinatorTickResultV0, error) {
	if deps.QueueReader == nil {
		return RunCoordinatorTickResultV0{}, fmt.Errorf("run_coordinator: queue_reader requerido")
	}
	if deps.Drainer == nil {
		return RunCoordinatorTickResultV0{}, fmt.Errorf("run_coordinator: drainer requerido")
	}
	candidates, err := deps.QueueReader.ListRunSchedulingCandidatesV0(ctx, queueReadRequestV0(command))
	if err != nil {
		return RunCoordinatorTickResultV0{}, err
	}

	policy := command.RankingPolicy
	if policy.Now.IsZero() {
		policy = orquestarunqueue.DefaultRunQueueRankingPolicyV0(command.OccurredAt)
	}
	ranked := orquestarunqueue.RankRunCandidatesV0(candidates, policy)
	result := RunCoordinatorTickResultV0{Ranked: compactRankedV0(ranked)}

	maxRuns := command.MaxRuns
	if maxRuns <= 0 {
		maxRuns = 1
	}
	excluded := stringSetV0(command.ExcludeRunRefs)
	for _, candidate := range ranked {
		if len(result.Executions) >= maxRuns {
			break
		}
		if _, ok := excluded[candidate.RunRef]; ok {
			result.Skips = append(result.Skips, excludedSkipV0(candidate))
			continue
		}
		state, readErr := readRunControlStateV0(ctx, deps.ControlReader, candidate.RunRef)
		if readErr != nil {
			return RunCoordinatorTickResultV0{}, readErr
		}
		evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
		if !evaluation.DispatchAllowed && !evaluation.StopAgentsAllowed {
			if err := syncControlBlockedQueueStatusV0(ctx, deps.QueueUpdater, candidate, command, state); err != nil {
				return RunCoordinatorTickResultV0{}, err
			}
			result.Skips = append(result.Skips, controlSkipV0(candidate, state))
			continue
		}
		executed, drainErr := deps.Drainer.DrainRunV0(ctx, drainRequestV0(candidate, command))
		if drainErr != nil {
			executed = runDrainResultWithErrorDiagnosticV0(candidate, executed, drainErr)
			if err := rotateExecutedRunV0(ctx, deps.QueueUpdater, candidate, command, executed); err != nil {
				return RunCoordinatorTickResultV0{}, err
			}
			result.Executions = append(result.Executions, executionSummaryV0(candidate, executed))
			if !command.ContinueOnDrainError {
				return result, drainErr
			}
			continue
		}
		if err := rotateExecutedRunV0(ctx, deps.QueueUpdater, candidate, command, executed); err != nil {
			return RunCoordinatorTickResultV0{}, err
		}
		result.Executions = append(result.Executions, executionSummaryV0(candidate, executed))
	}
	return result, nil
}

func queueReadRequestV0(command RunCoordinatorTickCommandV0) orquestarunqueue.RunQueueReadRequestV0 {
	return orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: strings.TrimSpace(command.QueueRef),
		AppRefs:  append([]string(nil), command.AppRefs...),
		Limit:    command.QueueLimit,
	}
}

func syncControlBlockedQueueStatusV0(
	ctx context.Context,
	updater orquestarunqueue.RunQueuePriorityWriterPortV0,
	candidate orquestarunqueue.RankedRunCandidateV0,
	command RunCoordinatorTickCommandV0,
	state orquestaruncontrol.RunControlStateV0,
) error {
	if updater == nil {
		return nil
	}
	status, ok := queueStatusForBlockedRunControlV0(state)
	if !ok || strings.TrimSpace(candidate.Status) == status {
		return nil
	}
	refs := append([]string(nil), candidate.EvidenceRefs...)
	refs = append(refs, "evidence-ref-run-coordinator-control-blocked-queue-sync")
	_, err := updater.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        candidate.RunRef,
		QueueRef:      command.QueueRef,
		AppRef:        candidate.AppRef,
		Status:        status,
		PriorityScore: candidate.PriorityScore,
		UpdatedAt:     command.OccurredAt,
		RequestedBy:   "orquesta-run-coordinator",
		Reason:        "run_control_blocked_queue_sync",
		EvidenceRefs:  refs,
	})
	return err
}

func queueStatusForBlockedRunControlV0(
	state orquestaruncontrol.RunControlStateV0,
) (string, bool) {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusPausedV0:
		return orquestarunqueue.RunStatusPausedV0, true
	case orquestaruncontrol.RunControlStatusStopRequestedV0, orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestarunqueue.RunStatusStoppedV0, true
	case orquestaruncontrol.RunControlStatusCancelRequestedV0, orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestarunqueue.RunStatusCanceledV0, true
	default:
		return "", false
	}
}

func readRunControlStateV0(
	ctx context.Context,
	reader orquestaruncontrol.RunControlReaderPortV0,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	if reader == nil {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	state, err := reader.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef})
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}

func drainRequestV0(
	candidate orquestarunqueue.RankedRunCandidateV0,
	command RunCoordinatorTickCommandV0,
) RunDrainRequestV0 {
	return RunDrainRequestV0{
		RunRef:        candidate.RunRef,
		AppRef:        candidate.AppRef,
		Rank:          candidate.Rank,
		OccurredAt:    command.OccurredAt,
		CorrelationID: strings.TrimSpace(command.CorrelationID),
		Limits:        command.DrainLimits,
	}
}

func rotateExecutedRunV0(
	ctx context.Context,
	updater orquestarunqueue.RunQueuePriorityWriterPortV0,
	candidate orquestarunqueue.RankedRunCandidateV0,
	command RunCoordinatorTickCommandV0,
	result RunDrainResultV0,
) error {
	if updater == nil || command.OccurredAt.IsZero() {
		return nil
	}
	refs := append([]string(nil), candidate.EvidenceRefs...)
	refs = append(refs, "evidence-ref-run-coordinator-executed")
	queueStatus := effectiveExecutedRunQueueStatusV0(candidate, result)
	_, err := updater.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        candidate.RunRef,
		QueueRef:      command.QueueRef,
		AppRef:        candidate.AppRef,
		Status:        queueStatus,
		PriorityScore: candidate.PriorityScore,
		UpdatedAt:     command.OccurredAt,
		RequestedBy:   "orquesta-run-coordinator",
		Reason:        "run_executed_rotation",
		EvidenceRefs:  refs,
	})
	return err
}

func effectiveExecutedRunQueueStatusV0(
	candidate orquestarunqueue.RankedRunCandidateV0,
	result RunDrainResultV0,
) string {
	status := strings.TrimSpace(result.QueueStatus)
	if status != "" {
		return status
	}
	if orquestarunqueue.IsExecutableRunStatusV0(candidate.Status) {
		return orquestarunqueue.RunStatusRunningV0
	}
	return strings.TrimSpace(candidate.Status)
}

func compactRankedV0(ranked []orquestarunqueue.RankedRunCandidateV0) []RankedRunSummaryV0 {
	summaries := make([]RankedRunSummaryV0, 0, len(ranked))
	for _, candidate := range ranked {
		summaries = append(summaries, RankedRunSummaryV0{
			RunRef:        candidate.RunRef,
			AppRef:        candidate.AppRef,
			Rank:          candidate.Rank,
			PriorityScore: candidate.PriorityScore,
			AgingBoost:    candidate.AgingBoost,
		})
	}
	return summaries
}

func controlSkipV0(
	candidate orquestarunqueue.RankedRunCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) RunSkipSummaryV0 {
	status := string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status))
	return RunSkipSummaryV0{
		RunRef: candidate.RunRef,
		AppRef: candidate.AppRef,
		Rank:   candidate.Rank,
		Reason: "run_control_blocked",
		Status: status,
	}
}

func excludedSkipV0(candidate orquestarunqueue.RankedRunCandidateV0) RunSkipSummaryV0 {
	return RunSkipSummaryV0{
		RunRef: candidate.RunRef,
		AppRef: candidate.AppRef,
		Rank:   candidate.Rank,
		Reason: "run_excluded",
		Status: candidate.Status,
	}
}

func stringSetV0(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}
