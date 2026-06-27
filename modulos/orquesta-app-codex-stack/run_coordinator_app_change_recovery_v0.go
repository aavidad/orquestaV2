package orquestaappcodexstack

import (
	"context"
	"encoding/json"
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
	staleOpenPlanAfterReview := codexStackHasRecoverableStaleOpenPlanAfterAcceptedReviewV0(run, recoverable)
	decisions, err := (orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
		Store: stack.Stores.AppChangeStore,
	}).ListDirectorAgentDecisionsV0(
		ctx,
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)
	if err != nil {
		return run, false, err
	}
	missingTaskRefs := codexStackRecoverableAppChangeAutoPlanTaskRefsV0(run, decisions)
	if !staleOpenPlanAfterReview && len(missingTaskRefs) == 0 {
		return run, false, nil
	}
	recovered := run
	if len(missingTaskRefs) > 0 {
		next, _, err := stack.recoverDurableAppChangeAutoPlanMicrotaskProjectionV0(
			ctx,
			recovered,
			missingTaskRefs,
		)
		if err != nil {
			return run, false, err
		}
		recovered = next
	}
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

func (stack StackV0) recoverDurableAppChangeAutoPlanMicrotaskProjectionV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRefs []string,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if stack.Ports.EventReader == nil || stack.Ports.RunStore == nil {
		return run, false, nil
	}
	refs := map[string]bool{}
	for _, taskRef := range compactStringsV0(taskRefs) {
		if strings.HasPrefix(taskRef, "task-ref-app-change-") {
			refs[taskRef] = true
		}
	}
	if len(refs) == 0 {
		return run, false, nil
	}
	events, err := stack.Ports.EventReader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return run, false, err
	}
	recovered := run
	changed := false
	for _, event := range events {
		if event.EventType != orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0 ||
			strings.TrimSpace(event.RunID) != strings.TrimSpace(run.RunID) {
			continue
		}
		taskRef, ok := codexStackMicrotaskCreatedEventTaskIDV0(event)
		if !ok || !refs[taskRef] || codexStackStringInSetV0(recovered.Tasks, taskRef) {
			continue
		}
		next, err := codexStackProjectDurableMicrotaskCreatedEventV0(recovered, event)
		if err != nil {
			return run, false, err
		}
		recovered = next
		changed = true
	}
	if !changed {
		return run, false, nil
	}
	if err := stack.Ports.RunStore.SaveRunV0(ctx, recovered); err != nil {
		return run, false, err
	}
	return recovered, true, nil
}

func codexStackProjectDurableMicrotaskCreatedEventV0(
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

func codexStackMicrotaskCreatedEventTaskIDV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (string, bool) {
	var payload orquestacoreworkflow.MicrotaskCreatedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", false
	}
	taskRef := strings.TrimSpace(payload.TaskID)
	return taskRef, taskRef != ""
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

func (stack StackV0) recoverBlockedDomainWorkOpenReviewRunV0(
	ctx context.Context,
	runRef string,
	correlationID string,
	occurredAt string,
) (bool, error) {
	if stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		stack.Stores.TaskStore == nil {
		return false, nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return false, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		return false, nil
	}
	recoverable, retained := codexStackSplitRecoverableDomainWorkOpenReviewBlockersV0(run.Blockers)
	if len(recoverable) == 0 || len(retained) > 0 {
		return false, nil
	}
	if drainRunHasPendingExternalAgentsV0(run, nil) {
		return false, nil
	}
	hasAcceptedSubmissions, err := stack.blockedDomainWorkOpenReviewHasAcceptedSubmissionsV0(ctx, run)
	if err != nil || !hasAcceptedSubmissions {
		return false, err
	}
	recovered := run
	for _, blocker := range recoverable {
		next, err := stack.resolveRunBlockerForDomainWorkOpenReviewRecoveryV0(
			ctx,
			recovered,
			blocker,
			correlationID,
			occurredAt,
		)
		if err != nil {
			return false, err
		}
		recovered = next
	}
	return true, nil
}

func (stack StackV0) recoverPartialDomainWorkReviewPhaseV0(
	ctx context.Context,
	runRef string,
	correlationID string,
	occurredAt string,
) (bool, error) {
	if stack.Ports.RunStore == nil ||
		stack.Ports.EventSink == nil ||
		stack.Stores.TaskStore == nil {
		return false, nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return false, err
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		len(compactStringsV0(run.Blockers)) > 0 ||
		!domainWorkReviewPhasePartiallyOpenV0(run) ||
		drainRunHasPendingExternalAgentsV0(run, nil) {
		return false, nil
	}
	hasAcceptedSubmissions, err := stack.blockedDomainWorkOpenReviewHasAcceptedSubmissionsV0(ctx, run)
	if err != nil || !hasAcceptedSubmissions {
		return false, err
	}
	repaired, err := stack.openRevisionPhaseForDomainWorkReviewRecoveryV0(ctx, run, correlationID, occurredAt)
	if err != nil {
		return false, err
	}
	return repaired.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseRevisionV0, nil
}

func (stack StackV0) openRevisionPhaseForDomainWorkReviewRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(run.RunID)
	open, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-domain-work-review-phase-repair-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-domain-work-review-phase-repair-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-domain-work-review-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "domain-work-review-phase-partial-projection-recovery",
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

func domainWorkReviewPhasePartiallyOpenV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if len(compactStringsV0(run.Reviews)) == 0 {
		return false
	}
	for _, phase := range run.Phases {
		if phase.ID != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
			continue
		}
		return phase.Status != orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 &&
			strings.TrimSpace(phase.OpenedAt) != ""
	}
	return false
}

