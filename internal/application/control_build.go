package application

import (
	"context"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) buildInitialControl(
	ctx context.Context,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	request ControlRequest,
	state *ApplyControlState,
) error {
	var err error
	policy := orchestrator.budgetPolicy.effectPolicy()
	if (request.Operation == ControlStop || request.Operation == ControlCancel) && !legacyGovernanceRecord(record) {
		policy, err = historicalEffectPolicy(record)
		if err != nil {
			return err
		}
	}
	switch request.Operation {
	case ControlPause, ControlResume:
		paused := request.Operation == ControlPause
		if request.Target == ControlTargetGoal {
			state.Goal, err = record.Goal.SetPaused(record.Goal.Revision(), paused, state.OperationAt)
		} else {
			state.Goal, err = record.Goal.SetWorkItemPaused(
				record.Goal.Revision(), item.Revision(), item.Ref(), paused, state.OperationAt,
			)
		}
		if err == nil && !paused {
			var scheduledEvents []EventRecord
			state.Executions, state.NewActions, scheduledEvents, err = orchestrator.scheduleHistoricalReady(
				ctx, record, state.Goal, record.Executions, state.OperationAt,
			)
			state.Events = append(state.Events, scheduledEvents...)
		}
		confirmLocalControl(&state.Control, state.OperationAt)
	case ControlStop:
		capabilities, capabilityErr := orchestrator.controller.ControlCapabilities(ctx)
		if capabilityErr != nil {
			return capabilityErr
		}
		if !ports.SupportsAgentStopMode(capabilities, request.Mode) {
			return errors.New("application.control_stop_unsupported")
		}
		if execution.State != ExecutionDispatching && execution.State != ExecutionRunning {
			return &StateError{Code: StateConflict}
		}
		state.Goal, err = record.Goal.AdvanceControl(record.Goal.Revision(), state.OperationAt)
		if err == nil {
			var action ActionRecord
			action, err = orchestrator.stopAction(policy, state.Control, state.Goal, item, execution, state.OperationAt)
			state.NewActions = []ActionRecord{action}
		}
	case ControlCancel:
		err = orchestrator.buildCancelControl(ctx, record, item, request, policy, state)
	case ControlRetry:
		err = orchestrator.buildRetryControl(ctx, record, item, execution, state)
	default:
		err = errors.New("application.control_operation_invalid")
	}
	if err != nil {
		return err
	}
	state.Events = append(
		controlEvents(record.Goal, state.Goal, state.Control, state.Executions, state.OperationAt),
		state.Events...,
	)
	return nil
}

func (orchestrator *Orchestrator) buildCancelControl(ctx context.Context, record GoalRecord, item goal.WorkItem, request ControlRequest, policy effectPolicySnapshot, state *ApplyControlState) error {
	var err error
	state.Goal, err = orchestrator.cancelRequest(record.Goal, item, request, state)
	if err != nil {
		return err
	}
	if len(cancelExecutions(record.Executions, request.Target, request.WorkItemRef)) > 0 {
		capabilities, capabilityErr := orchestrator.controller.ControlCapabilities(ctx)
		if capabilityErr != nil {
			return capabilityErr
		}
		mode, supported := cancelStopMode(capabilities)
		if !supported {
			return errors.New("application.control_cancel_unsupported")
		}
		state.Control.Mode = mode
	}
	for _, current := range record.Executions {
		if request.Target == ControlTargetWorkItem && current.WorkItemRef != request.WorkItemRef {
			continue
		}
		updatedItem, found := state.Goal.WorkItem(current.WorkItemRef)
		if !found {
			return &StateError{Code: StateConflict}
		}
		switch current.State {
		case ExecutionQueued, ExecutionAwaitingCommit, ExecutionAwaitingAttestation, ExecutionAwaitingIntegration:
			previousState := current.State
			current.State = ExecutionCanceled
			current.FinishedAt = state.OperationAt
			state.Executions = append(state.Executions, current)
			switch previousState {
			case ExecutionQueued:
				initialRef := "action:launch:" + current.Ref.String()
				if current.ExecutionWorkspaceRef.String() != "" {
					initialRef = "action:prepare-workspace:" + current.Ref.String()
				}
				state.RetireActionRefs = append(state.RetireActionRefs, initialRef)
			case ExecutionAwaitingCommit:
				state.RetireActionRefs = append(state.RetireActionRefs, "action:commit-change:"+current.Ref.String())
			case ExecutionAwaitingAttestation:
				state.RetireActionRefs = append(state.RetireActionRefs, "action:attest-test:"+current.Ref.String())
			case ExecutionAwaitingIntegration:
				integrationRefs, integrationErr := integrationActionRefsForCancellation(record, current)
				if integrationErr != nil {
					return integrationErr
				}
				state.RetireActionRefs = append(state.RetireActionRefs, integrationRefs...)
			}
		case ExecutionDispatching, ExecutionRunning:
			action, actionErr := orchestrator.stopAction(policy, state.Control, state.Goal, updatedItem, current, state.OperationAt)
			if actionErr != nil {
				return actionErr
			}
			state.NewActions = append(state.NewActions, action)
		}
	}
	if len(state.NewActions) == 0 {
		if request.Target == ControlTargetGoal {
			state.Goal, err = state.Goal.CompleteCancel(state.Goal.Revision(), state.OperationAt)
		} else if outcome, closable := state.Goal.ClosableOutcome(); closable {
			state.Goal, err = state.Goal.Close(state.Goal.Revision(), outcome, state.OperationAt)
		}
		if err != nil {
			return err
		}
		confirmLocalControl(&state.Control, state.OperationAt)
	}
	existing := replaceExecutions(record.Executions, state.Executions)
	newExecutions, newActions, scheduledEvents, scheduleErr := orchestrator.scheduleReady(ctx, state.Goal, existing, record.WorkItemAuthorities, policy, state.OperationAt)
	if scheduleErr != nil {
		return scheduleErr
	}
	state.Executions = append(state.Executions, newExecutions...)
	state.NewActions = append(state.NewActions, newActions...)
	state.Events = append(state.Events, scheduledEvents...)
	return nil
}

