package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func (stack StackV0) recoverBlockedOpenPhaseProjectionRunForDrainV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	state, err := stack.runControlStateForDirectorDecisionSourceRecoveryV0(ctx, run.RunID)
	if err != nil {
		return run, false, err
	}
	return stack.recoverBlockedOpenPhaseProjectionRunV0(
		ctx,
		strings.TrimSpace(request.CorrelationID),
		strings.TrimSpace(request.OccurredAt),
		state,
		run,
	)
}

func (stack StackV0) recoverBlockedOpenPhaseProjectionRunForQueueV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	return stack.recoverBlockedOpenPhaseProjectionRunV0(
		ctx,
		strings.TrimSpace(command.CorrelationID),
		formatStackCoordinatorTimeV0(command.OccurredAt),
		state,
		run,
	)
}

func (stack StackV0) recoverBlockedOpenPhaseProjectionRunV0(
	ctx context.Context,
	correlationID string,
	occurredAt string,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		stack.Ports.EventReader == nil {
		return run, false, nil
	}
	if !codexStackRunControlAllowsOpenPhaseProjectionRecoveryV0(state) {
		return run, false, nil
	}
	recoverable, retained := codexStackSplitRecoverableOpenPhaseProjectionBlockersV0(run.Blockers)
	if len(recoverable) == 0 || len(retained) > 0 {
		return run, false, nil
	}
	phaseEvent, ok, err := stack.recoverableOpenPhaseProjectionEventV0(ctx, run, recoverable)
	if err != nil || !ok {
		return run, false, err
	}
	recovered, err := codexStackProjectDurableOpenPhaseEventV0(run, phaseEvent)
	if err != nil {
		return run, false, err
	}
	for _, blocker := range recoverable {
		next, err := stack.resolveRunBlockerForOpenPhaseProjectionRecoveryV0(
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

func codexStackRunControlAllowsOpenPhaseProjectionRecoveryV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return evaluation.DispatchAllowed ||
		(state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
			stackRunControlCanAutoResumeQueuedCandidateV0(state))
}

func (stack StackV0) recoverableOpenPhaseProjectionEventV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	blockers []string,
) (orquestacoreworkflow.OrchestrationEventV0, bool, error) {
	events, err := stack.Ports.EventReader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return orquestacoreworkflow.OrchestrationEventV0{}, false, err
	}
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != orquestacoreworkflow.OrchestrationEventPhaseOpenedV0 ||
			strings.TrimSpace(event.RunID) != strings.TrimSpace(run.RunID) ||
			event.Sequence > run.LastSequence {
			continue
		}
		phaseID, ok := codexStackPhaseOpenedEventPhaseIDV0(event)
		if !ok || !codexStackOpenPhaseProjectionBlockersMatchPhaseV0(blockers, phaseID) {
			continue
		}
		if !codexStackRunHasWorkForOpenPhaseProjectionV0(run, phaseID) {
			continue
		}
		return event, true, nil
	}
	return orquestacoreworkflow.OrchestrationEventV0{}, false, nil
}

func codexStackProjectDurableOpenPhaseEventV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	lastEventID := run.LastEventID
	lastSequence := run.LastSequence
	projected, err := orquestacoreworkflow.ApplyEventV0(run, event)
	if err != nil {
		return run, err
	}
	projected.LastEventID = lastEventID
	projected.LastSequence = lastSequence
	return projected, nil
}

func (stack StackV0) resolveRunBlockerForOpenPhaseProjectionRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	blocker string,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(blocker)
	resolve, err := orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-phase-projection-resolve-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-open-phase-projection-resolve-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-open-phase-projection-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blocker,
			ReasonCode:   "open_phase_projection_recovered",
			Summary:      "Resolver bloqueo tecnico recuperable de apertura de fase ya persistida en historial.",
			EvidenceRefs: []string{"evidence-ref-open-phase-projection-blocker-recovered"},
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
	}
	if err := stack.Ports.RunStore.SaveRunV0(ctx, next); err != nil {
		return run, err
	}
	return next, nil
}

func codexStackSplitRecoverableOpenPhaseProjectionBlockersV0(
	blockers []string,
) ([]string, []string) {
	recoverable := make([]string, 0, len(blockers))
	retained := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		trimmed := strings.TrimSpace(blocker)
		if trimmed == "" {
			continue
		}
		if codexStackRecoverableOpenPhaseProjectionBlockerV0(trimmed) {
			recoverable = append(recoverable, trimmed)
			continue
		}
		retained = append(retained, trimmed)
	}
	return recoverable, retained
}

func codexStackRecoverableOpenPhaseProjectionBlockerV0(blocker string) bool {
	normalized := strings.ToLower(strings.TrimSpace(blocker))
	return strings.HasPrefix(normalized, "app-director-decision-") &&
		strings.Contains(normalized, "director-decision-apply-error") &&
		strings.Contains(normalized, "open-")
}

func codexStackPhaseOpenedEventPhaseIDV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationPhaseIDV0, bool) {
	var payload orquestacoreworkflow.PhaseOpenedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", false
	}
	phaseID := orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(payload.PhaseID))
	if err := orquestacoreworkflow.ValidateOrchestrationPhaseIDV0(phaseID); err != nil {
		return "", false
	}
	return phaseID, true
}

func codexStackOpenPhaseProjectionBlockersMatchPhaseV0(
	blockers []string,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	phase := strings.ToLower(strings.TrimSpace(string(phaseID)))
	if phase == "" {
		return false
	}
	for _, blocker := range blockers {
		if !strings.Contains(strings.ToLower(strings.TrimSpace(blocker)), "open-"+phase) {
			return false
		}
	}
	return true
}

func codexStackRunHasWorkForOpenPhaseProjectionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	switch phaseID {
	case orquestacoreworkflow.OrchestrationPhaseProgramacionV0:
		return len(compactStringsV0(run.Tasks)) > 0
	case orquestacoreworkflow.OrchestrationPhaseRevisionV0:
		return len(compactStringsV0(run.Deliveries)) > 0 ||
			len(compactStringsV0(run.DeliveredTasks)) > 0
	default:
		return true
	}
}
