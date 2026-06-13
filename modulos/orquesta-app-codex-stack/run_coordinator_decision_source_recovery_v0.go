package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const codexStackDecisionSourceRecoveryRequestedByV0 = "orquesta-app-codex-stack-decision-source-recovery"

func (stack StackV0) recoverBlockedDirectorDecisionSourceRunForDrainV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	state, err := stack.runControlStateForDirectorDecisionSourceRecoveryV0(ctx, run.RunID)
	if err != nil {
		return run, false, err
	}
	return stack.recoverBlockedDirectorDecisionSourceRunV0(
		ctx,
		strings.TrimSpace(request.CorrelationID),
		strings.TrimSpace(request.OccurredAt),
		state,
		run,
	)
}

func (stack StackV0) recoverBlockedDirectorDecisionSourceRunForQueueV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	return stack.recoverBlockedDirectorDecisionSourceRunV0(
		ctx,
		strings.TrimSpace(command.CorrelationID),
		formatStackCoordinatorTimeV0(command.OccurredAt),
		state,
		run,
	)
}

func (stack StackV0) recoverBlockedDirectorDecisionSourceRunV0(
	ctx context.Context,
	correlationID string,
	occurredAt string,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		stack.Ports.DirectorDecisionSource == nil {
		return run, false, nil
	}
	if !codexStackRunControlAllowsDirectorDecisionSourceRecoveryV0(state) {
		return run, false, nil
	}
	recoverable, retained := codexStackSplitRecoverableDirectorDecisionSourceBlockersV0(run.Blockers)
	if len(recoverable) == 0 || len(retained) > 0 {
		return run, false, nil
	}
	ok, err := stack.blockedDirectorDecisionSourceHasValidDecisionsV0(ctx, run, correlationID, occurredAt)
	if err != nil || !ok {
		return run, false, err
	}
	recovered := run
	for _, blocker := range recoverable {
		next, err := stack.resolveRunBlockerForDirectorDecisionSourceRecoveryV0(
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
	return recovered, true, nil
}

func (stack StackV0) runControlStateForDirectorDecisionSourceRecoveryV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	runRef = strings.TrimSpace(runRef)
	if stack.Stores.RunControl == nil {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
		}
		return orquestaruncontrol.RunControlStateV0{}, err
	}
	return state, nil
}

func codexStackRunControlAllowsDirectorDecisionSourceRecoveryV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return evaluation.DispatchAllowed ||
		(state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
			stackRunControlCanAutoResumeQueuedCandidateV0(state))
}

func (stack StackV0) blockedDirectorDecisionSourceHasValidDecisionsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	correlationID string,
	occurredAt string,
) (bool, error) {
	decisions, err := stack.Ports.DirectorDecisionSource.ListDirectorAgentDecisionsV0(
		ctx,
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run:           run,
			RequestKind:   "",
			ExecutionMode: orquestafactory.ExecutionModeNormalV0,
			OccurredAt:    strings.TrimSpace(occurredAt),
			CorrelationID: strings.TrimSpace(correlationID),
			RequestedBy:   codexStackDecisionSourceRecoveryRequestedByV0,
		},
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, ctxErr
		}
		return false, nil
	}
	return len(decisions) > 0, nil
}

func (stack StackV0) resolveRunBlockerForDirectorDecisionSourceRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	blocker string,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(blocker)
	resolve, err := orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-director-decision-source-resolve-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-director-decision-source-resolve-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-decision-source-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blocker,
			ReasonCode:   "director_decision_source_recovered",
			Summary:      "Resolver bloqueo tecnico recuperable de fuente de decisiones del director tras relectura valida.",
			EvidenceRefs: []string{"evidence-ref-director-decision-source-blocker-recovered"},
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

func codexStackSplitRecoverableDirectorDecisionSourceBlockersV0(
	blockers []string,
) ([]string, []string) {
	recoverable := make([]string, 0, len(blockers))
	retained := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		trimmed := strings.TrimSpace(blocker)
		if trimmed == "" {
			continue
		}
		if codexStackRecoverableDirectorDecisionSourceBlockerV0(trimmed) {
			recoverable = append(recoverable, trimmed)
			continue
		}
		retained = append(retained, trimmed)
	}
	return recoverable, retained
}

func codexStackRecoverableDirectorDecisionSourceBlockerV0(blocker string) bool {
	normalized := strings.ToLower(strings.TrimSpace(blocker))
	return strings.HasPrefix(normalized, "app-director-decision-") &&
		strings.Contains(normalized, "director-decision-source-error") &&
		strings.Contains(normalized, "director-decision-source")
}

func codexStackDecisionSourceRecoveryRequestedV0(requestedBy string) bool {
	return strings.TrimSpace(requestedBy) == codexStackDecisionSourceRecoveryRequestedByV0
}