func replaceExecutions(existing, updates []ExecutionRecord) []ExecutionRecord {
	replaced := append([]ExecutionRecord(nil), existing...)
	for _, update := range updates {
		replaced = replaceExecution(replaced, update)
	}
	return replaced
}

func integrationActionRefsForCancellation(record GoalRecord, execution ExecutionRecord) ([]string, error) {
	terminal := make(map[string]struct{}, len(record.ConsumptionReceipts))
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == ActionIntegrateChange && receipt.GoalRef == execution.GoalRef &&
			receipt.WorkItemRef == execution.WorkItemRef && receipt.ExecutionRef == execution.Ref &&
			receipt.PlanGeneration == execution.PlanGeneration &&
			(receipt.Outcome == ActionConsumedCompleted || receipt.Outcome == ActionConsumedQuarantined) {
			terminal[receipt.ActionRef] = struct{}{}
		}
	}

	seen := make(map[string]struct{})
	refs := make([]string, 0, 1)
	for _, intent := range record.EffectIntents {
		subject := intent.Subject
		if intent.ActionKind != ActionIntegrateChange || subject.ProjectRef != record.Goal.Project() ||
			subject.GoalRef != execution.GoalRef || subject.GoalRef != record.Goal.Ref() ||
			subject.WorkItemRef != execution.WorkItemRef || subject.ExecutionRef != execution.Ref ||
			subject.PlanGeneration != execution.PlanGeneration ||
			subject.AppSpecGeneration != execution.AppSpecGeneration || subject.SpecHash != execution.SpecHash ||
			subject.ActorRef != record.Goal.Actor() {
			continue
		}
		if err := ValidateEffectIntent(intent); err != nil {
			return nil, &StateError{Code: StateInvalid, Cause: err}
		}
		if _, consumed := terminal[intent.ActionRef]; consumed {
			continue
		}
		if _, duplicate := seen[intent.ActionRef]; duplicate {
			continue
		}
		seen[intent.ActionRef] = struct{}{}
		refs = append(refs, intent.ActionRef)
	}
	return refs, nil
}

func (orchestrator *Orchestrator) cancelRequest(
	current goal.Goal, item goal.WorkItem, request ControlRequest, state *ApplyControlState,
) (goal.Goal, error) {
	if request.Target == ControlTargetGoal {
		return current.RequestCancel(current.Revision(), state.OperationAt)
	}
	return current.RequestWorkItemCancel(current.Revision(), item.Revision(), item.Ref(), state.OperationAt)
}

