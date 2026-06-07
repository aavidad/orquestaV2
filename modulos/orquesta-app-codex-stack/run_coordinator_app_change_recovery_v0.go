package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func (stack StackV0) recoverBlockedAppChangeAutoPlanRunV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		stack.Stores.AppChangeStore == nil {
		return run, false, nil
	}
	if !codexStackRunControlAllowsAppChangeAutoPlanRecoveryV0(state) {
		return run, false, nil
	}
	recoverable, retained := codexStackSplitRecoverableAppChangeAutoPlanBlockersV0(run.Blockers)
	if len(recoverable) == 0 || len(retained) > 0 {
		return run, false, nil
	}
	decisions, err := (orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
		Store: stack.Stores.AppChangeStore,
	}).ListDirectorAgentDecisionsV0(
		ctx,
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)
	if err != nil {
		return run, false, err
	}
	if !codexStackHasRecoverableAppChangeAutoPlanDecisionV0(run, decisions) {
		return run, false, nil
	}
	recovered := run
	for _, blocker := range recoverable {
		next, err := stack.resolveRunBlockerForAppChangeAutoPlanRecoveryV0(
			ctx,
			command,
			recovered,
			blocker,
		)
		if err != nil {
			return run, false, err
		}
		recovered = next
	}
	return recovered, true, nil
}

func codexStackRunControlAllowsAppChangeAutoPlanRecoveryV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return evaluation.DispatchAllowed ||
		(state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
			stackRunControlCanAutoResumeQueuedCandidateV0(state))
}

func (stack StackV0) resolveRunBlockerForAppChangeAutoPlanRecoveryV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	blocker string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(blocker)
	resolve, err := orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-app-change-autoplan-resolve-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-app-change-autoplan-resolve-" + safe,
			CorrelationID:  command.CorrelationID,
			RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
			OccurredAt:     formatStackCoordinatorTimeV0(command.OccurredAt),
		},
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blocker,
			ReasonCode:   "app_change_autoplan_recovered",
			Summary:      "Resolver bloqueo tecnico recuperable de app-change autoplan.",
			EvidenceRefs: []string{"evidence-ref-app-change-autoplan-blocker-recovered"},
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

func codexStackSplitRecoverableAppChangeAutoPlanBlockersV0(
	blockers []string,
) ([]string, []string) {
	recoverable := make([]string, 0, len(blockers))
	retained := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		trimmed := strings.TrimSpace(blocker)
		if trimmed == "" {
			continue
		}
		if codexStackRecoverableAppChangeAutoPlanBlockerV0(trimmed) {
			recoverable = append(recoverable, trimmed)
			continue
		}
		retained = append(retained, trimmed)
	}
	return recoverable, retained
}

func codexStackRecoverableAppChangeAutoPlanBlockerV0(blocker string) bool {
	normalized := strings.ToLower(strings.TrimSpace(blocker))
	return strings.HasPrefix(normalized, "app-director-decision-") &&
		strings.Contains(normalized, "director-decision-apply-error") &&
		strings.Contains(normalized, "app-change-task")
}

func codexStackHasRecoverableAppChangeAutoPlanDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) bool {
	for _, decision := range decisions {
		if decision.CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 ||
			decision.CreateMicrotask == nil {
			continue
		}
		task := decision.CreateMicrotask.Task
		if strings.TrimSpace(task.RunID) == strings.TrimSpace(run.RunID) &&
			strings.HasPrefix(strings.TrimSpace(task.TaskID), "task-ref-app-change-") &&
			!codexStackStringInSetV0(run.Tasks, task.TaskID) {
			return true
		}
	}
	return false
}