func (stack StackV0) resolveRunBlockerForDomainWorkOpenReviewRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	blocker string,
	correlationID string,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	safe := codexStackOperationalClosureSafeRefV0(blocker)
	resolve, err := orquestacoreworkflow.NewResolveRunBlockerCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-domain-work-open-review-resolve-" + safe,
			RunID:          run.RunID,
			IdempotencyKey: "idem-domain-work-open-review-resolve-" + safe,
			CorrelationID:  strings.TrimSpace(correlationID),
			RequestedBy:    "orquesta-app-codex-stack-domain-work-review-reconciler",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(occurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.ResolveRunBlockerCommandPayloadV0{
			BlockerID:    blocker,
			ReasonCode:   "domain_work_open_review_recovered",
			Summary:      "Resolver bloqueo tecnico recuperable de apertura de revision con entrega de dominio aceptada.",
			EvidenceRefs: []string{"evidence-ref-domain-work-open-review-blocker-recovered"},
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

func (stack StackV0) blockedDomainWorkOpenReviewHasAcceptedSubmissionsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	records, err := stack.domainWorkSubmissionRecordsForRunV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	if len(records) == 0 {
		return false, nil
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return false, err
	}
	deliveredDomainWorkTasks := 0
	for _, task := range tasks {
		if !workflowTaskHasDomainWorkContractV0(task) ||
			!codexStackStringInSetV0(run.DeliveredTasks, task.TaskID) ||
			codexStackStringInSetV0(run.ClosedTasks, task.TaskID) {
			continue
		}
		deliveredDomainWorkTasks++
		if !domainWorkTaskHasAcceptedSubmissionForAnyDeliveryV0(records, run, task.TaskID) {
			return false, nil
		}
	}
	return deliveredDomainWorkTasks > 0, nil
}

func domainWorkTaskHasAcceptedSubmissionForAnyDeliveryV0(
	records []DomainWorkArtifactSubmissionRecordV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	for _, deliveryRef := range run.Deliveries {
		if domainWorkSubmissionRecordsContainAcceptedReceiptV0(records, run.RunID, taskRef, deliveryRef) {
			return true
		}
	}
	return false
}

func codexStackSplitRecoverableDomainWorkOpenReviewBlockersV0(
	blockers []string,
) ([]string, []string) {
	recoverable := make([]string, 0, len(blockers))
	retained := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		trimmed := strings.TrimSpace(blocker)
		if trimmed == "" {
			continue
		}
		if codexStackRecoverableDomainWorkOpenReviewBlockerV0(trimmed) {
			recoverable = append(recoverable, trimmed)
			continue
		}
		retained = append(retained, trimmed)
	}
	return recoverable, retained
}

func codexStackRecoverableDomainWorkOpenReviewBlockerV0(blocker string) bool {
	normalized := strings.ToLower(strings.TrimSpace(blocker))
	return strings.HasPrefix(normalized, "app-director-decision-") &&
		strings.Contains(normalized, "director-decision-apply-error") &&
		strings.Contains(normalized, "app-change-open-review")
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
		(strings.Contains(normalized, "app-change-task") ||
			strings.Contains(normalized, "app-change-open-plan"))
}

func codexStackHasRecoverableStaleOpenPlanAfterAcceptedReviewV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	blockers []string,
) bool {
	if drainRunHasPendingExternalAgentsV0(run, nil) ||
		len(compactStringsV0(run.Deliveries)) == 0 ||
		len(compactStringsV0(run.DeliveredTasks)) == 0 ||
		len(compactStringsV0(run.AcceptedReviews)) == 0 {
		return false
	}
	for _, blocker := range blockers {
		normalized := strings.ToLower(strings.TrimSpace(blocker))
		if strings.Contains(normalized, "app-change-open-plan") {
			return true
		}
	}
	return false
}

func codexStackRecoverableAppChangeAutoPlanTaskRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []string {
	refs := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		if decision.CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 ||
			decision.CreateMicrotask == nil {
			continue
		}
		task := decision.CreateMicrotask.Task
		if strings.TrimSpace(task.RunID) == strings.TrimSpace(run.RunID) &&
			strings.HasPrefix(strings.TrimSpace(task.TaskID), "task-ref-app-change-") &&
			!codexStackStringInSetV0(run.Tasks, task.TaskID) {
			refs = append(refs, task.TaskID)
		}
	}
	return compactStringsV0(refs)
}
