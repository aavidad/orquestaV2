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
			state.Executions, state.NewActions, scheduledEvents, err = orchestrator.scheduleReady(
				ctx, state.Goal, record.Executions, state.OperationAt,
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
			state.NewActions = []ActionRecord{
				stopAction(state.Control.Ref, state.Goal, item, execution, state.OperationAt),
			}
		}
	case ControlCancel:
		err = orchestrator.buildCancelControl(ctx, record, item, request, state)
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

func (orchestrator *Orchestrator) buildCancelControl(
	ctx context.Context,
	record GoalRecord,
	item goal.WorkItem,
	request ControlRequest,
	state *ApplyControlState,
) error {
	var err error
	if request.Target == ControlTargetGoal {
		state.Goal, err = record.Goal.RequestCancel(record.Goal.Revision(), state.OperationAt)
	} else {
		state.Goal, err = record.Goal.RequestWorkItemCancel(
			record.Goal.Revision(), item.Revision(), item.Ref(), state.OperationAt,
		)
	}
	if err != nil {
		return err
	}
	live := cancelExecutions(record.Executions, request.Target, request.WorkItemRef)
	if len(live) > 0 {
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
		case ExecutionQueued:
			current.State = ExecutionCanceled
			current.FinishedAt = state.OperationAt
			state.Executions = append(state.Executions, current)
			state.RetireActionRefs = append(state.RetireActionRefs, "action:launch:"+current.Ref.String())
		case ExecutionDispatching, ExecutionRunning:
			state.NewActions = append(state.NewActions, stopAction(
				state.Control.Ref, state.Goal, updatedItem, current, state.OperationAt,
			))
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
	existing := append([]ExecutionRecord(nil), record.Executions...)
	for _, updated := range state.Executions {
		existing = replaceExecution(existing, updated)
	}
	newExecutions, newActions, scheduledEvents, scheduleErr := orchestrator.scheduleReady(
		ctx, state.Goal, existing, state.OperationAt,
	)
	if scheduleErr != nil {
		return scheduleErr
	}
	state.Executions = append(state.Executions, newExecutions...)
	state.NewActions = append(state.NewActions, newActions...)
	state.Events = append(state.Events, scheduledEvents...)
	return nil
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
		// Retry only queues a fresh provider attempt. RecordLaunchPrepared is the
		// single durable frontier that may move it to dispatching after a worker
		// has claimed the launch and revalidated the effective pause gate.
		State: ExecutionQueued, ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey: "execution:" + replacementRef.String(), MaxOutputBytes: execution.MaxOutputBytes,
		CreatedAt: state.OperationAt,
	}
	updatedItem, _ := state.Goal.WorkItem(item.Ref())
	state.Executions = []ExecutionRecord{replacement}
	state.NewActions = []ActionRecord{{
		Ref: "action:launch:" + replacement.Ref.String(), Kind: ActionLaunchAgent,
		GoalRef: state.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacement.Ref,
		PlanGeneration: replacement.PlanGeneration, WorkItemGeneration: updatedItem.Revision(),
		AvailableAt: state.OperationAt,
	}}
	state.RequireMailboxClearForExecutionRef = execution.Ref
	confirmLocalControl(&state.Control, state.OperationAt)
	return nil
}

// exclusiveStopOwner keeps one effect owner per live Execution. The only
// transfer allowed is cooperative Stop -> forced Stop for the same exact
// Execution; cancel ownership and forced ownership are never preempted.
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
