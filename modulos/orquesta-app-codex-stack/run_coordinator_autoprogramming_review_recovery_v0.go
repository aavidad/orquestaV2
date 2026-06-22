package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func (stack StackV0) recoverBlockedAutoprogrammingOpenReviewRunV0(
	ctx context.Context,
	runRef string,
	correlationID string,
	occurredAt string,
) (bool, error) {
	if stack.Ports.RunStore == nil {
		return false, nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return false, err
	}
	_, recovered, err := stack.recoverBlockedAutoprogrammingOpenReviewRunSnapshotV0(
		ctx,
		correlationID,
		occurredAt,
		run,
	)
	return recovered, err
}

func (stack StackV0) recoverBlockedAutoprogrammingOpenReviewRunForDrainV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	state, err := stack.runControlStateForDirectorDecisionSourceRecoveryV0(ctx, run.RunID)
	if err != nil {
		return run, false, err
	}
	if !codexStackRunControlAllowsAutoprogrammingOpenReviewRecoveryV0(state) {
		return run, false, nil
	}
	return stack.recoverBlockedAutoprogrammingOpenReviewRunSnapshotV0(
		ctx,
		strings.TrimSpace(request.CorrelationID),
		strings.TrimSpace(request.OccurredAt),
		run,
	)
}

func (stack StackV0) recoverBlockedAutoprogrammingOpenReviewRunForQueueV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if !codexStackRunControlAllowsAutoprogrammingOpenReviewRecoveryV0(state) {
		return run, false, nil
	}
	return stack.recoverBlockedAutoprogrammingOpenReviewRunSnapshotV0(
		ctx,
		strings.TrimSpace(command.CorrelationID),
		formatStackCoordinatorTimeV0(command.OccurredAt),
		run,
	)
}

func (stack StackV0) recoverBlockedAutoprogrammingOpenReviewRunSnapshotV0(
	ctx context.Context,
	correlationID string,
	occurredAt string,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!stackDrainRunHasAllTasksDeliveredOrClosedV0(run) ||
		drainRunHasPendingExternalAgentsV0(run, nil) {
		return run, false, nil
	}
	hasOpenAutoprogrammingTask, err := RunHasOpenAutoprogrammingTasksV0(ctx, stack.Stores.TaskStore, run)
	if err != nil || !hasOpenAutoprogrammingTask {
		return run, false, err
	}
	recoverable, retained := codexStackSplitRecoverableAutoprogrammingOpenReviewBlockersV0(run.Blockers)
	if len(recoverable) == 0 || len(retained) > 0 {
		return run, false, nil
	}
	recovered := run
	for _, blocker := range recoverable {
		next, err := stack.resolveRunBlockerForAutoprogrammingOpenReviewRecoveryV0(
			ctx,
			recovered,
			blocker,
			correlationID,
			occurredAt,
		)
		if err != nil {
			return run, false, err
		}
		recovered = next
	}
	if recovered.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		recovered.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		return recovered, true, nil
	}
	opened, err := stack.openRevisionPhaseForAutoprogrammingOpenReviewRecoveryV0(
		ctx,
		recovered,
		correlationID,
		occurredAt,
	)
	if err != nil {
		return run, false, err
	}
	return opened, true, nil
}

func codexStackRunControlAllowsAutoprogrammingOpenReviewRecoveryV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return evaluation.DispatchAllowed ||
		(state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
			stackRunControlCanAutoResumeQueuedCandidateV0(state))
}

func (stack StackV0) resolveRunBlockerForAutoprogrammingOpenReviewRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	blocker string,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(blocker)
	resolve, err := orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-autoprogramming-open-review-resolve-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-autoprogramming-open-review-resolve-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-autoprogramming-review-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blocker,
			ReasonCode:   "autoprogramming_open_review_recovered",
			Summary:      "Resolver bloqueo tecnico recuperable de apertura de revision con ACK de autoprogramacion entregado.",
			EvidenceRefs: []string{"evidence-ref-autoprogramming-open-review-blocker-recovered"},
		},
	)
	if err != nil {
		return run, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, resolve)
	if err != nil {
		return run, err
	}
	next := run
	for _, event := range result.Events {
		applied, err := orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, err
		}
		next = applied
	}
	if len(result.Events) > 0 {
		if err := stack.Ports.EventSink.AppendRunEventsV0(ctx, run.RunID, result.Events); err != nil {
			return run, err
		}
		if err := stack.Ports.RunStore.SaveRunV0(ctx, next); err != nil {
			return run, err
		}
	}
	return next, nil
}

func (stack StackV0) openRevisionPhaseForAutoprogrammingOpenReviewRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(run.RunID)
	open, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-autoprogramming-review-phase-repair-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-autoprogramming-review-phase-repair-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-autoprogramming-review-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "autoprogramming-open-review-recovery",
		},
	)
	if err != nil {
		return run, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, open)
	if err != nil {
		return run, err
	}
	next := run
	for _, event := range result.Events {
		applied, err := orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, err
		}
		next = applied
	}
	if len(result.Events) > 0 {
		if err := stack.Ports.EventSink.AppendRunEventsV0(ctx, run.RunID, result.Events); err != nil {
			return run, err
		}
		if err := stack.Ports.RunStore.SaveRunV0(ctx, next); err != nil {
			return run, err
		}
	}
	return next, nil
}

func codexStackSplitRecoverableAutoprogrammingOpenReviewBlockersV0(
	blockers []string,
) ([]string, []string) {
	recoverable := make([]string, 0, len(blockers))
	retained := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		trimmed := strings.TrimSpace(blocker)
		if trimmed == "" {
			continue
		}
		if codexStackRecoverableAutoprogrammingOpenReviewBlockerV0(trimmed) {
			recoverable = append(recoverable, trimmed)
			continue
		}
		retained = append(retained, trimmed)
	}
	return recoverable, retained
}

func codexStackRecoverableAutoprogrammingOpenReviewBlockerV0(blocker string) bool {
	normalized := strings.ToLower(strings.TrimSpace(blocker))
	return strings.HasPrefix(normalized, "app-director-decision-") &&
		strings.Contains(normalized, "director-decision-apply-error") &&
		strings.Contains(normalized, "open-review")
}