func (orchestrator *Orchestrator) buildRetryControl(
	ctx context.Context,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	state *ApplyControlState,
) error {
	paused, found := record.Goal.EffectivePause(item.Ref())
	if !found || paused || record.Goal.IsTerminal() || item.State() != goal.WorkItemStateInterrupted ||
		execution.State != ExecutionStopped || execution.RecipientMailboxRetired ||
		execution.AttemptNo >= execution.MaxExecutionAttempts {
		return &StateError{Code: StateConflict}
	}
	authority, found := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !found {
		if len(record.WorkItemAuthorities) == 0 {
			return errors.New("governance.legacy_reauthorization_required")
		}
		return errors.New("application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	replacementRef, err := newExecutionRef(ctx, orchestrator.ids)
	if err != nil {
		return err
	}
	state.Goal, err = record.Goal.RetryWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref, replacementRef, state.OperationAt,
	)
	if err != nil {
		return err
	}
	replacement := ExecutionRecord{
		Ref: replacementRef, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: state.Goal.PlanGeneration(),
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		State: ExecutionQueued, ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey: "execution:" + replacementRef.String(), MaxOutputBytes: execution.MaxOutputBytes,
		CreatedAt: state.OperationAt,
	}
	if len(item.WriteSet()) != 0 {
		workspaceRef, workspaceErr := newExecutionWorkspaceRef(ctx, orchestrator.ids)
		if workspaceErr != nil {
			return workspaceErr
		}
		replacement.RepositoryRef = execution.RepositoryRef
		if replacement.RepositoryRef.String() == "" {
			replacement.RepositoryRef, err = orchestrator.state.ProjectRepository(ctx, state.Goal.Project())
			if err != nil {
				return err
			}
		}
		replacement.ExecutionWorkspaceRef = workspaceRef
	}
	updatedItem, _ := state.Goal.WorkItem(item.Ref())
	state.Executions = []ExecutionRecord{replacement}
	var action ActionRecord
	if len(updatedItem.WriteSet()) != 0 {
		action, err = orchestrator.prepareWorkspaceAction(
			policy, state.Goal, updatedItem, replacement, authority, state.OperationAt, state.OperationAt,
		)
	} else {
		action, err = orchestrator.launchAction(
			policy, state.Goal, updatedItem, replacement, authority, state.OperationAt, state.OperationAt,
		)
	}
	if err != nil {
		return err
	}
	state.NewActions = []ActionRecord{action}
	state.RequireMailboxClearForExecutionRef = execution.Ref
	confirmLocalControl(&state.Control, state.OperationAt)
	return nil
}

func exclusiveStopOwner(record GoalRecord, request ControlRequest) (ControlRecord, error) {
	var targets []ExecutionRecord
	switch request.Operation {
	case ControlStop:
		execution, found := executionByRef(record.Executions, request.ExecutionRef)
		if found && (execution.State == ExecutionDispatching || execution.State == ExecutionRunning) {
			targets = []ExecutionRecord{execution}
		}
	case ControlCancel:
		targets = cancelExecutions(record.Executions, request.Target, request.WorkItemRef)
	default:
		return ControlRecord{}, nil
	}
	var superseded ControlRecord
	for _, control := range record.Controls {
		if control.Status != ControlRequested ||
			(control.Operation != ControlStop && control.Operation != ControlCancel) {
			continue
		}
		for _, execution := range targets {
			if pendingControlOwnsExecution(control, execution) {
				eligible := request.Operation == ControlStop && request.Target == ControlTargetExecution &&
					request.Mode == ports.AgentStopForced && control.Operation == ControlStop &&
					control.Target == ControlTargetExecution && control.Mode == ports.AgentStopCooperative &&
					control.ExecutionRef == request.ExecutionRef &&
					control.WorkItemRef == request.WorkItemRef &&
					control.ExecutionAttempt == request.ExpectedExecutionAttempt
				if !eligible || superseded.Ref != "" {
					return ControlRecord{}, &StateError{Code: StateConflict}
				}
				superseded = control
			}
		}
	}
	return superseded, nil
}

func pendingControlOwnsExecution(control ControlRecord, execution ExecutionRecord) bool {
	if control.Operation == ControlStop {
		return control.ExecutionRef == execution.Ref
	}
	return control.Target == ControlTargetGoal ||
		(control.Target == ControlTargetWorkItem && control.WorkItemRef == execution.WorkItemRef)
}

func validateControlFences(
	record GoalRecord,
	projectRef goal.ProjectRef,
	request ControlRequest,
) (goal.WorkItem, ExecutionRecord, error) {
	if record.Goal.Project() != projectRef {
		return goal.WorkItem{}, ExecutionRecord{}, &StateError{Code: StateNotFound}
	}
	if record.Goal.Ref() != request.GoalRef || record.Goal.Revision() != request.ExpectedGoalRevision ||
		record.Goal.PlanGeneration() != request.ExpectedPlanGeneration ||
		record.Goal.AppSpec().Generation() != request.ExpectedAppSpecGeneration ||
		record.Goal.SpecHash() != request.ExpectedSpecHash || record.Goal.IsTerminal() {
		return goal.WorkItem{}, ExecutionRecord{}, &StateError{Code: StateConflict}
	}
	var item goal.WorkItem
	if request.WorkItemRef.String() != "" {
		var found bool
		item, found = record.Goal.WorkItem(request.WorkItemRef)
		if !found || item.Revision() != request.ExpectedWorkItemRevision {
			return goal.WorkItem{}, ExecutionRecord{}, &StateError{Code: StateConflict}
		}
	}
	var execution ExecutionRecord
	if request.ExecutionRef.String() != "" {
		var found bool
		for _, candidate := range record.Executions {
			if candidate.Ref == request.ExecutionRef {
				execution, found = candidate, true
				break
			}
		}
		bound, hasBinding := item.Execution()
		if !found || execution.GoalRef != request.GoalRef || execution.WorkItemRef != request.WorkItemRef ||
			execution.AttemptNo != request.ExpectedExecutionAttempt || !hasBinding || bound != execution.Ref {
			return goal.WorkItem{}, ExecutionRecord{}, &StateError{Code: StateConflict}
		}
	}
	return item, execution, nil
}
